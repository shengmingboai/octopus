#!/bin/bash

set -e

if [ -z "$ONEPANEL_URL" ]; then
    echo "::error::ONEPANEL_URL is not set"
    exit 1
fi

if [ -z "$ONEPANEL_API_KEY" ]; then
    echo "::error::ONEPANEL_API_KEY is not set"
    exit 1
fi

if [ -z "$IMAGE" ]; then
    echo "::error::IMAGE is not set"
    exit 1
fi

if [ -z "$CONTAINER_NAME" ]; then
    echo "::error::CONTAINER_NAME is not set"
    exit 1
fi

TASK_ID=${TASK_ID:-"unknown"}

TIMESTAMP=$(date +%s)
TOKEN=$(echo -n "1panel${ONEPANEL_API_KEY}${TIMESTAMP}" | md5sum | awk '{print $1}')

echo "Triggering 1Panel upgrade..."
echo "  Panel: ${ONEPANEL_URL}"
echo "  Image: ${IMAGE}"
echo "  Container: ${CONTAINER_NAME}"
echo "  TaskID: ${TASK_ID}"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${ONEPANEL_URL}/api/v2/upgrade/upgrade" \
    -H "1Panel-Token: ${TOKEN}" \
    -H "1Panel-Timestamp: ${TIMESTAMP}" \
    -H "Content-Type: application/json" \
    -d "{
        \"forcePull\": true,
        \"image\": \"${IMAGE}\",
        \"names\": [\"${CONTAINER_NAME}\"],
        \"taskID\": \"${TASK_ID}\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "Response (${HTTP_CODE}): ${BODY}"

if [ "$HTTP_CODE" -ne 200 ]; then
    echo "::error::1Panel API returned HTTP ${HTTP_CODE}"
    exit 1
fi

echo "1Panel upgrade triggered successfully"
