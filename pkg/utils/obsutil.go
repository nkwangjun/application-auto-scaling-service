package utils

import (
	"fmt"
	"io/ioutil"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/config"
)

func SendNodeIdsFileToOBS(clusterId, srcFile string) {
	var ak = config.AK
	var sk = config.SK
	var endpoint = config.ObsEndpoint

	// 创建 obs client
	var obsClient, err = obs.New(ak, sk, endpoint)
	if err != nil {
		klog.Errorf("Failed to get OBS client, err: %v", err)
		return
	}
	defer obsClient.Close()

	// 上传文件
	input := &obs.PutFileInput{}
	input.Bucket = config.ObsBucketName
	input.Key = fmt.Sprintf(config.ObsObjectKeyNodeIdsTemplate, clusterId)
	input.SourceFile = srcFile // localfile为待上传的本地文件路径，需要指定到具体的文件名
	output, err := obsClient.PutFile(input)
	if err != nil {
		klog.Errorf("Failed to send file[%s], err: %v", srcFile, err)
		return
	}
	klog.Infof("Success to send file[%s], RequestId[%s]", srcFile, output.RequestId)
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
