# ADR: Caching Decision

## Context
Our application requires a caching solution to improve performance by reducing the load on the database and speeding up data retrieval. The caching system needs to be fast, reliable, and capable of handling a high volume of requests.

## Decision
We have decided to use Redis for caching.

## Alternatives Considered
1. **Memcached**
   - **Pros:**
     - Simple and easy to set up.
     - High performance for simple key-value storage.
   - **Cons:**
     - Does not offer persistence, which can be a limitation for some caching needs.
     - Limited data structure support compared to Redis.

2. **In-Memory Caching with Application**
   - **Pros:**
     - No additional infrastructure required.
     - Directly integrated with the application.
   - **Cons:**
     - Limited scalability.
     - Increased memory usage on application servers.

## Consequences
- **Positive:**
  - Redis offers high-speed data access and supports various data structures, making it ideal for caching.
  - Provides persistence options, which can be beneficial for certain use cases.
  - Strong community support and extensive documentation.

## Rationale
Redis was chosen for its speed, flexibility, and support for a wide range of data structures. Its persistence capabilities and strong community support make it a reliable choice for our caching needs. Its also in the course you know 🐀

