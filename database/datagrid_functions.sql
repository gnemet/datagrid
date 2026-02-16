-- Datagrid Database Functions
-- This file contains all the functions needed for dynamic SQL generation
-- and consolidated data execution for the Datagrid application.
SET search_path TO datagrid;
-- Function to generate Grid SQL with CTE-based LOV joins, filtering, sorting, and pagination
CREATE OR REPLACE FUNCTION datagrid_get_grid_sql(p_data jsonb) RETURNS text AS $$
DECLARE v_sql text;
v_columns text;
v_ctes text := '';
v_joins text := '';
v_where text := ' WHERE true';
v_tableName text := quote_ident(p_data->>'tableName');
v_lov jsonb;
v_filter_key text;
v_filter_vals jsonb;
v_idx int := 1;
v_join_type text;
v_limit int := (p_data->>'limit')::int;
v_offset int := (p_data->>'offset')::int;
v_order text := p_data->>'order';
BEGIN -- Build columns list
SELECT string_agg(
        CASE
            WHEN (c->>'alias') IS NOT NULL THEN (c->>'name') || ' AS ' || quote_ident(c->>'alias')
            ELSE (c->>'name')
        END,
        ', '
    ) INTO v_columns
FROM jsonb_array_elements(p_data->'columns') AS c;
-- Build LOV CTEs and Joins
IF p_data ? 'lovs' THEN FOR v_lov IN
SELECT *
FROM jsonb_array_elements(p_data->'lovs') LOOP v_ctes := v_ctes || (
        CASE
            WHEN v_ctes = '' THEN 'WITH '
            ELSE ', '
        END
    ) || 'lov' || v_idx || ' AS (SELECT (elem->>''code'')::text as code, (elem->>''label'')::text as label FROM jsonb_array_elements(''' || (v_lov->'values')::text || '''::jsonb) elem)';
v_join_type := COALESCE(v_lov->>'join', 'LEFT');
v_joins := v_joins || ' ' || v_join_type || ' JOIN lov' || v_idx || ' ON src.' || quote_ident(v_lov->>'column') || ' = lov' || v_idx || '.code';
v_idx := v_idx + 1;
END LOOP;
END IF;
-- Cleanup trailing space for CTEs
IF v_ctes != '' THEN v_ctes := v_ctes || ' ';
END IF;
-- Build Filters
IF p_data ? 'filters' THEN FOR v_filter_key,
v_filter_vals IN
SELECT *
FROM jsonb_each(p_data->'filters') LOOP IF jsonb_typeof(v_filter_vals) = 'array'
    AND jsonb_array_length(v_filter_vals) > 0 THEN v_where := v_where || ' AND src.' || quote_ident(v_filter_key) || ' IN (' || (
        SELECT string_agg('''' || elem.value || '''', ', ')
        FROM jsonb_array_elements_text(v_filter_vals) AS elem
    ) || ')';
ELSIF jsonb_typeof(v_filter_vals) = 'string'
AND v_filter_vals != '""'::jsonb THEN v_where := v_where || ' AND src.' || quote_ident(v_filter_key) || ' = ' || quote_literal(v_filter_vals#>>'{}');
END IF;
END LOOP;
END IF;
v_sql := v_ctes || 'SELECT ' || v_columns || ' FROM ' || v_tableName || ' AS src' || v_joins || v_where;
IF v_order IS NOT NULL
AND v_order != '' THEN v_sql := v_sql || ' ' || v_order;
END IF;
IF v_limit > 0 THEN v_sql := v_sql || ' LIMIT ' || v_limit;
END IF;
IF v_offset > 0 THEN v_sql := v_sql || ' OFFSET ' || v_offset;
END IF;
RETURN v_sql;
END;
$$ LANGUAGE plpgsql;
-- Function to generate Pivot SQL with LOV support
CREATE OR REPLACE FUNCTION datagrid_get_pivot_sql(p_data jsonb) RETURNS text AS $$
DECLARE v_sql text;
v_dims text;
v_ctes text := '';
v_joins text := '';
v_measures text;
v_where text := ' WHERE true';
v_tableName text := p_data->>'tableName';
v_lov jsonb;
v_filter_key text;
v_filter_vals jsonb;
v_idx int := 1;
v_group_by text;
BEGIN -- Build dimensions (rows + cols)
SELECT string_agg(
        CASE
            WHEN (d->>'isLOV')::boolean THEN 'lov' || (d->>'lovIdx') || '.label'
            ELSE 'src.' || quote_ident(d->>'column')
        END || ' AS ' || quote_ident(d->>'column'),
        ', '
    ),
    string_agg(
        CASE
            WHEN (d->>'isLOV')::boolean THEN 'lov' || (d->>'lovIdx') || '.label'
            ELSE 'src.' || quote_ident(d->>'column')
        END,
        ', '
    ) INTO v_dims,
    v_group_by
FROM jsonb_array_elements(p_data->'dimensions') AS d;
-- Build LOV CTEs and Joins
IF p_data ? 'lovs' THEN FOR v_lov IN
SELECT *
FROM jsonb_array_elements(p_data->'lovs') LOOP v_ctes := v_ctes || (
        CASE
            WHEN v_ctes = '' THEN 'WITH '
            ELSE ', '
        END
    ) || 'lov' || v_idx || ' AS (SELECT (elem->>''code'')::text as code, (elem->>''label'')::text as label FROM jsonb_array_elements(''' || (v_lov->'values')::text || '''::jsonb) elem)';
v_joins := v_joins || ' LEFT JOIN lov' || v_idx || ' ON src.' || quote_ident(v_lov->>'column') || ' = lov' || v_idx || '.code';
v_idx := v_idx + 1;
END LOOP;
END IF;
-- Cleanup trailing space for CTEs
IF v_ctes != '' THEN v_ctes := v_ctes || ' ';
END IF;
-- Build measures
SELECT string_agg(
        (elem->>'func') || '(src.' || quote_ident(elem->>'column') || ') AS ' || quote_ident(elem->>'alias'),
        ', '
    ) INTO v_measures
FROM jsonb_array_elements(p_data->'measures') AS elem;
-- Build Filters
IF p_data ? 'filters' THEN FOR v_filter_key,
v_filter_vals IN
SELECT *
FROM jsonb_each(p_data->'filters') LOOP IF jsonb_typeof(v_filter_vals) = 'array'
    AND jsonb_array_length(v_filter_vals) > 0 THEN v_where := v_where || ' AND src.' || quote_ident(v_filter_key) || ' IN (' || (
        SELECT string_agg('''' || elem.value || '''', ', ')
        FROM jsonb_array_elements_text(v_filter_vals) AS elem
    ) || ')';
ELSIF jsonb_typeof(v_filter_vals) = 'string'
AND v_filter_vals != '""'::jsonb THEN v_where := v_where || ' AND src.' || quote_ident(v_filter_key) || ' = ' || quote_literal(v_filter_vals#>>'{}');
END IF;
END LOOP;
END IF;
v_sql := v_ctes || 'SELECT ' || v_dims || ', ' || v_measures || ' FROM ' || v_tableName || ' AS src' || v_joins || v_where || ' GROUP BY ' || v_group_by;
RETURN v_sql;
END;
$$ LANGUAGE plpgsql;
-- Function to execute Datagrid SQL and return data in specified format (JSON/CSV)
CREATE OR REPLACE FUNCTION datagrid_execute(
        p_data jsonb,
        p_view text DEFAULT 'grid',
        p_format text DEFAULT 'json'
    ) RETURNS text AS $$
DECLARE v_sql text;
v_res jsonb;
v_csv text;
BEGIN -- 1. Generate SQL
IF p_view = 'pivot' THEN v_sql := datagrid_get_pivot_sql(p_data);
ELSE v_sql := datagrid_get_grid_sql(p_data);
END IF;
-- 2. Return based on format
IF p_format = 'json' THEN EXECUTE 'SELECT json_agg(t) FROM (' || v_sql || ') t' INTO v_res;
RETURN COALESCE(v_res::text, '[]');
ELSIF p_format = 'csv' THEN -- Manual CSV construction for PL/pgSQL
EXECUTE 'SELECT string_agg(csv_line, chr(10)) FROM (
            SELECT string_agg(quote_nullable(val), '','') as csv_line FROM (
                SELECT (jsonb_each_text(to_jsonb(t))).value as val FROM (' || v_sql || ') t
            ) sub
        ) sub2' INTO v_csv;
RETURN COALESCE(v_csv, '');
ELSE RAISE EXCEPTION 'Unsupported format: %',
p_format;
END IF;
END;
$$ LANGUAGE plpgsql;
-- Function to execute a raw SQL query with a JSONB data parameter (using $1 for data)
CREATE OR REPLACE FUNCTION datagrid_execute_sql(p_sql text, p_data jsonb) RETURNS text AS $$
DECLARE v_res jsonb;
BEGIN EXECUTE 'SELECT jsonb_agg(t) FROM (' || p_sql || ') t' INTO v_res USING p_data;
RETURN COALESCE(v_res::text, '[]');
END;
$$ LANGUAGE plpgsql;