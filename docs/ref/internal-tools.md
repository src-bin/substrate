# Deciding where to host internal tools

It's easy to put all internal tools into one basket but, if you look closely, they can typically be cleanly put into two smaller groups. Some tools are, for lack of a better word, devops tools; they're concerned with cluster management, deployment, and the like. Other tools are part of your product; they understand and interact with your data model and your services. You should put devops tools in your Substrate account and you should put product tools in the appropriate domain accounts.

If you like, you can expose them all via your Intranet DNS domain name, though you'll want to namespace product tool URLs with an environment e.g. [https://example.com/production/user/123](https://example.com/production/user/123).

Finally, you are likely to encounter many situations in which tools running in your Substrate account need to access either control plane services or EC2 instances in service accounts. To facilitate this, your Substrate account is guaranteed to use networks in the 192.168.0.0/16 RFC 1918 address space, which means that security groups in your service accounts can feel free to allow access from 192.168.0.0/16 without any risk they're unknowingly allowing other service accounts a back door.

You should almost certainly [protect your internal tools](../mgmt/protecting-internal-tools.html) using your Intranet and identity provider.
