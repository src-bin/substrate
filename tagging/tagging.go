package tagging

// Keys.
const (

	// TODO Customer = "Customer"
	Domain      = "Domain"
	Environment = "Environment"
	Quality     = "Quality"

	Manager = "Manager"

	Name  = "Name"
	Owner = "Owner"

	Region           = "Region"           // only used by VPCs and subnets; probably unnecessary
	AvailabilityZone = "AvailabilityZone" // only used by subnets
	Connectivity     = "Connectivity"     // only used by subnets

	SubstrateAccountSelectors          = "SubstrateAccountSelectors"
	SubstrateAssumeRolePolicyFilenames = "SubstrateAssumeRolePolicyFilenames"
	SubstratePolicyAttachmentFilenames = "SubstratePolicyAttachmentFilenames"

	SubstrateSpecialAccount = "SubstrateSpecialAccount" // deprecated
	SubstrateType           = "SubstrateType"

	SubstrateVersion = "SubstrateVersion"
)

// Values.
const (
	Substrate = "Substrate"
)

type Map map[string]string

func Merge(maps ...Map) Map {
	m := make(Map)
	for i := 0; i < len(maps); i++ {
		for k, v := range maps[i] {
			m[k] = v
		}
	}
	return m
}
