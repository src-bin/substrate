package terraform

// managed by go generate; do not edit by hand

func makefileTemplate() string {
	return `# managed by Substrate; do not edit by hand

AUTO_APPROVE=
#AUTO_APPROVE=-auto-approve

GOBIN={{.GOBIN}}

all:

apply: init
	terraform apply $(AUTO_APPROVE)

destroy: init
	terraform destroy $(AUTO_APPROVE)

init:
	terraform init

plan:
	terraform plan

.PHONY: all apply init plan
`
}
