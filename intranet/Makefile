all:
	go install ../cmd/okta-authenticator
	touch -t 197001010000 $(GOBIN)/okta-authenticator
	zip -X -j okta-authenticator.zip $(GOBIN)/okta-authenticator
	go install ../cmd/okta-authorizer
	touch -t 197001010000 $(GOBIN)/okta-authorizer
	zip -X -j okta-authorizer.zip $(GOBIN)/okta-authorizer
	go install ../cmd/substrate-instance-factory
	touch -t 197001010000 $(GOBIN)/substrate-instance-factory
	zip -X -j substrate-instance-factory.zip $(GOBIN)/substrate-instance-factory

.PHONY: all
