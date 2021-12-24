package ecs

import(
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	ecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

var logger = logutil.GetLogger()

func NewEcsClient(endpoint string, ak string, sk string, projectId string) *ecs.EcsClient {
	credentials := basic.NewCredentialsBuilder().
		WithAk(ak).
		WithSk(sk).
		WithProjectId(projectId).
		Build()
	client := ecs.NewEcsClient(ecs.EcsClientBuilder().
		WithEndpoint(endpoint).
		WithCredential(credentials).
		Build())
	return client
}

func CreateServers(client *ecs.EcsClient, name string, imageRef string, flavorRef string, adminPass string,
	vpcId string, subnetId string, securityGroupId string, publicIpType string, bandWithSize int32) (*[]string, error) {
	req := &model.CreateServersRequest{
		Body: &model.CreateServersRequestBody{
			Server: &model.PrePaidServer{
				ImageRef: imageRef,
				FlavorRef: flavorRef,
				Name: name,
				AdminPass: &adminPass,
				Vpcid: vpcId,
				Nics: []model.PrePaidServerNic {
						{SubnetId: subnetId},
					},
				SecurityGroups: &[]model.PrePaidServerSecurityGroup {
						{Id: &securityGroupId},
					},
				Publicip: &model.PrePaidServerPublicip {
						Eip: &model.PrePaidServerEip{
							Iptype: publicIpType,
							Bandwidth: &model.PrePaidServerEipBandwidth {
								Size: &bandWithSize,
							},
						},
					},
				},
			},
		}
	rsp, err := client.CreateServers(req)
	if err != nil {
		logger.Errorf("CreateServer errors")
		return nil, err
	}

	return rsp.ServerIds, nil
}