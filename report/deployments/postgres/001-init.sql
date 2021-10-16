CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DROP TABLE IF EXISTS genres;
DROP TABLE IF EXISTS returns;

CREATE TABLE genres (
    id serial PRIMARY KEY,
    genre TEXT NOT NULL
);

CREATE TABLE returns (
    id serial PRIMARY KEY,
    user_uid uuid NOT NULL,
    on_time boolean NOT NULL
);

INSERT INTO returns(user_uid, on_time) VALUES
('bc39ffba-80b4-49fb-8101-35f514a438e9'::uuid, TRUE),
('bc39ffba-80b4-49fb-8101-35f514a438e9'::uuid, TRUE);


INSERT INTO genres (genre) VALUES
('Novel'),
('Novel'),
('Novel'),
('Detective');