package world

import "testing"

// TestSessionEndHandlerPlumbing verifica a ligação do gancho de desmonte: removeSession
// o executa enquanto a sessão E sua entidade ainda são endereçáveis (o
// desvínculo de grupo lê Leader/PartyList da entidade), exatamente uma vez por slot, e um
// gancho nil continua um no-op seguro.
func TestSessionEndHandlerPlumbing(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)
	const conn = 3
	s := &Session{Conn: conn, Mode: UserPlay, out: newOutQueue(), closeCh: make(chan struct{})}
	w.sessions[conn] = s
	w.entities[conn] = &Entity{ID: conn, Mode: MobUser, HP: 100, Leader: 7}

	calls, sawLeader := 0, 0
	w.SetSessionEndHandler(func(hw *World, hs *Session) {
		calls++
		if e := hw.Entity(hs.Conn); e != nil {
			sawLeader = e.Leader
		}
	})
	w.removeSession(s)
	if calls != 1 {
		t.Fatalf("onSessionEnd called %d times, want 1", calls)
	}
	if sawLeader != 7 {
		t.Fatalf("hook saw Leader %d, want 7 (the entity must still be live)", sawLeader)
	}
	if w.sessions[conn] != nil || w.entities[conn] != nil {
		t.Fatal("removeSession did not free the slot")
	}
	// Desmontar a mesma sessão (já liberada) de novo não pode reexecutar o gancho.
	w.removeSession(s)
	if calls != 1 {
		t.Fatalf("onSessionEnd ran again for a freed slot (calls = %d)", calls)
	}

	// Um handler nil é no-op seguro (ex.: testes só de transporte).
	w2 := New(Config{GridDim: 16}, slogDiscard(), nil, nil)
	s2 := &Session{Conn: 1, Mode: UserPlay, out: newOutQueue(), closeCh: make(chan struct{})}
	w2.sessions[1] = s2
	w2.entities[1] = &Entity{ID: 1, Mode: MobUser, HP: 100}
	w2.removeSession(s2) // não pode dar panic
}
