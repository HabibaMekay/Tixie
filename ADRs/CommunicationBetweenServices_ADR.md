# ADR: Mode of Integration and Communication Between Services

## Context

As our system grows and incorporates multiple microservices, we need a well-defined approach to service integration and communication. The choice of communication mode impacts system performance, reliability, and scalability.

Our architecture consists of various services such as authentication, payment, and booking, all interacting through an API Gateway. Additionally, background processes such as email notifications require reliable and scalable event-driven mechanisms.

To ensure a seamless user experience and maintain system efficiency, we must determine when to use synchronous communication versus asynchronous communication. This decision will directly influence responsiveness, fault tolerance, and overall system resilience.

By defining clear integration patterns, we aim to balance immediate feedback for user interactions with the benefits of asynchronous event-driven processing for background tasks.

---

## Decision

We will use a combination of **synchronous** and **asynchronous** communication between services, depending on the nature of the interaction and the system’s requirements. The specific modes of integration and communication for each interaction are as follows:

### User/Vendor → Client Application: Interacts with [HTTP] (Synchronous)

**Justification:** Users interact with the Client Application (ReactJS) via HTTP requests, requiring immediate feedback for responsiveness.

### Client Application → API-GATEWAY: Makes API request to [JSON/HTTP] (Synchronous)

**Justification:** The Client Application needs immediate responses for real-time user feedback (e.g., ticket purchase status, event submission).

### Client Application → Authentication Service: Makes API request to [JSON/HTTP] (Synchronous)

**Justification:** Immediate feedback is required during login and authentication (e.g., OAuth2 token issuance).

### API-GATEWAY → Authentication Service: Validates authentication [JSON/HTTP] (Synchronous)

**Justification:** Immediate response is required to authorize API requests before routing them to other microservices.

### API-GATEWAY → Payment Service: Sends payment details [JSON/HTTP] (Synchronous)

**Justification:** Immediate response is needed to confirm payment success or failure during ticket purchases.

### API-GATEWAY → Booking Service: Makes request to [JSON/HTTP] (Synchronous)

**Justification:** The booking must be confirmed immediately after a successful payment to provide real-time feedback to the user.

### Authentication Service → RabbitMQ: Sends UserRegistered message [Event Published] (Asynchronous)

**Justification:** When a new user registers, the Authentication Service publishes a **UserRegistered** event to RabbitMQ. This event triggers background tasks like sending a welcome email, ensuring **decoupling and scalability** without blocking user interactions.

### Payment Service → RabbitMQ: Sends PaymentSuccess message [Event Published] (Asynchronous)

**Justification:** After a successful payment, the Payment Service publishes a **PaymentSuccess** event to RabbitMQ. This event triggers background tasks like sending a payment receipt email, ensuring **decoupling and non-blocking behavior**.

### Booking Service → RabbitMQ: Sends BookingConfirmed message [Event Published] (Asynchronous)

**Justification:** After a booking is confirmed, the Booking Service publishes a **BookingConfirmed** event to RabbitMQ. This event triggers background tasks like sending a **booking confirmation email**, ensuring **decoupling**.

### RabbitMQ → Email System: Sends PaymentSuccess/BookingConfirmed/UserRegistered [Event Consumed] (Asynchronous)

**Justification:** The Email System consumes events from RabbitMQ to send email notifications for **user registration, payment success, and booking confirmations**. This centralizes event-driven notifications through RabbitMQ, ensuring **decoupling and reliability** for background tasks.

### Authentication Service → Database (Auth): Reads from and writes to [SQL] (Synchronous)

**Justification:** Immediate response is required for authentication validation (e.g., user login, token issuance) and **data consistency**.

### Payment Service → Database (Payment): Reads from and writes to [SQL] (Synchronous)

**Justification:** Ensures **reliable transactional access** to payment records for immediate confirmation.

### Booking Service → Database (Booking): Reads from and writes to [SQL] (Synchronous)

**Justification:** Ensures **data consistency** for event and ticket records during booking operations.

---

## Consequences

### Synchronous Communication (HTTP/JSON, SQL)

**Pros:**

- Immediate feedback for user-facing operations (e.g., login, ticket purchases).
- Simplifies implementation of request-response workflows.

**Cons:**

- Potential latency if a service is slow or unavailable, blocking the workflow.
- Requires careful handling of timeouts and retries to maintain reliability.

### Asynchronous Communication (RabbitMQ)

**Pros:**

- **Decouples services**, allowing independent scaling and fault tolerance (e.g., email notifications).
- **Ensures reliability** for background tasks by persisting events in RabbitMQ until processed.
- **Supports an event-driven architecture**, centralizing event handling through RabbitMQ.

**Cons:**

- Adds complexity in **tracking and debugging event-driven workflows**.
- Requires **additional infrastructure** (e.g., RabbitMQ brokers) and monitoring to manage event throughput.

---

## Alternatives Considered

### Fully Synchronous Communication

**Rejected Because:**

- Would tightly **couple services** and block user-facing operations, degrading **scalability and responsiveness**.
- For example, waiting for an **email to be sent synchronously** would delay the user registration response.

### Fully Asynchronous Communication

**Rejected Because:**

- Would **complicate user-facing operations** that require **immediate feedback**.
- For example, users expect **immediate confirmation** for transactions like ticket purchases, not delayed responses.

---

## Rationale

The combination of **synchronous and asynchronous communication** balances the system’s requirements:

- **Synchronous for User-Facing Operations:** Ensures **immediate feedback** for critical interactions like authentication, payment processing, and booking confirmations, providing a **responsive user experience**.
- **Asynchronous for Background Tasks:** Decouples services for **scalability and reliability**, using **RabbitMQ** to handle event-driven notifications (e.g., **UserRegistered, PaymentSuccess, BookingConfirmed**).
- **Microservices Best Practices:** Aligns with best practices by using **synchronous communication** for request-response workflows and **asynchronous event-driven communication** for background tasks, ensuring **both responsiveness and scalability**.
