# ADR: Mode of Integration and Communication Between Services

## Context

As our system grows and incorporates multiple microservices, we need a well-defined approach to service integration and communication. The choice of communication mode impacts system performance, reliability, and scalability.

Our architecture consists of various services such as authentication, payment, and booking, all interacting through an API Gateway. Additionally, background processes such as email notifications and user deletions require reliable and scalable event-driven mechanisms.

To ensure a seamless user experience and maintain system efficiency, we must determine when to use synchronous communication versus asynchronous communication. This decision will directly influence responsiveness, fault tolerance, and overall system resilience.

By defining clear integration patterns, we aim to balance immediate feedback for user interactions with the benefits of asynchronous event-driven processing for background tasks.

## Decision

We will use a combination of synchronous and asynchronous communication between services, depending on the nature of the interaction and the system’s requirements. The specific modes of integration and communication for each interaction as follows:

### User/Vendor → Client Application: Interacts with **[HTTP] (Synchronous)**

**Justification:** Users interact with the Client Application (ReactJS) via HTTP requests, requiring immediate feedback for responsiveness.

### Client Application → API-GATEWAY: Makes API request to **[JSON/HTTP] (Synchronous)**

**Justification:** The Client Application needs immediate responses for real-time user feedback.

### Client Application → Authentication Service: Makes API request to **[JSON/HTTP] (Synchronous)**

**Justification:** Immediate feedback is required during login authentication.

### API-GATEWAY → Authentication Service: Validates authentication **[JSON/HTTP] (Synchronous)**

**Justification:** Immediate response is required to authorize API requests.

### API-GATEWAY → Payment Service: Send payment details **[JSON/HTTP] (Synchronous)**

**Justification:** Immediate response is needed to confirm payment success or failure.

### API-GATEWAY → Booking Service: Makes request to **[JSON/HTTP] (Synchronous)**

**Justification:** The booking must be confirmed immediately after successful payment.

### API-GATEWAY → Apache Kafka: Receives the id of the user that needs to be deleted **[TCP] (Asynchronous)**

**Justification:** User deletion is a background process and benefits from Kafka’s publish-subscribe model.

### API-GATEWAY → Email System: Send email notifications **[JSON/HTTP] (Asynchronous)**

**Justification:** Email sending is a background task and does not require immediate feedback.

### Authentication Service → Email System: Send auth emails **[JSON/HTTP] (Asynchronous)**

**Justification:** Authentication-related emails (password reset, email verification) are non-blocking background tasks.

### Authentication Service → Database (Auth): Reads from and writes to **[SQL] (Synchronous)**

**Justification:** Immediate response is required for authentication validation and data consistency.

### Payment Service → Database (Payment): Reads from and writes to **[SQL] (Synchronous)**

**Justification:** Ensures reliable transactional access to payment records.

### Booking Service → Database (Booking): Reads from and writes to **[SQL] (Synchronous)**

**Justification:** Ensures data consistency for event and ticket records.

## Consequences

### Synchronous Communication (HTTP/JSON, SQL)

**Pros:**

- Immediate feedback for user-facing operations.
- Simplifies implementation of request-response workflows.

**Cons:**

- Potential latency if a service is slow or unavailable.
- Requires handling of timeouts and retries.

### Asynchronous Communication (Kafka, Email Notifications)

**Pros:**

- Decouples services, allowing independent scaling and fault tolerance.
- Ensures reliability for background tasks.

**Cons:**

- Adds complexity in tracking and debugging workflows.
- Requires additional infrastructure (e.g., Kafka brokers).

## 5. Alternatives Considered

### Fully Synchronous Communication

**Rejected Because:**

- Would tightly couple services and block user-facing operations.
- Would degrade scalability and responsiveness.

### Fully Asynchronous Communication

**Rejected Because:**

- Would complicate user-facing operations that require immediate feedback.
- Users expect immediate confirmation for transactions like ticket purchases.

## Rationale

The combination of synchronous and asynchronous communication balances the system’s requirements:

- **Synchronous for User-Facing Operations:** Ensures immediate feedback for transactions and authentication.
- **Asynchronous for Background Tasks:** Decouples services for scalability and reliability.
- **Microservices Best Practices:** Aligns with best practices by using event-driven architecture where appropriate.
