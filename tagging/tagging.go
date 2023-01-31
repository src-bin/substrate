package tagging

// Keys.
const (
	Connectivity = "Connectivity" // only used by subnets
	Domain       = "Domain"
	Environment  = "Environment"
	Quality      = "Quality"

	Manager = "Manager"

	Name = "Name"

	SubstrateSpecialAccount = "SubstrateSpecialAccount"
	SubstrateVersion        = "SubstrateVersion"
)

// Values.
const (
	Substrate                = "Substrate"
	SubstrateInstanceFactory = "substrate-instance-factory" // remove in 2022.10
	Unknown                  = "unknown"
)

type Map map[string]string
