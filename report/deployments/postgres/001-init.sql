CREATE TABLE genres (
    id serial PRIMARY KEY,
    genre TEXT NOT NULL
);

CREATE TABLE returns (
    id serial PRIMARY KEY,
    user_uid int NOT NULL,
    on_time boolean NOT NULL
);
