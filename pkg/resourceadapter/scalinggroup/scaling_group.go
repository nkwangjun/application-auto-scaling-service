package scalinggroup

type ScalingGroup interface {
	CountInstances(groupId string, status string) (error, int)
	ScaleOut(groupId string, instanceNum int) error
	ScaleIn(groupId string, instanceNum int) error
}