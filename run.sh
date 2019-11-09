#!/usr/bin/env sh

docker build -t python-saml . || exit 1
docker run -p 5000:5000 python-saml
