#!/bin/bash

set -xe -o pipefail

export AWS_CONFIG_FILE=../aws-config
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=my-access
export AWS_SECRET_ACCESS_KEY=really-long-key
export AWS_PROFILE=home-fern-test

aws kms \
  encrypt \
  --key-id alias/home-fern \
  --plaintext $(echo jim | base64) \
  --encryption-context KeyName1=string,KeyName2=string

aws kms \
  encrypt \
  --key-id alias/home-fern \
  --plaintext $(echo jim | base64)

aws kms \
  decrypt \
  --key-id alias/home-fern \
  --ciphertext-blob dXg5R2VQTmI4WlZ5VTQxMXZPUS9QVy9leHZTSDlYLzVhS24xczEzandIVT0= \
  --encryption-context KeyName1=string,KeyName2=string

aws kms \
  decrypt \
  --key-id alias/home-fern \
  --ciphertext-blob dXg5R2VQTmI4WlZ5VTQxMXZPUS9QVy9leHZTSDlYLzVhS24xczEzandIVT0=
