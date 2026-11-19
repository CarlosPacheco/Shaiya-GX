// Package npcquest maps the native NPC and quest catalogs onto their SQLite tables.
package npcquest

import (
	"fmt"
	"strings"
)

type ValueType uint8

const (
	U8 ValueType = iota
	U16
	U32
	I16
	I32
	Float
	Text
)

type Field struct {
	Name string
	Type ValueType
}

type Table struct {
	Name   string
	Keys   int
	Fields []Field
}

const (
	NPCs = iota
	Products
	Gates
	NPCLinks
	Quests
	ItemLinks
)

func fields(t ValueType, names string) []Field {
	var out []Field
	for _, name := range strings.Fields(names) {
		out = append(out, Field{name, t})
	}
	return out
}

func join(groups ...[]Field) []Field {
	var out []Field
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

var Tables = [...]Table{
	{"PS_GameDefs_NpcDefinitions", 2, join(fields(U16, "NpcGroup NpcRow"), fields(U8, "NPCType"), fields(U16, "NPCTypeID"),
		fields(I32, "Model MoveDistance MoveSpeed Faction"), fields(Text, "NpcName"), fields(U8, "MerchantType"))},
	{"PS_GameDefs_NpcStoreProducts", 3, join(fields(U16, "NpcGroup NpcRow"), fields(U32, "Slot"), fields(U8, "ItemType ItemTypeID"))},
	{"PS_GameDefs_NpcGateDestinations", 3, join(fields(U16, "NpcGroup NpcRow"), fields(U8, "Slot"), fields(U16, "MapID"),
		fields(Float, "X Y Z"), fields(U32, "Money"))},
	{"PS_GameDefs_NpcQuestLinks", 4, join(fields(U16, "NpcGroup NpcRow"), fields(U8, "Direction"), fields(U32, "Slot"), fields(U16, "QuestRow"))},
	{"PS_GameDefs_QuestDetails", 1, append(fields(U16, "QuestRow"), questFields()...)},
	{"PS_GameDefs_ItemQuestLinks", 4, join(fields(U8, "ItemType ItemTypeID Direction"), fields(U32, "Slot"), fields(U16, "QuestRow"))},
}

func questFields() []Field {
	out := join(fields(U16, "EmbeddedID"), fields(Text, "QuestName"), fields(U16, "MinLevel MaxLevel"),
		fields(U8, "Faction Mode Male Female AttackFighter DefenseFighter PatrolRogue ShooterRogue AttackMage DefenseMage"),
		fields(U16, "HG"), fields(I16, "VG"), fields(U8, "CG OG IG"), fields(U16, "PrevQuest"),
		fields(U8, "RequireParty PartyAttackFighter PartyDefenseFighter PartyPatrolRogue PartyShooterRogue PartyAttackMage PartyDefenseMage"),
		fields(U32, "MinMinutes MaxMinutes TickStart TickKeep TickReceive"), fields(U8, "StartType StartNPCType"),
		fields(U16, "StartNPCID"), fields(U8, "StartItemType StartItemID"))
	for i := 1; i <= 3; i++ {
		out = append(out, fields(U8, fmt.Sprintf("StartItem%dType StartItem%dID StartItem%dCount", i, i, i))...)
	}
	out = append(out, join(fields(U8, "EndType EndNPCType"), fields(U16, "EndNPCID"))...)
	for i := 1; i <= 3; i++ {
		out = append(out, fields(U8, fmt.Sprintf("FarmItem%dType FarmItem%dID FarmItem%dCount", i, i, i))...)
	}
	out = append(out, join(fields(U8, "PVPKills"), fields(U16, "Mob1ID"), fields(U8, "Mob1Count"),
		fields(U16, "Mob2ID"), fields(U8, "Mob2Count ResultType UserSelect"))...)
	for i := 1; i <= 3; i++ {
		prefix := fmt.Sprintf("Reward%d", i)
		for _, f := range join(fields(U16, "NeedMobID"), fields(U8, "NeedMobCount NeedItemID NeedItemCount"),
			fields(U32, "NeedTime"), fields(U16, "NeedHG"), fields(I16, "NeedVG"), fields(U8, "NeedOG"),
			fields(U32, "XP Gold"), fields(U8, "ItemType ItemTypeID"), fields(U16, "NextQuest")) {
			out = append(out, Field{prefix + f.Name, f.Type})
		}
	}
	return out
}

type Row []any
type Catalog [len(Tables)][]Row
