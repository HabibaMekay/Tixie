# ADR: Database Decision

## Context

We need a robust and scalable database solution to handle structured data for our Ticketing System application. The application demands high availability, consistency, and performance to ensure a seamless user experience across its microservices architecture.

The system consists of multiple services—Authentication Service, Payment Service, and Booking Service—each managing distinct types of structured data.

To support this microservices architecture, we must decide on a database solution that ensures data consistency, scalability, and performance while aligning with the system’s design principles. Additionally, we need to determine how to structure the databases to support the separation of concerns between services.

## Decision

We have decided to use **PostgreSQL** for structured data storage, with a **vertical partitioning** strategy to allocate a dedicated PostgreSQL database for each service:

- **Authentication Service Database**: Stores authentication-related data (e.g., user credentials, tokens).
- **Payment Service Database**: Stores payment-related data (e.g., transaction records).
- **Booking Service Database**: Stores booking-related data (e.g., events, tickets).

### Choice of PostgreSQL

**Justification**:

- PostgreSQL is selected as the database management system due to its robustness, feature set, and ability to handle complex queries and transactions efficiently, which aligns with the application’s requirements for structured data management.

### Vertical Partitioning Strategy

**Justification**:

- Vertical partitioning is implemented by assigning a separate PostgreSQL database to each service. This approach aligns with the microservices principle of separation of concerns, where each service manages its own data independently.
- Vertical partitioning ensures that each service’s data is isolated, reducing the risk of data conflicts and improving scalability by allowing each database to be optimized for its specific workload.

## Consequences

### Positive Consequences

- **Robustness and Feature Set with PostgreSQL**:
  - Provides strong ACID compliance, ensuring data consistency for critical operations like payment transactions and ticket bookings.
  - Offers advanced querying capabilities (e.g., complex joins, aggregations) and features like full-text search, which are beneficial for the Booking Service.
  - Supports extensions for additional functionality (e.g., PostGIS for geospatial data if needed in the future).
- **Open-Source Benefits**:
  - PostgreSQL is open-source, allowing the team to inspect the implementation of certain functionalities if needed and leverage a large community for support and guidance.
- **Service Isolation with Vertical Partitioning**:
  - Each service has its own database, aligning with the microservices principle of separation of concerns.
  - Improves scalability by allowing each database to be optimized for its specific workload.
  - Enhances fault isolation—if one database fails, other services can continue operating.
- **Performance and Consistency**:
  - Synchronous communication with each database ensures immediate responses for data operations, maintaining consistency during user-facing operations.

### Negative Consequences

- **Increased Operational Overhead with Vertical Partitioning**:
  - Managing multiple PostgreSQL databases requires more administrative effort (e.g., backups, monitoring, schema migrations for each database).
  - Cross-service queries are more complex and may require API calls or data replication.
- **Potential Scalability Limits**:
  - Each database may eventually face scalability limits if a service’s data volume grows significantly.
- **PostgreSQL Learning Curve**:
  - PostgreSQL’s advanced features may have a learning curve for the team.

## Alternatives Considered

### Alternative Database Systems

#### MySQL

**Pros:**

- Widely used with a large community.
- Simple to set up and manage.

**Cons:**

- Lacks some advanced features of PostgreSQL, such as full-text search and robust support for complex transactions, which are beneficial for the Ticketing System’s needs (e.g., searching events in the Booking Service).

**Rejected Because**: PostgreSQL’s advanced features and stronger ACID compliance better align with the system’s requirements for structured data management and complex querying.

#### MongoDB

**Pros:**

- Flexible schema design, suitable for unstructured data.
- Good for rapid prototyping and schema evolution.

**Cons:**

- Not as strong in handling complex queries and transactions as PostgreSQL, which is critical for the Ticketing System’s structured data (e.g., transactional consistency for payments and bookings).
- Not open-source, limiting transparency and community support compared to PostgreSQL.

**Rejected Because**: The Ticketing System primarily deals with structured data (e.g., user records, payment transactions, event/ticket data), where PostgreSQL’s relational model and transactional capabilities are more appropriate than MongoDB’s document-based model.

### Alternative Partitioning Strategies

#### Single Shared Database

**Pros:**

- Simplifies database management by centralizing all data in one PostgreSQL instance.
- Easier to perform cross-service queries (e.g., joining user data with booking data).

**Cons:**

- Violates the microservices principle of separation of concerns, as services would share a single database, leading to tight coupling.
- Increases the risk of data conflicts and contention (e.g., Authentication Service and Booking Service competing for database resources).
- A single point of failure—if the database goes down, all services are affected.

**Rejected Because**: A shared database undermines the independence of microservices, increases coupling, and reduces scalability, which conflicts with the system’s architectural goals.

#### Horizontal Partitioning (Sharding)

**Pros:**

- Distributes data across multiple database instances based on a shard key (e.g., user ID), improving scalability for very large datasets.
- Can enhance performance by distributing load across shards.

**Cons:**

- Adds complexity in managing sharded databases, including shard key selection, data distribution, and cross-shard queries.
- Not necessary for the Ticketing System’s current data volume, which is moderate and can be handled by vertical partitioning.
- Increases operational overhead (e.g., managing multiple shards, rebalancing data).

**Rejected Because**: The system’s data volume and workload do not justify the complexity of horizontal partitioning. Vertical partitioning provides sufficient scalability and isolation for the current and near-future needs of the system, with less operational overhead.

## Rationale

PostgreSQL with vertical partitioning was chosen because it best aligns with the system’s requirements and architectural principles, providing robustness, service isolation, scalability, performance, and simplicity compared to alternative solutions.
