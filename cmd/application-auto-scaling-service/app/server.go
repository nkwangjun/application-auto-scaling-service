package app

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
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

	// 捕获并记录异常
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("Panic recover, err: %+v, call stack: %s", err, debug.Stack())
		}
	}()

	// 启动 application-auto-scaling-service 服务
	ctx, cancel := context.WithCancel(context.Background())
	//if err = startService(ctx, conf, cancel); err != nil {
	//	logger.Errorf("Start service err: %+v", err)
	//	// return err
	//}
	logger.Info("=== Start application-auto-scaling-service success ===")

	startPolicy()

	// 监听退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		logger.Infof("Caught signal[%s], terminating...", sig)
	case <-ctx.Done():
		logger.Infof("Stop serving due to ctx.Done(), terminating...")
	}
	cancel()
	time.Sleep(time.Second) // 等待子goroutine退出
	return nil
}

func startPolicy() {
	go controller.NewPolicy("policy-001", "fleet-123", "PercentAvailableGameSessions",
		"RuleBased", "ChangeInCapacity",
		50, 1, "GreaterThanThreshold", 2).Start()
}

// 启动 application-auto-scaling-service 服务
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

	// 启动strategy controller，修改 cce 的 hpa策略
	go controller.NewStrategyController(&conf.StrategyConf).Start(ctx, cancel)

	// todo 初始化 http client（请求 GTM、GRM）

	// todo 启动 http server（给 conductor 提供分流策略）

	return nil
}
