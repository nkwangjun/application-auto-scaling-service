package controller

import (
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	servicev1 "nanto.io/application-auto-scaling-service/pkg/apis/batch/v1"
	clientset "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned"
	aassscheme "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned/scheme"
	"nanto.io/application-auto-scaling-service/pkg/client/informers/externalversions"
	autoscalinglisters "nanto.io/application-auto-scaling-service/pkg/client/listers/autoscaling/v1alpha1"
	batchlisters "nanto.io/application-auto-scaling-service/pkg/client/listers/batch/v1"
)

const controllerAgentName = "application-auto-scaling-service"

const (
	NamespaceDefault = "default"

	// SuccessSynced is used as part of the Event 'reason' when a ForecastTask is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a ForecastTask fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"
	// SuccessSynced is used as part of the Event 'reason' when a ForecastTask prediction operation is executed
	SuccessExecuted = "Executed"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by ForecastTask"
	// MessageResourceSynced is the message used for an Event fired when a ForecastTask
	// is synced successfully
	MessageResourceSynced = "ForecastTask synced successfully"
	// MessageResourceSynced is the message used for an Event fired when
	// ForecastTask prediction operation is executed
	MessagePredictionOperationExecuted = "Prediction operation is executed, changed the HPA values to isScaleUp[%t] " +
		"metricValue[%.2f] stepVal[%d]"
)

// Controller is the controller implementation for aass and chpa resources
type Controller struct {
	// kubeClientset is a standard kubernetes clientset
	kubeClientset kubernetes.Interface
	// crdClientset is a clientset for our own API group
	crdClientset clientset.Interface

	ftLister batchlisters.ForecastTaskLister
	ftSynced cache.InformerSynced

	chpaLister autoscalinglisters.CustomedHorizontalPodAutoscalerLister
	chpaSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new controller
func NewController(
	kubeClientset kubernetes.Interface,
	crdClientset clientset.Interface,
	crdInformerFactory externalversions.SharedInformerFactory) *Controller {

	// Create event broadcaster
	// Add forecast-task types to the default Kubernetes Scheme so Events can be
	// logged for forecast-task types.
	utilruntime.Must(aassscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	ftInformer := crdInformerFactory.Batch().V1().ForecastTasks()
	chpaInformer := crdInformerFactory.Autoscaling().V1alpha1().CustomedHorizontalPodAutoscalers()
	controller := &Controller{
		kubeClientset: kubeClientset,
		crdClientset:  crdClientset,
		ftLister:      ftInformer.Lister(),
		ftSynced:      ftInformer.Informer().HasSynced,
		chpaLister:    chpaInformer.Lister(),
		chpaSynced:    chpaInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
			"ForecastTasks"),
		recorder: recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when forecastTaskObj resources change
	ftInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueForecastTask,
		UpdateFunc: func(old, new interface{}) {
			oldFt := old.(*servicev1.ForecastTask)
			newFt := new.(*servicev1.ForecastTask)
			if oldFt.ResourceVersion == newFt.ResourceVersion {
				// If the version is consistent, no actual update is performed
				return
			}
			controller.enqueueForecastTask(new)
		},
		DeleteFunc: StopTaskCustomHPA,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.ftSynced, c.chpaSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch workers to process forecastTask resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// aass resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the apha resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the ft resource with this namespace/name
	ft, err := c.ftLister.ForecastTasks(namespace).Get(name)
	if err != nil {
		// The aass resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("aass '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	{ // debug
		bytes, _ := json.Marshal(ft)
		klog.Infof("=== ft marshal: %s", bytes)
	}

	// TODO(wangjun): 参数校验, 一个HPA只允许关联一个AHPA, 失败则更新AHPA状态和原因

	// TODO(wangjun): 如果AHPA作用的目标对象发生了变化, 则删除历史关联的后台任务

	klog.Infof("Receive ForecastTask obj, ScaleTargetRefs[%s/%s], ForecastWindow[%dmin]",
		NamespaceDefault, ft.Spec.ScaleTargetRefs[0].Name, *ft.Spec.ForecastWindow)
	RunTaskCustomHPA(ft, c.kubeClientset, c.crdClientset, c.recorder)

	c.recorder.Event(ft, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueueForecastTask takes an ft resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ft.
func (c *Controller) enqueueForecastTask(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
