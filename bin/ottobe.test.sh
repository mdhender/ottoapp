#!/bin/bash
# test some ottobe routes

set -e

PORT=29631
URL="http://localhost:${PORT}"

echo "🔍 Testing /api/health endpoint..."
curl -s "${URL}/api/health" | jq .
echo

echo "🔐 Testing /api/auth/login with demo credentials..."
LOGIN_RESPONSE=$(curl -s -X POST "${URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"demo"}')

echo "Login response:"
echo "${LOGIN_RESPONSE}" | jq .
echo

# Extract token from login response
TOKEN=$(echo "${LOGIN_RESPONSE}" | jq -r '.token')
USERID=$(echo "${LOGIN_RESPONSE}" | jq -r '.userId')
CLAN=$(echo "${LOGIN_RESPONSE}" | jq -r '.clan')

if [ "${TOKEN}" == "null" ] || [ -z "${TOKEN}" ]; then
  echo "❌ Login failed, no token received"
  exit 1
fi

echo "✅ Login successful. Token received for user ID: ${USERID} (Clan: ${CLAN})"
echo

echo "👤 Testing /api/auth/user endpoint with token..."
USER_RESPONSE=$(curl -s "${URL}/api/auth/user" \
  -H "Authorization: Bearer ${TOKEN}")

echo "User info response:"
echo "${USER_RESPONSE}" | jq .
echo

USER_EMAIL=$(echo "${USER_RESPONSE}" | jq -r '.email')
if [ "${USER_EMAIL}" == "null" ] || [ -z "${USER_EMAIL}" ]; then
  echo "❌ Getting user info failed"
  exit 1
fi

echo "✅ Successfully retrieved user info for: ${USER_EMAIL}"
echo

echo "📂 Testing /api/data endpoint with token..."
DATA_RESPONSE=$(curl -s "${URL}/api/data" \
  -H "Authorization: Bearer ${TOKEN}")

echo "User data response:"
echo "${DATA_RESPONSE}" | jq .
echo

DATA_PATH=$(echo "${DATA_RESPONSE}" | jq -r '.path')
if [ "${DATA_PATH}" == "null" ] || [ -z "${DATA_PATH}" ]; then
  echo "❌ Getting user data failed"
  exit 1
fi

echo "✅ Successfully retrieved data path: ${DATA_PATH}"
echo

echo "📊 Testing /api/data/turn endpoint with token..."
TURN_RESPONSE=$(curl -s "${URL}/api/data/turn?year=2025&month=5" \
  -H "Authorization: Bearer ${TOKEN}")

echo "Turn data response:"
echo "${TURN_RESPONSE}" | jq .
echo

TURN_EXISTS=$(echo "${TURN_RESPONSE}" | jq -r '.exists')
echo "Turn data exists: ${TURN_EXISTS}"
echo

echo "✅ All tests completed successfully!"
exit 0
