// Package npccfg carrega a configuração de NPC editada por moderador para o
// tmServer (npc-editing-plan.md). É um pacote de tipos folha: o dbclient
// implementa Source (buscando por gRPC + fazendo a resolução de templates), e o handler
// aplica um Snapshot dentro do laço de dono único. O mundo nunca escreve isto
// de volta — o fluxo é de mão única (web → Postgres → dbServer → tmServer).
package npccfg

import "context"

// ShopItem é um slot da loja de um mercador (sobrepõe o Carry[] do template). Slot é o
// índice posicional de MSG_ShopList (0..26). Preços NÃO estão aqui — eles são globais
// (Snapshot.PriceOverrides).
type ShopItem struct {
	Slot     int
	Index    uint16
	Quantity uint8
	Eff      [3][2]uint8
}

// Definition é um NPC totalmente resolvido, pronto para materializar: o template
// STRUCT_MOB cru de 816 bytes mais as sobreposições do moderador (visibilidade,
// posição, loja). Template é nil quando o arquivo referenciado não pôde ser
// carregado — tal definição é pulada pelo aplicador.
type Definition struct {
	Slug                                                     string
	Template                                                 []byte
	DisplayName                                              string
	Enabled                                                  bool
	X, Y                                                     int16
	RouteType                                                uint8
	Merchant                                                 uint8
	Origin                                                   string
	GeneratorIndex                                           int
	FollowerTemplate                                         []byte
	MinuteGenerate, MinGroup, MaxGroup, MaxNumMob, Formation int
	SegX, SegY, SegRange, SegWait                            [5]int16
	FightAction, DieAction                                   [4]string
	Shop                                                     []ShopItem // nil = usa o próprio Carry[] do template como loja
}

// Snapshot é a configuração completa numa dada versão: o conjunto de definições
// mais as sobreposições globais de preço por item (índice do item → preço).
type Snapshot struct {
	Version        int64
	Defs           []Definition
	PriceOverrides map[int]int32
}

// Source é o lado de leitura do tmServer para a configuração de NPC. Version é um
// poll barato; Snapshot busca o conjunto completo (chamado só quando a versão mudou).
type Source interface {
	Version(ctx context.Context) (int64, error)
	Snapshot(ctx context.Context) (Snapshot, error)
}
