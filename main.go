package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	clientset "nanto.io/application-auto-scaling-service/pkg/client/clientset/versioned"
	informers "nanto.io/application-auto-scaling-service/pkg/client/informers/externalversions"
	ftcontroller "nanto.io/application-auto-scaling-service/pkg/controller"
	"nanto.io/application-auto-scaling-service/pkg/signals"
)

var (
	masterURL           string
	kubeconfig          string
	trafficStrategyAddr string
	forecastWindow      int64
	nodepoolConfig      string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	crdClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	{ // Debug
		nodes, err := kubeClient.CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
		if err != nil {
			klog.Errorf("=== Get nodes err: %+v", err)
			return
		}
		nodeNum := len(nodes.Items)
		fmt.Printf("=== nodes info(total num[%d]):\n", nodeNum)
		for i, node := range nodes.Items {
			fmt.Printf("=== node[%d]: %s\n", i+1, node.Spec.ProviderID)
		}
		chpas, err := crdClient.AutoscalingV1alpha1().CustomedHorizontalPodAutoscalers(ftcontroller.NamespaceDefault).
			List(context.Background(), v1.ListOptions{})
		if err != nil {
			klog.Errorf("=== Get cce hpa err: %+v", err)
			return
		}
		chpaNum := len(chpas.Items)
		fmt.Printf("=== cce hpa info(total num[%d]):\n", chpaNum)
		for i, chpa := range chpas.Items {
			fmt.Printf("=== cce hpa[%d]: %s/%s\n", i+1, chpa.Namespace, chpa.Name)
		}
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	crdInformerFactory := informers.NewSharedInformerFactory(crdClient, time.Second*30)

	crdController := ftcontroller.NewController(kubeClient, crdClient, crdInformerFactory)

	// notice that there is no need to run Start methods in a separate goroutine.
	kubeInformerFactory.Start(stopCh)
	crdInformerFactory.Start(stopCh)

	//crdInformerFactory.ForResource(GVersion)
	if err = crdController.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

	// 注意上面 controller 的 Run 是阻塞的
	//server, err := trafficstrategy.NewWebServer(trafficStrategyAddr, forecastWindow, nodepoolConfig)
	//if err != nil {
	//	klog.Fatalf("Error running controller: %s", err.Error())
	//}
	//
	//// start traffic strategy webserver
	//go func() {
	//	server.Serve()
	//}()

}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&trafficStrategyAddr, "traffic-strategy-bind-address", ":6060", "The address the traffic strategy endpoint binds to.")
	flag.Int64Var(&forecastWindow, "forecast-window", 10, "The forecast window for traffic strategy.")
	flag.StringVar(&nodepoolConfig, "nodepool-config", "", "Path to a nodepoolconfig.")
}
