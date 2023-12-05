package features

// Define all feature flags here. By convention, the Go symbol and the string
// it refers to are the same so that uses in code and values in environment
// variables are the same. Set a feature flag by adding its string to the
// comma-delimited value of the SUBSTRATE_FEATURES environment variable.
const (
	APIGatewayV2                        feature = "APIGatewayV2"
	DelegatedOrganizationAdministration feature = "DelegatedOrganizationAdministration"
	IdentityCenter                      feature = "IdentityCenter"
	MacOSKeychain                       feature = "MacOSKeychain"
)
