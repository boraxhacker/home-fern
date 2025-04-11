#!/bin/bash

if [ -z "$1" ]
then
  echo "Must specify a service, ex ssm, route53"
  exit 1
fi

USER=my-access
PWD=really-long-key

curl --basic --user "${USER}:${PWD}" http://localhost:9080/dump/$1/keys
