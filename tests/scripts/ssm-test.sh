#!/bin/bash

set -xe -o pipefail

export AWS_CONFIG_FILE=../aws-config
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=my-access
export AWS_SECRET_ACCESS_KEY=really-long-key
export AWS_PROFILE=home-fern-test

aws ssm \
    delete-parameters \
    --names /user/jim /user/horace /user/larry /user/frances /user/jim/friends /user/trash-dump

aws ssm \
    put-parameter \
    --name /user/jim \
    --value jim \
    --type SecureString \
	--tags Key=LastName,Value=Jones

aws ssm \
	list-tags-for-resource \
	--resource-type Parameter \
	--resource-id /user/jim

aws ssm \
    put-parameter \
    --name /user/jim \
    --value "jim v2" \
    --type SecureString \
	--overwrite

aws ssm \
	list-tags-for-resource \
	--resource-type Parameter \
	--resource-id /user/jim

aws ssm \
    put-parameter \
    --name /user/horace \
    --value horace \
    --type SecureString

aws ssm \
    put-parameter \
    --name /user/larry \
    --value larry \
    --type SecureString

aws ssm \
    put-parameter \
    --name /user/frances \
    --value frances \
    --type SecureString

aws ssm \
    put-parameter \
    --name /user/jim/friends \
    --value horace,frances \
    --type StringList

aws ssm \
    get-parameter \
    --name /user/jim \
    --with-decryption

aws ssm \
    describe-parameters \
    --parameter-filters Key=Path,Option=OneLevel,Values=/user

aws ssm \
    describe-parameters \
    --parameter-filters Key=Path,Option=OneLevel,Values=/user/jim

aws ssm \
    describe-parameters \
    --parameter-filters Key=Path,Option=Recursive,Values=/user

aws ssm \
    get-parameters-by-path \
    --path /user

aws ssm \
    get-parameters-by-path \
    --path /user/jim \
    --recursive


