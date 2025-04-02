#!/bin/bash

set -xe -o pipefail

REGION=us-east-1
PROFILE=test-ssm
ENDPOINT=http://localhost:9080/ssm

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    delete-parameters \
    --names /user/jim /user/horace /user/larry /user/frances /user/jim/friends /user/trash-dump

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    put-parameter \
    --name /user/jim \
    --value jim \
    --type SecureString

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    put-parameter \
    --name /user/horace \
    --value horace \
    --type SecureString

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    put-parameter \
    --name /user/larry \
    --value larry \
    --type SecureString

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    put-parameter \
    --name /user/frances \
    --value frances \
    --type SecureString

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    put-parameter \
    --name /user/jim/friends \
    --value horace,frances \
    --type StringList

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    get-parameter \
    --name /user/jim \
    --with-decryption

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    describe-parameters \
    --parameter-filters Key=Path,Option=OneLevel,Values=/user

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    describe-parameters \
    --parameter-filters Key=Path,Option=OneLevel,Values=/user/jim

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    describe-parameters \
    --parameter-filters Key=Path,Option=Recursive,Values=/user

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    get-parameters-by-path \
    --path /user

aws ssm \
    --region ${REGION} \
    --profile ${PROFILE} \
    --endpoint "${ENDPOINT}" \
    --output json \
    get-parameters-by-path \
    --path /user/jim \
    --recursive


