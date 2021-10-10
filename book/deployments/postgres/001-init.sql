CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE table genres (
    id serial PRIMARY KEY,
    name varchar(40) NOT NULL
);

CREATE TABLE books (
    book_uid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name varchar(40) NOT NULL,
    author varchar(40) NOT NULL,
    genre_id integer REFERENCES genres (id)
);

INSERT INTO genres(id, name) VALUES
(1, 'Novel'),
(2, 'Detective');

INSERT INTO books(book_uid, name, author, genre_id) VALUES
('006ca255-f5e9-4153-9423-2ac188512e70'::uuid, 'a', 'a b', 1),
('08a31c8e-1a2c-4bd2-b87b-632377136d83'::uuid, 'b', 'c d', 2);