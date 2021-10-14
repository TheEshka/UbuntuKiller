CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DROP TABLE IF EXISTS genres CASCADE;
DROP TABLE IF EXISTS authors CASCADE;
DROP TABLE IF EXISTS books CASCADE;

CREATE table genres (
    id serial PRIMARY KEY,
    name varchar(40) NOT NULL
);

CREATE TABLE authors (
    id serial PRIMARY KEY,
    name varchar(40) NOT NULL,
    surname varchar(40),
    description varchar(200)
);

CREATE TABLE books (
    book_uid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name varchar(40) NOT NULL,
    author_id integer REFERENCES authors (id) NOT NULL,
    genre_id integer REFERENCES genres (id) NOT NULL
);

INSERT INTO authors(id, name, surname, description) VALUES
(1, 'Sanya', 'Pushkin', 'Russian author'),
(2, 'Misha', NULL, NULL);

INSERT INTO genres(id, name) VALUES
(1, 'Novel'),
(2, 'Detective');

INSERT INTO books(book_uid, name, author_id, genre_id) VALUES
('006ca255-f5e9-4153-9423-2ac188512e70'::uuid, 'Gold petuh', 1, 1),
('08a31c8e-1a2c-4bd2-b87b-632377136d83'::uuid, 'Dub', 2, 2);