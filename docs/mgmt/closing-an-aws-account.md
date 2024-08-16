# Closing an AWS account

In March of 2022, AWS introduced the oft-requested `organizations:CloseAccount` API which at long last allows administrators to close AWS accounts programmatically, saving them from a tedious process that formerly involved password resets, credit cards, and phone number verification.

To close an AWS account that's a member of your organization, follow these steps:

1. Note from `substrate.accounts.txt` the account number of the account you wish to close
2. **Make super-duper sure this is the account you wish close**
3. `substrate assume-role --management aws organizations close-account --account-id <account-number-noted-above>`

See [AWS' announcement](https://aws.amazon.com/blogs/mt/aws-organizations-now-provides-a-simple-scalable-and-more-secure-way-to-close-your-member-accounts/) for more information on using this API, its limitations, and your options should you wish to reopen a closed account.
