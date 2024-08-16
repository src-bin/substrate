Substrate
=========

Substrate is a command-line tool that helps teams build and operate secure, compliant, isolated AWS infrastructure. It's the right way to use AWS for startups and small teams.

Full documentation is available in [docs](https://github.com/substrate-maintainers/substrate/tree/main/docs).

As of 2024-08-14, the canonical version of Substrate comes from <https://github.com/substrate-maintainers/substrate>. Source & Binary, the company that initially developed Substrate, is winding down operations as of this date.

Development
-----------

Here's what you need to do Substrate development:

* Linux or MacOS
* Git
* The version of Go specified in `go.mod`
* `GOBIN` set (explicitly or implicitly - verify with `go env GOBIN`) to a writeable directory on your `PATH`
* Terraform (by running `substrate terraform install`, among other options)

Here's how to build and install `substrate` locally:

    make && make install

Use the following environment variables to control debugging features:

* `SUBSTRATE_DEBUG_AWS_LOGS`: Set to a non-empty string to get full request and response logs of every request made by the AWS SDK.
* `SUBSTRATE_DEBUG_AWS_RETRIES`: Set to an integer to control the maximum number of times a request will be retried by the AWS SDK.
* `SUBSTRATE_DEBUG_AWS_IAM_ASSUME_ROLE_POLICIES`: Set to a non-empty string to print the assume-role policy for every IAM role Substrate creates.
* `SUBSTRATE_FEATURES`: Set to a comma-separated list of feature names in `features/features.go` to enable.
