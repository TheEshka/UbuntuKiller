CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DROP TABLE IF EXISTS control CASCADE;

CREATE TABLE IF NOT EXISTS control (
    user_uid uuid PRIMARY KEY,
    current_count int NOT NULL,
    limit_count int NOT NULL
);

INSERT INTO control(user_uid, current_count, limit_count) VALUES
('bc39ffba-80b4-49fb-8101-35f514a438e9'::uuid, 2, 3),
('e48b2402-5f67-4706-9d27-9b29bebdddf9'::uuid, 1, 5);