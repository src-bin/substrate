package terraform

import (
	"fmt"
	"path/filepath"

	"github.com/src-bin/substrate/regions"
)

// Scaffold generates global.tf, global/{providers,substrate}.tf, regional.tf,
// and regional/{providers,substrate}.tf in dirname and places the substrate
// module directory in the current directory (presumed to be the root of the
// overall Substrate tree).  This creates enough structure to get the user
// started writing their own Terraform code that's domain-, environment-,
// quality-, and region-aware.
func Scaffold(dirname string) error {

	globalFile := NewFile()
	globalFile.Push(Module{
		Label: Q("global"),
		Providers: map[ProviderAlias]ProviderAlias{
			DefaultProviderAlias: GlobalProviderAlias,
			GlobalProviderAlias:  GlobalProviderAlias,
		},
		Source: Q("./global"),
	})
	if err := globalFile.WriteIfNotExists(filepath.Join(dirname, "global.tf")); err != nil {
		return err
	}

	globalProvidersFile := NewFile()
	globalProvidersFile.Push(Provider{Alias: "global"})
	if err := globalProvidersFile.Write(filepath.Join(dirname, "global/providers.tf")); err != nil {
		return err
	}

	globalSubstrateFile := NewFile()
	globalSubstrateFile.Push(Module{
		Label:     Q("substrate"),
		Providers: map[ProviderAlias]ProviderAlias{GlobalProviderAlias: GlobalProviderAlias},
		Source:    Q("../../substrate/global"),
	})
	if err := globalSubstrateFile.Write(filepath.Join(dirname, "global/substrate.tf")); err != nil {
		return err
	}

	regionalFile := NewFile()
	for _, region := range regions.Selected() {
		regionalFile.Push(Module{
			Label: Q(region),
			Providers: map[ProviderAlias]ProviderAlias{
				DefaultProviderAlias: ProviderAliasFor(region),
				GlobalProviderAlias:  GlobalProviderAlias,
				NetworkProviderAlias: ProviderAlias(fmt.Sprintf("aws.%s-network", region)),
			},
			Source: Q("./regional"),
		})
	}
	if err := regionalFile.WriteIfNotExists(filepath.Join(dirname, "regional.tf")); err != nil {
		return err
	}

	regionalProvidersFile := NewFile()
	regionalProvidersFile.Push(Provider{Alias: "global"})
	regionalProvidersFile.Push(Provider{Alias: "network"})
	if err := regionalProvidersFile.Write(filepath.Join(dirname, "regional/providers.tf")); err != nil {
		return err
	}

	regionalSubstrateFile := NewFile()
	regionalSubstrateFile.Push(Module{
		Label: Q("substrate"),
		Providers: map[ProviderAlias]ProviderAlias{
			GlobalProviderAlias:  GlobalProviderAlias,
			NetworkProviderAlias: NetworkProviderAlias,
		},
		Source: Q("../../substrate/regional"),
	})
	if err := regionalSubstrateFile.Write(filepath.Join(dirname, "regional/substrate.tf")); err != nil {
		return err
	}

	substrateGlobalModule := SubstrateGlobalModule()
	if err := substrateGlobalModule.Write("substrate/global"); err != nil {
		return err
	}
	substrateRegionalModule := SubstrateRegionalModule()
	if err := substrateRegionalModule.Write("substrate/regional"); err != nil {
		return err
	}

	return nil
}
