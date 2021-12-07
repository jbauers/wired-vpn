#!/bin/bash -e
interface="wired0"

down() {
	ip link del dev $interface type wireguard
}

trap down SIGINT EXIT
ip link add dev $interface type wireguard

/opt/wss -interface $interface
