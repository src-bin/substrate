package terraform

//go:generate go run ../tools/template/main.go -name intranetGlobalTemplate -o intranet-global.go modules/intranet/global
func IntranetGlobalModule() *Directory {
	return &Directory{
		Files: intranetGlobalTemplate(),
	}
}

//go:generate go run ../tools/template/main.go -name intranetRegionalTemplate -o intranet-regional.go modules/intranet/regional
func IntranetRegionalModule() *Directory {
	return &Directory{
		ConfigurationAliases: []ProviderAlias{NetworkProviderAlias},
		Files:                intranetRegionalTemplate(),
	}
}

//go:generate go run ../tools/template/main.go -name lambdaFunctionGlobalTemplate -o lambda-function-global.go modules/lambda-function/global
func LambdaFunctionGlobalModule() *Directory {
	return &Directory{
		Files: lambdaFunctionGlobalTemplate(),
	}
}

//go:generate go run ../tools/template/main.go -name lambdaFunctionRegionalTemplate -o lambda-function-regional.go modules/lambda-function/regional
func LambdaFunctionRegionalModule() *Directory {
	return &Directory{
		Files: lambdaFunctionRegionalTemplate(),
	}
}

//go:generate go run ../tools/template/main.go -name peeringConnectionTemplate -o peering-connection.go modules/peering-connection
func PeeringConnectionModule() *Directory {
	return &Directory{
		ConfigurationAliases: []ProviderAlias{"aws.accepter", "aws.requester"},
		Files:                peeringConnectionTemplate(),
	}
}

//go:generate go run ../tools/template/main.go -name substrateGlobalTemplate -o substrate-global.go modules/substrate/global
func SubstrateGlobalModule() *Directory {
	return &Directory{
		Files: substrateGlobalTemplate(),
	}
}

//go:generate go run ../tools/template/main.go -name substrateRegionalTemplate -o substrate-regional.go modules/substrate/regional
func SubstrateRegionalModule() *Directory {
	return &Directory{
		ConfigurationAliases: []ProviderAlias{NetworkProviderAlias},
		Files:                substrateRegionalTemplate(),
	}
}
