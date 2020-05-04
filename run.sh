#/usr/bin/env sh

docker build -t saml-wireguard .
docker run -p 80:8080 saml-wireguard
