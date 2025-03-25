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


