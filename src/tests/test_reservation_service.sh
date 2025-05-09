#!/bin/bash

# Exit on any error
set -e

# Base URLs for services
TICKET_SERVICE_URL=http://ticket-service-1:8082
RESERVE_SERVICE_URL=http://reservation-service-1:9081
USER_SERVICE_URL=http://user-service-1:8081

# TICKET_SERVICE_URL="http://localhost:8080"
# RESERVATION_SERVICE_URL="http://localhost:8082"
# USER_SERVICE_URL="http://localhost:8081"

info() {
  echo -e "\n=== $1 ==="
}

# Generate a random user email to avoid conflicts
RANDOM_PART=$((RANDOM % 10000))
USER_EMAIL="user${RANDOM_PART}@example.com"
USER_PASS="securepass123"chmod +x test_reservation_service.sh

info "Register user: $USER_EMAIL"
REGISTER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" \
  "${USER_SERVICE_URL}/auth/register")

echo "REGISTER_RESPONSE: $REGISTER_RESPONSE"

info "Login user to get JWT"
LOGIN_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" \
  "${USER_SERVICE_URL}/auth/login")

echo "LOGIN_RESPONSE: $LOGIN_RESPONSE"

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
if [[ -z "$ACCESS_TOKEN" || "$ACCESS_TOKEN" == "null" ]]; then
  echo "ERROR: Failed to obtain access_token!"
  exit 1
fi
echo "Got access_token: $ACCESS_TOKEN"

info "Create a ticket for an event"
CREATE_TICKET_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{"event_id":1,"user_id":1}' \
  "${TICKET_SERVICE_URL}/tickets")

echo "CREATE_TICKET_RESPONSE: $CREATE_TICKET_RESPONSE"

TICKET_ID=$(echo "$CREATE_TICKET_RESPONSE" | jq -r '.ticket_id')
if [[ -z "$TICKET_ID" || "$TICKET_ID" == "null" ]]; then
  echo "ERROR: Could not extract ticket_id!"
  exit 1
fi
echo "Ticket ID: $TICKET_ID"

info "Reserve the ticket"
RESERVE_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{\"ticket_id\":${TICKET_ID},\"user_id\":1}" \
  "${RESERVATION_SERVICE_URL}/reservations")

echo "RESERVE_RESPONSE: $RESERVE_RESPONSE"

RESERVATION_ID=$(echo "$RESERVE_RESPONSE" | jq -r '.reservation_id')
if [[ -z "$RESERVATION_ID" || "$RESERVATION_ID" == "null" ]]; then
  echo "ERROR: Could not extract reservation_id!"
  exit 1
fi
echo "Reservation ID: $RESERVATION_ID"

info "Polling reservation status until completed or failed"
while true; do
  RESERVATION_INFO=$(curl -s -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    "${RESERVATION_SERVICE_URL}/reservations/${RESERVATION_ID}")

  STATUS=$(echo "$RESERVATION_INFO" | jq -r '.status')
  echo "Current reservation status: $STATUS"
  echo "$RESERVATION_INFO"

  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]]; then
    echo "Final reservation info:"
    echo "$RESERVATION_INFO" | jq .
    break
  fi

  sleep 2
done

info "Get ticket details to confirm status"
TICKET_DETAILS=$(curl -s -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${TICKET_SERVICE_URL}/tickets/${TICKET_ID}")

echo "TICKET_DETAILS: $TICKET_DETAILS"
FINAL_TICKET_STATUS=$(echo "$TICKET_DETAILS" | jq -r '.status')
echo "Final ticket status: $FINAL_TICKET_STATUS"