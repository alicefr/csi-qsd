#!/bin/bash

set -xe 

CONT=qsd

docker run --name $CONT --rm -td -p 4444:4444 qsd/qsd
sleep 2
bin/qsd-client  create --image test0 --size 512000
bin/qsd-client snapshot create --image test0 --source test0 --name snap0
bin/qsd-client snapshot create --image snap0 --source test0 --name snap1
docker stop $CONT
