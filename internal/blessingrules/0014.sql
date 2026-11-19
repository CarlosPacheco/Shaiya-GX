-- Require matching data so consolidating the tables cannot discard edits.
CREATE TEMP TABLE GX_BlessingMergeCheck (
    Matched INTEGER NOT NULL CONSTRAINT BlessingLightAndFuryMustMatch CHECK (Matched = 1)
);
INSERT INTO GX_BlessingMergeCheck
SELECT NOT EXISTS (
    SELECT * FROM PS_GameDefs_BlessingLight
    EXCEPT SELECT * FROM PS_GameDefs_BlessingFury
) AND NOT EXISTS (
    SELECT * FROM PS_GameDefs_BlessingFury
    EXCEPT SELECT * FROM PS_GameDefs_BlessingLight
);
DROP TABLE GX_BlessingMergeCheck;

ALTER TABLE PS_GameDefs_BlessingLight RENAME TO PS_GameDefs_GoddessBlessing;
DROP TABLE PS_GameDefs_BlessingFury;
