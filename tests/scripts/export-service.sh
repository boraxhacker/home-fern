#!/bin/bash

if [ -z "$1" ]
then
  echo "Must specify a service, ex ssm, route53"
  exit 1
fi

USER=my-access
PWD=really-long-key
URL=http://localhost:9080

if [ "$1" == "all" ]
then
  curl --basic --user "${USER}:${PWD}" \
	--output /tmp/export-all.zip \
	${URL}/db/export/$1
else
  curl --basic --user "${USER}:${PWD}" \
	 ${URL}/db/export/$1
fi
