CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS mesh_blocks (
    mb_code TEXT PRIMARY KEY,
    sa3_code TEXT,
    sa3_name TEXT
);

CREATE TABLE IF NOT EXISTS addresses (
    address_detail_pid TEXT PRIMARY KEY,
    address_label TEXT NOT NULL,
    number_first TEXT,
    street_name TEXT,
    street_type TEXT,
    street_name_norm TEXT,
    street_type_norm TEXT,
    locality_name TEXT,
    locality_name_norm TEXT,
    state TEXT NOT NULL,
    postcode TEXT,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    mb_code_2021 TEXT,
    sa3_code TEXT,
    sa3_name TEXT
);

CREATE TABLE IF NOT EXISTS locality_centroids (
    state TEXT NOT NULL,
    postcode TEXT NOT NULL,
    locality_name TEXT NOT NULL,
    locality_name_norm TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    address_count INTEGER NOT NULL,
    sa3_code TEXT,
    sa3_name TEXT,
    PRIMARY KEY (state, postcode, locality_name_norm)
);

CREATE TABLE IF NOT EXISTS postcode_centroids (
    state TEXT NOT NULL,
    postcode TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    address_count INTEGER NOT NULL,
    sa3_code TEXT,
    sa3_name TEXT,
    PRIMARY KEY (state, postcode)
);

CREATE TABLE IF NOT EXISTS state_centroids (
    state TEXT PRIMARY KEY,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    address_count INTEGER NOT NULL,
    sa3_code TEXT,
    sa3_name TEXT
);
