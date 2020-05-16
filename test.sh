#!/bin/sh

go build src/handler.go || exit 1
sudo ./handler
