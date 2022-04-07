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
			Quality:     quality,
			Region:      region,
		},
	}
	if domain == "admin" {
		rs.Tags.Name = fmt.Sprintf("%s-%s", domain, quality)
	} else {
		rs.Tags.Name = fmt.Sprintf("%s-%s-%s", domain, environment, quality)
	}
	rs.Label = terraform.Label(rs.Tags)
	f.Push(rs)

	f.Push(terraform.PrincipalAssociation{
		Label:            terraform.Label(rs.Tags),
		Principal:        terraform.Q(aws.StringValue(account.Id)),
		Provider:         terraform.NetworkProviderAlias,
		ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
	})

	ts := terraform.TimeSleep{
		CreateDuration: terraform.Q("60s"),
		Label:          terraform.Q("share-before-tag"),
	}

	eqTags := terraform.Tags{
		Environment: environment,
		Quality:     quality,
	}

	dataVPC := terraform.DataVPC{
		Label:    terraform.Label(rs.Tags),
		Provider: terraform.NetworkProviderAlias,
		Tags:     eqTags,
	}
	f.Push(dataVPC)
	dataSubnetIds := terraform.DataSubnetIds{
		Label:    terraform.Label(rs.Tags),
		Provider: terraform.NetworkProviderAlias,
		Tags:     eqTags,
		VpcId:    terraform.U(dataVPC.Ref(), ".id"),
	}
	f.Push(dataSubnetIds)
	dataSubnet := terraform.DataSubnet{
		ForEach:  terraform.U(dataSubnetIds.Ref(), ".ids"),
		Id:       terraform.U("each.value"),
		Label:    terraform.Label(rs.Tags),
		Provider: terraform.NetworkProviderAlias,
	}
	f.Push(dataSubnet)

	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Connectivity),
		Label:      terraform.Label(rs.Tags, "subnet-connectivity"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.U(fmt.Sprintf("each.value.tags[\"%s\"]", tags.Connectivity)),
	})
	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Environment),
		Label:      terraform.Label(rs.Tags, "subnet-environment"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.Q(environment),
	})
	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Name),
		Label:      terraform.Label(rs.Tags, "subnet-name"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.U(fmt.Sprintf("\"%s-%s-${each.value.tags[\"%s\"]}-${each.value.availability_zone}\"", environment, quality, tags.Connectivity)),
	})
	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tags.Quality),
		Label:      terraform.Label(rs.Tags, "subnet-quality"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.Q(quality),
	})

	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		Key:        terraform.Q(tags.Environment),
		Label:      terraform.Label(rs.Tags, "vpc-environment"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(environment),
	})
	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		Key:        terraform.Q(tags.Name),
		Label:      terraform.Label(rs.Tags, "vpc-name"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(fmt.Sprintf("%s-%s", environment, quality)),
	})
	f.Push(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		Key:        terraform.Q(tags.Quality),
		Label:      terraform.Label(rs.Tags, "vpc-quality"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(quality),
	})

	ra := terraform.ResourceAssociation{
		ForEach:          terraform.U(dataSubnet.Ref()),
		Label:            terraform.Label(rs.Tags),
		Provider:         terraform.NetworkProviderAlias,
		ResourceArn:      terraform.U("each.value.arn"),
		ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
	}
	f.Push(ra)

	ts.DependsOn = terraform.ValueSlice{ra.Ref()}
	f.Push(ts)

}
