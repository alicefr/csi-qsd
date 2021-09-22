#!/bin/bash

bin/qsd-client  create --image test0 --size 512000
bin/qsd-client  create --image test2 --from test0
bin/qsd-client list --tree true
bin/qsd-client delete image --image test2
