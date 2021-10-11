CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DROP TABLE IF EXISTS ratings CASCADE;

CREATE TABLE IF NOT EXISTS ratings (
    user_uid uuid PRIMARY KEY,
    rate int NOT NULL
);

INSERT INTO ratings(user_uid, rate) VALUES
('bc39ffba-80b4-49fb-8101-35f514a438e9'::uuid, 3),
('e48b2402-5f67-4706-9d27-9b29bebdddf9'::uuid, 2);