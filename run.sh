#!/bin/bash

set -e 

echo "building..."
go build

echo "running..."
./api www/space-api.json www/auth-token.txt www/last-seen.txt :8081
