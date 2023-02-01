package networks

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
)

func ShareVPC(
	f *terraform.File,
	account *awsorgs.Account,
	domain, environment, quality string,
	region string,
) (dependsOn terraform.ValueSlice) {
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
	f.Add(rs)

	f.Add(terraform.PrincipalAssociation{
		Label:            terraform.Label(rs.Tags),
		Principal:        terraform.Q(aws.ToString(account.Id)),
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
	f.Add(dataVPC)
	dataSubnets := terraform.DataSubnets{
		Label:    terraform.Label(rs.Tags),
		Provider: terraform.NetworkProviderAlias,
		Tags:     eqTags,
		VpcId:    terraform.U(dataVPC.Ref(), ".id"),
	}
	f.Add(dataSubnets)
	dataSubnet := terraform.DataSubnet{
		ForEach:  terraform.U("toset(", dataSubnets.Ref(), ".ids", ")"),
		Id:       terraform.U("each.value"),
		Label:    terraform.Label(rs.Tags),
		Provider: terraform.NetworkProviderAlias,
	}
	f.Add(dataSubnet)

	t := terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tagging.Connectivity),
		Label:      terraform.Label(rs.Tags, "subnet-connectivity"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.U(fmt.Sprintf("each.value.tags[\"%s\"]", tagging.Connectivity)),
	}
	f.Add(t)
	dependsOn = append(dependsOn, t.Ref())
	t = terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tagging.Environment),
		Label:      terraform.Label(rs.Tags, "subnet-environment"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.Q(environment),
	}
	f.Add(t)
	dependsOn = append(dependsOn, t.Ref())
	f.Add(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tagging.Name),
		Label:      terraform.Label(rs.Tags, "subnet-name"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.U(fmt.Sprintf("\"%s-%s-${each.value.tags[\"%s\"]}-${each.value.availability_zone}\"", environment, quality, tagging.Connectivity)),
	})
	t = terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		ForEach:    terraform.U(dataSubnet.Ref()),
		Key:        terraform.Q(tagging.Quality),
		Label:      terraform.Label(rs.Tags, "subnet-quality"),
		ResourceId: terraform.U("each.value.id"),
		Value:      terraform.Q(quality),
	}
	f.Add(t)
	dependsOn = append(dependsOn, t.Ref())

	t = terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		Key:        terraform.Q(tagging.Environment),
		Label:      terraform.Label(rs.Tags, "vpc-environment"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(environment),
	}
	f.Add(t)
	dependsOn = append(dependsOn, t.Ref())
	f.Add(terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		Key:        terraform.Q(tagging.Name),
		Label:      terraform.Label(rs.Tags, "vpc-name"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(fmt.Sprintf("%s-%s", environment, quality)),
	})
	t = terraform.EC2Tag{
		DependsOn:  terraform.ValueSlice{ts.Ref()},
		Key:        terraform.Q(tagging.Quality),
		Label:      terraform.Label(rs.Tags, "vpc-quality"),
		ResourceId: terraform.U(dataVPC.Ref(), ".id"),
		Value:      terraform.Q(quality),
	}
	f.Add(t)
	dependsOn = append(dependsOn, t.Ref())

	ra := terraform.ResourceAssociation{
		ForEach:          terraform.U(dataSubnet.Ref()),
		Label:            terraform.Label(rs.Tags),
		Provider:         terraform.NetworkProviderAlias,
		ResourceArn:      terraform.U("each.value.arn"),
		ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
	}
	f.Add(ra)

	ts.DependsOn = terraform.ValueSlice{ra.Ref()}
	f.Add(ts)

	return dependsOn
}
