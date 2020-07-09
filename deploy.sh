#!/bin/bash
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# TODO one mode with deploying own stack with policies and lambda and passing subnets and security group as parameters
# TODO one mode providing ROLE_ARN, SUBNET_IDS and SECURITY_GROUP

# TODO jq is installed
# TODO options
STACK_NAME="thesis2"
LAMBDA_NAME="${STACK_NAME}-postgres-db-provider"
# thesis2
SUBNET_IDS="subnet-0f4f56fbaa07eeb81,subnet-09875bf7a54a32008"
SECURITY_GROUP_IDS="sg-0b1b018ecb9bea54f" # databaseaccesssg


set -e

echo -e "\nDeploying the lambda policy\n"
#aws cloudformation deploy \
  #--template-file "${ROOT}/cloudformation/stack.yaml" \
  #--stack-name "${STACK_NAME}" \
  #--capabilities CAPABILITY_IAM \
  #--no-fail-on-empty-changeset


ZIP_PATH=$("${ROOT}/build.sh")

ROLE_RESOURCE_ID="thesis2-CustomProviders-DT45RHP2F6Z5-LambdaRole-RAV6JBNHGXR"
#ROLE_RESOURCE_ID=$(aws cloudformation describe-stack-resource \
  #--stack-name "${STACK_NAME}" \
  #--logical-resource-id LambdaRole \
  #| jq -r ".StackResourceDetail.PhysicalResourceId")

ROLE_ARN=$( aws iam get-role \
  --role-name "${ROLE_RESOURCE_ID}" \
  | jq -r ".Role.Arn")

echo -e "\nUsing role with ARN: ${ROLE_ARN}\n"

echo -e "\nDeploying lambda function: ${LAMBDA_NAME}\n"

set +e
aws lambda get-function \
  --function-name "${LAMBDA_NAME}" >/dev/null 2>&1
RES=$?
set -e

if [ $RES -eq 0 ]; then

  echo -e "\nUpdating code of the lambda: ${LAMBDA_NAME}\n"
  aws lambda update-function-code \
    --function-name "${LAMBDA_NAME}" \
    --zip-file "fileb://${ZIP_PATH}" \
    --publish

  echo -e "\nUpdating the config of the lambda: ${LAMBDA_NAME}\n"
  aws lambda update-function-configuration \
    --function-name "${LAMBDA_NAME}" \
    --timeout 8 \
    --role "${ROLE_ARN}" \
    --vpc-config "SubnetIds=${SUBNET_IDS},SecurityGroupIds=${SECURITY_GROUP_IDS}"

else

  echo -e "\nCreating Lambda: ${LAMBDA_NAME}\n"


  aws lambda create-function \
    --function-name "${LAMBDA_NAME}" \
    --timeout 8 \
    --runtime go1.x \
    --zip-file "fileb://${ZIP_PATH}" \
    --handler main \
    --role "${ROLE_ARN}" \
    --vpc-config "SubnetIds=${SUBNET_IDS},SecurityGroupIds=${SECURITY_GROUP_IDS}"


fi

echo -e "\nDeleting lambda build artifacts in ${TMP}\n"

rm -R $(dirname "${ZIP_PATH}")
