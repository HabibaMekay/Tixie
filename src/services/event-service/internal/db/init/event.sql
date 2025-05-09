CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    date TEXT NOT NULL,
    venue TEXT NOT NULL,
    total_tickets INT NOT NULL,
    vendor_id INT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    sold_tickets INT NOT NULL DEFAULT 0,
    tickets_reserved INT NOT NULL DEFAULT 0,
    tickets_left INT GENERATED ALWAYS AS (total_tickets - sold_tickets - tickets_reserved) STORED,
    reservation_timeout INT NOT NULL DEFAULT 600 
);
-- the following is a chatgpt assisted code to allow for database updates without breaking or demounting the current implementation
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'events') AND 
       NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'events' AND column_name = 'tickets_reserved') THEN
        ALTER TABLE events ADD COLUMN tickets_reserved INT NOT NULL DEFAULT 0;
    END IF;
    
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'events') AND 
       NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'events' AND column_name = 'reservation_timeout') THEN
        ALTER TABLE events ADD COLUMN reservation_timeout INT NOT NULL DEFAULT 600;
    END IF;
    
    -- Handle the transition of tickets_left to a generated column
    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'events' AND column_name = 'tickets_left') THEN
        ALTER TABLE events DROP COLUMN tickets_left;
        ALTER TABLE events ADD COLUMN tickets_left INT GENERATED ALWAYS AS (total_tickets - sold_tickets - tickets_reserved) STORED;
    END IF;
END$$;

-- Only add sample data if the table is empty
INSERT INTO events (name, date, venue, total_tickets, vendor_id, price, sold_tickets, tickets_reserved, reservation_timeout)
SELECT 
    'Summer Music Festival', 
    '2023-07-15', 
    'Central Park Amphitheater', 
    1000, 
    42, 
    79.99, 
    150,
    0,
    600
WHERE NOT EXISTS (SELECT 1 FROM events);

INSERT INTO events (name, date, venue, total_tickets, vendor_id, price, sold_tickets, tickets_reserved, reservation_timeout)
SELECT 
    'Tech Innovators Summit', 
    '2023-09-20', 
    'Convention Center Hall A', 
    500, 
    17, 
    249.50, 
    320,
    0,
    600
WHERE NOT EXISTS (SELECT 1 FROM events OFFSET 1);