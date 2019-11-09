#!/usr/bin/env sh

docker build -t saml-wireguard . || exit 1
docker run -p 5000:5000 \
           --name wireguard-saml \
           --rm \
           saml-wireguard \
       || exit 1
