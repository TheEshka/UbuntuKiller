CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DROP TABLE IF EXISTS library CASCADE;
DROP TABLE IF EXISTS libraryBooks CASCADE;
DROP TABLE IF EXISTS takenBooks CASCADE;

CREATE TABLE IF NOT EXISTS library (
    id serial PRIMARY KEY,
    library_uid uuid DEFAULT uuid_generate_v4(),
    location varchar(40) NOT NULL
);

CREATE TABLE IF NOT EXISTS libraryBooks  (
    library_id int REFERENCES library (id),
    book_uid uuid NOT NULL,
    available_count integer NOT NULL,
    PRIMARY KEY (library_id, book_uid)
);


CREATE TABLE IF NOT EXISTS takenBooks (
    taken_book_uid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    book_uid uuid NOT NULL,
    user_uid uuid NOT NULL,
    library_id int REFERENCES library (id),
    take_date date DEFAULT CURRENT_DATE,
    status varchar(15) NOT NULL
);

INSERT INTO library(id, library_uid, location) VALUES
(1, '006ca255-f5e9-4153-9423-2ac188512e70'::uuid, 'kolotushkin strit'),
(2, '08a31c8e-1a2c-4bd2-b87b-632377136d83'::uuid, 'tutushkin prospect');

INSERT INTO libraryBooks(library_id, book_uid, available_count) VALUES
(1, '111ca255-f5e9-4153-9423-2ac188512e70'::uuid, 3),
(1, '22a31c8e-1a2c-4bd2-b87b-632377136d83'::uuid, 2),
(2, '111ca255-f5e9-4153-9423-2ac188512e70'::uuid, 0),
(2, '22a31c8e-1a2c-4bd2-b87b-632377136d83'::uuid, 0);

INSERT INTO takenBooks(book_uid, user_uid, library_id, status) VALUES
('111ca255-f5e9-4153-9423-2ac188512e70'::uuid, 'bc39ffba-80b4-49fb-8101-35f514a438e9'::uuid, 2, 'bad_condition'),
('22a31c8e-1a2c-4bd2-b87b-632377136d83'::uuid, 'bc39ffba-80b4-49fb-8101-35f514a438e9'::uuid, 2, 'new');
