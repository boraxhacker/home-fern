# home-fern

home-fern (Faux Endpoints for Resources and Networking) implements some commonly used 
APIs and technologies.

> **_NOTE:_** Use at your own risk. There isn't robust authorization, validation nor error checking. 
> The implementation is at best "mostly" consistent with the real thing.

## Overview

home-fern (sorta) implements 
* Terraform state HTTP backend 
* AWS SSM Stored Parameter API
* AWS Route53 API

The goal is to enable the use of well known frameworks, such as Terraform, in the home lab setting.
See terraform example under the tests folder.

For example, using home-fern, it's possible to: 
* put-parameter using AWS CLI 
* get-parameter using Terraform

AWS CLI

```shell

#!/bin/bash

REGION=us-east-1
PROFILE=home-fern
ENDPOINT=http://localhost:9080/ssm

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    put-parameter \
    --name /home/mydb/password \
    --type SecureString \
    --value some-long-password
```

Terraform:

```terraform
terraform {
  backend "http" {
    address        = "http://localhost:9080/tfstate/myapp"
    lock_address   = "http://localhost:9080/tfstate/myapp/lock"
    unlock_address = "http://localhost:9080/tfstate/myapp/unlock"

    username = "my-username"
    password = "really-long-password"
  }
}


provider "aws" {
  profile = "home-fern"

  s3_use_path_style = true

  # Skip AWS related checks and validations
  skip_credentials_validation = true
  skip_requesting_account_id = true
  skip_metadata_api_check = true
  skip_region_validation = true
}

data "aws_ssm_parameter" "database_password" {
   name = "/home/mydb/password"
   with_decryption = true
}
```
## Configuration

### home-fern-config.yaml

Region and one value for each stanza, credentials and keys, is required. 
The application uses the first Key as the default key.

Region, AccessKey, SecretKey, Username, Alias, KeyId are arbitrary though 
you'll probably want consistency with ~/.aws/{config,credentials} files. 

The credentials are used for authenticating the v4sig headers; no PBAC is enforced.

The Key data value must be base64 encoded; see comments below. Parameters of type SecureString 
are encrypted and decrypted based on the KeyId argument. Config ID and Alias values are used 
for lookup of the KeyId argument. 

```yaml
region: us-east-1

credentials:
  - accessKey: "my-access"
    secretKey: "really-long-key"
    username: "John.Doe"

# AES-256 uses a 32-byte (256-bit) key
# openssl rand -base64 32
kms:
  - alias: aws/ssm
    id: 844c1364-08b8-11f0-aeb7-33cf4b255e16
    key: DkVsBYNRbORxQ6vtjUCex54YdfYfxd3c5PcP/ZruwUs=
  - alias: home-ssm
    id:  d0c49d70-4fae-4a20-84f0-d03fb6d670cb
    key: rvl7SbrNObB5MMQDUUAoInJXpyCA3QDqELyuwa2G48M=

dns:
  soa: ns-1.example.com. admin.example.com. (1 3600 180 604800 1800)
  nameServers:
    - ns-1.example.com
    - ns-2.example.com
    - ns-3.example.com
    - ns-4.example.com
```

## Execution

```shell
 ./home-fern --help
Usage of ./home-fern:
  -config string
        Path to the home-fern config file. (default ".home-fern-config.yaml")
  -data-path string
        Path to data store folder. (default ".home-fern-data")
```
