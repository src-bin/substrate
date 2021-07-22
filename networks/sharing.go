package networks

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/terraform"
)

func ShareVPC(
	f *terraform.File,
	account *awsorgs.Account,
	domain, environment, quality string,
	region string,
) {
	rs := terraform.ResourceShare{
		Provider: terraform.NetworkProviderAlias,
		Tags: terraform.Tags{
			Environment: environment,
			Name:        fmt.Sprintf("%s-%s-%s", domain, environment, quality),
			Quality:     quality,
			//Region:      region,
		},
	}
	rs.Label = terraform.Label(rs.Tags)
	f.Push(rs)

	f.Push(terraform.PrincipalAssociation{
		Label:            terraform.Label(rs.Tags),
		Principal:        terraform.Q(aws.StringValue(account.Id)),
		Provider:         terraform.NetworkProviderAlias,
		ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
	})

	eqTags := terraform.Tags{
		Environment: environment,
		Quality:     quality,
	}

	dataVPC := terraform.DataVPC{
		Label:    terraform.Q("network"),
		Provider: terraform.NetworkProviderAlias,
		Tags:     eqTags,
	}
	f.Push(dataVPC)
	dataSubnetIds := terraform.DataSubnetIds{
		Label:    terraform.Q("network"),
		Provider: terraform.NetworkProviderAlias,
		Tags:     eqTags,
		VpcId:    terraform.U(dataVPC.Ref(), ".id"),
	}
	f.Push(dataSubnetIds)
	dataSubnet := terraform.DataSubnet{
		ForEach:  terraform.U(dataSubnetIds.Ref(), ".ids"),
		Id:       terraform.U("each.value"),
		Label:    terraform.Q("network"),
		Provider: terraform.NetworkProviderAlias,
	}
	f.Push(dataSubnet)

	// TODO 2021.07 share the appropriate VPC using the aws.network provider.

	f.Push(terraform.EC2Tag{
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Connectivity),
		Label:      terraform.Q("subnet-connectivity"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.U(fmt.Sprintf("each.value.tags[\"%s\"]", tags.Connectivity)),
	})
	f.Push(terraform.EC2Tag{
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Environment),
		Label:      terraform.Q("subnet-environment"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.Q(environment),
	})
	f.Push(terraform.EC2Tag{
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Name),
		Label:      terraform.Q("subnet-name"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.U(fmt.Sprintf("\"%s-%s-${each.value.tags[\"%s\"]}-${each.value.availability_zone}\"", environment, quality, tags.Connectivity)),
	})
	f.Push(terraform.EC2Tag{
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Quality),
		Label:      terraform.Q("subnet-quality"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.Q(quality),
	})

	f.Push(terraform.EC2Tag{
		Key:        terraform.Q(tags.Environment),
		Label:      terraform.Q("vpc-environment"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(environment),
	})
	f.Push(terraform.EC2Tag{
		Key:        terraform.Q(tags.Name),
		Label:      terraform.Q("vpc-name"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(fmt.Sprintf("%s-%s", environment, quality)),
	})
	f.Push(terraform.EC2Tag{
		Key:        terraform.Q(tags.Quality),
		Label:      terraform.Q("vpc-quality"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(quality),
	})
}
