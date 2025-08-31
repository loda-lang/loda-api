#!/bin/bash

set -e

export GOROOT=/opt/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

pushd $GOPATH/src/github.com/loda-lang/loda-api/cmd

pushd programs
go build -o /programs_server
popd

pushd sequences
go build -o /sequences_server
popd

pushd stats
go build -o /stats_server
popd

popd > /dev/null
