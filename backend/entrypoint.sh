#!/bin/bash
export WG_SERVER_IP="${WG_SERVER_CIDR%%/*}"

sed -i "s/IP_FORWARDING=No/IP_FORWARDING=YES/g" /etc/shorewall/shorewall.conf
sed -i "s/CLAMPMSS=No/CLAMPMSS=YES/" /etc/shorewall/shorewall.conf
sed -i "s/STARTUP_ENABLED=No/STARTUP_ENABLED=YES/" /etc/shorewall/shorewall.conf

ip link add dev ${WG_SERVER_INTERFACE} type wireguard
ip address add dev ${WG_SERVER_INTERFACE} ${WG_SERVER_CIDR}
ip link set dev ${WG_SERVER_INTERFACE} up

exec /opt/backend
