#!/bin/bash

bin/qsd-client  create --image test0 --size 512000
bin/qsd-client  create --image test1 --from test0
bin/qsd-client  snapshot create  --source test0 --name snap0
bin/qsd-client  snapshot create  --source test0 --name snap2
bin/qsd-client  snapshot create  --source test1 --name snap3
bin/qsd-client  create --image test2 --from snap3
bin/qsd-client list --tree true
bin/qsd-client  delete --image test0
bin/qsd-client  delete --image test1
bin/qsd-client  snapshot delete  --source test0 --name snap0
bin/qsd-client  snapshot delete  --source test0 --name snap2
bin/qsd-client list --tree true
bin/qsd-client  snapshot delete  --source test1 --name snap3
bin/qsd-client list --tree true
bin/qsd-client  delete --image test2


bin/qsd-client list --tree true
