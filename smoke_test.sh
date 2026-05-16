#!/usr/bin/env bash
# Smoke test - hits every provider end-to-end with curl.
# Run `./nimbus &` (or `docker compose up -d`) first.
set -e
BASE=${BASE:-http://localhost:4566}

echo "=== Health ==="
curl -s $BASE/_nimbus/health
curl -s $BASE/_nimbus/providers

echo
echo "=== AWS S3 ==="
curl -s -X PUT "$BASE/mybucket" -H "x-amz-content-sha256: x"
curl -s -X PUT "$BASE/mybucket/hello.txt" -H "x-amz-content-sha256: x" --data "hello world"
curl -s "$BASE/mybucket/hello.txt" -H "x-amz-content-sha256: x"
echo

echo "=== AWS DynamoDB ==="
curl -s -X POST $BASE/ \
  -H "X-Amz-Target: DynamoDB_20120810.PutItem" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -d '{"TableName":"users","Item":{"id":{"S":"1"},"name":{"S":"alice"}}}'
echo
curl -s -X POST $BASE/ \
  -H "X-Amz-Target: DynamoDB_20120810.GetItem" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -d '{"TableName":"users","Key":{"id":{"S":"1"}}}'
echo

echo "=== GCP Storage ==="
curl -s -X POST "$BASE/upload/storage/v1/b/mybucket/o?name=hi.txt" --data "hi from gcp"
curl -s "$BASE/storage/v1/b/mybucket/o/hi.txt?alt=media"
echo

echo "=== Azure Blob ==="
curl -s -X PUT "$BASE/mycontainer/hi.txt" -H "x-ms-version: 2020-04-08" --data "hi from azure"
curl -s "$BASE/mycontainer/hi.txt" -H "x-ms-version: 2020-04-08"
echo
echo "=== Done ==="
