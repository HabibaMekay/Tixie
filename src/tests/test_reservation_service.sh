#!/bin/bash

# Exit on any error
set -e

# Base URLs for services
TICKET_SERVICE_URL="http://ticket-service-1:8082"
RESERVATION_SERVICE_URL="http://reservation-service-1:9081"
USER_SERVICE_URL="http://user-service-1:8081"
EVENT_SERVICE_URL="http://event-service-1:8080"

info() {
  echo -e "\n=== $1 ==="
}

error() {
  echo -e "\n[ERROR] $1" >&2
  exit 1
}

# Check if jq is installed, install if not
if ! command -v jq &> /dev/null; then
  echo "jq is not installed. Installing..."
  if command -v apt &> /dev/null; then
    sudo apt update && sudo apt install -y jq
  elif command -v yum &> /dev/null; then
    sudo yum install -y jq
  else
    error "Please install jq manually (e.g., 'sudo apt install jq' or 'brew install jq')"
  fi
fi

# Generate a random user email to avoid conflicts
RANDOM_PART=$((RANDOM % 10000))
USER_EMAIL="user${RANDOM_PART}@example.com"
USER_PASS="securepass123"

info "Register user: $USER_EMAIL"
REGISTER_RESPONSE=$(curl -s --max-time 5 -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" \
  "${USER_SERVICE_URL}/v1" || echo "ERROR")

if [[ "$REGISTER_RESPONSE" == "ERROR" ]]; then
  error "Failed to register user. Is the user service running at ${USER_SERVICE_URL}?"
fi
echo "REGISTER_RESPONSE: $REGISTER_RESPONSE"

info "Login user to get JWT"
LOGIN_RESPONSE=$(curl -s --max-time 5 -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" \
  "${USER_SERVICE_URL}/login" || echo "ERROR")

if [[ "$LOGIN_RESPONSE" == "ERROR" ]]; then
  error "Failed to login. Check user credentials or service at ${USER_SERVICE_URL}."
fi
echo "LOGIN_RESPONSE: $LOGIN_RESPONSE"

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token' 2>/dev/null)
if [[ -z "$ACCESS_TOKEN" || "$ACCESS_TOKEN" == "null" ]]; then
  error "Failed to obtain access_token! Response: $LOGIN_RESPONSE"
fi
echo "Got access_token: $ACCESS_TOKEN"

info "Create a ticket for an event"
CREATE_TICKET_RESPONSE=$(curl -s --max-time 5 -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{"event_id":1,"user_id":1}' \
  "${TICKET_SERVICE_URL}/v1" || echo "ERROR")

if [[ "$CREATE_TICKET_RESPONSE" == "ERROR" ]]; then
  error "Failed to create ticket. Is the ticket service running at ${TICKET_SERVICE_URL}?"
fi
echo "CREATE_TICKET_RESPONSE: $CREATE_TICKET_RESPONSE"

TICKET_ID=$(echo "$CREATE_TICKET_RESPONSE" | jq -r '.ticket_id' 2>/dev/null)
if [[ -z "$TICKET_ID" || "$TICKET_ID" == "null" ]]; then
  error "Could not extract ticket_id! Response: $CREATE_TICKET_RESPONSE"
fi
echo "Ticket ID: $TICKET_ID"

info "Reserve the ticket"
RESERVE_RESPONSE=$(curl -s --max-time 5 -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{\"ticket_id\":${TICKET_ID},\"user_id\":1}" \
  "${RESERVATION_SERVICE_URL}/v1" || echo "ERROR")

if [[ "$RESERVE_RESPONSE" == "ERROR" ]]; then
  error "Failed to reserve ticket. Is the reservation service running at ${RESERVATION_SERVICE_URL}?"
fi
echo "RESERVE_RESPONSE: $RESERVE_RESPONSE"

RESERVATION_ID=$(echo "$RESERVE_RESPONSE" | jq -r '.reservation_id' 2>/dev/null)
if [[ -z "$RESERVATION_ID" || "$RESERVATION_ID" == "null" ]]; then
  error "Could not extract reservation_id! Response: $RESERVE_RESPONSE"
fi
echo "Reservation ID: $RESERVATION_ID"

info "Polling reservation status until completed or failed"
while true; do
  RESERVATION_INFO=$(curl -s --max-time 5 -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    "${RESERVATION_SERVICE_URL}/v1/${RESERVATION_ID}" || echo "ERROR")

  if [[ "$RESERVATION_INFO" == "ERROR" ]]; then
    error "Failed to poll reservation status. Is the reservation service running? Note: GET /v1/:id route may be commented out in SetupRoutes."
  fi

  STATUS=$(echo "$RESERVATION_INFO" | jq -r '.status' 2>/dev/null)
  echo "Current reservation status: $STATUS"
  echo "$RESERVATION_INFO"

  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]]; then
    echo "Final reservation info:"
    echo "$RESERVATION_INFO" | jq .
    break
  fi

  sleep 2
done

info "Verify the reservation"
VERIFY_RESPONSE=$(curl -s --max-time 5 -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{\"reservation_id\":${RESERVATION_ID}}" \
  "${RESERVATION_SERVICE_URL}/v1/verify" || echo "ERROR")

if [[ "$VERIFY_RESPONSE" == "ERROR" ]]; then
  error "Failed to verify reservation. Is the reservation service running?"
fi
echo "VERIFY_RESPONSE: $VERIFY_RESPONSE"

info "Get ticket details to confirm status"
TICKET_DETAILS=$(curl -s --max-time 5 -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${TICKET_SERVICE_URL}/v1/${TICKET_ID}" || echo "ERROR")

if [[ "$TICKET_DETAILS" == "ERROR" ]]; then
  error "Failed to get ticket details. Is the ticket service running?"
fi
echo "TICKET_DETAILS: $TICKET_DETAILS"
FINAL_TICKET_STATUS=$(echo "$TICKET_DETAILS" | jq -r '.status' 2>/dev/null)
echo "Final ticket status: $FINAL_TICKET_STATUS"

info "Test completed."