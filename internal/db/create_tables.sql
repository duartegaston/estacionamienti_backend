-- Tabla de reservas
CREATE TABLE reservations (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) UNIQUE NOT NULL,
    user_name VARCHAR(100) NOT NULL,
    user_email VARCHAR(150) NOT NULL,
    user_phone VARCHAR(20) NOT NULL,
    vehicle_type_id INT NOT NULL REFERENCES vehicle_types(id),
    vehicle_plate VARCHAR(20) NOT NULL,
    vehicle_model VARCHAR(50) NOT NULL,
    payment_method VARCHAR(50) NOT NULL, -- on site or online
    status VARCHAR(20) DEFAULT 'active',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Tabla de administradores
CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    user_name VARCHAR(150) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de tipos de vehículos
CREATE TABLE vehicle_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL -- auto, moto, camioneta
);

CREATE TABLE  reservation_times (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL -- hora, semana, mes
);

CREATE TABLE vehicle_prices (
    id SERIAL PRIMARY KEY,
    vehicle_type_id INT NOT NULL REFERENCES vehicle_types(id),
    reservation_time_id INT NOT NULL REFERENCES reservation_times(id),
    price INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE vehicle_spaces (
    id SERIAL PRIMARY KEY,
    vehicle_type_id INT NOT NULL REFERENCES vehicle_types(id),
    spaces INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
