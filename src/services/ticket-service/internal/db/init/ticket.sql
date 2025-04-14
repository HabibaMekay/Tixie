CREATE TABLE ticket (
    id SERIAL PRIMARY KEY,
    event_id INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP--,
    -- FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);


INSERT INTO ticket (event_id, price, status) VALUES (1, 50.00, 'available');
INSERT INTO ticket (event_id, price, status) VALUES (2, 50.00, 'not');
INSERT INTO ticket (event_id, price, status) VALUES (3, 50.00, 'available');
INSERT INTO ticket (event_id, price, status) VALUES (4, 50.00, 'available');
INSERT INTO ticket (event_id, price, status) VALUES (5, 50.00, 'available');

