-- Preserve the existing EXP gain of each consumable when used on its own.
UPDATE PS_GameDefs_Skills SET AbilityValue1=50
WHERE SkillID IN (222,223,250,251,272) AND SkillLevel=1 AND AbilityType1=35 AND AbilityValue1=1;
UPDATE PS_GameDefs_Skills SET AbilityType1=35, AbilityValue1=20
WHERE SkillID IN (231,232,233) AND SkillLevel=1 AND AbilityType1=SkillID-191 AND AbilityValue1=1;

-- Sun EXP Stone already advertises the 24-hour EXP spell.
UPDATE PS_GameDefs_Items SET Range=272
WHERE ItemID=100135 AND Range=16 AND AttackTime=1;

-- Store the effective percentage points directly for legacy oversized modifiers.
UPDATE PS_GameDefs_Skills SET AbilityValue1=100, AbilityValue2=100
WHERE SkillID=136 AND AbilityType1=20 AND AbilityType2=21 AND AbilityValue1=500 AND AbilityValue2=500;
UPDATE PS_GameDefs_Skills SET AbilityValue1=100, AbilityValue2=100
WHERE SkillID=189 AND AbilityType1=29 AND AbilityType2=30
  AND ((SkillLevel=2 AND AbilityValue1=180 AND AbilityValue2=180)
    OR (SkillLevel=3 AND AbilityValue1=240 AND AbilityValue2=240));
UPDATE PS_GameDefs_Skills SET AbilityValue1=100
WHERE SkillID IN (212,213,214) AND SkillLevel=1 AND AbilityType1=SkillID-192 AND AbilityValue1=300;

-- Ill Fate's three reductions are all 20 points in its description.
UPDATE PS_GameDefs_NpcSkills SET AbilityValue1=20, AbilityValue2=20, AbilityValue3=20
WHERE SkillID=190 AND Rank=1 AND SkillName='Ill Fate'
  AND AbilityType1=20 AND AbilityType2=21 AND AbilityType3=22
  AND AbilityValue1=25 AND AbilityValue2=25 AND AbilityValue3=25;
