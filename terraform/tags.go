package terraform

import "github.com/src-bin/substrate/version"

type Tags struct {
	Domain, Environment, Quality string
	Name                         string
	Special                      string
}

func (Tags) Manager() string { return "Terraform" }

func (Tags) SubstrateVersion() string { return version.Version }
