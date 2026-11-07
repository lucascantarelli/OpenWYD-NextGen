package world

// GuildInfo são os metadados mínimos de guilda não modelados na própria Entity: o
// nome e a pontuação de fama, indexados por id de guilda (issue #131). Só em memória — não
// existe ainda fluxo de criação de guilda para persistir (guildRelay, chat.go
// leaveGuild); populado hoje só pelos comandos admin /gm guildname|guildfame,
// espelhando a ferramenta legada só-de-GM "+guildfame set"
// (Source/Comandos GM.txt).
type GuildInfo struct {
	Name string
	Fame int32
}

// GuildInfo devolve o nome/fama registrados para um id de guilda, ou false se
// nada foi registrado para ele ainda. Só no loop.
func (w *World) GuildInfo(id uint16) (GuildInfo, bool) {
	gi, ok := w.guilds[id]
	return gi, ok
}

// SetGuildName define (ou cria) o nome registrado de uma guilda. Só no loop.
func (w *World) SetGuildName(id uint16, name string) {
	gi := w.guilds[id]
	gi.Name = name
	w.guilds[id] = gi
}

// SetGuildFame define (ou cria) a fama registrada de uma guilda. Só no loop.
func (w *World) SetGuildFame(id uint16, fame int32) {
	gi := w.guilds[id]
	gi.Fame = fame
	w.guilds[id] = gi
}
