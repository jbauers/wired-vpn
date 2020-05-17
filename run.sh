#/usr/bin/env sh

down() {
	docker-compose down
}

trap down TERM SIGINT

docker-compose up --build --force-recreate -V
