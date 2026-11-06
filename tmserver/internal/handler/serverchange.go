package handler

import (
	"strconv"
	"strings"

	"openwyd-nextgen/internal/crossover"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// maxServerNumber is how many channels the client's server table holds
// (10 grupos × 11 células, canais.md §8.1.1).
const maxServerNumber = 10

// changeServer handles "/srv <n>": the in-game channel switch, the counterpart of
// the legacy command (_MSG_MessageWhisper.cpp:75-95). The legacy asks the dbServer
// for the crossover token (MSG_DBServerChange → MSG_DBCNFServerChange); here the
// token is signed locally, so the same frame reaches the client without a new RPC
// (canais.md §8.1.2).
//
// The client is the one that reconnects: the frame carries the account name and
// the token, and the next MSG_AccountLogin brings it back in Zero[52] — which is
// exactly the field the legacy compares against TempKey (CFileDB.cpp:662-812).
func (d *Dispatcher) changeServer(w *world.World, s *world.Session, args []byte) {
	if w.Entity(s.Conn) == nil {
		return
	}
	raw := strings.TrimSpace(cstr(args))
	if raw == "" {
		d.sendClientMessage(w, s, "troca de canal: use /srv <numero do canal>")
		return
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > maxServerNumber {
		d.sendClientMessage(w, s, "troca de canal: numero invalido")
		return
	}
	if n == d.serverIndex+1 {
		d.sendClientMessage(w, s, "troca de canal: voce ja esta neste canal")
		return
	}
	if d.crossoverKey == "" {
		d.log.Warn("server change: crossover key not configured", "conn", s.Conn)
		d.sendClientMessage(w, s, "troca de canal desativada no servidor")
		return
	}
	token := crossover.Build(d.crossoverKey, s.AccountName, s.Slot, n, d.now())
	body := protocol.EncodeCNFServerChangeBody(s.AccountName, token)
	// HEADER.ID = own conn, like the legacy forwarding (ProcessDBMessage.cpp:1474).
	w.SendTo(s, protocol.Header{Type: protocol.MsgCNFServerChange, ID: uint16(s.Conn)}, body)
	d.log.Info("server change: token issued", "conn", s.Conn, "account", s.AccountName,
		"slot", s.Slot, "from", d.serverIndex+1, "to", n)
}
