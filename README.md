Use `homedns` to set DNS records in AWS Route53 to point at a dynamic IP address. Every 5 minutes,
`homedns` will check the dynamic IP address. If it has changed, the DNS records will be updated.

## Install

```
go get github.com/leftmike/homedns
go build
```

You need AWS API access; see below to configure restricted permissions or just use default
permissions.

## Usage

Run `homedns` with the name(s) of the hosts you want to use the dynamic IP address
as arguments. The
[ipify](https://www.ipify.org/) service is used to determine the current IP address. Route 53
will be consulted to make sure the hosts have the current IP address; if they don't, they
will be updated.

## Restricted AWS Permissions

Using AWS IAM, create a user with only programmatic access and don't set any permissions. Save
the access key ID and secret access key; if you forget, you can always add another later.
Give that user the inline policy below; don't forget to change 'YOURZONEID' to the zone id for
your domain. Use Route 53 and then Hosted zones to find the zone id.

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "route53:ListResourceRecordSets",
                "route53:ChangeResourceRecordSets"
            ],
            "Resource": "arn:aws:route53:::hostedzone/YOURZONEID"
        },
        {
            "Effect": "Allow",
            "Action": "route53:ListHostedZonesByName",
            "Resource": "*"
        }
    ]
}
```


