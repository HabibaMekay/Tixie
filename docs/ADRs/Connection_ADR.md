# ADR: Connection Decision

## Context
Our application requires a connection that ensures real-time updates while maintaining efficiency. The solution must prevent double booking when two users attempt to book the same ticket simultaneously.

## Decision
We have decided to use WebSockets.

## Alternatives Considered
1. **Server-Sent Events (SSEs)**
- **Pros:**
  - Efficient for unidirectional data flow (server to client)
  - Lower overhead compared to WebSockets
- **Cons:**
  - Limited to one-way communication
  - Not ideal for bidirectional real-time interactions

2. **Long Polling**
- **Pros:**
  - Works with legacy systems and older browsers
  - Easier to implement than WebSockets
- **Cons:**
  - Higher resource consumption due to frequent HTTP requests
  - Increased server load and latency compared to WebSockets

## Consequences
- **Positive:**
  - Stateful communication
  - Real-time updates
  - Uses fewer resources compared to several other solutions

## Rationale
WebSockets provide a reliable, real-time connection while efficiently maintaining session state. This ensures immediate updates and prevents double booking.