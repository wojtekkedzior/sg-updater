# sg-updater
Security Group updater based on AWS SNS topic.  Upload this as a lambda function and make sure the exceution role has permission to RevokeSecurityGroupIngress and AuthorizeSecurityGroupIngress.

The AWS SNS topic is described here: https://aws.amazon.com/blogs/aws/subscribe-to-aws-public-ip-address-changes-via-amazon-sns/

This tool searches for all security groups tagged with a key "sg-updater".  The value is irrelevant.  The SG capacity is hardcoded to 30 and should be adjusted according to your accound limits.  


