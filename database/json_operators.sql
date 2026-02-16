-- Comprehensive JSONB Utility Suite for Datagrid
-- Ported and adapted from Zafir/MKE/MKS logic
-- Supports: text, boolean, numeric, integer, timestamp, tstzrange (Scalar & Array, Key & Path)
SET search_path TO public,
    datagrid;
-------------------------------------------------------------------------------
-- 1. Helper Functions (Modern SQL Syntax)
-------------------------------------------------------------------------------
-- 1.1 Scalar Extractions
CREATE OR REPLACE FUNCTION public.jsonb_extract_numeric(j jsonb, key text) RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j->>key)::numeric;
CREATE OR REPLACE FUNCTION public.jsonb_extract_bigint(j jsonb, key text) RETURNS bigint LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j->>key)::bigint;
CREATE OR REPLACE FUNCTION public.jsonb_extract_integer(j jsonb, key text) RETURNS integer LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j->>key)::integer;
CREATE OR REPLACE FUNCTION public.jsonb_extract_boolean(j jsonb, key text) RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j->>key)::boolean;
CREATE OR REPLACE FUNCTION public.jsonb_extract_timestamp(j jsonb, key text) RETURNS timestamp LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j->>key)::timestamp;
CREATE OR REPLACE FUNCTION public.jsonb_extract_tstzrange(j jsonb, key text) RETURNS tstzrange LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j->>key)::tstzrange;
-- 1.2 Path Extractions
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_numeric(j jsonb, path text []) RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j#>>path)::numeric;
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_bigint(j jsonb, path text []) RETURNS bigint LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j#>>path)::bigint;
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_integer(j jsonb, path text []) RETURNS integer LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j#>>path)::integer;
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_boolean(j jsonb, path text []) RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j#>>path)::boolean;
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_timestamp(j jsonb, path text []) RETURNS timestamp LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j#>>path)::timestamp;
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_tstzrange(j jsonb, path text []) RETURNS tstzrange LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN (j#>>path)::tstzrange;
-- 1.3 Array Extractions (with flattening if needed)
CREATE OR REPLACE FUNCTION public.jsonb_extract_text_array(j jsonb, key text) RETURNS text [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN j IS NULL THEN NULL
        WHEN jsonb_typeof(j) = 'array' THEN CASE
            WHEN key = '' THEN ARRAY(
                SELECT jsonb_array_elements_text(j)
            )
            ELSE ARRAY(
                SELECT (elem->>key)
                FROM jsonb_array_elements(j) elem
            )
        END
        ELSE ARRAY(
            SELECT jsonb_array_elements_text(j->key)
        )
    END;
CREATE OR REPLACE FUNCTION public.jsonb_extract_numeric_array(j jsonb, key text) RETURNS numeric [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN j IS NULL THEN NULL
        WHEN jsonb_typeof(j) = 'array' THEN CASE
            WHEN key = '' THEN ARRAY(
                SELECT jsonb_array_elements_text(j)::numeric
            )
            ELSE ARRAY(
                SELECT (elem->>key)::numeric
                FROM jsonb_array_elements(j) elem
            )
        END
        ELSE ARRAY(
            SELECT jsonb_array_elements_text(j->key)::numeric
        )
    END;
CREATE OR REPLACE FUNCTION public.jsonb_extract_int_array(j jsonb, key text) RETURNS integer [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN j IS NULL THEN NULL
        WHEN jsonb_typeof(j) = 'array' THEN CASE
            WHEN key = '' THEN ARRAY(
                SELECT jsonb_array_elements_text(j)::integer
            )
            ELSE ARRAY(
                SELECT (elem->>key)::integer
                FROM jsonb_array_elements(j) elem
            )
        END
        ELSE ARRAY(
            SELECT jsonb_array_elements_text(j->key)::integer
        )
    END;
CREATE OR REPLACE FUNCTION public.jsonb_extract_boolean_array(j jsonb, key text) RETURNS boolean [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN j IS NULL THEN NULL
        WHEN jsonb_typeof(j) = 'array' THEN CASE
            WHEN key = '' THEN ARRAY(
                SELECT jsonb_array_elements_text(j)::boolean
            )
            ELSE ARRAY(
                SELECT (elem->>key)::boolean
                FROM jsonb_array_elements(j) elem
            )
        END
        ELSE ARRAY(
            SELECT jsonb_array_elements_text(j->key)::boolean
        )
    END;
CREATE OR REPLACE FUNCTION public.jsonb_extract_timestamp_array(j jsonb, key text) RETURNS timestamp [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN j IS NULL THEN NULL
        WHEN jsonb_typeof(j) = 'array' THEN CASE
            WHEN key = '' THEN ARRAY(
                SELECT jsonb_array_elements_text(j)::timestamp
            )
            ELSE ARRAY(
                SELECT (elem->>key)::timestamp
                FROM jsonb_array_elements(j) elem
            )
        END
        ELSE ARRAY(
            SELECT jsonb_array_elements_text(j->key)::timestamp
        )
    END;
CREATE OR REPLACE FUNCTION public.jsonb_extract_tstzrange_array(j jsonb, key text) RETURNS tstzrange [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN j IS NULL THEN NULL
        WHEN jsonb_typeof(j) = 'array' THEN CASE
            WHEN key = '' THEN ARRAY(
                SELECT jsonb_array_elements_text(j)::tstzrange
            )
            ELSE ARRAY(
                SELECT (elem->>key)::tstzrange
                FROM jsonb_array_elements(j) elem
            )
        END
        ELSE ARRAY(
            SELECT jsonb_array_elements_text(j->key)::tstzrange
        )
    END;
-- 1.4 Path Array Extractions
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_text_array(j jsonb, path text []) RETURNS text [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN ARRAY(
        SELECT jsonb_array_elements_text(j#>path)
    );
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_numeric_array(j jsonb, path text []) RETURNS numeric [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN ARRAY(
        SELECT jsonb_array_elements_text(j#>path)::numeric
    );
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_int_array(j jsonb, path text []) RETURNS integer [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN ARRAY(
        SELECT jsonb_array_elements_text(j#>path)::integer
    );
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_boolean_array(j jsonb, path text []) RETURNS boolean [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN ARRAY(
        SELECT jsonb_array_elements_text(j#>path)::boolean
    );
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_timestamp_array(j jsonb, path text []) RETURNS timestamp [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN ARRAY(
        SELECT jsonb_array_elements_text(j#>path)::timestamp
    );
CREATE OR REPLACE FUNCTION public.jsonb_path_extract_tstzrange_array(j jsonb, path text []) RETURNS tstzrange [] LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN ARRAY(
        SELECT jsonb_array_elements_text(j#>path)::tstzrange
    );
-------------------------------------------------------------------------------
-- 2. Advanced JSONB Functions (Unordered Equality, Intersection, Merge)
-------------------------------------------------------------------------------
-- 2.1 Unordered Equality (===)
CREATE OR REPLACE FUNCTION public.jsonb_array_equal_ignore_order(j1 jsonb, j2 jsonb) RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT (
        (
            SELECT jsonb_agg(
                    e
                    ORDER BY e
                )
            FROM jsonb_array_elements(j1) AS e
        ) = (
            SELECT jsonb_agg(
                    e
                    ORDER BY e
                )
            FROM jsonb_array_elements(j2) AS e
        )
    ) $$;
-- 2.2 Intersection (&&)
CREATE OR REPLACE FUNCTION public.jsonb_has_intersection(j1 jsonb, j2 jsonb) RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT EXISTS (
        SELECT 1
        FROM jsonb_array_elements(j1) AS a_val
            INNER JOIN jsonb_array_elements(j2) AS b_val ON a_val.value = b_val.value
    ) $$;
-- 2.3 Deep Merge (|||)
CREATE OR REPLACE FUNCTION public.jsonb_deep_merge(in_frst jsonb, in_scnd jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        jsonb_typeof(in_frst)
        WHEN 'object' THEN CASE
            jsonb_typeof(in_scnd)
            WHEN 'object' THEN (
                SELECT jsonb_object_agg(
                        jkey,
                        CASE
                            WHEN scnd.jkey IS NULL
                            AND frst.jkey IS NOT NULL THEN frst.jvalue
                            WHEN frst.jkey IS NULL
                            AND scnd.jkey IS NOT NULL THEN scnd.jvalue
                            WHEN frst.jvalue = scnd.jvalue IS TRUE THEN frst.jvalue
                            WHEN frst.jvalue = scnd.jvalue IS FALSE THEN public.jsonb_deep_merge(frst.jvalue, scnd.jvalue)
                        END
                    )
                FROM jsonb_each(in_frst) AS frst(jkey, jvalue)
                    FULL JOIN jsonb_each(in_scnd) AS scnd(jkey, jvalue) USING (jkey)
            )
            ELSE in_scnd
        END
        WHEN 'array' THEN CASE
            WHEN in_frst->0 ? 'id' THEN (
                SELECT jsonb_agg(
                        elem.value
                        ORDER BY elem.value->'id'
                    ) FILTER (
                        WHERE elem.value IS NOT NULL
                    )
                FROM (
                        SELECT CASE
                                WHEN prnt IS NULL
                                AND chld IS NOT NULL THEN chld
                                WHEN chld IS NULL
                                AND prnt IS NOT NULL THEN prnt
                                WHEN prnt IS NOT NULL
                                AND chld IS NOT NULL THEN public.jsonb_deep_merge(prnt, chld)
                            END
                        FROM jsonb_array_elements (in_frst) AS prnt
                            FULL JOIN jsonb_array_elements (in_scnd) AS chld ON chld->'id' = prnt->'id'
                    ) AS elem(value)
            )
            ELSE CASE
                WHEN jsonb_typeof(in_scnd) = 'array' THEN in_frst || in_scnd
                ELSE in_scnd
            END
        END
        ELSE in_scnd
    END $$;
-- 2.4 Aggregate for Deep Merge
CREATE OR REPLACE AGGREGATE public.jsonb_deep_merge_agg(jsonb) (
        SFUNC = public.jsonb_deep_merge,
        STYPE = jsonb,
        INITCOND = '{}'
    );
-- 2.5 Key filtering (+ operator)
CREATE OR REPLACE FUNCTION public.jsonb_filter_keys(j jsonb, keys text []) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        jsonb_typeof(j)
        WHEN 'string' THEN j
        WHEN 'object' THEN (
            SELECT jsonb_object_agg(
                    o_elem.key,
                    public.jsonb_filter_keys(o_elem.value, keys)
                )
            FROM jsonb_each(j) WITH ORDINALITY AS o_elem(key, value)
            WHERE o_elem.key = ANY (keys)
        )
        WHEN 'array' THEN (
            SELECT jsonb_agg(public.jsonb_filter_keys(a_elem, keys))
            FROM jsonb_array_elements(j) AS a_elem
        )
        ELSE j
    END $$;
-- 2.6 Simple whitelist filter (scalar text)
CREATE OR REPLACE FUNCTION public.jsonb_filter_keys(j jsonb, key text) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN public.jsonb_filter_keys(j, ARRAY [key]);
-------------------------------------------------------------------------------
-- 3. Utility Functions (camelCase, unarray)
-------------------------------------------------------------------------------
-- 3.1 Lowercase first letter
CREATE OR REPLACE FUNCTION public.mks_lc_first(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN lower(left($1, 1)) || right($1, -1);
-- 3.2 camelCase helper (from Zafir)
CREATE OR REPLACE FUNCTION public.mks_camel_case(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN public.mks_lc_first(replace(initcap(replace($1, '_', ' ')), ' ', ''));
-- 3.3 Recursive camelCase
CREATE OR REPLACE FUNCTION public.jsonb_recursive_camel_case(j jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'object' THEN (
            SELECT jsonb_object_agg (
                    public.mks_camel_case (o.key),
                    public.jsonb_recursive_camel_case (o.value)
                )
            FROM jsonb_each (j) AS o (key, value)
        )
        WHEN jsonb_typeof(j) = 'array' THEN (
            SELECT COALESCE (
                    jsonb_agg (public.jsonb_recursive_camel_case (a_elem)),
                    '[]'
                )
            FROM jsonb_array_elements (j) AS a_elem
        )
        ELSE j
    END $$;
-- 3.4 Smart unarray
CREATE OR REPLACE FUNCTION public.jsonb_unarray(j jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN jsonb_array_length(public.jsonb_safe_array(j)) = 1 THEN j->0
        ELSE j
    END;
-- 3.5 Safe Type Wrappers
CREATE OR REPLACE FUNCTION public.jsonb_safe_object(j jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN jsonb_typeof(j) = 'object' THEN j
        ELSE '{}'::jsonb
    END;
CREATE OR REPLACE FUNCTION public.jsonb_safe_array(j jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN CASE
        WHEN jsonb_typeof(j) = 'array' THEN j
        ELSE '[]'::jsonb
    END;
CREATE OR REPLACE FUNCTION public.jsonb_safe_boolean(j jsonb, def boolean = false) RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN COALESCE((j)::boolean, def);
-- 3.6 Key Extraction & Values
CREATE OR REPLACE FUNCTION public.jsonb_get_keys(j jsonb) RETURNS text [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT ARRAY(
        SELECT jsonb_object_keys(public.jsonb_safe_object(j))
    ) $$;
CREATE OR REPLACE FUNCTION public.jsonb_get_keys_by_value(j jsonb, val jsonb) RETURNS text [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT ARRAY(
        SELECT key
        FROM jsonb_each(public.jsonb_safe_object(j))
        WHERE value = val
    ) $$;
-- 3.7 Collection Stats
CREATE OR REPLACE FUNCTION public.jsonb_array_stats(j jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT jsonb_build_object(
        'count',
        count(v::numeric),
        'sum',
        sum(v::numeric),
        'avg',
        avg(v::numeric),
        'max',
        max(v::numeric),
        'min',
        min(v::numeric)
    )
FROM jsonb_array_elements_text(public.jsonb_safe_array(j)) v $$;
-- 3.8 Distinct Array Elements
CREATE OR REPLACE FUNCTION public.jsonb_distinct_array(j jsonb, key text) RETURNS jsonb LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT jsonb_agg(
        v
        ORDER BY idx
    )
FROM (
        SELECT DISTINCT ON (value->>key) value as v,
            ordinality as idx
        FROM jsonb_array_elements(public.jsonb_safe_array(j)) WITH ORDINALITY
    ) q $$;
-------------------------------------------------------------------------------
-- 4. Fuzzy & Similarity Logic
-------------------------------------------------------------------------------
-- 4.1 Array Similarity (Ported from mke_array_in_second_percent)
CREATE OR REPLACE FUNCTION public.jsonb_array_similarity(j1 jsonb, j2 jsonb) RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT COALESCE(
        (
            SELECT count(a)::numeric / NULLIF(jsonb_array_length(j1), 0)::numeric
            FROM jsonb_array_elements_text(j1) AS a
                JOIN jsonb_array_elements_text(j2) AS b ON b = a
            GROUP BY j1
        ),
        CASE
            WHEN jsonb_array_length(j1) = 0 THEN 1.0
            ELSE 0.0
        END
    ) $$;
-- 4.2 Similarity Operator Proc (Ported from mke_array_in_second)
CREATE OR REPLACE FUNCTION public.jsonb_array_similar(j1 jsonb, j2 jsonb) RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT public.jsonb_array_similarity(j1, j2) >= 0.6 $$;
-------------------------------------------------------------------------------
-- 4. Cast Definitions (Explicitly Listed for Visibility)
-------------------------------------------------------------------------------
-- jsonb to text[]
CREATE OR REPLACE FUNCTION public.jsonb_cast_text_array(j jsonb) RETURNS text [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'array' THEN (
            SELECT COALESCE(array_agg(v), '{}')
            FROM jsonb_array_elements_text(j) v
        )
        ELSE NULL::text []
    END $$;
DROP CAST IF EXISTS (jsonb AS text []);
CREATE CAST (jsonb AS text []) WITH FUNCTION public.jsonb_cast_text_array(jsonb) AS ASSIGNMENT;
-- jsonb to numeric[]
CREATE OR REPLACE FUNCTION public.jsonb_cast_numeric_array(j jsonb) RETURNS numeric [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'array' THEN (
            SELECT COALESCE(array_agg(v::numeric), '{}')
            FROM jsonb_array_elements_text(j) v
        )
        ELSE NULL::numeric []
    END $$;
DROP CAST IF EXISTS (jsonb AS numeric []);
CREATE CAST (jsonb AS numeric []) WITH FUNCTION public.jsonb_cast_numeric_array(jsonb) AS ASSIGNMENT;
-- jsonb to int[]
CREATE OR REPLACE FUNCTION public.jsonb_cast_int_array(j jsonb) RETURNS int [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'array' THEN (
            SELECT COALESCE(array_agg(v::int), '{}')
            FROM jsonb_array_elements_text(j) v
        )
        ELSE NULL::int []
    END $$;
DROP CAST IF EXISTS (jsonb AS int []);
CREATE CAST (jsonb AS int []) WITH FUNCTION public.jsonb_cast_int_array(jsonb) AS ASSIGNMENT;
-- jsonb to bigint[]
CREATE OR REPLACE FUNCTION public.jsonb_cast_bigint_array(j jsonb) RETURNS bigint [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'array' THEN (
            SELECT COALESCE(array_agg(v::bigint), '{}')
            FROM jsonb_array_elements_text(j) v
        )
        ELSE NULL::bigint []
    END $$;
DROP CAST IF EXISTS (jsonb AS bigint []);
CREATE CAST (jsonb AS bigint []) WITH FUNCTION public.jsonb_cast_bigint_array(jsonb) AS ASSIGNMENT;
-- jsonb to boolean[]
CREATE OR REPLACE FUNCTION public.jsonb_cast_boolean_array(j jsonb) RETURNS boolean [] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'array' THEN (
            SELECT COALESCE(array_agg(v::boolean), '{}')
            FROM jsonb_array_elements_text(j) v
        )
        ELSE NULL::boolean []
    END $$;
DROP CAST IF EXISTS (jsonb AS boolean []);
CREATE CAST (jsonb AS boolean []) WITH FUNCTION public.jsonb_cast_boolean_array(jsonb) AS ASSIGNMENT;
-- jsonb to daterange
CREATE OR REPLACE FUNCTION public.jsonb_cast_daterange(j jsonb) RETURNS daterange LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'string' THEN (j->>0)::daterange
        WHEN jsonb_typeof(j) = 'array'
        AND jsonb_array_length(j) = 2 THEN daterange((j->>0)::date, (j->>1)::date)
        ELSE NULL::daterange
    END $$;
DROP CAST IF EXISTS (jsonb AS daterange);
CREATE CAST (jsonb AS daterange) WITH FUNCTION public.jsonb_cast_daterange(jsonb) AS ASSIGNMENT;
-- jsonb to int4range
CREATE OR REPLACE FUNCTION public.jsonb_cast_int4range(j jsonb) RETURNS int4range LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT CASE
        WHEN jsonb_typeof(j) = 'string' THEN (j->>0)::int4range
        WHEN jsonb_typeof(j) = 'array'
        AND jsonb_array_length(j) = 2 THEN int4range((j->>0)::int, (j->>1)::int)
        ELSE NULL::int4range
    END $$;
DROP CAST IF EXISTS (jsonb AS int4range);
CREATE CAST (jsonb AS int4range) WITH FUNCTION public.jsonb_cast_int4range(jsonb) AS ASSIGNMENT;
-------------------------------------------------------------------------------
-- 5. Operator Registration
-------------------------------------------------------------------------------
DO $$
DECLARE row record;
BEGIN FOR row IN
SELECT *
FROM (
        VALUES ('->#', 'public.jsonb_extract_numeric', 'text'),
            ('->^', 'public.jsonb_extract_bigint', 'text'),
            ('->&', 'public.jsonb_extract_boolean', 'text'),
            ('->@', 'public.jsonb_extract_timestamp', 'text'),
            ('->%', 'public.jsonb_extract_tstzrange', 'text'),
            (
                '->>>>',
                'public.jsonb_extract_text_array',
                'text'
            ),
            (
                '->##',
                'public.jsonb_extract_numeric_array',
                'text'
            ),
            ('->^^', 'public.jsonb_extract_int_array', 'text'),
            (
                '->&&',
                'public.jsonb_extract_boolean_array',
                'text'
            ),
            (
                '->@@',
                'public.jsonb_extract_timestamp_array',
                'text'
            ),
            (
                '->%%',
                'public.jsonb_extract_tstzrange_array',
                'text'
            ),
            (
                '#>#',
                'public.jsonb_path_extract_numeric',
                'text[]'
            ),
            (
                '#>^',
                'public.jsonb_path_extract_bigint',
                'text[]'
            ),
            (
                '#>&',
                'public.jsonb_path_extract_boolean',
                'text[]'
            ),
            (
                '#>@',
                'public.jsonb_path_extract_timestamp',
                'text[]'
            ),
            (
                '#>%',
                'public.jsonb_path_extract_tstzrange',
                'text[]'
            ),
            (
                '#>>>>',
                'public.jsonb_path_extract_text_array',
                'text[]'
            ),
            (
                '#>##',
                'public.jsonb_path_extract_numeric_array',
                'text[]'
            ),
            (
                '#>^^',
                'public.jsonb_path_extract_int_array',
                'text[]'
            ),
            (
                '#>&&',
                'public.jsonb_path_extract_boolean_array',
                'text[]'
            ),
            (
                '#>@@',
                'public.jsonb_path_extract_timestamp_array',
                'text[]'
            ),
            (
                '#>%%',
                'public.jsonb_path_extract_tstzrange_array',
                'text[]'
            ),
            (
                '===',
                'public.jsonb_array_equal_ignore_order',
                'jsonb'
            ),
            (
                '==@',
                'public.jsonb_array_similar',
                'jsonb'
            ),
            ('&&', 'public.jsonb_has_intersection', 'jsonb'),
            ('|||', 'public.jsonb_deep_merge', 'jsonb'),
            ('+', 'public.jsonb_filter_keys', 'text[]'),
            ('+', 'public.jsonb_filter_keys', 'text')
    ) AS t(op, func, arg) LOOP BEGIN EXECUTE format(
        'DROP OPERATOR IF EXISTS public.%s (jsonb, %s) CASCADE',
        row.op,
        row.arg
    );
EXCEPTION
WHEN OTHERS THEN NULL;
END;
EXECUTE format(
    'CREATE OPERATOR public.%s (PROCEDURE = %s, LEFTARG = jsonb, RIGHTARG = %s)',
    row.op,
    row.func,
    row.arg
);
END LOOP;
END $$;