-- PostgreSQL Test Bloat Generation Script
-- This script creates test tables with bloat to simulate vacuum scenarios

-- Create test schema
CREATE SCHEMA IF NOT EXISTS bloat_test;
SET search_path TO bloat_test, public;

-- Drop existing test tables if they exist
DROP TABLE IF EXISTS bloat_test.high_bloat_table;
DROP TABLE IF EXISTS bloat_test.critical_bloat_table;
DROP TABLE IF EXISTS bloat_test.update_heavy_table;

-- Table 1: High Bloat Table (simulates 15% bloat)
CREATE TABLE bloat_test.high_bloat_table (
    id SERIAL PRIMARY KEY,
    data VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert initial data
INSERT INTO bloat_test.high_bloat_table (data)
SELECT 'Initial data row ' || generate_series(1, 10000)
FROM generate_series(1, 10000);

-- Update every row to create dead tuples (this creates bloat)
UPDATE bloat_test.high_bloat_table
SET data = data || ' - updated', updated_at = NOW();

-- Delete some rows to create more bloat
DELETE FROM bloat_test.high_bloat_table
WHERE id % 10 = 0;

-- Insert more data after deletes to fragment the table
INSERT INTO bloat_test.high_bloat_table (data)
SELECT 'New data after delete ' || generate_series(1, 5000)
FROM generate_series(1, 5000);

-- Table 2: Critical Bloat Table (simulates 35% bloat)
CREATE TABLE bloat_test.critical_bloat_table (
    id SERIAL PRIMARY KEY,
    payload TEXT,
    metadata JSONB,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert lots of data with large payloads
INSERT INTO bloat_test.critical_bloat_table (payload, metadata, status)
SELECT
    repeat('Lorem ipsum dolor sit amet ' || generate_series, 100) as payload,
    jsonb_build_object(
        'seq', generate_series,
        'data', md5(random()::text)
    ) as metadata,
    'active' as status
FROM generate_series(1, 50000);

-- Perform multiple UPDATE cycles to create massive bloat
DO $$
DECLARE
    i INT;
BEGIN
    FOR i IN 1..5 LOOP
        UPDATE bloat_test.critical_bloat_table
        SET payload = payload || ' - cycle ' || i,
            updated_at = NOW()
        WHERE id % 3 = 0;

        DELETE FROM bloat_test.critical_bloat_table
        WHERE id % 7 = 0 AND id < 40000;

        -- Insert new rows after deletions
        INSERT INTO bloat_test.critical_bloat_table (payload, metadata, status)
        SELECT
            repeat('New data cycle ' || i || ' row ' || generate_series, 50) as payload,
            jsonb_build_object('seq', generate_series, 'cycle', i) as metadata,
            'active' as status
        FROM generate_series(1, 5000);
    END LOOP;
END $$;

-- Table 3: Update-Heavy Table (simulates table that needs vacuum)
CREATE TABLE bloat_test.update_heavy_table (
    id SERIAL PRIMARY KEY,
    counter INT DEFAULT 0,
    name VARCHAR(200),
    description TEXT,
    last_modified TIMESTAMP DEFAULT NOW()
);

-- Insert initial data
INSERT INTO bloat_test.update_heavy_table (name, description)
SELECT
    'Item ' || generate_series,
    'Description for item ' || generate_series || ' with some extra text'
FROM generate_series(1, 20000);

-- Perform updates to create moderate bloat
UPDATE bloat_test.update_heavy_table
SET counter = counter + 1,
    last_modified = NOW()
WHERE id % 5 = 0;

-- Update again
UPDATE bloat_test.update_heavy_table
SET description = description || ' - modified',
    last_modified = NOW()
WHERE id % 3 = 0;

-- Analyze tables to update statistics
ANALYZE bloat_test.high_bloat_table;
ANALYZE bloat_test.critical_bloat_table;
ANALYZE bloat_test.update_heavy_table;

-- Display bloat information
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_stat_get_dead_tuples(c.oid) AS dead_tuples,
    (pg_stat_get_dead_tuples(c.oid)::FLOAT / NULLIF(pg_stat_get_live_tuples(c.oid) + pg_stat_get_dead_tuples(c.oid), 0) * 100)::NUMERIC(5,2) AS dead_tuple_percent
FROM pg_tables t
JOIN pg_class c ON c.relname = t.tablename
WHERE t.schemaname = 'bloat_test'
ORDER BY tablename;

-- Show vacuum information
SELECT
    schemaname,
    tablename,
    last_vacuum,
    last_autovacuum,
    vacuum_count,
    autovacuum_count
FROM pg_stat_user_tables
WHERE schemaname = 'bloat_test'
ORDER BY tablename;

-- Instructions for manual vacuum
-- Uncomment below lines to manually vacuum and see the difference:

-- VACUUM (VERBOSE, ANALYZE) bloat_test.high_bloat_table;
-- VACUUM (VERBOSE, FULL, ANALYZE) bloat_test.critical_bloat_table;
-- VACUUM (VERBOSE, ANALYZE) bloat_test.update_heavy_table;

-- Check bloat again after vacuum
-- SELECT
--     schemaname,
--     tablename,
--     pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS table_size,
--     pg_stat_get_dead_tuples(c.oid) AS dead_tuples
-- FROM pg_tables t
-- JOIN pg_class c ON c.relname = t.tablename
-- WHERE t.schemaname = 'bloat_test'
-- ORDER BY tablename;
