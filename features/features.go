package features

// Define all feature flags here. By convention, the Go symbol and the string
// it refers to are the same so that uses in code and values in environment
// variables are the same. Set a feature flag by adding its string to the
// comma-delimited value of the SUBSTRATE_FEATURES environment variable.
const (
	DelegatedOrganizationAdministration feature = "DelegatedOrganizationAdministration"
	IdentityCenter                      feature = "IdentityCenter"
	IgnoreMacOSKeychain                 feature = "IgnoreMacOSKeychain"
	ProxyTelemetry                      feature = "ProxyTelemetry"
)
