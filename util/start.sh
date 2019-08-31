#!/usr/bin/env bash

set -e

docker-compose down
docker volume rm giveanet_app-static || true
docker-compose up
