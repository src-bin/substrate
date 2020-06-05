all:
	touch -t 197001010000 $(GOBIN)/substrate-okta-authenticator
	zip -X -j substrate-okta-authenticator.zip $(GOBIN)/substrate-okta-authenticator
	touch -t 197001010000 $(GOBIN)/substrate-okta-authorizer
	zip -X -j substrate-okta-authorizer.zip $(GOBIN)/substrate-okta-authorizer
	touch -t 197001010000 $(GOBIN)/substrate-instance-factory
	zip -X -j substrate-instance-factory.zip $(GOBIN)/substrate-instance-factory

.PHONY: all
