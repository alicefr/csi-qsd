#!/bin/bash

bin/qsd-client  create --image test0 --size 512000
bin/qsd-client  snapshot create  --source test0 --name snap0
bin/qsd-client  create --image test1 --from snap0

bin/qsd-client list --tree true

bin/qsd-client  delete --image test0
bin/qsd-client  snapshot delete  --source test0 --name snap0
bin/qsd-client  delete --image test1

bin/qsd-client list --tree true
