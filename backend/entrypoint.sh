#!/bin/bash
source /wireguard.env
export WG_SERVER_IP="${WG_NETWORK%%/*}"

ip link add dev ${WG_INTERFACE} type wireguard
ip address add dev ${WG_INTERFACE} ${WG_NETWORK}

exec /opt/backend
