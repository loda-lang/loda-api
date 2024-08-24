#!/bin/bash

set -e

export GOROOT=/opt/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

pushd $GOPATH/src/github.com/loda-lang/loda-api/cmd

pushd oeis
go build -o /oeis_server
popd

pushd programs
go build -o /programs_server
popd

pushd stats
go build -o /stats_server
popd

popd > /dev/null
