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
	Substrate = "Substrate"
)

type Map map[string]string
