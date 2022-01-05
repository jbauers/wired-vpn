#!/bin/bash -e
# Bypass auth proxy during testing.
docker exec server_control_1 \
	curl -v \
	  -H "X-Wired-User: test0@example.com" \
	  -H "X-Wired-Group: Infrastructure" \
	  -H "X-Wired-Public-Key: 9TKwZcutg7jaL0CGKj+LhKrSfvTGigfO9AwULMBRu0E=" \
	control:9000
sleep 1
docker exec server_control_1 \
	curl -v \
	  -H "X-Wired-User: test1@example.com" \
	  -H "X-Wired-Group: Infrastructure" \
	  -H "X-Wired-Public-Key: $(wg genkey)" \
	control:9000
sleep 1
docker exec server_control_1 \
	curl -v \
	  -H "X-Wired-User: test2@example.com" \
	  -H "X-Wired-Group: Marketing" \
	  -H "X-Wired-Public-Key: WFNaRV5UC9FN0rVMCo3qyRctz64SXuDlAgqsPIuJsmY=" \
	control:9000
sleep 1

docker exec server_vpn0_1 wg
echo "---------------------"
docker exec server_vpn1_1 wg
