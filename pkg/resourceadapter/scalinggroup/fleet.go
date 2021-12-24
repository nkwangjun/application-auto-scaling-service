package scalinggroup

import "nanto.io/application-auto-scaling-service/pkg/utils/logutil"

var logger = logutil.GetLogger()


type GameFleet struct {
	regionId string
}

// GameFleet implement ScalingGroup
var _ ScalingGroup = &GameFleet{}

func NewGameFleet(regionId string) *GameFleet {
	return &GameFleet{
		regionId: regionId,
	}
}

func (client *GameFleet) CountInstances(fleetId string, status string) (error, int) {
	panic("implement me")
}


func (client *GameFleet) ScaleOut(fleetId string, instanceNum int) error {
	// TODO(wj): 执行扩容操作(异步)
	panic("implement me")
}

func (client *GameFleet) ScaleIn(fleetId string, instanceNum int) error {
	// TODO(wj): 执行减容操作(异步)
	panic("implement me")
}
