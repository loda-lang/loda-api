#!/bin/bash

set -e

export GOROOT=/opt/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

pushd $GOPATH/src/github.com/loda-lang/loda-api
go build -o /oeis_server cmd/oeis/oeis_server.go
go build -o /programs_server cmd/programs/programs_server.go
go build -o /stats_server cmd/stats/stats_server.go
popd
