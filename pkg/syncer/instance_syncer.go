package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nanto.io/application-auto-scaling-service/pkg/confutil"
	"nanto.io/application-auto-scaling-service/pkg/k8sclient"
	"nanto.io/application-auto-scaling-service/pkg/logutil"
	"nanto.io/application-auto-scaling-service/pkg/obsutil"
)

var logger = logutil.GetLogger()

// InstanceSyncer 周期同步 X实例 信息给 Vega
type InstanceSyncer struct {
	obsCli    *obsutil.ObsClient
	clusterId string
	// obs bucket name
	bucket string
	// 上传文件本地路径
	srcPath string
	// 上传文件目标路径（obs）
	targetPath     string
	intervalMinute time.Duration
}

func NewInstanceSyncer(obsCli *obsutil.ObsClient, obsConfig *confutil.ObsConf, clusterId string) *InstanceSyncer {
	return &InstanceSyncer{
		obsCli:         obsCli,
		clusterId:      clusterId,
		bucket:         obsConfig.BucketName,
		srcPath:        fmt.Sprintf(obsConfig.SourceFileNodeIdsTemplate, clusterId),
		targetPath:     fmt.Sprintf(obsConfig.ObjectKeyNodeIdsTemplate, clusterId),
		intervalMinute: time.Duration(obsConfig.SyncNodeIdsToOBSIntervalMinute) * time.Minute,
	}
}

// SyncInstanceToOBS 同步实例信息(nodeId)到 obs，供 Vega 获取
func (s *InstanceSyncer) SyncInstanceToOBS(ctx context.Context) {
	if err := s.syncInstanceToOBS(); err != nil {
		logger.Errorf("SyncInstanceToOBS err: %+v", err)
	}
	ticker := time.NewTicker(s.intervalMinute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.syncInstanceToOBS(); err != nil {
				logger.Errorf("SyncInstanceToOBS err: %+v", err)
				continue
			}
		case <-ctx.Done():
			logger.Info("=== SyncInstanceToOBS exit ===")
			return
		}
	}
}

func (s *InstanceSyncer) syncInstanceToOBS() error {
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
	nodes, err := k8sclient.GetKubeClientSet().CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "clientset get nodes err")
	}

	nodeIds := []string{}
	for _, node := range nodes.Items {
		nodeIds = append(nodeIds, node.Spec.ProviderID)
	}

	return nodeIds, nil
}

type instanceData struct {
	ClusterId string
	NodeIds   []string
}

func newInstanceData(clusterId string, nodeIds []string) *instanceData {
	return &instanceData{ClusterId: clusterId, NodeIds: nodeIds}
}

func writeNodeIdsFile(clusterId string, nodeIds []string, filePath string) error {
	var (
		dataFile  *os.File
		dataBytes []byte
		err       error
	)
	data := newInstanceData(clusterId, nodeIds)
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
