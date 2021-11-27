package obsutil

import (
	"fmt"
	"io/ioutil"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/pkg/errors"

	"nanto.io/application-auto-scaling-service/config"
	"nanto.io/application-auto-scaling-service/pkg/logutil"
)

const (
	defaultOBSMaxConnectionTimeoutSecond = 30
	defaultOBSMaxRetryCont               = 3
)

var logger = logutil.GetLogger()

type ObsConfig struct {
	Endpoint                  string `ini:"endpoint"`
	BucketName                string `ini:"bucket_name"`
	SourceFileNodeIdsTemplate string `ini:"source_file_node_ids_template"`
	// nodeIds文件路径
	ObjectKeyNodeIdsTemplate string `ini:"object_key_node_ids_template"`
	// 伸缩策略文件路径
	ObjectKeyStrategiesTemplate    string `ini:"object_key_strategies_template"`
	SyncNodeIdsToOBSIntervalMinute int    `ini:"sync_node_ids_to_obs_interval_minute"`
}

// ObsClient ...
type ObsClient struct {
	ObsCli *obs.ObsClient
}

// NewObsClient 创建 obs client
func NewObsClient(endpoint, ak, sk string) (*ObsClient, error) {
	obsCli, err := obs.New(ak, sk, endpoint,
		obs.WithConnectTimeout(defaultOBSMaxConnectionTimeoutSecond),
		obs.WithMaxRetryCount(defaultOBSMaxRetryCont))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get obs client")
	}
	return &ObsClient{ObsCli: obsCli}, nil
}

// UploadObj 上传文件，入参 srcPath 和 targetPath 需要指定到文件名
func (c *ObsClient) UploadObj(bucket, srcPath, targetPath string) error {
	input := &obs.PutFileInput{}
	input.Bucket = bucket
	input.Key = targetPath
	input.SourceFile = srcPath
	output, err := c.ObsCli.PutFile(input)
	if err != nil {
		return errors.Wrapf(err, "failed to send file[%s]", srcPath)
	}
	logger.Infof("Success to send file[%s], RequestId[%s]", srcPath, output.RequestId)
	return nil
}

func GetStrategiesFromTianCe(clusterId string) ([]byte, error) {
	var ak = config.AK
	var sk = config.SK
	var endpoint = config.ObsEndpoint

	// 创建 obs client
	var obsClient, err = obs.New(ak, sk, endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get OBS client")
	}
	defer obsClient.Close()

	input := &obs.GetObjectInput{}
	input.Bucket = config.ObsBucketName
	input.Key = fmt.Sprintf(config.ObsObjectKeyStrategiesTemplate, clusterId)
	output, err := obsClient.GetObject(input)
	if err != nil {
		return nil, errors.Wrapf(err, "Get obj[%s/%s] from obs err", input.Bucket, input.Key)
	}
	defer output.Body.Close()
	bytes, err := ioutil.ReadAll(output.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Read file err")
	}
	return bytes, nil
}
