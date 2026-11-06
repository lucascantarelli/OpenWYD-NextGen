package handler

import (
	"log/slog"
	"testing"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// TestApplyMobExpUsesExpPanel locks the EXP feedback channel (QA 06/11). The gain
// must ride MSG_Exp_Msg_Panel_ (0x5000, legacy SendMsgExp at MobKilled.cpp:2606,
// TNColor::Default), because MsgMessageChat makes the client render the line as
// the character speaking it.
func TestApplyMobExpUsesExpPanel(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 16}, log, nil, nil)
	s := &world.Session{Conn: 1, Mode: world.UserPlay}
	// Already past every level gate: the gain must not cross a level, so any
	// MsgMessageChat in the session could only be the EXP line itself.
	e := &world.Entity{ID: 1, HP: 100, MaxHP: 100, Level: 300}

	d.applyMobExp(w, s, e, 500)

	if e.Exp != 500 {
		t.Fatalf("exp = %d, want 500", e.Exp)
	}
	if got := w.SentOfType(s, protocol.MsgExpPanel); got != 1 {
		t.Errorf("MSG_Exp_Msg_Panel_ frames = %d, want 1", got)
	}
	if got := w.SentOfType(s, protocol.MsgMessageChat); got != 0 {
		t.Errorf("MsgMessageChat frames = %d, want 0 (the gain must not be spoken)", got)
	}
}
