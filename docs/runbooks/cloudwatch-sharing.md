# Sharing CloudWatch data between accounts

AWS paves the way for the CloudWatch console to view metrics cross-account and cross-region which is very handy in a multi-account and potentially multi-region deployment managed by Substrate. Substrate does the majority of the work of managing IAM roles and configuration to make this easy.

To use this facility, follow these steps:

1. Visit the [cross-account cross-region settings in the CloudWatch console](https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#settings:/xaxr/view)
2. Check the “Show selector in the console” box
3. Click **Save changes**
