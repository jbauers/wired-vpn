#/bin/bash

i=0
while true; do
	# sleep 1
	echo ""
	echo "test$i@example.com"
	echo ""
	docker exec wired-vpn_frontend_1 curl -H "Authenticated-User: test$i@example.com" http://localhost:3000/frontend
	((i++))
done
