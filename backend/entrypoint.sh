#!/bin/bash

sed -i "s/IP_FORWARDING=No/IP_FORWARDING=YES/g" /etc/shorewall/shorewall.conf
sed -i "s/CLAMPMSS=No/CLAMPMSS=YES/" /etc/shorewall/shorewall.conf
sed -i "s/STARTUP_ENABLED=No/STARTUP_ENABLED=YES/" /etc/shorewall/shorewall.conf

exec /opt/backend
