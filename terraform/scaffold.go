package terraform

import (
	"fmt"
	"path/filepath"

	"github.com/src-bin/substrate/regions"
)

// Scaffold generates global.tf, global/{providers,substrate}.tf, regional.tf,
// and regional/{providers,substrate}.tf in rootDirname and places the
// substrate module directory in the current directory (presumed to be the
// root of the overall Substrate tree).  This creates enough structure to get
// the user started writing their own Terraform code that's domain-,
// environment-, quality-, and region-aware.
func Scaffold(domain, rootDirname string) error {

	domainGlobalProvidersFile := NewFile()
	domainGlobalProvidersFile.Push(Provider{Alias: "global"})
	if err := domainGlobalProvidersFile.Write(filepath.Join(domain, "global/providers.tf")); err != nil {
		return err
	}

	domainGlobalSubstrateFile := NewFile()
	domainGlobalSubstrateFile.Push(Module{
		Label:     Q("substrate"),
		Providers: map[ProviderAlias]ProviderAlias{GlobalProviderAlias: GlobalProviderAlias},
		Source:    Q("../../substrate/global"),
	})
	if err := domainGlobalSubstrateFile.Write(filepath.Join(domain, "global/substrate.tf")); err != nil {
		return err
	}

	domainRegionalProvidersFile := NewFile()
	domainRegionalProvidersFile.Push(Provider{Alias: "global"})
	domainRegionalProvidersFile.Push(Provider{Alias: "network"})
	if err := domainRegionalProvidersFile.Write(filepath.Join(domain, "regional/providers.tf")); err != nil {
		return err
	}

	domainRegionalSubstrateFile := NewFile()
	domainRegionalSubstrateFile.Push(Module{
		Label: Q("substrate"),
		Providers: map[ProviderAlias]ProviderAlias{
			GlobalProviderAlias:  GlobalProviderAlias,
			NetworkProviderAlias: NetworkProviderAlias,
		},
		Source: Q("../../substrate/regional"),
	})
	if err := domainRegionalSubstrateFile.Write(filepath.Join(domain, "regional/substrate.tf")); err != nil {
		return err
	}

	globalFile := NewFile()
	globalFile.Push(Module{
		Label: Q("global"),
		Providers: map[ProviderAlias]ProviderAlias{
			DefaultProviderAlias: GlobalProviderAlias,
			GlobalProviderAlias:  GlobalProviderAlias,
		},
		Source: Q("./global"),
	})
	if err := globalFile.WriteIfNotExists(filepath.Join(rootDirname, "global.tf")); err != nil {
		return err
	}

	globalDomainFile := NewFile()
	globalDomainFile.Push(Module{
		Label: Q("global"),
		Providers: map[ProviderAlias]ProviderAlias{
			DefaultProviderAlias: GlobalProviderAlias,
			GlobalProviderAlias:  GlobalProviderAlias,
		},
		Source: Qf("../../%s/global", domain),
	})
	if err := globalDomainFile.Write(filepath.Join(rootDirname, "global", domain+".tf")); err != nil {
		return err
	}

	globalProvidersFile := NewFile()
	globalProvidersFile.Push(Provider{Alias: "global"})
	if err := globalProvidersFile.Write(filepath.Join(rootDirname, "global/providers.tf")); err != nil {
		return err
	}

	globalSubstrateFile := NewFile()
	globalSubstrateFile.Push(Module{
		Label:     Q("substrate"),
		Providers: map[ProviderAlias]ProviderAlias{GlobalProviderAlias: GlobalProviderAlias},
		Source:    Q("../../substrate/global"),
	})
	if err := globalSubstrateFile.Write(filepath.Join(rootDirname, "global/substrate.tf")); err != nil {
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
	if err := regionalFile.WriteIfNotExists(filepath.Join(rootDirname, "regional.tf")); err != nil {
		return err
	}

	regionalDomainFile := NewFile()
	regionalDomainFile.Push(Module{
		Label: Q("regional"),
		Providers: map[ProviderAlias]ProviderAlias{
			DefaultProviderAlias: DefaultProviderAlias,
			GlobalProviderAlias:  GlobalProviderAlias,
			NetworkProviderAlias: NetworkProviderAlias,
		},
		Source: Qf("../../%s/regional", domain),
	})
	if err := regionalDomainFile.Write(filepath.Join(rootDirname, "regional", domain+".tf")); err != nil {
		return err
	}

	regionalProvidersFile := NewFile()
	regionalProvidersFile.Push(Provider{Alias: "global"})
	regionalProvidersFile.Push(Provider{Alias: "network"})
	if err := regionalProvidersFile.Write(filepath.Join(rootDirname, "regional/providers.tf")); err != nil {
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
	if err := regionalSubstrateFile.Write(filepath.Join(rootDirname, "regional/substrate.tf")); err != nil {
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
