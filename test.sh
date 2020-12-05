#/bin/bash

i=0
while true; do
	# sleep 1
	echo ""
	echo "test$i@example.com"
	echo ""
	docker exec wired-vpn_backend_1 curl -H "X-Wired-User: test$i@example.com" http://localhost:9000
	((i++))
done
