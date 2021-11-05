package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/pkg/apis/autoscaling/v1alpha1"
	batchv1 "nanto.io/application-auto-scaling-service/pkg/apis/batch/v1"
	clientset "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned"
)

const (
	OperationTypeScaleUp   = "ScaleUp"
	OperationTypeScaleDown = "ScaleDown"
	RuleTypeMetric         = "Metric"
	MetricOptScaleUp       = ">"
	MetricOptScaleDown     = "<"
)

var task *TaskCustomHPA

type TaskCustomHPA struct {
	forecastTaskObj      *batchv1.ForecastTask
	forecastWindowMinute int32
	RefNames             []string
	kubeClientset        kubernetes.Interface
	crdClientset         clientset.Interface
	recorder             record.EventRecorder
	cancelFunc           context.CancelFunc
}

func RunTaskCustomHPA(forecastTask *batchv1.ForecastTask, kubeClientset kubernetes.Interface, crdClientset clientset.Interface,
	recorder record.EventRecorder) {
	ctx, cancel := context.WithCancel(context.Background())
	task = &TaskCustomHPA{
		forecastTaskObj:      forecastTask,
		RefNames:             []string{forecastTask.Spec.ScaleTargetRefs[0].Name},
		forecastWindowMinute: *forecastTask.Spec.ForecastWindow,
		kubeClientset:        kubeClientset,
		crdClientset:         crdClientset,
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
			klog.Info("Task exit")
			return
		}
	}
}

func (t *TaskCustomHPA) stop() {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}

func (t *TaskCustomHPA) forecastAndUpdateHPA() {
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
	chpa, err := t.crdClientset.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
		Get(ctx, t.RefNames[0], metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get CustomHPA err: %v", err)
		return
	}

	modifyRule(&chpa.Spec.Rules[0], isScaleUp, metricValue, stepVal)
	update, err := t.crdClientset.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(NamespaceDefault).
		Update(ctx, chpa, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Update CustomHPA err: %v", err)
		return
	}

	t.recorder.Eventf(t.forecastTaskObj, corev1.EventTypeNormal, SuccessExecuted, MessagePredictionOperationExecuted,
		isScaleUp, metricValue, stepVal)
	klog.Infof("Update HPA success, current HPA info: %+v", update)
}

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
*/
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
