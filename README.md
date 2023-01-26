Substrate
=========

Substrate is an opinionated suite of tools that manage secure, reliable, and compliant cloud infrastructure in AWS.

Full documentation is available at <https://src-bin.com/substrate/manual/>.

Substrate is licensed under your Master Services Agreement with Source & Binary or under the [Substrate License](https://src-bin.com/substrate/license/).

Development
-----------

Here's what you need to do Substrate development:

* Linux or MacOS
* Git
* The version of Go specified in `go.mod`
* `GOBIN` set (explicitly or implicitly - verify with `go env GOBIN`) to a writeable directory on your `PATH`
* Terraform (by running `substrate terraform`, among other options)

Use the following environment variables to control debugging features:

* `SUBSTRATE_DEBUG_AWS_LOGS`: Set to a non-empty string to get full request and response logs of every request made by the AWS SDK.
* `SUBSTRATE_DEBUG_AWS_RETRIES`: Set to an integer to control the maximum number of times a request will be retried by the AWS SDK.
