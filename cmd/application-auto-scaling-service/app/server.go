package app

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/config"
	apiextensionsclientset "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned"
	informers "nanto.io/application-auto-scaling-service/pkg/client/informers/externalversions"
	"nanto.io/application-auto-scaling-service/pkg/controller"
	"nanto.io/application-auto-scaling-service/pkg/vega"
)

var (
	masterURL  string
	kubeconfig string
	//trafficStrategyAddr string
	//nodepoolConfig      string
)

func Run() error {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. "+
		"Overrides any value in kubeconfig. Only required if out-of-cluster.")
	//flag.StringVar(&trafficStrategyAddr, "traffic-strategy-bind-address", ":6060", "The address the traffic strategy endpoint binds to.")
	//flag.StringVar(&nodepoolConfig, "nodepool-config", "", "Path to a nodepoolconfig.")
	flag.Parse()
	//fmt.Println(kubeconfig, masterURL, trafficStrategyAddr, nodepoolConfig)

	clusterId := os.Getenv("CLUSTER_ID")
	if clusterId == "" {
		clusterId = config.ClusterId
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopCh := ctx.Done()
	// 启动 application-auto-scaling-service 服务
	if err := startService(clusterId, stopCh, cancel); err != nil {
		klog.Errorf("Start service err: %+v", err)
	}

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	for run := true; run; {
		select {
		case sig := <-sigCh:
			klog.Infof("Caught signal[%s], terminating...", sig)
			run = false
		case <-stopCh:
			klog.Infof("Stop serving due to ctx.Done(), terminating...")
			run = false
		}
	}
	cancel()
	controller.GetCron().Stop()
	klog.Flush()
	time.Sleep(time.Second)
	return nil
}

func startService(clusterId string, stopCh <-chan struct{}, cancel context.CancelFunc) error {
	// 初始化 k8s client set
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return errors.Wrap(err, "Error building kubeconfig")
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "Error building kubernetes clientset")
	}
	crdClient, err := apiextensionsclientset.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "Error building example clientset")
	}

	{ // [Debug] 打印 cce cluster 中的 customed hpa 信息
		chpas, err := crdClient.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(controller.NamespaceDefault).
			List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "Get cce hpa err")
		}
		chpaNum := len(chpas.Items)
		for i, chpa := range chpas.Items {
			klog.Infof("=== cce hpa[%d/%d]: %s/%s", i+1, chpaNum, chpa.Namespace, chpa.Name)
		}
	}

	// 同步 X实例 信息给 Vega
	go vega.SyncNodeIdsToOBS(clusterId, kubeClient, stopCh)

	// 启动 Informer 和 controller
	startInformerAndController(clusterId, kubeClient, crdClient, stopCh, cancel)

	// 初始化定时任务（扩缩容策略需要定时修改）
	controller.InitCron()

	// todo 初始化 北向 api client

	// todo 启动 http server（给 conductor 提供分流策略）
	//startHttpServer(stopCh)

	return nil
}

func startInformerAndController(clusterId string, kubeClient kubernetes.Interface,
	crdClient apiextensionsclientset.Interface, stopCh <-chan struct{}, cancel context.CancelFunc) {

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	crdInformerFactory := informers.NewSharedInformerFactory(crdClient, time.Second*30)

	crdController := controller.NewController(clusterId, kubeClient, crdClient, crdInformerFactory)

	kubeInformerFactory.Start(stopCh)
	crdInformerFactory.Start(stopCh)

	crdController.Run(2, stopCh, cancel)
}

//func startHttpServer() {
//	server, err := trafficstrategy.NewWebServer(trafficStrategyAddr, forecastWindow, nodepoolConfig)
//	if err != nil {
//		klog.Fatalf("Error running controller: %s", err.Error())
//	}
//
//	// start traffic strategy webserver
//	go func() {
//		server.Serve()
//	}()
//}
