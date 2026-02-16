-- Database Initialization for Datagrid Component
CREATE SCHEMA IF NOT EXISTS datagrid;
SET search_path TO datagrid;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- LOV Table for Departments
DROP TABLE IF EXISTS personnel CASCADE;
DROP TABLE IF EXISTS lov_departments CASCADE;
CREATE TABLE lov_departments (
    code TEXT PRIMARY KEY,
    name_en TEXT,
    name_hu TEXT
);
INSERT INTO lov_departments (code, name_en, name_hu)
VALUES ('ENG', 'Engineering', 'Mérnökség'),
    ('MGT', 'Management', 'Vezetőség'),
    ('DSG', 'Design', 'Tervezés'),
    ('HR', 'Human Resources', 'HR'),
    ('OPS', 'Operations', 'Üzemeltetés'),
    ('FIN', 'Finance', 'Pénzügy'),
    ('SALES', 'Sales', 'Értékesítés') ON CONFLICT (code) DO NOTHING;
-- Table 1: Personnel
CREATE TABLE IF NOT EXISTS personnel (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    department TEXT REFERENCES lov_departments(code),
    gender TEXT,
    role TEXT,
    salary NUMERIC(15, 2),
    bonus NUMERIC(15, 2),
    rating INTEGER,
    tenure INTEGER,
    status TEXT DEFAULT 'employed',
    is_valid BOOLEAN DEFAULT TRUE,
    is_active BOOLEAN DEFAULT TRUE,
    data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO personnel (
        name,
        email,
        department,
        gender,
        role,
        salary,
        bonus,
        rating,
        tenure,
        is_valid,
        is_active,
        data
    )
VALUES (
        'Molnar Sandor',
        'sandor.molnar@example.com',
        'ENG',
        'Male',
        'QA Engineer',
        129510,
        15000,
        4,
        3,
        true,
        true,
        '{"tags": ["js", "docker"]}'
    ),
    (
        'Balog Gabor',
        'gabor.balog@example.com',
        'FIN',
        'Male',
        'Financial Analyst',
        118488,
        5000,
        5,
        20,
        true,
        true,
        '{"experience": 20}'
    ),
    (
        'Meszaros Zoltan',
        'zoltan.meszaros@example.com',
        'HR',
        'Male',
        'HR Specialist',
        129462,
        2000,
        3,
        13,
        true,
        true,
        '{"experience": 13}'
    ),
    (
        'Takacs Peter',
        'peter.takacs@example.com',
        'HR',
        'Male',
        'HR Specialist',
        74235,
        1000,
        2,
        17,
        true,
        false,
        '{"experience": 17}'
    ),
    (
        'Nagy Peter',
        'peter.nagy@example.com',
        'SALES',
        'Male',
        'Account Manager',
        102010,
        8000,
        4,
        5,
        true,
        true,
        '{"experience": 5}'
    );
-- Personnel 2 - Demo data
CREATE TABLE IF NOT EXISTS personnel_2 (
    id SERIAL PRIMARY KEY,
    name TEXT,
    dept TEXT,
    salary NUMERIC
);
INSERT INTO personnel_2 (name, dept, salary)
VALUES ('John Doe', 'ENG', 100000),
    ('Jane Doe', 'DSG', 110000);
-- Table 2: Responsibility
DROP TABLE IF EXISTS responsibility CASCADE;
CREATE TABLE responsibility (
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
        'Web Platform',
        'UI',
        'molnar.sandor',
        4,
        1
    ),
    (
        'Project Beta',
        'Mobile App',
        'API',
        'anna.szabo',
        1,
        1
    );