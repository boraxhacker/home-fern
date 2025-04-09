#!/bin/bash

set -xe -o pipefail

export AWS_CONFIG_FILE=../aws-config
export AWS_PROFILE=home-fern

aws ssm \
    delete-parameters \
    --names /user/jim /user/horace /user/larry /user/frances /user/jim/friends /user/trash-dump

aws ssm \
    put-parameter \
    --name /user/jim \
    --value jim \
    --type SecureString

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


