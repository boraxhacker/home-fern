#!/bin/bash

if [ -z "$1" ]
then
  echo "Must specify a service, ex ssm, route53"
  exit 1
fi

if [ -z "$2" ]
then
  echo "Must specify PUT or POST"
  exit 1
fi

if [ -z "$3" ]
then
  echo "Must specify a data file"
  exit 1
fi

USER=my-access
PWD=really-long-key
URL=http://localhost:9080


if [ "$1" == "all" ]
then
  curl --basic --user "${USER}:${PWD}" \
	--request POST \
	--form file=@$3 \
	${URL}/db/import/$1
else
  curl --basic --user "${USER}:${PWD}" \
	--request $2 \
	--data @$3 \
	${URL}/db/import/$1 
fi
