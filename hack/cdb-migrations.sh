#!/bin/bash
set -e

NAME=$1
HOST=$2

echo "CREATE DATABASE IF NOT EXISTS $1; USE $1;" > /tmp/migrations.sql
cat /migrations/*.up* >> /tmp/migrations.sql
until cat /tmp/migrations.sql | ./cockroach sql --insecure --echo-sql --host $HOST; do 
	sleep 5; 
done

