package confutil

import (
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
	"k8s.io/klog/v2"
	"nanto.io/application-auto-scaling-service/pkg/logutil"
	"nanto.io/application-auto-scaling-service/pkg/obsutil"
)

type Config struct {
	ClusterId   string            `ini:"cluster_id"`
	ClusterName string            `ini:"cluster_name"`
	KubeConfig  string            `ini:"kubeconfig"`
	LogConf     logutil.LogConf   `ini:"log"`
	ObsConfig   obsutil.ObsConfig `ini:"obs"`
}

// LoadConfig 加载配置文件
func LoadConfig(configFile string) (*Config, error) {
	config := GetDefaultConfig()
	if err := readConfig(configFile, config); err != nil {
		return nil, err
	}
	return config, nil
}

func readConfig(configFile string, config *Config) error {
	klog.Infof("Reading config file: %s", configFile)
	conf, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, configFile)
	if err != nil {
		return errors.Wrapf(err, "read conf file[%s] err", configFile)
	}
	if err = conf.MapTo(config); err != nil {
		return errors.Wrapf(err, "invalid config from file[%s]", configFile)
	}
	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		ClusterId:   "default_cluster_id",
		ClusterName: "default_cluster_name",
		KubeConfig:  "./resources/kubeconfig.json",
		LogConf: logutil.LogConf{
			Level:      "info",
			Path:       "/opt/cloud/logs/application-auto-scaling-service/application-auto-scaling-service.conf",
			MaxSize:    20,
			MaxBackups: 50,
			MaxDays:    90,
			Compress:   true,
		},
	}
}
