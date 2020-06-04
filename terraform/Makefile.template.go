package terraform

// managed by go generate; do not edit by hand

func makefileTemplate() string {
	return `AUTO_APPROVE=
#AUTO_APPROVE=-auto-approve

all:

apply: init
	terraform apply $(AUTO_APPROVE)

destroy: init
	terraform destroy $(AUTO_APPROVE)

init: .terraform

plan:
	terraform plan

.PHONY: all apply init plan

.terraform:
	terraform init
`
}
