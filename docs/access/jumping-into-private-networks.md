# Jumping into private networks

Substrate manages VPCs for each of your environments, as documented in [networking](../ref/networking.md), and each of these is constructed with three public and three private subnets. The private subnets, being private, aren't directly accessible from the Internet. And though the public subnets are, it's recommended that the traffic they do allow directly from the public Internet be handled first by ALB, NLB, API Gateway, or the like.

We recommend that you only allow SSH traffic from the public Internet into your Substrate account. Conveniently, this is exactly how the Instance Factory is meant to be used. So in order to jump into your private networks via SSH, use an EC2 instance from the Instance Factory as a jump box.

## Security groups in your private networks

Substrate takes care of peering VPCs such that your Substrate account will be able to reach all your other networks. It's up to you, however, to configure security groups to allow access where you want to.

To allow SSH from the Instance Factory, create security group rules that allow TCP traffic on port 22 from 192.168.0.0/16. (Your Substrate account is guaranteed to use that CIDR prefix and all your service accounts are guaranteed not to.)

## In two commands

First, SSH into an EC2 instance from the Instance Factory using the command it gives you that looks like this: `ssh -A ec2-user@ec2-5-6-7-8.us-west-2.compute.amazonaws.com`

Once you're there, you'll be able to SSH anywhere else in any of your networks that allow SSH from the Instance Factory.

## In one command

You can rewrite the SSH command generated by the Instance Factory to jump through it to your actual destination: `ssh -A -Jec2-user@ec2-5-6-7-8.us-west-2.compute.amazonaws.com DESTINATION`
