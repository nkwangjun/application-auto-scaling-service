package controller

import "nanto.io/application-auto-scaling-service/pkg/apis/autoscaling/v1alpha1"

type StrategiesInfo struct {
	ClusterId string `yaml:"clusterId"`
	// 策略创建时间，时间戳（毫秒）
	CreateTime int64 `yaml:"createTime"`
	// 目标HPA
	TargetHPA  string     `yaml:"targetHPA"`
	Strategies []Strategy `yaml:"strategies"`
}

type Strategy struct {
	// 生效时间段，eg："0:00-09:30"
	ValidTime string                                       `yaml:"validTime"`
	Spec      v1alpha1.CustomedHorizontalPodAutoscalerSpec `yaml:"spec"`
}
