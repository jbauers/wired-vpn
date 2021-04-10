#!/bin/bash -e
pushd server

HTTP_ENDPOINT="$(jq -r '.oidc.http_endpoint' ./settings.json)"

mkdir -p ./ssl
if [[ ! -f "./ssl/cert.crt" ]]; then
    openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 \
      -subj "/CN=${HTTP_ENDPOINT}" \
      -keyout ./ssl/cert.key \
      -out ./ssl/cert.crt
fi

popd
