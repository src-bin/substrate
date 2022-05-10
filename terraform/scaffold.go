package terraform

import (
	"path/filepath"

	"github.com/src-bin/substrate/regions"
)

// Scaffold generates modules/domain/{global,regional}, both setup with the
// substrate module already instantiated.  These are the best places to put
// your own Terraform code to make it domain-, environment-, quality-, and
// region-aware.
func Scaffold(domain string) error {
	{
		dirname := filepath.Join(ModulesDirname, domain, regions.Global)

		if err := NewFile().WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			return err
		}

		substrateFile := NewFile()
		substrateFile.Push(Module{
			Label:  Q("substrate"),
			Source: Q("../../substrate/global"),
		})
		if err := substrateFile.Write(filepath.Join(dirname, "substrate.tf")); err != nil {
			return err
		}

		if err := versions(
			dirname,
			[]ProviderAlias{
				DefaultProviderAlias,
				UsEast1ProviderAlias,
			},
			false,
		); err != nil {
			return err
		}

	}
	{
		dirname := filepath.Join(ModulesDirname, domain, "regional")

		if err := NewFile().WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			return err
		}

		substrateFile := NewFile()
		substrateFile.Push(Module{
			Label: Q("substrate"),
			Providers: map[ProviderAlias]ProviderAlias{
				DefaultProviderAlias: DefaultProviderAlias,
				NetworkProviderAlias: NetworkProviderAlias,
			},
			Source: Q("../../substrate/regional"),
		})
		if err := substrateFile.Write(filepath.Join(dirname, "substrate.tf")); err != nil {
			return err
		}

		if err := versions(dirname, []ProviderAlias{NetworkProviderAlias}, false); err != nil {
			return err
		}

	}

	substrateGlobalModule := SubstrateGlobalModule()
	if err := substrateGlobalModule.Write(filepath.Join(ModulesDirname, "substrate/global")); err != nil {
		return err
	}
	substrateRegionalModule := SubstrateRegionalModule()
	if err := substrateRegionalModule.Write(filepath.Join(ModulesDirname, "substrate/regional")); err != nil {
		return err
	}

	return nil
}
