#!/usr/bin/env bash

set -e

[ -z "$1" ] && echo "Need template path" && exit 1

EPOCH=$(date +'%s')

# Replace "CACHEBUSTER" with build timestamp
find "$1" -type f -exec sed -i "s/CACHEBUSTER/${EPOCH}/g" {} +
