#!/bin/bash

export AWS_CONFIG_FILE=../aws-config
export AWS_PROFILE=home-fern

aws ${@}

