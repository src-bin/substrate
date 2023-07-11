package tagging

// Keys.
const (
	Connectivity = "Connectivity" // only used by subnets

	// TODO Customer = "Customer"
	Domain      = "Domain"
	Environment = "Environment"
	Quality     = "Quality"

	Manager = "Manager"

	Name = "Name"

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
