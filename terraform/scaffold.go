package terraform

import (
	"path/filepath"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/regions"
)

// Scaffold generates modules/domain/{global,regional}, both setup with both
// modules/{common,substrate} already instantiated if commonAndSubstrateModules
// is true. These are the best places to put your own Terraform code to make it
// domain-, environment-, quality-, and region-aware.
func Scaffold(domain string, commonAndSubstrateModule bool) (err error) {

	// modules/DOMAIN, possibly with references to modules{common,substrate} if
	// we're a fully-fledged domain module.
	{
		dirname := filepath.Join(ModulesDirname, domain, regions.Global)

		file := NewFile()
		if commonAndSubstrateModule {
			file.Add(Module{
				Label: Q(naming.Common),
				Providers: map[ProviderAlias]ProviderAlias{
					DefaultProviderAlias: DefaultProviderAlias,
					UsEast1ProviderAlias: UsEast1ProviderAlias,
				},
				Source: Q("../../", naming.Common, "/global"),
			})
		}
		if err = file.WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			return
		}

		if commonAndSubstrateModule {
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

		file := NewFile()
		if commonAndSubstrateModule {
			file.Add(Module{
				Label: Q(naming.Common),
				Providers: map[ProviderAlias]ProviderAlias{
					DefaultProviderAlias: DefaultProviderAlias,
					NetworkProviderAlias: NetworkProviderAlias,
				},
				Source: Q("../../", naming.Common, "/regional"),
			})
		}
		if err = file.WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			return
		}

		if commonAndSubstrateModule {
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

	// modules/common and modules/substrate, if we're a fully-fledged domain
	// module that needs to reference both of these.
	if commonAndSubstrateModule {
		{
			dirname := filepath.Join(ModulesDirname, naming.Common, regions.Global)

			if err = NewFile().WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
				return
			}

			substrateFile := NewFile()
			substrateFile.Add(Module{
				Label:  Q("substrate"),
				Source: Q("../../substrate/global"),
			})
			if err = substrateFile.Write(filepath.Join(dirname, "substrate.tf")); err != nil {
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
			dirname := filepath.Join(ModulesDirname, naming.Common, "regional")

			if err = NewFile().WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
				return
			}

			substrateFile := NewFile()
			substrateFile.Add(Module{
				Label: Q("substrate"),
				Providers: map[ProviderAlias]ProviderAlias{
					DefaultProviderAlias: DefaultProviderAlias,
					NetworkProviderAlias: NetworkProviderAlias,
				},
				Source: Q("../../substrate/regional"),
			})
			if err = substrateFile.Write(filepath.Join(dirname, "substrate.tf")); err != nil {
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
	}

	return
}
