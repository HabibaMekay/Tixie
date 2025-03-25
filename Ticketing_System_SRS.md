# Ticketing System

**Powerpuffs & Mjjj**

## Functional Requirements

- The system must allow users and vendors to create an account
- The system shall allow users to login as normal users or vendors securely
- The system must allow users to search for events
- The system must allow users to filter events by price
- The system must allow users to browse events
- The system shall allow users to browse and search for tickets for any event by artist, venue name, or event name
- The system shall allow users to filter search results based on date or price
- The system shall allow users to filter events based on type (concert, conference, gaming, educational, etc)
- The system shall allow users to buy e-tickets that contain a scannable QR code
- The system must allow users to make payments using different payment methods
- The system must allow vendors to view ticket sales to track earnings
- The system shall allow vendor users to create and add events to sell tickets
- The system shall allow vendor users to select from multiple payment methods when creating events (Visa, mastercard, cash on arrival, etc)
- The system shall grant users loyalty points that can be redeemed for discounts, ticket upgrades, or free tickets
- The system shall provide recommendations for users using machine learning models for a more personalized experience
- The system shall provide push notifications such as event reminders, event suggestions, etc
- The system shall allow users to upload pictures of their attended events/ venue for future users booking tickets in the same venue/ same artist.
- The system must allow users and vendors to log out to secure their accounts

## User Stories

- As a user I want to create an account so I can book tickets and manage my events
- As a user I want to book tickets so I can attend events
- As a user I want to search for specific tickets so I can find the event I want
- As a user I want to filter the results based on price so I can find tickets that fit my budget
- As a user I want to pay using different methods so I can choose what works best for me
- As a user I want to create a profile so I can keep track of my bookings and data
- As a user I want to log in so I can access my account
- As a user I want to gain loyalty points so I can redeem discounts or free tickets
- As a user I want to browse events so I can choose the most suitable one for me
- As a user I want to logout of my account so i can keep my account secure
- As a user I want to upload event pictures so i can share my experience
- As a user I want to receive recommendations based on my preferences so I don't have to search for a long time.
- As a user I want to receive a confirmation email after booking so I can have proof of my ticket.
- As a vendor I want to create an account so I can add events and sell tickets.
- As a vendor I want to log in so i can access my account
- As a vendor I want to add an event so i can sell tickets
- As a vendor i want to add available payment method so customers can pay easily
- As a vendor I want to view ticket sales so i can track my earnings
- As a vendor I want to log out so i can secure my account
- As a vendor I want to manage my payment methods so I can ensure smooth transactions for customers

# Use Cases
## Non-functional Requirements

### Scalability:
- The system should handle 200 users at the same time without slowing down.
- The system should use database vertical partitioning caching (Redis) to optimize query performance and reduce database load.
### Response Time:
- APIs must respond within 3 seconds for 80% of requests to ensure smooth third-party interactions.
### Availability:
- The system should be up and running 80% of the time.
- In case of a server failure, the system shall recover within 30 seconds using backup services.
### Security:
- The system shall implement OAuth 2.0 authentication to ensure secure user login and protect user credentials from unauthorized access.
### Fault Tolerance:
- The system must implement Circuit Breaker patterns to gracefully degrade features.
- The system must implement a retry mechanism with exponential backoff for failed API calls.
## Back-of-the-Envelope Testing

### 1. Load Testing Calculations

#### Concurrent Users
- **Peak Concurrent Users**: 200 (based on requirements)
- **Total Registered Users**: ~5,000 (estimated total user base)
- **Average Session Duration**: ~15-20 minutes (realistic browsing time)
- **Requests per User per Session**: ~30-40 (including searches, filters, and page loads)
- **Total Requests per Hour**: 200 users × 35 requests = 7,000 requests/hour
- **Requests per Second**: 7,000/3600 ≈ 2 requests/second

*Got these numbers from similar systems. Might need to bump up during big events.*

### 2. Database Load (PostgreSQL)

#### Data Size Estimation
| Data Type | Count | Size per Item | Total Size |
|-----------|-------|---------------|------------|
| Users | 5,000 | 2KB | 10MB |
| Events | 500 | 10KB | 5MB |
| Tickets | 50,000 | 3KB | 150MB |
| Images | 25,000 | 1MB | 25GB |
| **Total** | | | **~25.2GB** |

*Image sizes are rough - depends on quality settings*

### 3. Caching Requirements (Redis)

#### Cache Size Estimation
| Cache Type | Count | Size per Item | Total Size |
|------------|-------|---------------|------------|
| Active Events | 500 | 10KB | 5MB |
| Popular Event Details | 50 | 20KB | 1MB |
| User Sessions | 200 | 5KB | 1MB |
| **Total Cache Size** | | | **~7MB** |

*Doubled the numbers to be safe*

### 4. Response Time Budget

#### 3-Second Response Time Target (80% of requests)
| Component | Time Budget |
|-----------|-------------|
| Database Query | 800ms |
| Cache Lookup | 100ms |
| Business Logic | 1.5s |
| Network Latency | 300ms |
| Buffer | 300ms |
| **Total** | **3s** |

*These are ideal numbers - reality might be slower*

### 5. Storage Requirements

#### Monthly Growth
| Data Type | Monthly Growth | Size per Item | Monthly Storage |
|-----------|----------------|---------------|-----------------|
| New Users | 250 | 2KB | 500KB |
| New Events | 50 | 10KB | 500KB |
| New Tickets | 5,000 | 3KB | 15MB |
| New Images | 2,500 | 1MB | 2.5GB |
| **Total Monthly Growth** | | | **~2.5GB** |
| **Annual Growth** | | | **~30GB** |

*Expect double these numbers during peak season*

### 6. Bandwidth Requirements

#### Per Request Analysis
- **Average Response Size**: 100KB (including images and assets)
- **Requests per Second**: 2
- **Bandwidth**: 2 × 100KB = 200KB/s
- **Peak Hour Bandwidth**: 200KB × 3600 = 720MB/hour
- **Daily Bandwidth**: 720MB × 24 = 17.28GB/day

*Double this for peak times*

### 7. Service Scaling Points

#### Microservices Deployment
| Service | Instances | Purpose |
|---------|-----------|---------|
| Booking Service | 2 | Primary + Backup |
| Payment Service | 2 | Primary + Backup |
| Analytics Service | 1 | Single Instance |
| Image Service | 1 | Single Instance |
| **Total Instances** | **6** | |

*Might need more during busy times*

### 8. Cost Estimation (Monthly)

#### Infrastructure Costs
| Component | Quantity | Cost per Unit | Monthly Cost |
|-----------|----------|---------------|--------------|
| Compute | 6 instances | $75 | $450 |
| Database | 1 instance | $200 | $200 |
| Cache | 1 instance | $100 | $100 |
| Storage | 100GB | $0.15/GB | $15 |
| Bandwidth | 1TB | $0.15/GB | $150 |
| **Total Monthly Cost** | | | **~$915** |

*Cloud costs vary by region*

### 9. Failure Scenarios

#### Recovery Time Budget
| Scenario | Recovery Time |
|----------|---------------|
| Database Failover | 45-60 seconds |
| Service Restart | 15-20 seconds |
| Cache Rebuild | 10-15 minutes |
| Image Service Recovery | 30-45 seconds |

*Add buffer time for real-world scenarios*

### 10. Security Considerations

#### Token Storage Requirements
| Token Type | Size |
|------------|------|
| JWT Token | ~2KB |
| Refresh Token | ~2KB |
| Session Data | ~5KB |
| **Total per User** | **~9KB** |

*Token size might increase with new features*

### Quick Notes
- Numbers are rough estimates
- Double everything for peak season
- Keep an eye on actual usage
- Might need to adjust as we add features

### References
*Checked these out for numbers:*
- [Google Cloud Docs](https://cloud.google.com/architecture/load-testing)
- [AWS Docs](https://aws.amazon.com/architecture/well-architected/)
- [PostgreSQL Docs](https://www.postgresql.org/docs/current/performance-tips.html)
- [Redis Docs](https://redis.io/topics/memory-optimization)
## Architectural Drivers (Quality Attributes)

### Performance & Scalability
- **Circuit Breaker Patterns** to gracefully degrade features (e.g., if AI recommendations fail, the system should default to basic event recommendations).
- **Database Optimization**: 
  - Vertical partitioning
  - Redis caching for handling large-scale queries efficiently
- **Concurrent User Support**: System must handle **200+ concurrent users** searching, booking tickets, and making payments without performance degradation
- **Response Time**: Ticket booking and payment confirmation must happen in **under 3 seconds** to prevent user drop-off

### Security & Authentication
- **OAuth 2.0** authentication for secure user login
- **Data encryption** for storing sensitive information

### Architecture & Integration
- **Microservices architecture** to allow independent scaling of:
  - Booking services
  - Payment services
  - Analytics services
- **API-first approach** for easy third-party integrations (e.g., social sharing, affiliate resellers)

## Endpoints

| Function | Method | Endpoint |
|---|---|---|
| Create Account | POST | /register |
| Login | POST | /login |
| Logout | POST | /logout |
| Browse Events | GET | /events |
| Search | GET | /events?search={X} |
| Sort by price (low to high) | GET | /events?sort=price_asc |
| Sort by price (high to low) | GET | /events?sort=price_desc |
| Event Details | GET | /events/{eventId} |
| View Tickets for an Event | GET | /events/{eventId}/tickets |
| Buy a Ticket | POST | /tickets/book |
| Refund | POST | /tickets/refund |
| Make Payment | POST | /payment |
| Choose Payment Method | GET | /vendors/payment-methods |
| View Loyalty Points | GET | /loyalty-points |
| Add Event | POST | /events |
| Add Payment Method | POST | /events/{eventId}/payment-methods |
| View Ticket Sales | GET | /vendors/{vendorId}/sales |
| Recommendations | GET | /recommendations |
| Upload Event Pictures | POST | /users/{userId}/upload-image |
| View Uploaded Pictures | GET | /users/{userId}/images |
| Updating an Event (name,date,etc..) | PUT | /events/{eventId} |
| Update Payment Method | PUT | /vendors/payment-methods/{methodId} |
| Update User Profile | PUT | /users/{userId} |
| Delete an Event | DELETE | /events/{eventId} |
| Cancel a ticket | DELETE | /tickets/{ticketId} |
| Remove a Payment Method | DELETE | /vendors/payment-methods/{methodId} |
| Delete Uploaded image | DELETE | /users/{userId}/images/{imageId} |
| Check ticket states (cancellation) | HEAD | /tickets/{ticketId} |
| Check Ticket Availability | HEAD | /events/{eventId}/tickets |

**Note:** HEAD is faster so we could use it to check ticket availability before booking to help prevent race conditions.

## **System Circuit Breaker Resilience Strategies**

Since our system will utilize both *HTTP requests* and *WebSockets* (check [ADRs/CommunicationBetweenServices_ADR.md](ADRs/CommunicationBetweenServices_ADR.md) and [ADRs/Connection_ADR.md](ADRs/Connection_ADR.md), respectively), so we need to have **two different circuit breakers**:  

### **1. HTTP Request Circuit Breaker**  
- **Retries**: Apply retry with **exponential backoff**, starting at **5 seconds**.  
  - **Justification**: Starting with a small backoff and increasing the timeout with every retry helps lighten the load on the service, allowing it to recover.  
- **Fallback**: The service will be deemed **unavailable after 4 failed retries**.  
  - **Justification**: If the service fails **4 consecutive times**, it will be marked unavailable in the application, only affecting dependent services. *(e.g., if ticket browsing fails, all booking operations will be stopped until browsing is available.)* 
- **Cooldown**: the system should wait **2 minutes** before attempting to reintegrate the failed service.  
  - **Justification**:  This ensures the service has a reasonable time period to stabilize before resuming operations, preventing unnecessary retries and load.

### **2. WebSocket Circuit Breaker**  
- **Retries**: Apply retry **every 40 seconds**.  
  - **Justification**: Since each retry involves multiple pings to the WebSocket, a **longer timeout** balances efficiency and system load.  
- **Fallback**: The service will be deemed **unavailable after 2 failed retries**.  
  - **Justification**: Given the **long retry interval**, the service has ample time to recover. If it **fails twice consecutively**, it will be marked unavailable, only affecting dependent services. *(e.g., if ticket booking fails, all payments will be stopped until booking is available.)*  
- **Cooldown**: the system should wait **5 minutes** before attempting to reintegrate the failed service.  
  - **Justification**: Since WebSockets involve persistent connections, a longer cooldown gives the service more time to stabilize.

Further details can be found in [ADRs/SystemResilience_ADR.md](ADRs/SystemResilience_ADR.md).  
