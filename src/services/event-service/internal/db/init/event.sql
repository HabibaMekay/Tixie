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