#!/bin/bash

set -xe 

CONT=qsd

bin/qsd-client  create --image test0 --size 512000
bin/qsd-client snapshot create --source test0 --name snap0
bin/qsd-client snapshot create  --source test0 --name snap1
bin/qsd-client snapshot create  --source test0 --name snap2
bin/qsd-client snapshot create  --source test0 --name snap3
bin/qsd-client snapshot create --source test0 --name snap4
bin/qsd-client snapshot create --source test0 --name snap5
bin/qsd-client snapshot delete --source test0 --name snap0
bin/qsd-client snapshot delete  --source test0 --name snap1
bin/qsd-client snapshot delete  --source test0 --name snap2
bin/qsd-client snapshot delete  --source test0 --name snap3
bin/qsd-client snapshot delete --source test0 --name snap4
bin/qsd-client delete --image test0
bin/qsd-client snapshot delete --source test0 --name snap5

