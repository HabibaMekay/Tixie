CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    date TEXT NOT NULL,
    venue TEXT NOT NULL,
    total_tickets INT NOT NULL,
    vendor_id INT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    sold_tickets INT NOT NULL DEFAULT 0,
    tickets_left INT
);




INSERT INTO events (name, date, venue, total_tickets, vendor_id, price, sold_tickets, tickets_left)
VALUES (
    'Summer Music Festival', 
    '2023-07-15', 
    'Central Park Amphitheater', 
    1000, 
    42, 
    79.99, 
    150, 
    1000 - 150
);

INSERT INTO events (name, date, venue, total_tickets, vendor_id, price, sold_tickets, tickets_left)
VALUES (
    'Tech Innovators Summit', 
    '2023-09-20', 
    'Convention Center Hall A', 
    500, 
    17, 
    249.50, 
    320, 
    500 - 320
);