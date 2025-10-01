#!/bin/bash

set -e

export GOROOT=/opt/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

pushd $GOPATH/src/github.com/loda-lang/loda-api/cmd

pushd programs
go build -o /programs
popd

pushd sequences
go build -o /sequences
popd

pushd stats
go build -o /stats
popd

popd > /dev/null
