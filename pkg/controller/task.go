package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	v1 "k8s.io/application-aware-controller/pkg/apis/appawarecontroller/v1"

	"k8s.io/client-go/tools/record"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Task interface {
	ID() string
	Name() string
	SetID(id string)
	Equals(task Task) bool
	Run() (msg string, err error)
}

/*type TargetRef struct {
	RefName      string
	RefNamespace string
	//RefKind      string
	//RefGroup     string
	//RefVersion   string
}
*/

var taskHPA *TaskHPA

type TaskHPA struct {
	//TargetRef            *TargetRef
	hpaNamespace         string
	hpaName              string
	forecastPeriodMinute int32
	ahpa                 *v1.AppawareHorizontalPodAutoscaler
	//HPARef       *v1beta1.CronHorizontalPodAutoscaler
	//id           string
	//name         string
	//DesiredSize  int32
	//Plan         string
	//RunOnce      bool
	//scaler       scaleclient.ScalesGetter
	//mapper       apimeta.RESTMapper
	//excludeDates []string
	kubeclientset kubernetes.Interface
	recorder      record.EventRecorder
	cancelFunc    context.CancelFunc
}

func RunTaskHPA(ahpa *v1.AppawareHorizontalPodAutoscaler, kubeclientset kubernetes.Interface,
	recorder record.EventRecorder) {
	ctx, cancel := context.WithCancel(context.Background())
	taskHPA = &TaskHPA{
		hpaNamespace:         NamespaceDefault,
		hpaName:              ahpa.Spec.ScaleTargetRef.Name,
		forecastPeriodMinute: *ahpa.Spec.ForecastWindow,
		ahpa:                 ahpa,
		kubeclientset:        kubeclientset,
		recorder:             recorder,
		cancelFunc:           cancel,
	}
	go taskHPA.run(ctx)
}

func StopTaskHPA(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.Info("删除ahpa:", key)
	if taskHPA == nil {
		return
	}
	taskHPA.stop()
	taskHPA = nil
}

func (t *TaskHPA) stop() {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}

func (t *TaskHPA) run(ctx context.Context) {
	t.forecastAndUpdateHPA()
	ticker := time.NewTicker(time.Duration(t.forecastPeriodMinute) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go t.forecastAndUpdateHPA()
		case <-ctx.Done():
			klog.Info("TaskHPA exit")
			return
		}
	}
}

func (t *TaskHPA) forecastAndUpdateHPA() {
	// todo 请求天策、global-scheduler api
	forecastReplicas := int32(rand.IntnRange(1, 6))
	klog.Info("预测worker数：", forecastReplicas)
	ctx := context.Background()
	hpa, err := t.kubeclientset.AutoscalingV1().HorizontalPodAutoscalers(t.hpaNamespace).
		Get(ctx, t.hpaName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get HPA err: %v", err)
		return
	}
	if hpa.Status.CurrentReplicas == forecastReplicas {
		klog.Info("当前worker数满足预测，无需更新hpa")
		return
	}
	// todo 这里当前副本 ！= 预测副本，会不会hpa已经改了，如果已经改了，再次update会有什么结果，测试下
	klog.Infof("当前hpa[%s]对应的worker数[%d] != 预测worker数[%d]",
		t.hpaName, hpa.Status.CurrentReplicas, forecastReplicas)
	hpa.Spec.MinReplicas = &forecastReplicas
	hpa.Spec.MaxReplicas = forecastReplicas
	update, err := t.kubeclientset.AutoscalingV1().HorizontalPodAutoscalers(t.hpaNamespace).
		Update(ctx, hpa, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Update HPA err: %v", err)
		return
	}
	//t.recorder.Eventf(t.ahpa, corev1.EventTypeNormal)
	t.recorder.Eventf(t.ahpa, corev1.EventTypeNormal, SuccessExecuted, MessagePredictionOperationExecuted,
		*update.Spec.MinReplicas, update.Spec.MaxReplicas)

	klog.Infof("Update HPA success, HPA minReplicas[%d] maxReplicas[%d]",
		*update.Spec.MinReplicas, update.Spec.MaxReplicas)
}
