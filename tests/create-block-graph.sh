#!/bin/bash

set -xe 

CONT=qsd

bin/qsd-client  create --image test0 --size 512000
bin/qsd-client snapshot create --source test0 --name snap0
bin/qsd-client snapshot create  --source test0 --name snap1
bin/qsd-client snapshot create  --source test0 --name snap2
bin/qsd-client snapshot create  --source test0 --name snap3
bin/qsd-client  create --image test1 --from snap2
bin/qsd-client  create --image test2 --from test0
bin/qsd-client snapshot create --source test1 --name snap4
bin/qsd-client snapshot create --source test1 --name snap5

bin/qsd-client  create --image test3 --size 512000
bin/qsd-client snapshot create --source test3 --name snap6

bin/qsd-client  create --image test4 --size 512000
bin/qsd-client snapshot create --source test4 --name snap7
