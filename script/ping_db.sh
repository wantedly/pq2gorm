#!/bin/bash

docker-compose exec db psql -U postgres -d test -c 'select 1;' 2>&1 > /dev/null

if [[ $? -eq 0 ]]; then
  echo "Connection established."
  exit 0
fi

for i in `seq 1 5`; do
  echo "Wait for 5 seconds..."
  sleep 5

  docker-compose exec db psql -U postgres -d test -c 'select 1;' 2>&1 > /dev/null

  if [[ $? -eq 0 ]]; then
    echo "Connection established."
    exit 0
  fi
done

echo "Failed to connect to database."
exit 1
