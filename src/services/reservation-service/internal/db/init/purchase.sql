CREATE TABLE purchases (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    event_id INTEGER NOT NULL,
    purchase_date TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
);



INSERT INTO purchases (ticket_id, user_id, event_id, purchase_date, status) 
VALUES 
    (1, 1, 1, '2025-04-17 10:00:00', 'confirmed'),
    (2, 2, 1, '2025-04-17 11:00:00', 'pending');