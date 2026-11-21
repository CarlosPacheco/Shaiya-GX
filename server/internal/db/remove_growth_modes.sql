-- Credit the difference in level awards once, preserving spent and unspent points.
UPDATE PS_GameData_Chars
SET StatPoint = MIN(65535, StatPoint + MAX(0, Level - 1) * CASE Grow WHEN 0 THEN 4 WHEN 1 THEN 4 WHEN 2 THEN 2 ELSE 0 END),
    SkillPoint = MIN(65535, SkillPoint + MAX(0, Level - 1) * CASE Grow WHEN 0 THEN 2 WHEN 1 THEN 2 WHEN 2 THEN 1 ELSE 0 END),
    Grow = 0;
UPDATE PS_GameData_UserMaxGrow SET MaxGrow = 0;
UPDATE PS_GameDefs_Skills SET Grow = 0 WHERE Grow <> 0;
