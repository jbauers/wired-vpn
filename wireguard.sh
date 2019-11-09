#!/bin/sh

# Small script using Wireguard native tools to add a client block to a server
# configuration file, and to create a client configuration file to be served by
# Flask.
# This script is called when a '# Client: <CLIENT_EMAIL>' line is NOT
# found in the server configuration file. If the client block already exists,
# Flask will NOT call this script to create a new client configuration, it will
# serve the existing one instead.

tmp_dir=$(mktemp -d)
cd ${tmp_dir}

clean() {
    rm -rf ${tmp_dir}
}

trap clean EXIT

# Create new key pairs.
psk=$(wg genpsk)
wg genkey | tee privatekey | wg pubkey > publickey

# Add new client block to server configuration file.
cat << EOF >> ${WORKDIR}/wireguard/wireguard.conf
[Peer]
# Client: ${1}
PublicKey = $(cat publickey)
PresharedKey = ${psk}
AllowedIPs = ${2}/32

EOF

# Create new client configuration file.
cat << EOF > ${WORKDIR}/wireguard/${1}.conf
[Interface]
Address = ${2}/24
PrivateKey = $(cat privatekey)
DNS = ${WG_DNS}

[Peer]
PublicKey = $(cat ${WORKDIR}/publickey)
PresharedKey = ${psk}
AllowedIPs = ${WG_ALLOWED_IPS}
Endpoint = ${WG_SERVER_IP}:${WG_SERVER_PORT}
EOF

