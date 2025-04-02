#!/bin/bash

REGION=us-east-1
PROFILE=test-ssm
ENDPOINT=http://localhost:9080/ssm

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    $@

