package terraform

import "github.com/src-bin/substrate/version"

type Tags struct {
	Environment, Quality, Special string
}

func (Tags) Manager() string { return "Terraform" }

func (Tags) SubstrateVersion() string { return version.Version }
