CREATE TABLE IF NOT EXISTS users (
    id         INT AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) UNIQUE NOT NULL,
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sessions (
    id         INT AUTO_INCREMENT PRIMARY KEY,
    user_id    INT NOT NULL,
    token      VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS itineraries (
    id              INT AUTO_INCREMENT PRIMARY KEY,
    user_id         INT NOT NULL,
    slug            VARCHAR(255) UNIQUE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    currency        VARCHAR(10) DEFAULT 'IDR',
    cover_image_url MEDIUMTEXT,
    estimated_cost  DOUBLE DEFAULT 0,
    is_public       TINYINT(1) DEFAULT 0 NOT NULL,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS activities (
    id               INT AUTO_INCREMENT PRIMARY KEY,
    itinerary_id     INT NOT NULL,
    activity_type    ENUM(
                         'Attraction','Beach','Bus','Car','Culinary',
                         'Culture','Cycling','Event','Explore','Ferry',
                         'Flight','Hiking','Motorscooter','Nature','Other',
                         'Shopping','Spa','Sport','Stay','Taxi','Train'
                     ) NOT NULL,
    identifier       VARCHAR(255),
    name             VARCHAR(255),
    location_name    VARCHAR(255),
    location_address TEXT,
    latitude         DOUBLE,
    longitude        DOUBLE,
    activity_date    DATE NOT NULL,
    start_time       TIME NOT NULL,
    cost             DOUBLE DEFAULT 0,
    ticket_status    ENUM('Secured','Unbooked','Go Show') DEFAULT 'Unbooked',
    details          TEXT,
    sort_order       INT DEFAULT 0,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (itinerary_id) REFERENCES itineraries(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_itineraries_user_id ON itineraries(user_id);
CREATE INDEX idx_activities_itinerary_id ON activities(itinerary_id);
CREATE INDEX idx_activities_date ON activities(itinerary_id, activity_date, start_time);
