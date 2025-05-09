CREATE TABLE reservations (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expiration_time TIMESTAMP NOT NULL,
    CONSTRAINT valid_status CHECK (status IN ('pending', 'completed', 'expired'))
);

-- this index was used to optimize query times, since this data is structured i contemplated using redis for it, but i decided against it because 1- it is indeed very structure, 2- we need consistency & durability
-- for this part of the system as it is critical to the processing of tickets. redis could crash, flush data, etc. but database centers are usually managed a lot better.
CREATE INDEX idx_reservation_status_expiration ON reservations (status, expiration_time); 