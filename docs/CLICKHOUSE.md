# ClickHouse Setup Guide

This guide helps you set up ClickHouse for use with the PlayTracker simulator.

## Installation

### Docker (Recommended)
```bash
# Run ClickHouse server
docker run -d \
  --name clickhouse-server \
  -p 8123:8123 \
  -p 9000:9000 \
  --ulimit nofile=262144:262144 \
  clickhouse/clickhouse-server:latest

# Run ClickHouse client
docker exec -it clickhouse-server clickhouse-client
```

### Local Installation
```bash
# Ubuntu/Debian
curl https://packages.clickhouse.com/rpm/lts/repodata/repomd.xml.key | sudo apt-key add -
echo "deb https://packages.clickhouse.com/deb stable main" | sudo tee /etc/apt/sources.list.d/clickhouse.list
sudo apt-get update
sudo apt-get install -y clickhouse-server clickhouse-client
sudo service clickhouse-server start

# macOS
brew install clickhouse
```

## Configuration

### Default Configuration
The simulator works with default ClickHouse settings:
- **Host**: localhost
- **Port**: 9000 (native protocol)
- **Username**: default  
- **Password**: (empty)
- **Database**: playtracker (will be created)

### Custom Configuration
Update `config/config.yaml`:
```yaml
clickhouse:
  host: "your-clickhouse-host"
  port: 9000
  database: "your_database"
  username: "your_username"  
  password: "your_password"
  batch_size: 10000
  max_open_conns: 10
```

## Table Creation

### Automatic (Recommended)
```bash
# Creates tables automatically
./simulator load --content --use-clickhouse --create-tables
```

### Manual
```bash
# Run the SQL script
clickhouse-client --multiquery < sql/clickhouse_schema.sql

# Or execute queries individually
clickhouse-client -q "CREATE DATABASE IF NOT EXISTS playtracker"
```

## Usage Examples

### Load Content
```bash
# Load movies and episodes with table creation
./simulator load --content --use-clickhouse --create-tables

# Load without table creation (tables must exist)
./simulator load --content --use-clickhouse
```

### Load Watch History
```bash
# Load first 10 batches with table creation
./simulator load --watch-history --start-batch=1 --end-batch=10 --use-clickhouse --create-tables

# Load specific batch range
./simulator load --watch-history --start-batch=50 --end-batch=100 --use-clickhouse

# Load using Makefile
make load-watch-history-batch-clickhouse START=1 END=50
```

### Verify Data
```bash
# Check table row counts
clickhouse-client -q "SELECT count() FROM playtracker.content_items"
clickhouse-client -q "SELECT count() FROM playtracker.watch_history"

# Show table sizes
clickhouse-client -q "SELECT table, formatReadableSize(sum(bytes)) as size FROM system.parts WHERE database='playtracker' GROUP BY table"
```

## Performance Tips

### Batch Size Optimization
- **Small datasets** (< 1M records): batch_size: 5000
- **Medium datasets** (1M - 10M): batch_size: 10000 (default)
- **Large datasets** (> 10M): batch_size: 50000

### Connection Pool
- **Light load**: max_open_conns: 5, max_idle_conns: 2
- **Heavy load**: max_open_conns: 20, max_idle_conns: 10

### Memory Settings
Add to ClickHouse `users.xml`:
```xml
<profiles>
    <default>
        <max_memory_usage>20000000000</max_memory_usage>
        <use_uncompressed_cache>1</use_uncompressed_cache>
    </default>
</profiles>
```

## Sample Analytics Queries

### Content Performance
```sql
-- Top 10 most watched content
SELECT 
    content_id,
    content_type,
    count() as total_views,
    avg(watch_duration) as avg_watch_time
FROM playtracker.watch_history 
GROUP BY content_id, content_type 
ORDER BY total_views DESC 
LIMIT 10;
```

### User Engagement
```sql
-- User engagement by country
SELECT 
    country,
    count() as total_views,
    uniq(customer_id) as unique_users,
    avg(watch_duration) as avg_watch_time,
    countIf(watch_status = 'completed') / count() as completion_rate
FROM playtracker.watch_history 
WHERE country IS NOT NULL
GROUP BY country 
ORDER BY total_views DESC;
```

### Time Series Analysis
```sql
-- Daily viewing patterns
SELECT 
    toDate(watched_at) as date,
    count() as views,
    uniq(customer_id) as unique_users,
    avg(watch_duration) as avg_duration
FROM playtracker.watch_history
WHERE watched_at >= '2024-01-01'
GROUP BY date
ORDER BY date;
```

### Genre Analysis
```sql
-- Most popular genres
SELECT 
    arrayJoin(genres) as genre,
    count() as views,
    uniq(customer_id) as unique_viewers
FROM playtracker.watch_history 
GROUP BY genre
ORDER BY views DESC;
```

## Troubleshooting

### Connection Issues
```bash
# Test connection
clickhouse-client --host=localhost --port=9000 --user=default

# Check if service is running
sudo service clickhouse-server status

# View logs
sudo tail -f /var/log/clickhouse-server/clickhouse-server.log
```

### Permission Issues
```bash
# Create database with proper permissions
clickhouse-client -q "CREATE DATABASE IF NOT EXISTS playtracker"
clickhouse-client -q "GRANT ALL ON playtracker.* TO default"
```

### Performance Issues
- Increase `batch_size` in config for faster inserts
- Monitor memory usage with `clickhouse-client -q "SELECT * FROM system.metrics WHERE metric LIKE '%Memory%'"`
- Use `OPTIMIZE TABLE` after large inserts for better compression