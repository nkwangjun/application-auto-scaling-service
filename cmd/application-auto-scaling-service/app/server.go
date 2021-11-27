package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/config"
	apiextensionsclientset "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned"
	informers "nanto.io/application-auto-scaling-service/pkg/client/informers/externalversions"
	"nanto.io/application-auto-scaling-service/pkg/confutil"
	"nanto.io/application-auto-scaling-service/pkg/k8s"
	"nanto.io/application-auto-scaling-service/pkg/logutil"
	"nanto.io/application-auto-scaling-service/pkg/obsutil"
	"nanto.io/application-auto-scaling-service/pkg/vega"
)

var (
	logger = logutil.GetLogger()

	//conf *confutil.Config
)

func Run(configFile string) error {
	// 读取配置、初始化log
	conf, err := confutil.LoadConfig(configFile)
	if err != nil {
		return err
	}
	logutil.Init(&conf.LogConf)
	logger.Infof("Load config: %+v", conf)

	ctx, cancel := context.WithCancel(context.Background())
	stopCh := ctx.Done()
	// 启动 application-auto-scaling-service 服务
	if err := startService(ctx, conf, cancel); err != nil {
		klog.Errorf("Start service err: %+v", err)
		return err
	}
	klog.Info("=== Start application-auto-scaling-service success ===")

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
	k8s.GetCron().Stop()
	klog.Flush()
	time.Sleep(time.Second)
	return nil
}

func startService(ctx context.Context, conf *confutil.Config, cancel context.CancelFunc) error {
	// 初始化 k8s client set
	if err := k8s.InitK8sClientSet(conf.KubeConfig); err != nil {
		return err
	}

	{ // [Debug] 打印 cce cluster 中的 customed hpa 信息
		chpas, err := k8s.GetCrdClientSet().AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(k8s.NamespaceDefault).
			List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "get cce hpa err")
		}
		chpaNum := len(chpas.Items)
		for i, chpa := range chpas.Items {
			klog.Infof("=== cce hpa[%d/%d]: %s/%s", i+1, chpaNum, chpa.Namespace, chpa.Name)
		}
	}

	// 初始化 ObsClient
	obsCli, err := obsutil.NewObsClient(conf.ObsConfig.Endpoint, config.AK, config.SK)
	if err != nil {
		return err
	}
	// 同步 X实例 信息给 Vega
	go vega.NewNodeIdsSyncer(obsCli, &conf.ObsConfig, conf.ClusterId).SyncNodeIdsToOBS(ctx)

	// 启动 Informer 和 controller
	//startInformerAndController(conf.ClusterId, k8s.GetKubeClientSet(), k8s.GetCrdClientSet(), stopCh, cancel)

	// 初始化定时任务（扩缩容策略需要定时修改）
	k8s.InitCron()

	// todo 初始化 api http client
	//NewHttpClient

	// todo 启动 http server（给 conductor 提供分流策略）
	//startHttpServer(stopCh)

	return nil
}

func startInformerAndController(clusterId string, kubeClient kubernetes.Interface,
	crdClient apiextensionsclientset.Interface, stopCh <-chan struct{}, cancel context.CancelFunc) {

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	crdInformerFactory := informers.NewSharedInformerFactory(crdClient, time.Second*30)

	crdController := k8s.NewController(clusterId, kubeClient, crdClient, crdInformerFactory)
	{
		cmInformer := kubeInformerFactory.Core().V1().ConfigMaps()
		cmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				if newObj.(*v1.ConfigMap).Name == "aass-cm" {
					fmt.Println("aass-cm 配置刷新了")
				}
			},
		})
	}

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
