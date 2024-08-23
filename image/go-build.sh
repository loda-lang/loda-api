#!/bin/bash

set -e

export GOROOT=/opt/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

pushd $GOPATH/src/github.com/loda-lang/loda-api > /dev/null
go build -o /oeis_server cmd/oeis/main.go
go build -o /programs_server cmd/programs/main.go
go build -o /stats_server cmd/stats/main.go
popd > /dev/null
