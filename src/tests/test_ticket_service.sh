#!/bin/bash

# Exit on any error
set -e

# Base URLs for services
RESERVE_SERVICE_URL="http://reservation-service-1:9081"
USER_SERVICE_URL="http://user-service-1:8081"
TICKET_SERVICE_URL="http://event-service-1:8080"

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

info "Create a ticket for event 1"
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

info "Get ticket details by ID"
GET_TICKET_RESPONSE=$(curl -s --max-time 5 -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${TICKET_SERVICE_URL}/v1/${TICKET_ID}" || echo "ERROR")

if [[ "$GET_TICKET_RESPONSE" == "ERROR" ]]; then
  error "Failed to get ticket. Is the ticket service running?"
fi
echo "GET_TICKET_RESPONSE: $GET_TICKET_RESPONSE"

info "Update ticket status to 'used'"
UPDATE_STATUS_RESPONSE=$(curl -s --max-time 5 -X PUT -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{"status":"used"}' \
  "${TICKET_SERVICE_URL}/v1/${TICKET_ID}/status" || echo "ERROR")

if [[ "$UPDATE_STATUS_RESPONSE" == "ERROR" ]]; then
  error "Failed to update ticket status. Is the ticket service running?"
fi
echo "UPDATE_STATUS_RESPONSE: $UPDATE_STATUS_RESPONSE"

info "Get tickets by event ID"
GET_TICKETS_RESPONSE=$(curl -s --max-time 5 -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${TICKET_SERVICE_URL}/v1?event_id=1" || echo "ERROR")

if [[ "$GET_TICKETS_RESPONSE" == "ERROR" ]]; then
  error "Failed to get tickets by event. Is the ticket service running?"
fi
echo "GET_TICKETS_RESPONSE: $GET_TICKETS_RESPONSE"

info "Get events with tickets"
GET_EVENTS_RESPONSE=$(curl -s --max-time 5 -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${TICKET_SERVICE_URL}/v1/events-with-tickets" || echo "ERROR")

if [[ "$GET_EVENTS_RESPONSE" == "ERROR" ]]; then
  error "Failed to get events with tickets. Is the ticket service running?"
fi
echo "GET_EVENTS_RESPONSE: $GET_EVENTS_RESPONSE"

info "Get ticket by code (using ticket code from creation)"
TICKET_CODE=$(echo "$CREATE_TICKET_RESPONSE" | jq -r '.ticket_code' 2>/dev/null)
if [[ -z "$TICKET_CODE" || "$TICKET_CODE" == "null" ]]; then
  error "Could not extract ticket_code! Response: $CREATE_TICKET_RESPONSE"
fi
GET_CODE_RESPONSE=$(curl -s --max-time 5 -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${TICKET_SERVICE_URL}/v1/verify/${TICKET_CODE}" || echo "ERROR")

if [[ "$GET_CODE_RESPONSE" == "ERROR" ]]; then
  error "Failed to get ticket by code. Is the ticket service running?"
fi
echo "GET_CODE_RESPONSE: $GET_CODE_RESPONSE"

info "Attempt WebSocket connection for events with tickets"
# Note: WebSocket testing is limited in shell; this initiates a connection
wscat -c "${TICKET_SERVICE_URL}/v1/ws/events-with-tickets" --auth "Bearer ${ACCESS_TOKEN}" || {
  echo "WebSocket connection failed or wscat not installed. Install wscat (npm install -g wscat) for full testing."
}

info "Test completed."