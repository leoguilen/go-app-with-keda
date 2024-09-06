CREATE TYPE order_status AS ENUM (
    'Pending',
    'Processing',
    'Done'
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    customerId INT NOT NULL,
    date TIMESTAMP NOT NULL,
    amount NUMERIC(18, 6) NOT NULL,
    status order_status NOT NULL
);