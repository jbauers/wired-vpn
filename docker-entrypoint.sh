#!/bin/sh

source "${WORKDIR}/.env"
settings_file="${WORKDIR}/saml/settings.json"

sed -i "s/SERVER_PROTO/${SERVER_PROTO}/g" $settings_file
sed -i "s/SERVER_HOSTNAME/${SERVER_HOSTNAME}/g" $settings_file

sed -i "s/ONELOGIN_DOMAIN/${ONELOGIN_DOMAIN}/g" $settings_file
sed -i "s/ONELOGIN_CONNECTOR_ID/${ONELOGIN_CONNECTOR_ID}/g" $settings_file
sed -i "s/ONELOGIN_CONNECTOR_CERT/${ONELOGIN_CONNECTOR_CERT}/g" $settings_file

cat $settings_file

exec python -m flask run --host=0.0.0.0
