#!/bin/sh

# All our variable needs set in '.env'.
source "${WORKDIR}/.env"

#
# Set up SAML.
#

settings_file="${WORKDIR}/saml/settings.json"

sed -i "s/SERVER_PROTO/${SERVER_PROTO}/g" $settings_file
sed -i "s/SERVER_HOSTNAME/${SERVER_HOSTNAME}/g" $settings_file
sed -i "s/ONELOGIN_DOMAIN/${ONELOGIN_DOMAIN}/g" $settings_file
sed -i "s/ONELOGIN_CONNECTOR_ID/${ONELOGIN_CONNECTOR_ID}/g" $settings_file
sed -i "s|ONELOGIN_CONNECTOR_CERT|${ONELOGIN_CONNECTOR_CERT}|g" $settings_file

cat $settings_file

#
# Set up Wireguard.
#

wg_interface="wireguard"
wg_config="${WORKDIR}/wireguard/${wg_interface}.conf"
wg_ip="$(echo ${WG_ADDRESS} | cut -d. -f1-3).1"

if [ -f ${wg_config} ]; then
    echo "Using previous configuration ${wg_config}"
else
    echo "Adding new configuration..."
    wg genkey | tee privatekey | wg pubkey > publickey
    cat << EOF > "${wg_config}"
[Interface]
PrivateKey = $(cat privatekey)
Address = ${wg_ip}
ListenPort = ${WG_SERVER_PORT}
PostUp = iptables -A FORWARD -i ${wg_interface} -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i ${wg_interface} -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
SaveConfig = true

EOF
    chmod 0600 "${wg_config}"
fi

# We will call 'wireguard.sh' from Python.
export WG_SERVER_IP
export WG_SERVER_PORT
export WG_ADDRESS
export WG_ALLOWED_IPS
export WG_DNS

# Run Flask.
exec python -m flask run --host=0.0.0.0
