package controller

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/pkg/apis/autoscaling/v1alpha1"
	batchv1 "nanto.io/application-auto-scaling-service/pkg/apis/batch/v1"
	clientset "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned"
	"nanto.io/application-auto-scaling-service/pkg/utils"
)

var task *TaskCustomHPA

type TaskCustomHPA struct {
	clusterId            string
	forecastTaskObj      *batchv1.ForecastTask
	forecastWindowMinute int32
	RefNames             []string
	kubeClientset        kubernetes.Interface
	crdClientset         clientset.Interface
	recorder             record.EventRecorder
	cancelFunc           context.CancelFunc
	// todo 后面放到syncer
	StrategiesCreateTime int64
}

func RunTaskCustomHPA(forecastTask *batchv1.ForecastTask, clusterId string, kubeClient kubernetes.Interface, crdClient clientset.Interface,
	recorder record.EventRecorder) {
	ctx, cancel := context.WithCancel(context.Background())
	task = &TaskCustomHPA{
		clusterId:            clusterId,
		forecastTaskObj:      forecastTask,
		RefNames:             []string{forecastTask.Spec.ScaleTargetRefs[0].Name},
		forecastWindowMinute: *forecastTask.Spec.ForecastWindow,
		kubeClientset:        kubeClient,
		crdClientset:         crdClient,
		recorder:             recorder,
		cancelFunc:           cancel,
	}
	go task.run(ctx)
}

func StopTaskCustomHPA(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.Info("Stop forecast task:", key)
	task.stop()
	if task != nil && task.cancelFunc != nil {
		task.cancelFunc()
	}
}

func (t *TaskCustomHPA) run(ctx context.Context) {
	t.forecastAndUpdateHPA()
	ticker := time.NewTicker(time.Duration(t.forecastWindowMinute) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go t.forecastAndUpdateHPA()
		case <-ctx.Done():
			klog.Info("=== Task exit ===")
			return
		}
	}
}

func (t *TaskCustomHPA) stop() {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}

type StrategiesInfo struct {
	ClusterId  string `json:"cluster_id"`
	CreateTime int64  `json:"create_time"` // 策略创建时间，时间戳（毫秒）
	Strategies []Strategy
}

type Strategy struct {
	StartTime string                                       `json:"start_time"`
	Spec      v1alpha1.CustomedHorizontalPodAutoscalerSpec `json:"spec"`
}

func (t *TaskCustomHPA) forecastAndUpdateHPA() {
	// 从 天策 获取扩缩容策略
	strategiesBytes, err := utils.GetStrategiesFromTianCe(t.clusterId)
	if err != nil {
		klog.Errorf("GetStrategiesFromTianCe err: %+v", err)
		// todo 记录 event
		return
	}
	strategiesInfo := &StrategiesInfo{}
	err = json.Unmarshal(strategiesBytes, strategiesInfo)
	if err != nil {
		klog.Errorf("Unmarshal strategiesBytes err: %v", err)
		return
	}
	klog.Infof("Strategies info: %+v", strategiesInfo)

	if strategiesInfo.CreateTime == t.StrategiesCreateTime {
		klog.Infof("No change in strategies")
		return
	}

	// 刷新定时任务
	GetCron().Stop()
	RemoveAllCronEntries()
	for _, s := range strategiesInfo.Strategies {
		_, err := GetCron().AddFunc(s.StartTime, genCronFunc(t, s.Spec))
		if err != nil {
			klog.Errorf("jobCron add func err: %v", err)
			return
		}
	}
	GetCron().Start()
	// 更新执行当前策略
	jobExecNow, err := FindJobNeedExecNow()
	if err != nil {
		klog.Errorf("FindJobNeedExecNow err: %+v", err)
		return
	}
	jobExecNow.Run()

	// 更新策略时间戳记录
	t.StrategiesCreateTime = strategiesInfo.CreateTime
}

func genCronFunc(t *TaskCustomHPA, newSpec v1alpha1.CustomedHorizontalPodAutoscalerSpec) cron.FuncJob {
	return func() {
		ctx := context.Background()
		curHpa, err := t.crdClientset.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
			Get(ctx, t.RefNames[0], metav1.GetOptions{})
		if err != nil {
			t.recorder.Eventf(t.forecastTaskObj, corev1.EventTypeWarning, CronTaskError,
				"Get current customHPA err: %v", err)
			klog.Errorf("Get current customHPA err: %v", err)
			return
		}

		newSpec.ScaleTargetRef = curHpa.Spec.ScaleTargetRef
		newSpec.DeepCopyInto(&curHpa.Spec)

		update, err := t.crdClientset.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
			Update(ctx, curHpa, metav1.UpdateOptions{})
		if err != nil {
			t.recorder.Eventf(t.forecastTaskObj, corev1.EventTypeWarning, CronTaskError,
				"Update CustomHPA err: %v", err)
			klog.Errorf("Update CustomHPA err: %v", err)
			return
		}

		bytes, err := json.Marshal(update.Spec)
		if err != nil {
			klog.Fatalf("Marshal hpa spec[%+v] err: %v", update.Spec, err)
		}
		t.recorder.Eventf(t.forecastTaskObj, corev1.EventTypeNormal, SuccessRefreshStrategies,
			MessageStrategiesRefreshedFmt, bytes)
		klog.Infof("Update HPA success, current HPA info: %+v", update.Spec)
	}
}

/*
const (
	OperationTypeScaleUp   = "ScaleUp"
	OperationTypeScaleDown = "ScaleDown"
	RuleTypeMetric         = "Metric"
	MetricOptScaleUp       = ">"
	MetricOptScaleDown     = "<"
)
	// todo 请求天策预测接口
	isScaleUp := time.Now().Second() > 30
	if isScaleUp {
		klog.Info("=== 预测趋势：扩容")
	} else {
		klog.Info("=== 预测趋势：缩容")
	}
	metricValue := float32(rand.IntnRange(1, 99)) / 100
	klog.Info("=== 预测指标 metricValue：", metricValue)
	stepVal := int32(rand.IntnRange(1, 6))
	klog.Info("=== 预测步长 stepVal：", stepVal)

	ctx := context.Background()
	chpa, err := t.crdClient.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
		Get(ctx, t.RefNames[0], metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get CustomHPA err: %v", err)
		return
	}

	// 修改特定字段
	//modifyRule(&chpa.Spec.Rules[0], isScaleUp, metricValue, stepVal)

	update, err := t.crdClient.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
		Update(ctx, chpa, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Update CustomHPA err: %v", err)
		return
	}

	t.recorder.Eventf(t.forecastTaskObj, corev1.EventTypeNormal, SuccessExecuted, MessageStrategiesRefreshedFmt,
		isScaleUp, metricValue, stepVal)
	klog.Infof("Update HPA success, current HPA info: %+v", update)
*/
/*
	rule example:
	v1alpha1.Rule{
		Actions: []v1alpha1.Action{
			{
				MetricRange:   fmt.Sprintf("%.2f,+Infinity", metricValue),
				OperationType: OperationTypeScaleUp,
				//OperationUnit:  "Task",
				OperationValue: &stepVal,
			},
		},
		Disable: &ruleDisableFalse,
		MetricTrigger: v1alpha1.MetricTrigger{
			//HitThreshold:    1,
			//MetricName:      "CPURatioToRequest",
			MetricOperation: MetricOptScaleUp,
			MetricValue:     &metricValue,
			//PeriodSeconds:   60,
			//Statistic:       "instantaneous",
		},
		//RuleName: "up",
		//RuleType: "Metric",
	}

		// 修改特定字段
		//modifyRule(&oldHpa.Spec.Rules[0], isScaleUp, metricValue, stepVal)

func modifyRule(rule *v1alpha1.Rule, isScaleUp bool, metricValue float32, stepVal int32) {
	if isScaleUp {
		rule.Actions[0].MetricRange = fmt.Sprintf("%.2f,+Infinity", metricValue)
		rule.Actions[0].OperationType = OperationTypeScaleUp
		rule.MetricTrigger.MetricOperation = MetricOptScaleUp
	} else {
		rule.Actions[0].MetricRange = fmt.Sprintf("0.00,%.2f", metricValue)
		rule.Actions[0].OperationType = OperationTypeScaleDown
		rule.MetricTrigger.MetricOperation = MetricOptScaleDown
	}

	rule.Actions[0].OperationValue = &stepVal
	rule.MetricTrigger.MetricValue = &metricValue
}
*/
