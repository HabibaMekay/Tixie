# ADR: Database Decision

## Context
We need a robust and scalable database solution to handle structured data for our application. The application demands high availability, consistency, and performance to ensure a seamless user experience.

## Decision
We have decided to use PostgreSQL for structured data storage.

## Alternatives Considered
1. **MySQL**
   - **Pros:**
     - Widely used with a large community.
     - Simple to set up and manage.
   - **Cons:**
     - Lacks some advanced features of PostgreSQL, such as full-text search.

2. **MongoDB**
   - **Pros:**
     - Flexible schema design.
     - Good for unstructured data.
   - **Cons:**
     - Not as strong in handling complex queries and transactions as PostgreSQL.
     - not open source

## Consequences
- **Positive:**
  - PostgreSQL provides strong ACID compliance, advanced querying capabilities, and support for complex transactions.
  - Offers a wide range of features like full-text search, and extensions for additional functionality.

## Rationale
PostgreSQL was chosen for its robustness, feature set, and ability to handle complex queries and transactions efficiently. Its also open source so in times of confusion we can always peer into the actual implementation of certain functionalities, or reach out to their community for guidance. Its advanced features align with our application's requirements for structured data management.
