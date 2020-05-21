#!/bin/bash
source /wireguard.env
export WG_SERVER_IP="${WG_SERVER_CIDR%%/*}"

ip link add dev ${WG_SERVER_INTERFACE} type wireguard
ip address add dev ${WG_SERVER_INTERFACE} ${WG_SERVER_CIDR}

exec /opt/backend
