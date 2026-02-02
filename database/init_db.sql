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
    salary NUMERIC(15, 2),
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
        salary,
        is_valid,
        is_active,
        data
    )
VALUES (
        'Molnar Sandor',
        'sandor.molnar@example.com',
        'ENG',
        129510,
        true,
        true,
        '{"role": "QA Engineer", "experience": 3, "tags": ["js", "docker", "kubernetes", "aws"]}'
    ),
    (
        'Balog Gabor',
        'gabor.balog@example.com',
        'FIN',
        118488,
        true,
        true,
        '{"role": "Financial Analyst", "experience": 20}'
    ),
    (
        'Meszaros Zoltan',
        'zoltan.meszaros@example.com',
        'HR',
        129462,
        true,
        true,
        '{"role": "HR Specialist", "experience": 13}'
    ),
    (
        'Meszaros Cecil',
        'cecil.meszaros@example.com',
        'DSG',
        139358,
        true,
        true,
        '{"role": "UI Designer", "experience": 18}'
    ),
    (
        'Meszaros Bela',
        'bela.meszaros@example.com',
        'ENG',
        145887,
        true,
        true,
        '{"role": "Developer", "experience": 16, "tags": ["sql", "docker"]}'
    ),
    (
        'Takacs Peter',
        'peter.takacs@example.com',
        'HR',
        74235,
        true,
        false,
        '{"role": "HR Specialist", "experience": 17}'
    ),
    (
        'Kelemen Sara',
        'sara.kelemen@example.com',
        'OPS',
        86752,
        true,
        true,
        '{"role": "Sysadmin", "experience": 9}'
    ),
    (
        'Hegedus Laszlo',
        'laszlo.hegedus@example.com',
        'MGT',
        60239,
        true,
        true,
        '{"role": "Project Manager", "experience": 19}'
    ),
    (
        'Nagy Peter',
        'peter.nagy@example.com',
        'SALES',
        102010,
        true,
        true,
        '{"role": "Account Manager", "experience": 5}'
    ),
    (
        'Simon Bela',
        'bela.simon@example.com',
        'SALES',
        113478,
        true,
        true,
        '{"role": "Sales Rep", "experience": 4}'
    ),
    (
        'Takacs Luca',
        'luca.takacs@example.com',
        'ENG',
        79387,
        true,
        true,
        '{"role": "QA Engineer", "experience": 19, "tags": ["java", "go", "js", "python"]}'
    ),
    (
        'Simon Andras',
        'andras.simon@example.com',
        'ENG',
        114437,
        true,
        true,
        '{"role": "Senior Developer", "experience": 17, "tags": ["go", "aws", "python", "js"]}'
    ),
    (
        'Molnar Mari',
        'mari.molnar@example.com',
        'FIN',
        88632,
        true,
        true,
        '{"role": "Accountant", "experience": 10}'
    ),
    (
        'Meszaros Panna',
        'panna.meszaros@example.com',
        'ENG',
        78294,
        false,
        true,
        '{"role": "Senior Developer", "experience": 7, "tags": ["python", "aws"]}'
    ),
    (
        'Kovacs David',
        'david.kovacs@example.com',
        'OPS',
        56236,
        true,
        true,
        '{"role": "Sysadmin", "experience": 11}'
    ),
    (
        'Papp Luca',
        'luca.papp@example.com',
        'HR',
        144903,
        false,
        true,
        '{"role": "HR Specialist", "experience": 11}'
    ),
    (
        'Nemeth Zoltan',
        'zoltan.nemeth@example.com',
        'OPS',
        147555,
        true,
        true,
        '{"role": "DevOps Engineer", "experience": 18}'
    ),
    (
        'Simon Zsuzsanna',
        'zsuzsanna.simon@example.com',
        'OPS',
        130008,
        true,
        false,
        '{"role": "DevOps Engineer", "experience": 14}'
    ),
    (
        'Szabo Andras',
        'andras.szabo@example.com',
        'SALES',
        60135,
        true,
        true,
        '{"role": "Account Manager", "experience": 16}'
    ),
    (
        'Balog Elemer',
        'elemer.balog@example.com',
        'SALES',
        135870,
        false,
        true,
        '{"role": "Sales Rep", "experience": 3}'
    ),
    (
        'Hegedus Sandor',
        'sandor.hegedus@example.com',
        'OPS',
        90403,
        false,
        true,
        '{"role": "Sysadmin", "experience": 10}'
    ),
    (
        'Nemeth Bela',
        'bela.nemeth@example.com',
        'DSG',
        98053,
        true,
        true,
        '{"role": "UX Researcher", "experience": 2}'
    ),
    (
        'Papp Sandor',
        'sandor.papp@example.com',
        'MGT',
        51278,
        true,
        true,
        '{"role": "Team Lead", "experience": 3}'
    ),
    (
        'Balog Sandor',
        'sandor.balog@example.com',
        'FIN',
        63753,
        true,
        true,
        '{"role": "Accountant", "experience": 9}'
    ),
    (
        'Molnar Luca',
        'luca.molnar@example.com',
        'ENG',
        86300,
        false,
        true,
        '{"role": "Architect", "experience": 14, "tags": ["vue", "java", "aws"]}'
    ),
    (
        'Takacs Tamas',
        'tamas.takacs@example.com',
        'MGT',
        86821,
        true,
        true,
        '{"role": "Project Manager", "experience": 16}'
    ),
    (
        'Kiss Janos',
        'janos.kiss@example.com',
        'DSG',
        114535,
        true,
        true,
        '{"role": "UI Designer", "experience": 7}'
    ),
    (
        'Szabo Luca',
        'luca.szabo@example.com',
        'ENG',
        90050,
        true,
        true,
        '{"role": "Developer", "experience": 12, "tags": ["js"]}'
    ),
    (
        'Lakatos Timea',
        'timea.lakatos@example.com',
        'ENG',
        84686,
        true,
        true,
        '{"role": "Architect", "experience": 3, "tags": ["kubernetes", "python", "vue"]}'
    ),
    (
        'Simon Andras',
        'andras.simon@example.com',
        'ENG',
        115083,
        true,
        true,
        '{"role": "Senior Developer", "experience": 16, "tags": ["java", "kubernetes", "python", "js"]}'
    ),
    (
        'Varga Adam',
        'adam.varga@example.com',
        'SALES',
        60206,
        true,
        true,
        '{"role": "Sales Rep", "experience": 10}'
    ),
    (
        'Simon Orsolya',
        'orsolya.simon@example.com',
        'FIN',
        122423,
        true,
        true,
        '{"role": "Financial Analyst", "experience": 14}'
    ),
    (
        'Meszaros Bela',
        'bela.meszaros@example.com',
        'MGT',
        115911,
        true,
        true,
        '{"role": "Project Manager", "experience": 2, "certifications": ["PMP"]}'
    ),
    (
        'Kiss Nora',
        'nora.kiss@example.com',
        'MGT',
        110228,
        true,
        true,
        '{"role": "Product Owner", "experience": 18, "certifications": ["AWS"]}'
    ),
    (
        'Szabo Kata',
        'kata.szabo@example.com',
        'HR',
        144481,
        true,
        true,
        '{"role": "Recruiter", "experience": 7}'
    ),
    (
        'Varga Janos',
        'janos.varga@example.com',
        'ENG',
        120849,
        true,
        true,
        '{"role": "QA Engineer", "experience": 16, "tags": ["docker"]}'
    ),
    (
        'Szabo Ferenc',
        'ferenc.szabo@example.com',
        'DSG',
        80948,
        true,
        true,
        '{"role": "UI Designer", "experience": 13}'
    ),
    (
        'Varga David',
        'david.varga@example.com',
        'MGT',
        125632,
        true,
        true,
        '{"role": "Product Owner", "experience": 12, "certifications": ["CISSP", "SCRUM"]}'
    ),
    (
        'Kelemen Gabor',
        'gabor.kelemen@example.com',
        'DSG',
        141665,
        true,
        true,
        '{"role": "UI Designer", "experience": 15}'
    ),
    (
        'Szabo Orsolya',
        'orsolya.szabo@example.com',
        'ENG',
        145386,
        true,
        true,
        '{"role": "Senior Developer", "experience": 10, "tags": ["go", "aws"]}'
    ),
    (
        'Nagy Gyorgy',
        'gyorgy.nagy@example.com',
        'MGT',
        59937,
        true,
        true,
        '{"role": "Team Lead", "experience": 3}'
    ),
    (
        'Juhasz Janos',
        'janos.juhasz@example.com',
        'DSG',
        80911,
        true,
        true,
        '{"role": "UI Designer", "experience": 5}'
    ),
    (
        'Molnar Gabor',
        'gabor.molnar@example.com',
        'HR',
        60405,
        true,
        true,
        '{"role": "Recruiter", "experience": 3}'
    ),
    (
        'Molnar Laszlo',
        'laszlo.molnar@example.com',
        'FIN',
        129079,
        true,
        true,
        '{"role": "Accountant", "experience": 14}'
    ),
    (
        'Farkas Hugo',
        'hugo.farkas@example.com',
        'OPS',
        49721,
        true,
        true,
        '{"role": "Sysadmin", "experience": 2}'
    ),
    (
        'Hegedus Zsuzsanna',
        'zsuzsanna.hegedus@example.com',
        'OPS',
        77693,
        true,
        true,
        '{"role": "Sysadmin", "experience": 20}'
    ),
    (
        'Molnar David',
        'david.molnar@example.com',
        'FIN',
        112048,
        true,
        true,
        '{"role": "Accountant", "experience": 2}'
    ),
    (
        'Simon Adam',
        'adam.simon@example.com',
        'ENG',
        45010,
        true,
        true,
        '{"role": "Developer", "experience": 18, "tags": ["react", "go", "python"]}'
    ),
    (
        'Kelemen Adam',
        'adam.kelemen@example.com',
        'DSG',
        119162,
        true,
        true,
        '{"role": "UI Designer", "experience": 9}'
    ),
    (
        'Toth Attila',
        'attila.toth@example.com',
        'MGT',
        148844,
        true,
        true,
        '{"role": "Product Owner", "experience": 19}'
    ),
    (
        'Nagy Zsuzsanna',
        'zsuzsanna.nagy@example.com',
        'SALES',
        122125,
        true,
        false,
        '{"role": "Account Manager", "experience": 9}'
    ),
    (
        'Takacs Ferenc',
        'ferenc.takacs@example.com',
        'HR',
        94610,
        true,
        true,
        '{"role": "Recruiter", "experience": 14}'
    ),
    (
        'Toth Elemer',
        'elemer.toth@example.com',
        'ENG',
        85808,
        true,
        true,
        '{"role": "Architect", "experience": 4, "tags": ["docker", "java"]}'
    ),
    (
        'Hegedus Hugo',
        'hugo.hegedus@example.com',
        'ENG',
        78303,
        true,
        false,
        '{"role": "Architect", "experience": 7, "tags": ["sql"]}'
    ),
    (
        'Nemeth Luca',
        'luca.nemeth@example.com',
        'OPS',
        52212,
        true,
        true,
        '{"role": "DevOps Engineer", "experience": 12}'
    ),
    (
        'Kiss Bela',
        'bela.kiss@example.com',
        'DSG',
        75589,
        true,
        true,
        '{"role": "UI Designer", "experience": 1}'
    ),
    (
        'Simon Luca',
        'luca.simon@example.com',
        'SALES',
        77881,
        true,
        true,
        '{"role": "Account Manager", "experience": 17}'
    ),
    (
        'Farkas Panna',
        'panna.farkas@example.com',
        'MGT',
        88955,
        true,
        true,
        '{"role": "Project Manager", "experience": 10}'
    ),
    (
        'Farkas Tamas',
        'tamas.farkas@example.com',
        'OPS',
        107008,
        true,
        true,
        '{"role": "Sysadmin", "experience": 12}'
    ),
    (
        'Varga Cecil',
        'cecil.varga@example.com',
        'FIN',
        102784,
        true,
        true,
        '{"role": "Accountant", "experience": 3}'
    ),
    (
        'Lakatos Mari',
        'mari.lakatos@example.com',
        'FIN',
        50856,
        true,
        true,
        '{"role": "Financial Analyst", "experience": 16}'
    ),
    (
        'Takacs Attila',
        'attila.takacs@example.com',
        'FIN',
        131891,
        true,
        true,
        '{"role": "Accountant", "experience": 12}'
    ),
    (
        'Molnar Reka',
        'reka.molnar@example.com',
        'ENG',
        135641,
        true,
        true,
        '{"role": "QA Engineer", "experience": 6, "tags": ["vue", "react", "go", "kubernetes"]}'
    ),
    (
        'Simon Andras',
        'andras.simon@example.com',
        'ENG',
        131729,
        false,
        true,
        '{"role": "Senior Developer", "experience": 7, "tags": ["aws"]}'
    ),
    (
        'Nemeth Peter',
        'peter.nemeth@example.com',
        'HR',
        145173,
        true,
        true,
        '{"role": "Recruiter", "experience": 13}'
    ),
    (
        'Meszaros Jozsef',
        'jozsef.meszaros@example.com',
        'MGT',
        83662,
        true,
        true,
        '{"role": "Project Manager", "experience": 2}'
    ),
    (
        'Szabo Kata',
        'kata.szabo@example.com',
        'SALES',
        45663,
        true,
        true,
        '{"role": "Account Manager", "experience": 13}'
    ),
    (
        'Varga Sandor',
        'sandor.varga@example.com',
        'DSG',
        129843,
        true,
        true,
        '{"role": "UX Researcher", "experience": 3}'
    ),
    (
        'Papp Laszlo',
        'laszlo.papp@example.com',
        'MGT',
        56399,
        true,
        true,
        '{"role": "Project Manager", "experience": 17}'
    ),
    (
        'Balog Elemer',
        'elemer.balog@example.com',
        'DSG',
        99556,
        false,
        true,
        '{"role": "Graphic Designer", "experience": 4}'
    ),
    (
        'Balog David',
        'david.balog@example.com',
        'ENG',
        47152,
        true,
        true,
        '{"role": "Architect", "experience": 1, "tags": ["aws", "docker", "go", "js"]}'
    ),
    (
        'Balog Cecil',
        'cecil.balog@example.com',
        'OPS',
        147452,
        false,
        true,
        '{"role": "Sysadmin", "experience": 3}'
    ),
    (
        'Kiss Laszlo',
        'laszlo.kiss@example.com',
        'HR',
        50149,
        true,
        true,
        '{"role": "Recruiter", "experience": 2}'
    ),
    (
        'Simon Attila',
        'attila.simon@example.com',
        'HR',
        118748,
        true,
        true,
        '{"role": "Recruiter", "experience": 20}'
    ),
    (
        'Toth Jozsef',
        'jozsef.toth@example.com',
        'DSG',
        115549,
        false,
        true,
        '{"role": "UX Researcher", "experience": 5}'
    ),
    (
        'Balog Hugo',
        'hugo.balog@example.com',
        'MGT',
        49614,
        true,
        true,
        '{"role": "Product Owner", "experience": 11}'
    ),
    (
        'Balog Reka',
        'reka.balog@example.com',
        'DSG',
        107133,
        true,
        true,
        '{"role": "UI Designer", "experience": 9}'
    ),
    (
        'Nagy Zsuzsanna',
        'zsuzsanna.nagy@example.com',
        'MGT',
        140473,
        true,
        true,
        '{"role": "Team Lead", "experience": 7, "certifications": ["AWS"]}'
    ),
    (
        'Molnar Adam',
        'adam.molnar@example.com',
        'DSG',
        82421,
        true,
        true,
        '{"role": "Graphic Designer", "experience": 20}'
    ),
    (
        'Toth Cecil',
        'cecil.toth@example.com',
        'DSG',
        84572,
        true,
        false,
        '{"role": "UI Designer", "experience": 8}'
    ),
    (
        'Szabo Tamas',
        'tamas.szabo@example.com',
        'SALES',
        88151,
        true,
        true,
        '{"role": "Sales Rep", "experience": 15}'
    ),
    (
        'Farkas Adam',
        'adam.farkas@example.com',
        'ENG',
        55154,
        true,
        true,
        '{"role": "Senior Developer", "experience": 16, "tags": ["docker", "kubernetes"]}'
    ),
    (
        'Kelemen Bela',
        'bela.kelemen@example.com',
        'ENG',
        59978,
        true,
        true,
        '{"role": "Senior Developer", "experience": 6, "tags": ["python", "js"]}'
    ),
    (
        'Balog Mari',
        'mari.balog@example.com',
        'ENG',
        106216,
        true,
        true,
        '{"role": "Architect", "experience": 11, "tags": ["sql", "js", "vue", "java"]}'
    ),
    (
        'Nagy Timea',
        'timea.nagy@example.com',
        'SALES',
        142975,
        true,
        true,
        '{"role": "Account Manager", "experience": 9}'
    ),
    (
        'Balog Peter',
        'peter.balog@example.com',
        'DSG',
        61714,
        true,
        true,
        '{"role": "UX Researcher", "experience": 12}'
    ),
    (
        'Juhasz Timea',
        'timea.juhasz@example.com',
        'HR',
        137795,
        false,
        true,
        '{"role": "HR Specialist", "experience": 12}'
    ),
    (
        'Toth Zoltan',
        'zoltan.toth@example.com',
        'FIN',
        141620,
        true,
        true,
        '{"role": "Accountant", "experience": 4}'
    ),
    (
        'Molnar Andras',
        'andras.molnar@example.com',
        'DSG',
        136601,
        true,
        true,
        '{"role": "UX Researcher", "experience": 13}'
    ),
    (
        'Meszaros Luca',
        'luca.meszaros@example.com',
        'FIN',
        119485,
        true,
        true,
        '{"role": "Financial Analyst", "experience": 10}'
    ),
    (
        'Takacs Orsolya',
        'orsolya.takacs@example.com',
        'SALES',
        133271,
        true,
        true,
        '{"role": "Account Manager", "experience": 1}'
    ),
    (
        'Nemeth Cecil',
        'cecil.nemeth@example.com',
        'DSG',
        52919,
        true,
        false,
        '{"role": "UX Researcher", "experience": 10}'
    ),
    (
        'Balog Kata',
        'kata.balog@example.com',
        'FIN',
        120126,
        true,
        true,
        '{"role": "Financial Analyst", "experience": 17}'
    ),
    (
        'Papp Sara',
        'sara.papp@example.com',
        'HR',
        102900,
        true,
        true,
        '{"role": "HR Specialist", "experience": 16}'
    ),
    (
        'Szabo Zoltan',
        'zoltan.szabo@example.com',
        'HR',
        77247,
        false,
        true,
        '{"role": "Recruiter", "experience": 1}'
    ),
    (
        'Simon Zsuzsanna',
        'zsuzsanna.simon@example.com',
        'ENG',
        61118,
        true,
        true,
        '{"role": "QA Engineer", "experience": 6, "tags": ["vue", "js", "react"]}'
    ),
    (
        'Lakatos Attila',
        'attila.lakatos@example.com',
        'SALES',
        126430,
        true,
        true,
        '{"role": "Sales Rep", "experience": 13}'
    ),
    (
        'Balog Istvan',
        'istvan.balog@example.com',
        'SALES',
        145349,
        true,
        false,
        '{"role": "Account Manager", "experience": 19}'
    ),
    (
        'Kovacs Ferenc',
        'ferenc.kovacs@example.com',
        'OPS',
        130097,
        false,
        true,
        '{"role": "Sysadmin", "experience": 17}'
    ),
    (
        'Balog Bela',
        'bela.balog@example.com',
        'DSG',
        120595,
        true,
        true,
        '{"role": "UI Designer", "experience": 13}'
    ) ON CONFLICT (id) DO NOTHING;
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