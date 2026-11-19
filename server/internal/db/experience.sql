ALTER TABLE PS_GameDefs_ExpDefs RENAME TO GX_LegacyExpDefs;
CREATE TABLE PS_GameDefs_ExpDefs (
    Level INTEGER PRIMARY KEY CHECK (Level BETWEEN 1 AND 65535),
    Exp INTEGER NOT NULL CHECK (Exp BETWEEN 1 AND 4294967295)
) STRICT;

-- Keep the highest cumulative threshold at each level, rounded to about
-- three significant digits without losing the original precision below 100,000.
INSERT INTO PS_GameDefs_ExpDefs(Level, Exp)
WITH caps AS (
    SELECT Level, MAX(Exp) AS Highest FROM GX_LegacyExpDefs GROUP BY Level
), rounded AS (
    SELECT Level, Highest, CASE
        WHEN Highest >= 100000000 THEN 1000000
        WHEN Highest >= 10000000 THEN 100000
        WHEN Highest >= 1000000 THEN 10000
        WHEN Highest >= 100000 THEN 1000
        ELSE 1 END AS Step
    FROM caps
)
SELECT Level, MIN(4294967295, (Highest + Step / 2) / Step * Step) FROM rounded;

-- Transfer a character's fraction of progress through its current level.
-- Leave its level and unspent points intact.
WITH old_curve AS (
    SELECT Level, Grow, MAX(Exp) AS Exp FROM GX_LegacyExpDefs GROUP BY Level, Grow
), bounds AS MATERIALIZED (
    SELECT c.CharID, c.Level, c.Exp,
        COALESCE(old_prev.Exp, 0) AS OldStart, COALESCE(old_next.Exp, 0) AS OldEnd,
        COALESCE(new_prev.Exp, 0) AS NewStart, new_next.Exp AS NewEnd
    FROM PS_GameData_Chars c
    LEFT JOIN old_curve old_prev ON old_prev.Level = c.Level - 1 AND old_prev.Grow = c.Grow + 1
    LEFT JOIN old_curve old_next ON old_next.Level = c.Level AND old_next.Grow = c.Grow + 1
    LEFT JOIN PS_GameDefs_ExpDefs new_prev ON new_prev.Level = c.Level - 1
    LEFT JOIN PS_GameDefs_ExpDefs new_next ON new_next.Level = c.Level
    WHERE c.Level BETWEEN 1 AND (SELECT MAX(Level) FROM PS_GameDefs_ExpDefs)
)
UPDATE PS_GameData_Chars AS c
SET Exp = CASE
    WHEN b.Level = (SELECT MAX(Level) FROM PS_GameDefs_ExpDefs) THEN b.NewStart
    WHEN b.OldEnd <= b.OldStart THEN b.NewStart
    ELSE b.NewStart + MIN(b.NewEnd - b.NewStart - 1, CAST(
        MAX(0, b.Exp - b.OldStart) * 1.0 / (b.OldEnd - b.OldStart)
            * (b.NewEnd - b.NewStart) AS INTEGER))
    END
FROM bounds b WHERE c.CharID = b.CharID;

DROP TABLE GX_LegacyExpDefs;
