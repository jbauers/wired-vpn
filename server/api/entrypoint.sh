#!/bin/bash -e

if [[ "${LOCAL}" == true ]]; then
	ETH0_IP="$(ifconfig eth0 | grep -w inet | cut -d':' -f2 | cut -d' ' -f1)"
else
	ETH0_IP="$(jq -r '.oidc.http_endpoint' /settings.json.tpl)" # FIXME: Assuming they're on the same machine, which is not good.
fi
sed "s/ETH0_IP/${ETH0_IP}/g" /settings.json.tpl > /settings.json

/opt/mq & /opt/backend
