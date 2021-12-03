package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nanto.io/application-auto-scaling-service/pkg/config"
	"nanto.io/application-auto-scaling-service/pkg/controller"
	"nanto.io/application-auto-scaling-service/pkg/k8sclient"
	"nanto.io/application-auto-scaling-service/pkg/syncer"
	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
	"nanto.io/application-auto-scaling-service/pkg/utils/obsutil"
)

var (
	logger = logutil.GetLogger()
)

func Run(configFile string) error {
	// 读取配置、初始化log
	conf, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}
	logutil.Init(&conf.LogConf)
	logger.Infof("Load config: %+v", conf)

	// 启动 application-auto-scaling-service 服务
	ctx, cancel := context.WithCancel(context.Background())
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
		case <-ctx.Done():
			logger.Infof("Stop serving due to ctx.Done(), terminating...")
			run = false
		}
	}
	cancel()
	// 接收到信号退出场景，等待子goroutine退出
	time.Sleep(time.Second)
	return nil
}

func startService(ctx context.Context, conf *config.Config, cancel context.CancelFunc) error {
	var (
		obsCli *obsutil.ObsClient
		err    error
	)

	// 初始化 k8s client set
	if err = k8sclient.InitK8sClientSet(conf.K8sConf.Kubeconfig); err != nil {
		return err
	}

	// 同步 X实例 信息给 Vega（目前通过obs）
	if conf.SyncInstanceToVega {
		if obsCli, err = obsutil.NewObsClient(conf.ObsConf.Endpoint); err != nil {
			return err
		}
		go syncer.NewInstanceSyncer(obsCli, &conf.ObsConf, conf.ClusterId).SyncInstanceToOBS(ctx)
	}

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
