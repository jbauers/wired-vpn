#/bin/bash

i=0
while true; do
	# sleep 1
	echo ""
	echo "test$i@example.com"
	echo ""
	docker exec wired-vpn_backend_1 curl -H "X-Wired-User: test$i@example.com" -H "X-Wired-Group: Engineering" http://localhost:9000
	((i++))
	docker exec wired-vpn_backend_1 curl -H "X-Wired-User: test$i@example.com" -H "X-Wired-Group: Sales" http://localhost:9000
	((i++))
done
