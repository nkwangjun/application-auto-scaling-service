package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nanto.io/application-auto-scaling-service/pkg/config"
	"nanto.io/application-auto-scaling-service/pkg/k8sclient"
	"nanto.io/application-auto-scaling-service/pkg/k8sclient/apis/autoscaling/v1alpha1"
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
	// 本地策略yaml文件 md5值
	localDataKey string
}

func NewStrategyController(conf *config.StrategyConf) *StrategyController {
	return &StrategyController{
		StrategySource: conf.Source,
		LocalPath:      conf.LocalPath,
	}
}

// todo 目前只有local策略
// Start 启动controller，修改cce的配置，并监听strategies.yaml策略文件修改
func (s *StrategyController) Start(ctx context.Context, cancel context.CancelFunc) {
	// 初始化定时任务
	cronutil.InitCron()
	defer cronutil.GetCron().Stop()

	// 执行当前配置的策略
	if err := s.execLocalStrategies(); err != nil {
		logger.Errorf("Exec local strategies err: %+v", err)
		cancel()
		return
	}

	// 监听策略配置文件的修改
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !s.isStrategiesFileModified() {
				logger.Info("local strategies is not modified")
				continue
			}
			logger.Info("local strategies is modified, refresh cron tasks")
			// 注销之前的定时修改任务
			cronutil.GetCron().Stop()
			cronutil.RemoveAllCronEntries()
			// 执行当前配置的策略
			if err := s.execLocalStrategies(); err != nil {
				logger.Errorf("Exec local strategies err: %+v", err)
				cancel()
				return
			}
		case <-ctx.Done():
			logger.Info("=== Strategies controller exit ===")
			return
		}
	}
}

// Start 启动controller，监听strategies.yaml策略文件修改；修改cce的配置
func (s *StrategyController) isStrategiesFileModified() bool {
	hashMd5, err := utils.FileHashMd5(s.LocalPath)
	if err != nil {
		logger.Panicf("Get file[%s] md5 err: %+v", s.LocalPath, err)
		return false
	}
	return s.localDataKey != hashMd5
}

// execLocalStrategies 执行本地策略：编排定时任务，更新当前时间段的策略
func (s *StrategyController) execLocalStrategies() error {
	var (
		strategiesInfo *StrategiesInfo
		jobExecNow     cron.Job
		cronSpec       string
		err            error
	)

	// 从本地文件获取策略
	if strategiesInfo, err = s.getLocalStrategies(s.LocalPath); err != nil {
		return err
	}

	// 校验目标 CCE HPA 是否存在
	if err = checkRefCustomedHPA(strategiesInfo.TargetHPA); err != nil {
		return err
	}

	// 编排、启动 cron任务
	for _, s := range strategiesInfo.Strategies {
		if cronSpec, err = genStartTimeSpec(s.ValidTime); err != nil {
			return err
		}
		if _, err = cronutil.GetCron().AddFunc(cronSpec, genCronFunc(strategiesInfo.TargetHPA, s.Spec)); err != nil {
			return errors.Wrap(err, "add cron func err")
		}
		logger.Infof("Add cron task success, cron spec[%s]", cronSpec)
	}
	cronutil.GetCron().Start()

	// 更新当前策略
	if jobExecNow, err = cronutil.FindJobNeedExecNow(); err != nil {
		return err
	}
	jobExecNow.Run()
	return nil
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
		return errors.Errorf("customed HPA[%s] is not exist, current customed hpas include %v",
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

		// 仅记录日志用
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

func (s *StrategyController) getLocalStrategies(path string) (*StrategiesInfo, error) {
	var (
		bytes []byte
		err   error
	)

	// 读取本地配置，记录md5
	if bytes, err = ioutil.ReadFile(path); err != nil {
		return nil, errors.Wrap(err, "read local strategies file err")
	}
	s.localDataKey = utils.DataHashMd5(bytes)

	// 反序列化、校验并补全参数
	info := &StrategiesInfo{}
	if err = yaml.Unmarshal(bytes, &info); err != nil {
		return nil, errors.Wrapf(err, "yaml unmarshal err, file content: %s", bytes)
	}
	if err = checkAndCompleteInfo(info); err != nil {
		return nil, errors.Wrap(err, "check strategies info err")
	}

	// 仅记录日志用
	bytes, err = json.Marshal(info)
	if err != nil {
		logger.Panicf("Marshal StrategiesInfo err: %v", err)
	}
	logger.Infof("Read strategies from local file, detail: %s", bytes)

	return info, nil
}
