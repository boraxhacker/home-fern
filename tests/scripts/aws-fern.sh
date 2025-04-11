#!/bin/bash

export AWS_CONFIG_FILE=../aws-config
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=my-access
export AWS_SECRET_ACCESS_KEY=really-long-key
export AWS_PROFILE=home-fern-test

aws ${@}

