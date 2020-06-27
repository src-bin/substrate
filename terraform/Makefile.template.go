package terraform

// managed by go generate; do not edit by hand

func makefileTemplate() string {
	return `# managed by Substrate; do not edit by hand

AUTO_APPROVE=
#AUTO_APPROVE=-auto-approve

GOBIN={{.GOBIN}}

all:

apply:
	aws sts get-caller-identity
	terraform apply $(AUTO_APPROVE)

destroy:
	aws sts get-caller-identity
	terraform destroy $(AUTO_APPROVE)

init:
	terraform init

plan:
	aws sts get-caller-identity
	terraform plan

.PHONY: all apply init plan
`
}
