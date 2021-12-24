package scalinggroup

import (
	"nanto.io/application-auto-scaling-service/pkg/resourceadapter/ecs"
)

type GameFleetMock struct {
	FleetInstance map[string][]string
}

// GameFleetMock implement ScalingGroup
var _ ScalingGroup = &GameFleetMock{}


func (g *GameFleetMock) CountInstances(groupId string, status string) (error, int) {
	logger.Debugf("Do count scalinggroup instances, gruopId:%s", groupId)
	instanceList, ok := g.FleetInstance[groupId]
	if ok {
		return nil, len(instanceList)
	}

	// else
	return nil, 0
}

func (g *GameFleetMock) ScaleOut(groupId string, instanceNum int) error {
	logger.Debugf("Do scale out %d scalinggroup, gruopId:%s", instanceNum, groupId)
	instanceList, ok := g.FleetInstance[groupId]
	if !ok {
		instanceList = []string{}
		g.FleetInstance[groupId] = instanceList
	}

	// Mock fleet scale out process
	for {
		serverIds, err := ecs.CreateServers(
				ecs.NewEcsClient("", "", "", ""),
				groupId + "-server-001",
				"image-id",
				"s6.small.2",
				"AAAbbb@@@123",
				"vpc001",
				"subnet001",
				"security001",
				"publicIp001",
				1,
				)
		if err != nil {
			return err
		}
		instanceList = append(instanceList, (*serverIds)[0])
	}

	return nil
}

func (g *GameFleetMock) ScaleIn(groupId string, instanceNum int) error {
	logger.Debugf("Do scale in %d scalinggroup, gruopId:%s", instanceNum, groupId)
	return nil
}


