package obsutil

import (
	"os"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/pkg/errors"

	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

const (
	defaultOBSMaxConnectionTimeoutSecond = 30
	defaultOBSMaxRetryCont               = 3
)

var logger = logutil.GetLogger()

// ObsClient ...
type ObsClient struct {
	ObsCli *obs.ObsClient
}

// NewObsClient 创建 obs client
func NewObsClient(endpoint string) (*ObsClient, error) {
	ak := os.Getenv("ak")
	sk := os.Getenv("sk")
	if ak == "" || sk == "" {
		return nil, errors.New("ak/sk invalid")
	}
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
