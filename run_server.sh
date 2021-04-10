#/usr/bin/env sh

# "./run.sh" to spin up a fresh environment during
# development. Use "docker-compose up" if persistence
# is wanted.

down() {
	docker-compose down
	popd
}

trap down TERM SIGINT

pushd server
docker-compose up --build --force-recreate -V
