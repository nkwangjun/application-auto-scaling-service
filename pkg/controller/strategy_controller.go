package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nanto.io/application-auto-scaling-service/pkg/apis/autoscaling/v1alpha1"
	"nanto.io/application-auto-scaling-service/pkg/config"
	"nanto.io/application-auto-scaling-service/pkg/k8sclient"
	"nanto.io/application-auto-scaling-service/pkg/utils"
	"nanto.io/application-auto-scaling-service/pkg/utils/cronutil"
	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

const (
	NamespaceDefault = "default"
)

var logger = logutil.GetLogger()

type StrategyController struct {
	// 策略来源
	StrategySource string
	// 策略本地文件路径
	LocalPath string
}

func NewStrategyController(conf *config.StrategyConf) *StrategyController {
	return &StrategyController{
		StrategySource: conf.Source,
		LocalPath:      conf.LocalPath,
	}
}

// todo 这里逻辑后面会改，先按最简单，定时修改，中间状态不维护
// 后面周期修改 or watch 到 hpa 策略被修改后，强制覆盖
func (s *StrategyController) SyncStrategyToCCE(ctx context.Context, cancel context.CancelFunc) {
	// 从本地文件获取策略
	strategiesInfo, err := getLocalStrategies(s.LocalPath)
	if err != nil {
		logger.Errorf("Get local strategies err: %+v", err)
		cancel()
		return
	}

	// 校验目标 CCE HPA 是否存在
	targetHPA := strategiesInfo.TargetHPA
	if err = checkRefCustomedHPA(targetHPA); err != nil {
		logger.Errorf("Check ref customed hpa[%s] err: %+v", targetHPA, err)
		cancel()
		return
	}

	// 注册、启动定时任务
	cronutil.InitCron()
	for _, s := range strategiesInfo.Strategies {
		// todo 后面可以考虑将 valid_time 验证与 生成cron 逻辑分开
		cronSpec, err := genStartTimeSpec(s.ValidTime)
		if err != nil {
			logger.Errorf("Generate start time spec err for valid time[%s]", s.ValidTime)
			// 这里不能忽略这个错而 continue
			continue
		}
		_, err = cronutil.GetCron().AddFunc(cronSpec, genCronFunc(targetHPA, s.Spec))
		if err != nil {
			logger.Errorf("Add cron func err: %v", err)
			return
		}
		logger.Infof("Add cron task success, cron spec[%s]", cronSpec)
	}
	cronutil.GetCron().Start()

	// 更新执行当前策略
	jobExecNow, err := cronutil.FindJobNeedExecNow()
	if err != nil {
		logger.Errorf("FindJobNeedExecNow err: %+v", err)
		cancel()
		return
	}
	jobExecNow.Run()

	<-ctx.Done()
	cronutil.GetCron().Stop()
	logger.Info("=== Strategies controller exit ===")
}

func getAllCustomedHPAName() ([]string, error) {
	chpas, err := k8sclient.GetCrdClientSet().AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
		List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get cce hpa err")
	}
	chpaNames := []string{}
	for _, chpa := range chpas.Items {
		chpaNames = append(chpaNames, chpa.Name)
	}
	return chpaNames, nil
}

func checkRefCustomedHPA(chpaName string) error {
	chpaNames, err := getAllCustomedHPAName()
	if err != nil {
		return err
	}
	if !utils.IsInStrSlice(chpaNames, chpaName) {
		return errors.Errorf("Customed HPA[%s] is not exist, current customed hpas include %v",
			chpaName, chpaNames)
	}
	return nil
}

func genCronFunc(targetHPA string, newSpec v1alpha1.CustomedHorizontalPodAutoscalerSpec) cron.FuncJob {
	return func() {
		ctx := context.Background()
		curHpa, err := k8sclient.GetCrdClientSet().AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
			Get(ctx, targetHPA, metav1.GetOptions{})
		if err != nil {
			logger.Errorf("Get current customHPA err: %v", err)
			return
		}

		newSpec.ScaleTargetRef = curHpa.Spec.ScaleTargetRef
		newSpec.DeepCopyInto(&curHpa.Spec)

		update, err := k8sclient.GetCrdClientSet().AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
			Update(ctx, curHpa, metav1.UpdateOptions{})
		if err != nil {
			logger.Errorf("Update CustomHPA err: %v", err)
			return
		}

		bytes, err := json.Marshal(update.Spec)
		if err != nil {
			logger.Fatalf("Marshal hpa spec[%+v] err: %v", update.Spec, err)
		}
		logger.Infof("Update HPA success, current HPA info: %s", bytes)
	}
}

// genStartTimeSpec 生成策略生效起始时间的 cron 表达式
func genStartTimeSpec(validTime string) (string, error) {
	// todo 正则表达式改写
	// validTime例子: 0:00-09:30
	times := strings.Split(validTime, "-")
	if len(times) != 2 {
		return "", errors.Errorf("illegal validTime filed[%s]", validTime)
	}

	hourAndMinute := strings.Split(times[0], ":")
	if len(times) != 2 {
		return "", errors.Errorf("illegal validTime filed[%s]", validTime)
	}
	// todo hourAndMinute 元素内容也要判断下是否合法
	cronSpec := fmt.Sprintf("0 %s %s * * ?", hourAndMinute[1], hourAndMinute[0])
	return cronSpec, nil
}

func getLocalStrategies(path string) (*StrategiesInfo, error) {
	var (
		bytes []byte
		err   error
	)

	if bytes, err = ioutil.ReadFile(path); err != nil {
		return nil, errors.Wrap(err, "read local strategies file err")
	}

	info := &StrategiesInfo{}
	if err = yaml.Unmarshal(bytes, &info); err != nil {
		return nil, errors.Wrapf(err, "yaml unmarshal err, file content: %s", bytes)
	}
	if err = info.CheckAndCompleteInfo(); err != nil {
		return nil, errors.Wrap(err, "check strategies info err")
	}

	bytes, err = json.Marshal(info)
	if err != nil {
		logger.Panicf("Marshal StrategiesInfo err: %v", err)
	}
	logger.Infof("Read strategies from local file, detail: %s", bytes)

	return info, nil
}
