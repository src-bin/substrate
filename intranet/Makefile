all:
	touch -t 197001010000 $(GOBIN)/okta-authenticator
	zip -X -j okta-authenticator.zip $(GOBIN)/okta-authenticator
	touch -t 197001010000 $(GOBIN)/okta-authorizer
	zip -X -j okta-authorizer.zip $(GOBIN)/okta-authorizer
	touch -t 197001010000 $(GOBIN)/substrate-instance-factory
	zip -X -j substrate-instance-factory.zip $(GOBIN)/substrate-instance-factory

.PHONY: all
