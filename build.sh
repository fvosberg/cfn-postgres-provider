#!/bin/bash
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

CWD=$(pwd)

set -e

VERBOSE=false

while test $# -gt 0; do
  case "$1" in
    -v|--verbose)
      shift
      VERBOSE=true
      ;;

    --out)
      shift
      if test $# -gt 0; then
        OUTPUT_PATH="$1"
        #OUTPUT_PATH="$(cd "$(dirname "$1")" >/dev/null 2>&1 && pwd)/${1}"
      else
        echo -e "\n\nNo output path specified\n\n"
        exit 1
      fi
      shift
      ;;

    *)
      echo -e "\nUnknown arg $1\n"
      exit 1
      break
      ;;
  esac
done

if [ -z "${OUTPUT_PATH}" ]; then
  OUTPUT_PATH="$(mktemp -d -t 'postgres-provisioner-lambda.XXXXX')/function.zip"
fi
OUTPUT_DIR=$(dirname ${OUTPUT_PATH})

if [ "$VERBOSE" = true ]; then
  echo -e "\nBuilding lambda package in: ${OUTPUT_DIR}\n"
fi

cd "${ROOT}"

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${OUTPUT_DIR}/main" .
zip -qq -j "${OUTPUT_PATH}" "${OUTPUT_DIR}/main"

rm "${OUTPUT_DIR}/main"

echo "${OUTPUT_PATH}"
