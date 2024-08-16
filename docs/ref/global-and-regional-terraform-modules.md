# Global and regional Terraform modules

Substrate generates a lot of Terraform code in modules with leaf directory names of `global` and `regional`. This is necessary because certain AWS services are global - IAM, notably - and thus their resources cannot be successfully managed from two AWS regions in two regional Terraform statefiles. Doing so will cause conflicts that generally aren't detected in Terraform plans but nonetheless fail when changes are applied. It's not simply a matter of importing resources, either, because every change will cause those imported copies to drift and require manual intervention. Thus, `global` and `regional` modules are the best structure.

Terraform resources for global AWS services like CloudFront, IAM, and Route 53 should be placed in a `global` module. (If you're in doubt as to whether an AWS service is global or regional, check for a region in its resources' ARNs.)

Terraform resources for pseudo-global services like Lambda@Edge, ACM when certificates are used with CloudFront distributions, Route 53 Domains, and possibly others should be placed in a `global` module _and_ specifiy `provider = aws.us-east-1` as these services _must_ be managed from us-east-1 and only from us-east-1.

If you have global singleton resources, even from AWS serivces that are regional, and you want them to exist in your default region or us-east-1, you may define them in a `global` module. If you want singleton resources to exist in another region, put them directly in the appropriate leaf directory in the `root-modules/` tree.

It's common, once you've created resources in a `global` module, to need to reference these resources from regional modules. Use Terraform data sources to lookup these resource by their name, tags, or the like in the accompanying `regional` module.

## Alternatively, namespace global resources in regional modules

Of course, with every rule there are exceptions. Sometimes it's not possible (or is significantly harder) to create a global resource without first knowing something about a regional resource. A typical example would be an IAM role used as a service account for EKS pods; the IAM role's trust policy needs to know the identity of the cluster and its OAuth OIDC provider before it can be written. In such cases, the correct practice is to namespace the global resource (e.g. the IAM role) with at least the AWS region in which it's being created.

```
data "aws_region" "current" {}

resource "aws_iam_role" "example" {
    name = "example-${data.aws_region.current.name}"
}
```
