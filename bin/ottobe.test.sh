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

# Test admin functionality with admin credentials
echo "👑 Testing admin functionality..."
echo "🔐 Logging in as admin..."
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "${URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin"}')

echo "Admin login response:"
echo "${ADMIN_LOGIN_RESPONSE}" | jq .
echo

# Extract admin token
ADMIN_TOKEN=$(echo "${ADMIN_LOGIN_RESPONSE}" | jq -r '.token')

if [ "${ADMIN_TOKEN}" == "null" ] || [ -z "${ADMIN_TOKEN}" ]; then
  echo "❌ Admin login failed, skipping admin tests"
else
  echo "✅ Admin login successful"
  echo
  
  echo "🔄 Testing route logging toggle..."
  TOGGLE_RESPONSE=$(curl -s -X POST "${URL}/api/admin/debug/log-all-routes" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}")
  
  echo "Toggle response:"
  echo "${TOGGLE_RESPONSE}" | jq .
  echo
  
  LOGGING_STATUS=$(echo "${TOGGLE_RESPONSE}" | jq -r '.logging')
  if [ "${LOGGING_STATUS}" == "null" ] || [ -z "${LOGGING_STATUS}" ]; then
    echo "❌ Route logging toggle failed"
  else
    echo "✅ Route logging ${LOGGING_STATUS}"
    
    # Toggle it back
    echo "🔄 Toggling route logging back..."
    TOGGLE_BACK_RESPONSE=$(curl -s -X POST "${URL}/api/admin/debug/log-all-routes" \
      -H "Authorization: Bearer ${ADMIN_TOKEN}")
    
    LOGGING_STATUS=$(echo "${TOGGLE_BACK_RESPONSE}" | jq -r '.logging')
    echo "✅ Route logging now ${LOGGING_STATUS}"
  fi
fi

echo

echo "📮 Testing /api/version endpoint..."
VERSION_RESPONSE=$(curl -s "${URL}/api/version")

echo "Version response:"
echo "${VERSION_RESPONSE}" | jq .
echo

VERSION_STRING=$(echo "${VERSION_RESPONSE}" | jq -r '.version')
if [ "${VERSION_STRING}" == "null" ] || [ -z "${VERSION_STRING}" ]; then
  echo "❌ Version endpoint failed"
  exit 1
fi

echo "✅ Server version: ${VERSION_STRING}"
echo

echo "✅ All tests completed successfully!"
exit 0
