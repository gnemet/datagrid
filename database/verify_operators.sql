-- Verification Script for JSONB Utility Suite
SELECT '1. Typed Extraction' AS test;
SELECT '{"a": 12.3}'::jsonb-># 'a' AS numeric_val,
    '{"b": true}'::jsonb->& 'b' AS boolean_val,
    '{"c": [1,2,3]}'::jsonb->>>> '' AS text_array,
    '{"d": {"e": 42}}'::jsonb#># '{d,e}' AS path_numeric;
SELECT '2. Unordered Equality (===)' AS test;
SELECT jsonb_array_equal_ignore_order('[1,2,3]'::jsonb, '[3,2,1]'::jsonb) AS equal_true,
    jsonb_array_equal_ignore_order('[1,2,3]'::jsonb, '[1,2]'::jsonb) AS equal_false;
SELECT '3. Deep Merge (|||)' AS test;
SELECT jsonb_deep_merge(
        '{"a":1, "x":{"y":2}}'::jsonb,
        '{"b":3, "x":{"z":4}}'::jsonb
    ) AS merged_object,
    jsonb_deep_merge(
        '[{"id":1, "v":"old"}]'::jsonb,
        '[{"id":1, "v":"new"}, {"id":2, "v":"fresh"}]'::jsonb
    ) AS merged_array;
SELECT '4. Automated Casts' AS test;
SELECT '["apple", "banana"]'::jsonb::text [] AS cast_text_array,
    '[1.1, 2.2, 3.3]'::jsonb::numeric [] AS cast_numeric_array,
    '[true, false]'::jsonb::boolean [] AS cast_boolean_array,
    '["2025-01-01", "2025-12-31"]'::jsonb::daterange AS cast_daterange;
SELECT '5. Transformations' AS test;
SELECT jsonb_recursive_camel_case(
        '{"first_name": "John", "last_name": "Doe", "sub_obj": {"user_id": 1}}'::jsonb
    ) AS camel_case,
    jsonb_unarray('[42]'::jsonb) AS unarray_scalar,
    jsonb_filter_keys('{"a":1, "b":2, "c":3}'::jsonb, '{a,c}'::text []) AS filtered;
SELECT '6. Advanced Similarity & Collection' AS test;
SELECT jsonb_array_similar('[1,2,3]'::jsonb, '[1,2,4]'::jsonb) AS similar_true,
    jsonb_array_similarity('[1,2,3]'::jsonb, '[1,2,4,5]'::jsonb) AS similarity_score,
    jsonb_array_stats('[1,2,3,4,5]'::jsonb)->>'avg' AS average_val,
    jsonb_distinct_array(
        '[{"id":1, "n":"a"}, {"id":1, "n":"b"}, {"id":2, "n":"c"}]'::jsonb,
        'id'
    ) AS distinct_by_id;