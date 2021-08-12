#!/bin/bash

set -ex
tests/create-block-graph.sh

bin/qsd-client delete image --image test0
bin/qsd-client delete image --image test1
bin/qsd-client delete image --image test2
