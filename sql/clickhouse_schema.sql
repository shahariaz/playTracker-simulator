-- ClickHouse table creation script for PlayTracker Simulator
-- Run this script to create the necessary tables manually
-- Or use the --create-tables flag with the simulator

-- Create database (run as admin user)
CREATE DATABASE IF NOT EXISTS playtracker;

-- Use the database
-- USE playtracker; -- Uncomment if running interactively

-- Content Items Table
-- Stores movies and TV episodes with metadata
CREATE TABLE IF NOT EXISTS playtracker.content_items (
    content_id String,
    series_id String,
    season_id String,
    episode_number UInt64,
    title String,
    type String,
    content_type String,
    language String,
    age_rating UInt8,
    content_access String,
    publish_date String,
    release_date DateTime,
    duration UInt64,
    metas Array(String),
    genres Array(String),
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (content_id, type)
SETTINGS index_granularity = 8192;

-- Watch History Table  
-- Stores user viewing events and behavior data
CREATE TABLE IF NOT EXISTS playtracker.watch_history (
    uuid String,
    customer_id String,
    profile_id String,
    date_of_birth Nullable(String),
    gender LowCardinality(Nullable(String)),
    subscription_type LowCardinality(Nullable(String)),
    content_id String,
    series_id Nullable(String),
    content_type LowCardinality(String),
    content_duration UInt64,
    genres Array(LowCardinality(String)),
    casts Array(String),
    metas Array(LowCardinality(String)),
    provider_name LowCardinality(Nullable(String)),
    language LowCardinality(Nullable(String)),
    release_year Nullable(UInt16),
    release_date Nullable(Date),
    watch_status LowCardinality(Nullable(String)),
    watch_duration UInt64,
    watched_at DateTime,
    city LowCardinality(Nullable(String)),
    country LowCardinality(Nullable(String)),
    ip_address Nullable(String),
    device_id Nullable(String),
    device_type LowCardinality(Nullable(String)),
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (content_type, content_id, profile_id, watched_at)
PARTITION BY toYYYYMM(watched_at)
SETTINGS index_granularity = 8192;

-- Add bloom filter indexes for better performance (matching production backend)
ALTER TABLE playtracker.watch_history ADD INDEX IF NOT EXISTS idx_country country TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE playtracker.watch_history ADD INDEX IF NOT EXISTS idx_genres genres TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE playtracker.watch_history ADD INDEX IF NOT EXISTS idx_language language TYPE bloom_filter(0.01) GRANULARITY 4;

-- Create indexes for common queries
-- Content lookup by type and genre  
ALTER TABLE playtracker.content_items ADD INDEX IF NOT EXISTS idx_content_type type TYPE minmax GRANULARITY 1;
ALTER TABLE playtracker.content_items ADD INDEX IF NOT EXISTS idx_genres genres TYPE bloom_filter(0.01) GRANULARITY 1;

-- Sample queries for analytics

-- Top content by views
-- SELECT content_id, content_type, count() as views 
-- FROM playtracker.watch_history 
-- GROUP BY content_id, content_type 
-- ORDER BY views DESC 
-- LIMIT 10;

-- User engagement by country
-- SELECT country, 
--        count() as total_views,
--        avg(watch_duration) as avg_watch_time,
--        countIf(watch_status = 'completed') / count() as completion_rate
-- FROM playtracker.watch_history 
-- WHERE country IS NOT NULL
-- GROUP BY country 
-- ORDER BY total_views DESC;

-- Daily viewing patterns
-- SELECT toDate(watched_at) as date,
--        count() as views,
--        uniq(customer_id) as unique_users
-- FROM playtracker.watch_history
-- GROUP BY date
-- ORDER BY date;

-- Genre popularity by demographics
-- SELECT g.genre,
--        country,
--        count() as views
-- FROM playtracker.watch_history
-- ARRAY JOIN genres as g.genre
-- WHERE country IS NOT NULL
-- GROUP BY g.genre, country
-- ORDER BY views DESC;