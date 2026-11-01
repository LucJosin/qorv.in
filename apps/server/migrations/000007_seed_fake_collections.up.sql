-- Temporary development seed: 50 collections per active workspace.
-- Requires an existing active workspace; otherwise inserts no rows.
-- Repeated timestamps exercise ID tie-breaking; deleted rows test exclusion.

INSERT INTO collections (
    workspace_id,
    name,
    description,
    category,
    is_public,
    color,
    icon,
    public_stats,
    pinned,
    deleted_at,
    created_at,
    updated_at
)
WITH
    workspace AS (
        SELECT id
        FROM workspaces
        WHERE deleted_at IS NULL
    ),

    ones AS (
        SELECT 0 AS n
        UNION ALL
        SELECT 1
        UNION ALL
        SELECT 2
        UNION ALL
        SELECT 3
        UNION ALL
        SELECT 4
        UNION ALL
        SELECT 5
        UNION ALL
        SELECT 6
        UNION ALL
        SELECT 7
        UNION ALL
        SELECT 8
        UNION ALL
        SELECT 9
    ),

    tens AS (
        SELECT 0 AS n
        UNION ALL
        SELECT 1
        UNION ALL
        SELECT 2
        UNION ALL
        SELECT 3
        UNION ALL
        SELECT 4
    ),

    seed AS (
        SELECT ones.n + tens.n * 10 + 1 AS n
        FROM ones
            CROSS JOIN tens
    )

SELECT
    workspace.id,
    CONCAT('Demo collection ', LPAD(seed.n, 2, '0')) AS name,
    CONCAT('Fake collection ', seed.n, ' for local pagination testing.')
        AS description,
    CASE MOD(seed.n, 5)
        WHEN 0 THEN 'Development'
        WHEN 1 THEN 'Design'
        WHEN 2 THEN 'Reading'
        WHEN 3 THEN 'Research'
        ELSE 'Tools'
    END AS category,
    MOD(seed.n, 2) = 0 AS is_public,
    CASE MOD(seed.n, 3)
        WHEN 0 THEN '#3b82f6'
        WHEN 1 THEN '#22c55e'
        ELSE '#f97316'
    END AS color,
    CASE MOD(seed.n, 3)
        WHEN 0 THEN 'folder'
        WHEN 1 THEN 'bookmark'
        ELSE 'star'
    END AS icon,
    MOD(seed.n, 4) = 0 AS public_stats,
    MOD(seed.n, 7) = 0 AS pinned,
    CASE
        WHEN MOD(seed.n, 11) = 0 THEN '2026-10-01 13:00:00'
    END AS deleted_at,
    TIMESTAMP('2026-10-01 12:00:00') - INTERVAL FLOOR((seed.n - 1) / 3) MINUTE
        AS created_at,
    TIMESTAMP('2026-10-01 12:00:00') AS updated_at
FROM workspace
    CROSS JOIN seed;
