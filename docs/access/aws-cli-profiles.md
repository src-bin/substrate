# Using AWS CLI profiles

The AWS CLI is deceptively powerful and ubiquitous but can be tough to configure in a multi-account organization and the most obvious way to configure it — using an access key ID and secret access key — is far and away the most risky.

Fortunately, there are two less-well-known configurations that can help you address both: Profiles and the `credential_process` directive.

## Using Substrate as a “credential process”

When you use the AWS CLI or an SDK without any additional configuration, it'll read `~/.aws/config` and use the default profile it finds there. You can configure the default profile (or any other name by changing _default_ to something else) to use Substrate to get credentials as follows:

```
[profile default]
credential_process = substrate credentials --format json --quiet
```

This will save you having to run `eval $(substrate credentials)` yourself but will open a browser window each and every time you use the AWS CLI or SDK. Most users should prefer to use `eval $(substrate credentials)` to put AWS credentials that last 12 hours into environment variables.

## Using Substrate to assume roles in named profiles

Once you have AWS credentials in your environment, you can choose to use profiles to save yourself some typing. Define profiles in `~/.aws/config`, naming them whatever you like, that defer credential management to Substrate via `substrate assume-role`:

```
[profile whatever-you-want-to-call-it]
credential_process = substrate assume-role --format json --quiet --domain <domain> --environment <environment> --quality <quality>
```

Note well that, in order for this to succeed, you'll need to have already run `eval $(substrate credentials)` to prime the environment to have any access to AWS at all.

Use your profile thus:

```shell-session
eval $(substrate credentials)
aws sts get-caller-identity --profile whatever-you-want-to-call-it
```

This is considerably shorter than `substrate assume-role --format json --quiet --domain <domain> --environment <environment> --quality <quality> aws sts get-caller-identity` but the profile is local to your machine and not shared amongst your teammates the way domains, environments, and qualities are which makes collaboration harder. Nonetheless, profiles are a part of the AWS CLI and SDK that Substrate supports so use whichever tool suits you in every situation — there's no need to commit to one exclusively. You can even check out [Granted](https://granted.dev/) to navigate the profiles you configure in `~/.aws/config`.
