package vega

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/pkg/k8s"
	"nanto.io/application-auto-scaling-service/pkg/logutil"
	"nanto.io/application-auto-scaling-service/pkg/obsutil"
)

var logger = logutil.GetLogger()

// NodeIdsSyncer 周期同步 X实例 信息给 Vega
type NodeIdsSyncer struct {
	obsCli    *obsutil.ObsClient
	clusterId string
	// obs bucket
	bucket string
	// 上传文件本地路径
	srcPath string
	// 上传文件目标路径（obs）
	targetPath     string
	intervalMinute time.Duration
}

func NewNodeIdsSyncer(obsCli *obsutil.ObsClient, obsConfig *obsutil.ObsConfig, clusterId string) *NodeIdsSyncer {
	return &NodeIdsSyncer{
		obsCli:         obsCli,
		clusterId:      clusterId,
		bucket:         obsConfig.BucketName,
		srcPath:        fmt.Sprintf(obsConfig.SourceFileNodeIdsTemplate, clusterId),
		targetPath:     fmt.Sprintf(obsConfig.ObjectKeyNodeIdsTemplate, clusterId),
		intervalMinute: time.Duration(obsConfig.SyncNodeIdsToOBSIntervalMinute) * time.Minute,
	}
}

func (s *NodeIdsSyncer) SyncNodeIdsToOBS(ctx context.Context) {
	if err := s.syncNodeIdsToOBS(); err != nil {
		logger.Errorf("SyncNodeIdsToOBS err: %+v", err)
	}
	ticker := time.NewTicker(s.intervalMinute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.syncNodeIdsToOBS(); err != nil {
				logger.Errorf("SyncNodeIdsToOBS err: %+v", err)
				continue
			}
		case <-ctx.Done():
			klog.Info("=== SyncNodeIdsToOBS exit ===")
			return
		}
	}
}

func (s *NodeIdsSyncer) syncNodeIdsToOBS() error {
	var (
		nodeIds []string
		err     error
	)
	nodeIds, err = getNodeIds()
	if err != nil {
		return err
	}
	logger.Infof("Get nodeIds: %v", nodeIds)

	// todo 做缓存，先判断是否修改，再上传

	if err = writeNodeIdsFile(s.clusterId, nodeIds, s.srcPath); err != nil {
		return err
	}
	if err = s.obsCli.UploadObj(s.bucket, s.srcPath, s.targetPath); err != nil {
		return err
	}
	return nil
}

func getNodeIds() ([]string, error) {
	nodes, err := k8s.GetKubeClientSet().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "clientset get nodes err")
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
