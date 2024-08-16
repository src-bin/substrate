# Cost management

Substrate makes it standard operating procedure to use multiple AWS accounts. Each of those AWS accounts is an unavoidable cost allocation bucket. If you choose your domains reasonably well, cost allocation and management in a Substrate-managed AWS organization is an almost trivial problem. In such an AWS organization, AWS Cost Explorer, which usually gets a bad rap, is more than up to the task of helping you explore and manage costs.

To use Cost Explorer, first ensure you've followed the steps to **Delegate access to billing data** in the [getting started](../bootstrapping/finishing.html#delegate-access-to-billing-data) guide. Then, alas, you'll have to wait up to 24 hours for AWS to get you enrolled in Cost Explorer. From that second day onward, though, it'll be up and running.

The most important facet to use when dissecting your bill is AWS account. If you're using [domains, environments, and qualities](../ref/domains-environments-qualities.html) effectively, you won't need to stress greatly over tagging strategies in order to quickly derive meaningful insights on your cloud infrastructure costs.

Be mindful, though, that Cost Explorer works for all accounts in your AWS organization. This can be useful, of course, because it allows domain owners to dissect their own costs endlessly without being distracted by costs associated with other domains. To get a view of the entire organization, though, assume a role like OrganizationAdministrator in your master account.

Once Cost Explorer is enabled, you may consider additionally setting up Cost Anomaly Detection. See their [getting started](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/getting-started-ad.html) guide for more information and instructions.

## Frequently asked questions

### What if I have credits in a legacy account that I invite to an organization?

The credits get applied to the organization, eventually. This may take some time.

[https://aws.amazon.com/premiumsupport/knowledge-center/consolidated-billing-credits/](https://aws.amazon.com/premiumsupport/knowledge-center/consolidated-billing-credits/)
