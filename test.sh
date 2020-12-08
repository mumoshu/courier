#!/usr/bin/env bash

curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" \
  -d "{\"RequestType\":\"Create\",\"ResponseURL\":\"${RESPONSE_URL}\",\"ResourceProperties\":{\"Echo\":\"Hello World!\"}}"
