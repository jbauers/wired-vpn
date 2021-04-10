#!/bin/bash -e
export HTTP_ENDPOINT="$(jq -r '.oidc.http_endpoint' ./server/settings.json)"

fyne_cross="$(go env GOPATH)/bin/fyne-cross"
$fyne_cross version
pushd client
$fyne_cross linux -ldflags "-X main.endpoint=${HTTP_ENDPOINT}" .
$fyne_cross windows -ldflags "-X main.endpoint=${HTTP_ENDPOINT}" .
popd

cp -f client/fyne-cross/bin/linux-amd64/client wired
cp -f client/fyne-cross/bin/windows-amd64/client.exe wired.exe

cleanup() {
	sudo ip link del dev wired0 type wireguard
}

trap cleanup SIGINT EXIT

sudo ip link add dev wired0 type wireguard

# CAP_NET_ADMIN to allow configuring WireGuard interfaces.
# CAP_NET_RAW to allow the tenus lib to ping.
sudo setcap cap_net_admin,cap_net_raw+ep ./wired

./wired
