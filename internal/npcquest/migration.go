package npcquest

import _ "embed"

//go:embed schema.sql
var MigrationSQL string

// The distributable catalog comes from the server's shipped NpcQuest.
//
//go:embed seed.sql
var SeedSQL string
