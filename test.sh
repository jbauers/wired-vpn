#!/bin/bash -e
# Bypass auth proxy during testing.
docker exec server_api_1 \
	curl -v \
	  -H "X-Wired-User: test@example.com" \
	  -H "X-Wired-Group: Infrastructure" \
	  -H "X-Wired-Public-Key: 9TKwZcutg7jaL0CGKj+LhKrSfvTGigfO9AwULMBRu0E=" \
	api:9000
