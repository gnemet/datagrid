-- Database Initialization for Datagrid Component
CREATE SCHEMA IF NOT EXISTS datagrid;
SET search_path TO datagrid;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- LOV Table for Departments
CREATE TABLE IF NOT EXISTS lov_departments (
    code TEXT PRIMARY KEY,
    name_en TEXT,
    name_hu TEXT
);
INSERT INTO lov_departments (code, name_en, name_hu)
VALUES ('ENG', 'Engineering', 'Mérnökség'),
    ('MGT', 'Management', 'Vezetőség'),
    ('DSG', 'Design', 'Tervezés'),
    ('HR', 'Human Resources', 'HR') ON CONFLICT (code) DO NOTHING;
-- Table 1: Personnel
CREATE TABLE IF NOT EXISTS personnel (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    department TEXT REFERENCES lov_departments(code),
    salary NUMERIC(15, 2),
    status TEXT DEFAULT 'active',
    is_valid BOOLEAN DEFAULT TRUE,
    data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO personnel (
        name,
        email,
        department,
        salary,
        status,
        is_valid,
        data
    )
VALUES (
        'Gábor Német',
        'gabor.nemet@example.com',
        'ENG',
        120000,
        'active',
        true,
        '{"role": "Lead Architect"}'
    ),
    (
        'Kovács István',
        'istvan.kovacs@example.com',
        'MGT',
        95000,
        'active',
        true,
        '{"role": "PM"}'
    ),
    (
        'Szabó Anna',
        'anna.szabo@example.com',
        'DSG',
        85000,
        'active',
        true,
        '{"role": "UI/UX"}'
    ) ON CONFLICT (id) DO NOTHING;
-- Table 2: Responsibility
CREATE TABLE IF NOT EXISTS responsibility (
    id SERIAL PRIMARY KEY,
    uly_projekt_name TEXT,
    admin_projekt_name TEXT,
    admin_komponens_name TEXT,
    user_name TEXT,
    user_rank INTEGER,
    is_valid INTEGER DEFAULT 1
);
INSERT INTO responsibility (
        uly_projekt_name,
        admin_projekt_name,
        admin_komponens_name,
        user_name,
        user_rank,
        is_valid
    )
VALUES (
        'Project Alpha',
        'Admin Core',
        'Database',
        'gnemet',
        1,
        1
    ),
    (
        'Project Beta',
        'Mobile App',
        'API',
        'anna.szabo',
        1,
        1
    ) ON CONFLICT (id) DO NOTHING;