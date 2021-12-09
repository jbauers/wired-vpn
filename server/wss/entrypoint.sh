#!/bin/bash -eu
interface="$WG_INTERFACE"
network="$WG_NETWORK"
port="$WG_PORT"

down() {
	ip link del dev $interface type wireguard
}
trap down SIGINT EXIT

ip link add dev $interface type wireguard
ip address add dev $interface $network
ip link set dev $interface up

/opt/wss -interface $interface -port $port -network $network
