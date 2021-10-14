CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


DROP TABLE IF EXISTS accounts CASCADE;
DROP TYPE IF EXISTS user_role;

CREATE TYPE user_role AS ENUM ('admin', 'user');

CREATE table accounts (
    user_uid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    login TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'user'
);

INSERT INTO accounts(login, password, role) VALUES
('misha', crypt('libpass', gen_salt('bf')), 'admin'),
('pasha', crypt('gatewaypass', gen_salt('bf')), 'user');
