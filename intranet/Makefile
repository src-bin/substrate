all:
	(cd ../cmd/okta-authenticator && go build -o ../../bin/okta-authenticator)
	touch -t 197001010000 ../bin/okta-authenticator
	zip -X -j okta-authenticator.zip ../bin/okta-authenticator
	(cd ../cmd/okta-authorizer && go build -o ../../bin/okta-authorizer)
	touch -t 197001010000 ../bin/okta-authorizer
	zip -X -j okta-authorizer.zip ../bin/okta-authorizer
	(cd ../cmd/substrate-instance-factory && go build -o ../../bin/substrate-instance-factory)
	touch -t 197001010000 ../bin/substrate-instance-factory
	zip -X -j substrate-instance-factory.zip ../bin/substrate-instance-factory

.PHONY: all
