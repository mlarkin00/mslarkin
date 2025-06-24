WITH instances AS (
	SELECT COALESCE(SUM(instance_delta), 0) instances	
	FROM collector_events),
loaders AS (
    SELECT COALESCE(SUM(concurrency), 0) as clients
	FROM loader_instances
    ),
requests AS (
    SELECT 
        COALESCE(SUM(total_requests), 0) as total_requests, 
        COALESCE(SUM(failed_requests), 0) as failed_requests,
        COALESCE(SUM(rate_per_second), 0) as rate_per_second        
    FROM loader_request_totals
),
duration AS (
    SELECT         
        AVG(duration) as duration
    FROM loader_request_totals WHERE duration > 0
)
SELECT 
	i.instances, l.clients, r.total_requests, r.failed_requests, r.rate_per_second, COALESCE(d.duration,0)
FROM loaders l, requests r, instances i, duration d;