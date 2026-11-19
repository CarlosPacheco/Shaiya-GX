CREATE TABLE PS_GameDefs_WorldDrops (
    "Index" INTEGER PRIMARY KEY CHECK ("Index" > 0),
    Description TEXT NOT NULL,
    Grade INTEGER NOT NULL CHECK (Grade BETWEEN 1 AND 1024),
    MinLevel INTEGER NOT NULL CHECK (MinLevel BETWEEN 1 AND 65535),
    MaxLevel INTEGER NOT NULL CHECK (MaxLevel BETWEEN MinLevel AND 65535),
    DropChance INTEGER NOT NULL CHECK (DropChance BETWEEN 0 AND 10000000000)
) STRICT;

-- DropChance stores units of 0.00000001 percent. 10000 units = 0.0001%.
INSERT INTO PS_GameDefs_WorldDrops VALUES
    (1, 'Elemental Lapis Lv1', 34, 20, 69, 10000),
    (2, 'Lapis Lv5', 36, 1, 19, 10000),
    (3, 'Heroic and elemental weapons, shields and special lapis', 39, 20, 69, 10000),
    (4, 'Status Lapis Lv6', 41, 40, 49, 10000),
    (5, 'Worship weapons and shields', 125, 1, 19, 10000),
    (6, 'Legendary weapons and shields (grade 156)', 156, 20, 29, 10000),
    (7, 'Legendary weapons and shields (grade 186)', 186, 30, 39, 10000),
    (8, 'Armor and protectors', 305, 50, 69, 10000),
    (9, 'Mount summoning stones (grade 388)', 388, 20, 69, 10000),
    (10, 'Mount summoning stones (grade 389)', 389, 20, 69, 10000),
    (11, '[1000]Etin', 390, 1, 69, 10000),
    (12, 'Weapon and Armor Lapisia', 394, 1, 69, 10000);
