package obsutil

const (
	defaultOBSMaxConnectionTimeoutSecond = 30
	defaultOBSMaxRetryCont               = 3
)

// ObsClient ...
type ObsClient struct {

}

// NewObsClient 创建 obs client
func NewObsClient(endpoint string) (*ObsClient, error) {
	return nil, nil
}

// UploadObj 上传文件，入参 srcPath 和 targetPath 需要指定到文件名
func (c *ObsClient) UploadObj(bucket, srcPath, targetPath string) error {
	return nil
}
