-- Add complex JSON data to personnel for testing custom operators
SET search_path TO datagrid,
    public;
UPDATE personnel
SET data = '{"salary": 120000, "is_senior": true, "tags": ["go", "sql"], "joined": "2023-01-15T10:00:00Z"}'
WHERE name = 'Molnar Sandor';
UPDATE personnel
SET data = '{"salary": 95000, "is_senior": false, "tags": ["fin", "excel"], "joined": "2024-05-20T09:00:00Z"}'
WHERE name = 'Balog Gabor';
UPDATE personnel
SET data = '{"salary": 110000, "is_senior": true, "tags": ["hr", "management"], "joined": "2022-03-10T08:30:00Z"}'
WHERE name = 'Meszaros Zoltan';
-- Verify operators
SELECT name,
    data->#'salary' as sal_num,
    data->& 'is_senior' as senior_bool,
    data->@'joined' as join_ts
FROM personnel;