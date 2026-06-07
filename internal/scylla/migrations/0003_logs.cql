CREATE TABLE IF NOT EXISTS logs (
    channel_id text,
    timestamp timestamp,
    event_name text,
    event_payload text,
    metadata text,
    PRIMARY KEY ((channel_id), timestamp)
) WITH CLUSTERING ORDER BY (timestamp DESC)
