#/usr/bin/env sh

# "./run.sh" to spin up a fresh environment during
# development. Use "docker-compose up" if persistence
# is wanted.

down() {
	docker-compose down
}

trap down TERM SIGINT

docker-compose up --build --force-recreate -V
