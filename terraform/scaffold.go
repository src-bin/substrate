package terraform

import (
	"path/filepath"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/regions"
)

// Scaffold generates modules/domain/{global,regional}, both setup with the
// substrate module already instantiated if substrateModule is true. These are
// the best places to put your own Terraform code to make it domain-,
// environment-, quality-, and region-aware.
func Scaffold(domain string, substrateModule bool) (err error) {
	{
		dirname := filepath.Join(ModulesDirname, domain, regions.Global)

		if err = NewFile().WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			return
		}

		if substrateModule {
			substrateFile := NewFile()
			substrateFile.Add(Module{
				Label:  Q("substrate"),
				Source: Q("../../substrate/global"),
			})
			err = substrateFile.Write(filepath.Join(dirname, "substrate.tf"))
		} else {
			err = fileutil.Remove(filepath.Join(dirname, "substrate.tf"))
		}
		if err != nil {
			return
		}

		if err = versions(
			dirname,
			[]ProviderAlias{
				DefaultProviderAlias,
				UsEast1ProviderAlias,
			},
			false,
		); err != nil {
			return
		}

	}
	{
		dirname := filepath.Join(ModulesDirname, domain, "regional")

		if err = NewFile().WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			return
		}

		if substrateModule {
			substrateFile := NewFile()
			substrateFile.Add(Module{
				Label: Q("substrate"),
				Providers: map[ProviderAlias]ProviderAlias{
					DefaultProviderAlias: DefaultProviderAlias,
					NetworkProviderAlias: NetworkProviderAlias,
				},
				Source: Q("../../substrate/regional"),
			})
			err = substrateFile.Write(filepath.Join(dirname, "substrate.tf"))
		} else {
			err = fileutil.Remove(filepath.Join(dirname, "substrate.tf"))
		}
		if err != nil {
			return
		}

		if err = versions(dirname, []ProviderAlias{NetworkProviderAlias}, false); err != nil {
			return
		}

	}

	substrateGlobalModule := SubstrateGlobalModule()
	if err = substrateGlobalModule.Write(filepath.Join(ModulesDirname, "substrate/global")); err != nil {
		return
	}
	substrateRegionalModule := SubstrateRegionalModule()
	if err = substrateRegionalModule.Write(filepath.Join(ModulesDirname, "substrate/regional")); err != nil {
		return
	}

	return
}
