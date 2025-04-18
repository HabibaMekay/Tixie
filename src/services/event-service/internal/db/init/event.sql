CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    date TEXT NOT NULL,
    venue TEXT NOT NULL,
    total_tickets INT NOT NULL,
    vendor_id INT NOT NULL
);
