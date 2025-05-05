CREATE TABLE purchases (
    purchase_id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    event_id INTEGER NOT NULL,
    purchase_date TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    CONSTRAINT valid_status CHECK (status IN ('pending', 'confirmed', 'cancelled'))
);

