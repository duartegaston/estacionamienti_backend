-- Table: vehicle_spaces
CREATE TABLE vehicle_spaces (
    id SERIAL PRIMARY KEY,
    vehicle_type VARCHAR(20) UNIQUE NOT NULL,
    total_spaces INT NOT NULL,
    available_spaces INT NOT NULL
);

-- Table: reservations
CREATE TABLE reservations (
    id SERIAL PRIMARY KEY,
    reservation_code VARCHAR(10) UNIQUE NOT NULL,
    entry_time TIMESTAMP NOT NULL,
    exit_time TIMESTAMP NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    license_plate VARCHAR(20) NOT NULL,
    vehicle_model VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, cancelled
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);