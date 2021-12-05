#!/bin/bash -e

docker build -t wss .
img=$(docker create wss)
docker cp $img:/tmp/wss/wss .
