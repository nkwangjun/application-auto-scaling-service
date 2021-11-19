package vega

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/config"
	"nanto.io/application-auto-scaling-service/pkg/utils"
)

func SyncNodeIdsToOBS(clusterId string, kubeClient *kubernetes.Clientset, stopCh <-chan struct{}) {
	var (
		nodeIds []string
		err     error
	)
	localFilePath := fmt.Sprintf(config.ObsSourceFileTemplate, clusterId)
	ticker := time.NewTicker(config.SyncNodeIdsToOBSInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			nodeIds, err = getNodeIds(kubeClient)
			if err != nil {
				return
			}
			klog.Infof("Get nodeIds: %v", nodeIds)

			if err = writeNodeIdsFile(clusterId, nodeIds, localFilePath); err != nil {
				klog.Errorf("Write node ids to file err: %+v", err)
				continue
			}

			utils.SendNodeIdsFileToOBS(clusterId, localFilePath)
		case <-stopCh:
			klog.Info("=== SyncNodeIdsToOBS exit ===")
			return
		}
	}

}

func getNodeIds(kubeClient *kubernetes.Clientset) ([]string, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		klog.Errorf("Clientset get nodes err: %+v", err)
		return nil, err
	}

	nodeIds := []string{}
	for _, node := range nodes.Items {
		nodeIds = append(nodeIds, node.Spec.ProviderID)
	}

	return nodeIds, nil
}

type obsFileData struct {
	ClusterId string
	NodeIds   []string
}

func newObsFileData(clusterId string, nodeIds []string) *obsFileData {
	return &obsFileData{ClusterId: clusterId, NodeIds: nodeIds}
}

func writeNodeIdsFile(clusterId string, nodeIds []string, filePath string) error {
	var (
		dataFile  *os.File
		dataBytes []byte
		err       error
	)
	data := newObsFileData(clusterId, nodeIds)
	if dataBytes, err = json.Marshal(data); err != nil {
		return errors.Wrapf(err, "Marshal data[%+v] err", data)
	}
	if dataFile, err = os.Create(filePath); err != nil {
		return errors.Wrapf(err, "create file[%s] failed", filePath)
	}
	// chmod 避免umask覆盖
	if err = os.Chmod(filePath, 0640); err != nil {
		return errors.Wrapf(err, "chmod file[%s] failed", filePath)
	}
	if _, err = dataFile.WriteString(string(dataBytes)); err != nil {
		return errors.Wrapf(err, "create file[%s] failed", filePath)
	}
	return nil
}
