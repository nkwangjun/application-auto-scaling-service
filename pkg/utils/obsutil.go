package utils

import (
	"fmt"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"k8s.io/klog/v2"

	"nanto.io/application-auto-scaling-service/resources"
)

func SendFileToOBS(clusterId, srcFile string) {
	var ak = resources.AK
	var sk = resources.SK
	var endpoint = resources.ObsEndpoint

	// 创建 obs client
	var obsClient, err = obs.New(ak, sk, endpoint)
	if err != nil {
		klog.Errorf("Failed to get OBS client, err: %v", err)
		return
	}
	defer obsClient.Close()

	// 上传文件
	input := &obs.PutFileInput{}
	input.Bucket = resources.ObsBucketName
	input.Key = fmt.Sprintf(resources.ObsObjectKeyTemplate, clusterId)
	input.SourceFile = srcFile // localfile为待上传的本地文件路径，需要指定到具体的文件名
	output, err := obsClient.PutFile(input)
	if err != nil {
		klog.Errorf("Failed to send file[%s], err: %v", srcFile, err)
		return
	}
	klog.Infof("Success to send file[%s], RequestId[%s]", srcFile, output.RequestId)
}
