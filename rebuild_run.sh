#!/bin/bash
# rebuild_run.sh
go build -o testapp ./cmd/testapp/main.go
fuser -k 8085/tcp || true
./testapp
