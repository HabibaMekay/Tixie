# Gateway Service

This is a centralized API Gateway for our microservices architecture. It serves as the single entry point to route, throttle, and monitor traffic between clients and backend services.

The gateway uses **Redis-based middleware** to enforce three layers of request management — all based on the **authenticated user's JWT token**, not the IP address:

- **Concurrency Limiting**: Prevents a user from flooding the system with simultaneous requests.
- **Throttling**: Restricts how frequently a user can make requests in a short time window.
- **Sliding Window Rate Limiting**: Caps the total number of requests over a longer period.

### JWT-Aware Middleware
Requests must include a valid **JWT token** in the `Authorization` header. The user's ID extracted from this token is used to track and manage limits consistently across all connected services.

### 🌐 HAProxy Integration
The gateway is fronted by an **HAProxy load balancer**, which:
- Distributes traffic across multiple gateway instances using **least connections** policy
- Is the **only component exposed to the public network**
- Ensures gateway instances remain internal to the Docker network for security

### 🔜 Planned Enhancements
- **Application-Level Load Balancing**: Future releases will support  routing between backend services within the gateway itself — based on traffic type.

