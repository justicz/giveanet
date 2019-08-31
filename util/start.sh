#!/usr/bin/env bash

set -e

docker-compose down
docker volume rm millionnets_app-static || true
docker-compose up
