CREATE TABLE ticket (
    ticket_id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    ticket_code VARCHAR(32) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    CONSTRAINT valid_status CHECK (status IN ('active', 'used', 'cancelled'))
);

INSERT INTO ticket (event_id, user_id, ticket_code, status)
VALUES (1, 1, '123e4567-e89b-12d3-a456-426614174000', 'active');
INSERT INTO ticket (event_id, user_id, ticket_code, status)
VALUES (1, 2, '987fcdeb-51a2-43b7-9c1d-7f8e9a123456', 'used');
INSERT INTO ticket (event_id, user_id, ticket_code, status)
VALUES (2, 3, 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'active');
INSERT INTO ticket (event_id, user_id, ticket_code, status)
VALUES (3, 1, '456789ab-cdef-1234-5678-901234567890', 'cancelled');
INSERT INTO ticket (event_id, user_id, ticket_code, status)
VALUES (2, 4, 'bcdef123-4567-89ab-cdef-123456789abc', 'active');