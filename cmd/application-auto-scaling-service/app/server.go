package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nanto.io/application-auto-scaling-service/pkg/confutil"
	"nanto.io/application-auto-scaling-service/pkg/controller"
	"nanto.io/application-auto-scaling-service/pkg/k8sclient"
	"nanto.io/application-auto-scaling-service/pkg/logutil"
	"nanto.io/application-auto-scaling-service/pkg/obsutil"
	"nanto.io/application-auto-scaling-service/pkg/syncer"
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
		logger.Errorf("Start service err: %+v", err)
		return err
	}
	logger.Info("=== Start application-auto-scaling-service success ===")

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	for run := true; run; {
		select {
		case sig := <-sigCh:
			logger.Infof("Caught signal[%s], terminating...", sig)
			run = false
		case <-stopCh:
			logger.Infof("Stop serving due to ctx.Done(), terminating...")
			run = false
		}
	}
	cancel()
	time.Sleep(time.Second)
	return nil
}

func startService(ctx context.Context, conf *confutil.Config, cancel context.CancelFunc) error {
	// 初始化 k8s client set
	if err := k8sclient.InitK8sClientSet(conf.KubeConfig); err != nil {
		return err
	}

	{ // [Debug代码，上线删除] 打印 cce cluster 中的 customed hpa 信息
		chpas, err := k8sclient.GetCrdClientSet().AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(controller.NamespaceDefault).
			List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "get cce hpa err")
		}
		chpaNum := len(chpas.Items)
		for i, chpa := range chpas.Items {
			logger.Infof("=== cce hpa[%d/%d]: %s/%s", i+1, chpaNum, chpa.Namespace, chpa.Name)
		}
	}

	// 初始化 ObsClient
	obsCli, err := obsutil.NewObsClient(conf.ObsConf.Endpoint)
	if err != nil {
		return err
	}
	// 同步 X实例 信息给 Vega
	go syncer.NewInstanceSyncer(obsCli, &conf.ObsConf, conf.ClusterId).SyncInstanceToOBS(ctx)

	// 修改 cce 的扩缩策略
	go controller.NewStrategyController(&conf.StrategyConf).SyncStrategyToCCE(ctx, cancel)

	return nil

	// 初始化定时任务（扩缩容策略需要定时修改）
	//cronutil.InitCron()

	// todo 添加对 configmap 的监听
	// 启动 Informer 和 controller
	//startInformerAndController(conf.ClusterId, k8s.GetKubeClientSet(), k8s.GetCrdClientSet(), stopCh, cancel)

	// todo 初始化 http client（请求 GTM、GRM）
	//NewHttpClient

	// todo 启动 http server（给 conductor 提供分流策略）
	//startHttpServer(stopCh)
}

//func startInformerAndController(clusterId string, kubeClient kubernetes.Interface,
//	crdClient apiextensionsclientset.Interface, stopCh <-chan struct{}, cancel context.CancelFunc) {
//
//	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
//	crdInformerFactory := informers.NewSharedInformerFactory(crdClient, time.Second*30)
//
//	crdController := k8sclient.NewController(clusterId, kubeClient, crdClient, crdInformerFactory)
//	{
//		cmInformer := kubeInformerFactory.Core().V1().ConfigMaps()
//		cmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
//			UpdateFunc: func(oldObj, newObj interface{}) {
//				if newObj.(*v1.ConfigMap).Name == "aass-cm" {
//					fmt.Println("aass-cm 配置刷新了")
//				}
//			},
//		})
//	}
//
//	kubeInformerFactory.Start(stopCh)
//	crdInformerFactory.Start(stopCh)
//
//	crdController.Run(2, stopCh, cancel)
//}

//func startHttpServer() {
//	server, err := trafficstrategy.NewWebServer(trafficStrategyAddr, forecastWindow, nodepoolConfig)
//	if err != nil {
//		logger.Fatalf("Error running controller: %s", err.Error())
//	}
//
//	// start traffic strategy webserver
//	go func() {
//		server.Serve()
//	}()
//}
