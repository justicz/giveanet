#!/usr/bin/env bash

set -e

echo "Waiting for app server to start up to run integration tests"
while true
do
  STATUS=$(curl -s -o /dev/null -w '%{http_code}' http://millionnets-single-proxy/health) || true
  if [ $STATUS -eq 200 ]; then
    break
  else
    printf '.'
    sleep 1
  fi
done

go test -v -mod=vendor github.com/justicz/giveanet/test/integration/
