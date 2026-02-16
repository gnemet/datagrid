-- Database Unit Tests for Datagrid SQL Functions
-- These tests verify that datagrid_get_grid_sql, datagrid_get_pivot_sql
-- and datagrid_execute generate valid SQL and data.
SET search_path TO datagrid;
-- 1. Test Grid SQL with Multiple LOVs and Filters
DO $$
DECLARE v_data jsonb := '{
        "tableName": "personnel",
        "columns": [
            {"name": "src.id", "alias": "id"},
            {"name": "src.name", "alias": "name"},
            {"name": "lov1.label", "alias": "dept_label"},
            {"name": "lov2.label", "alias": "gender_label"}
        ],
        "lovs": [
            {
                "column": "department",
                "join": "LEFT",
                "values": [
                    {"code": "ENG", "label": "Engineering"},
                    {"code": "SALES", "label": "Sales"}
                ]
            },
            {
                "column": "gender",
                "join": "LEFT",
                "values": [
                    {"code": "Male", "label": "M"},
                    {"code": "Female", "label": "F"}
                ]
            }
        ],
        "filters": {
            "department": ["ENG"]
        }
    }';
v_sql text;
BEGIN v_sql := datagrid_get_grid_sql(v_data);
RAISE NOTICE 'Grid SQL with Multiple LOVs: %',
v_sql;
-- Basic validation
IF v_sql NOT LIKE '%lov1 AS%' THEN RAISE EXCEPTION 'lov1 missing';
END IF;
IF v_sql NOT LIKE '%lov2 AS%' THEN RAISE EXCEPTION 'lov2 missing';
END IF;
IF v_sql NOT LIKE '%LEFT JOIN lov1%' THEN RAISE EXCEPTION 'Join 1 missing';
END IF;
IF v_sql NOT LIKE '%LEFT JOIN lov2%' THEN RAISE EXCEPTION 'Join 2 missing';
END IF;
END $$;
-- 2. Test Pivot SQL with Multiple Dimensions
DO $$
DECLARE v_data jsonb := '{
        "tableName": "personnel",
        "dimensions": [
            {"column": "department", "isLOV": true, "lovIdx": 1},
            {"column": "gender", "isLOV": true, "lovIdx": 2}
        ],
        "measures": [
            {"column": "salary", "func": "SUM", "alias": "total_salary"}
        ],
        "lovs": [
            {
                "column": "department",
                "values": [{"code": "ENG", "label": "Engineering"}]
            },
            {
                "column": "gender",
                "values": [{"code": "Male", "label": "M"}]
            }
        ]
    }';
v_sql text;
BEGIN v_sql := datagrid_get_pivot_sql(v_data);
RAISE NOTICE 'Pivot SQL with Multiple LOVs: %',
v_sql;
IF v_sql NOT LIKE '%lov1.label, lov2.label%' THEN RAISE EXCEPTION 'Dimensions missing or malformed';
END IF;
END $$;
-- 3. Test datagrid_execute (JSON)
DO $$
DECLARE v_data jsonb := '{
        "tableName": "personnel",
        "columns": [
            {"name": "src.name", "alias": "n"},
            {"name": "lov1.label", "alias": "d"}
        ],
        "lovs": [
            {
                "column": "department",
                "values": [{"code": "ENG", "label": "Engineering"}]
            }
        ]
    }';
v_res text;
BEGIN v_res := datagrid_execute(v_data, 'grid', 'json');
RAISE NOTICE 'Execution result (JSON): %',
v_res;
IF v_res NOT LIKE '[{%}]' THEN RAISE EXCEPTION 'JSON format invalid';
END IF;
END $$;
-- 4. Test datagrid_execute (CSV)
DO $$
DECLARE v_data jsonb := '{
        "tableName": "personnel",
        "columns": [
            {"name": "src.name", "alias": "n"}
        ]
    }';
v_res text;
BEGIN v_res := datagrid_execute(v_data, 'grid', 'csv');
RAISE NOTICE 'Execution result (CSV): %',
v_res;
IF v_res = '' THEN RAISE EXCEPTION 'CSV result empty';
END IF;
END $$;