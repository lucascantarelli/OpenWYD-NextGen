package main

import (
	"testing"

	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/content"
)

// TestBuildItems checks that the importer row carries exactly what the runtime
// derives from the CSV line — price, grade, unique, position, requirements and
// the EF_ effects. A drift here means the catalog would disagree with the file
// the tmServer used to read.
func TestBuildItems(t *testing.T) {
	// Columns follow ItemList.csv as the content accessors read it:
	// 0 index, 1 Name, 2 ("837.0", unmapped), 3 ReqLvl.ReqStr.ReqInt.ReqDex.ReqCon,
	// 4 unique, 5 Price, 6 pos, 7 Extra, 8 Grade, then EF_<name>,value pairs.
	row := []string{
		"831", "Garra", "837.0", "4.10.0.0.11", "43", "530", "64", "0", "4",
		"EF_DAMAGE", "40", "EF_RANGE", "2",
	}
	l := content.NewItemList([]content.ItemEntry{{Index: 831, Name: "Garra", Fields: row}})

	items := buildItems(l)
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Index != 831 || it.Name != "Garra" {
		t.Errorf("index/name = %d/%q", it.Index, it.Name)
	}
	if it.Price != 530 {
		t.Errorf("price = %d, want 530", it.Price)
	}
	if it.Pos != 64 {
		t.Errorf("pos = %d, want 64", it.Pos)
	}
	if it.Grade != 4 {
		t.Errorf("grade = %d, want 4", it.Grade)
	}
	if it.UniqueID != 43 {
		t.Errorf("unique = %d, want 43", it.UniqueID)
	}
	if it.AttackRange == nil || *it.AttackRange != 2 {
		t.Errorf("attack_range = %v, want 2", it.AttackRange)
	}
	if it.Volatile != nil {
		t.Errorf("volatile = %v, want nil", *it.Volatile)
	}
	if len(it.Effects) != 1 || it.Effects[0].Eff != 2 || it.Effects[0].Val != 40 {
		t.Errorf("effects = %+v, want EF_DAMAGE 40", it.Effects)
	}
	if it.SHA256 == "" {
		t.Error("sha256 vazio")
	}
	if len(it.Fields) != len(row) {
		t.Errorf("fields = %d colunas, want %d", len(it.Fields), len(row))
	}
}

// TestBuildItemsRequirement checks the dotted requirement column is split into
// the five typed columns.
func TestBuildItemsRequirement(t *testing.T) {
	row := []string{"1", "Espada", "837.0", "40.30.0.10.5", "0", "100", "64", "0", "0"}
	l := content.NewItemList([]content.ItemEntry{{Index: 1, Name: "Espada", Fields: row}})
	it := buildItems(l)[0]
	if it.ReqLevel != 40 || it.ReqStr != 30 || it.ReqInt != 0 || it.ReqDex != 10 || it.ReqCon != 5 {
		t.Errorf("req = %d/%d/%d/%d/%d", it.ReqLevel, it.ReqStr, it.ReqInt, it.ReqDex, it.ReqCon)
	}
}

// TestBuildSkills checks every field survives the conversion, including the
// already-divided AffectTime (the catalog stores the runtime value).
func TestBuildSkills(t *testing.T) {
	sp := content.Spell{
		Index: 3, SkillPoint: 2, TargetType: 1, ManaSpent: 12, Delay: 9, Range: 5,
		InstanceType: 2, InstanceValue: 30, TickType: 20, TickValue: 7,
		AffectType: 17, AffectValue: 4, AffectTime: 8, InstanceAttribute: 1, TickAttribute: 2,
		Aggressive: 1, MaxTarget: 3, BParty: 1, AffectResist: 1, Passive: 0, Name: "Fogo",
	}
	skills := buildSkills(content.NewSkillData([]content.Spell{sp}))
	if len(skills) != 1 {
		t.Fatalf("want 1 skill, got %d", len(skills))
	}
	got := skills[0]
	if got.Index != 3 || got.Name != "Fogo" || got.ManaSpent != 12 || got.AffectTime != 8 {
		t.Errorf("skill = %+v", got)
	}
	if got.MaxTarget != 3 || got.BParty != 1 || got.InstanceValue != 30 {
		t.Errorf("skill campos = %+v", got)
	}
}

// TestBuildWorldItems pins the column order (Index,PosX,PosY,Rotate).
func TestBuildWorldItems(t *testing.T) {
	out := buildWorldItems([]content.InitItem{{Index: 115, PosX: 2100, PosY: 2032, Rotate: 3}})
	if len(out) != 1 {
		t.Fatalf("want 1, got %d", len(out))
	}
	if out[0].Index != 115 || out[0].X != 2100 || out[0].Y != 2032 || out[0].Rotate != 3 {
		t.Errorf("world item = %+v", out[0])
	}
}

// TestParseSets rejects unknown sets instead of silently importing nothing.
func TestParseSets(t *testing.T) {
	if _, err := parseSets("item,skill"); err != nil {
		t.Fatalf("parseSets valid: %v", err)
	}
	if _, err := parseSets("item,nope"); err == nil {
		t.Fatal("parseSets aceitou conjunto desconhecido")
	}
	if _, err := parseSets(" , "); err == nil {
		t.Fatal("parseSets aceitou lista vazia")
	}
}
