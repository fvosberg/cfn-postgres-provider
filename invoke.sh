#!/bin/bash

STACK_NAME="thesis"
LAMBDA_NAME="${STACK_NAME}-postgres-db-provider"

if [ ! -f "payload.json" ]; then
  echo -e "\nMissing payload.json. You can copy it from the cloudwatch logs, which log the request payload\n"
  exit 1
fi

aws lambda invoke --function-name "${LAMBDA_NAME}" \
  --payload "$(cat payload.json)" \
  out \
  --log-type Tail \
  --output text \
  --query "LogResult" \
  | base64 -d
