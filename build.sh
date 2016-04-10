#!/usr/bin/env bash

cat > ./Dockerfile.example <<DOCKERFILE
FROM gliderlabs/logspout:master
DOCKERFILE

cat > ./modules.go <<MODULES
package main
import (
	_ "github.com/gliderlabs/logspout/adapters/raw"
	_ "github.com/gliderlabs/logspout/adapters/syslog"
	_ "github.com/gliderlabs/logspout/httpstream"
	_ "github.com/gliderlabs/logspout/routesapi"
	_ "github.com/gliderlabs/logspout/transports/tcp"
	_ "github.com/gliderlabs/logspout/transports/udp"
	_ "github.com/gliderlabs/logspout/transports/tls"
    _ "github.com/lhdomenech/logspout-beat"
)
MODULES

docker build --no-cache -t lhdomenech/logspout-beat -f Dockerfile.example .

rm -f Dockerfile.example modules.go
