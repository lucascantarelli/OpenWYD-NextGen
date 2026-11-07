package handler

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// As poeiras de refino, classificadas (como toda ação de _MSG_UseItem) pelo
// EF_VOLATILE do item de origem e não pelo índice. OriLacto = Vol-4
// (_MSG_UseItem.cpp:849).
const (
	volDustOri = 4 // Poeira de Oriharucon (412)
	volDustLac = 5 // Poeira de Lactolerium (413)
)

const (
	// itemLacto100 (Lactolerium_100) carrega EF_VOLATILE 5 como uma poeira Lac
	// comum mas força a linha do Âmago, um 100% fixo (_MSG_UseItem.cpp:850).
	itemLacto100 = 4141
	// item769 tem teto +9 por nome (_MSG_UseItem.cpp:802).
	item769 = 769

	// Os tetos (Ori +6, Lac +9, o passo da gema +10 e a muralha +11) costumavam ser
	// constantes aqui. Agora vivem em Common/Settings/RefineRate.txt e no
	// overlay do moderador como content.RefinePolicy (QA 05/11) para um moderador poder definir
	// até onde cada poeira refina sem rebuild. Só a regra de piedade continua em
	// código: é uma peculiaridade do sorteio, não um ajustável.
	pityOriMaxLevel = 5 // Ori acumula piedade só até +5 (_MSG_UseItem.cpp:943)

	// refineRollModulo é o sorteio da poeira: um simples rand()%100
	// (_MSG_UseItem.cpp:851). NÃO o rand()%115 com o desconto de -15 do combine.Roll —
	// os dois caminhos usam distribuições diferentes.
	refineRollModulo = 100
	// pityRollModulo: rand()%4 <= 2, chance de 3 em 4 de acumular piedade numa falha
	// (_MSG_UseItem.cpp:943).
	pityRollModulo = 4
	pityRollMax    = 2
)

// Ovos de montaria (2300..2329) chocam na montaria em sIndex+30 quando refinados.
const (
	eggLo       = 2300
	eggHi       = 2330
	eggHatchAdd = 30
	// eggIncuDelayBase é o cooldown marcado num refino de ovo bem-sucedido:
	// rand()%4 + 6 (_MSG_UseItem.cpp:899).
	eggIncuDelayBase = 6
	// incuLevelExempt: o gate de incubação só se aplica abaixo do nível 999
	// (_MSG_UseItem.cpp:842).
	incuLevelExempt = 999

	// Uma montaria recém-chocada reinterpreta seus slots de efeito como dados de montaria
	// (BASE_GetItemAbility, Basedef.cpp:1758-1770): stEffect[0].sValue = MOUNTHP,
	// [1].cEffect = MOUNTSANC, [1].cValue = MOUNTLIFE, [2].cEffect = MOUNTFEED,
	// [2].cValue = MOUNTKILL.
	hatchMountHP   = 20000
	hatchMountSanc = 1
	hatchMountFeed = 30
	hatchMountKill = 1
	hatchLifeBase  = 10 // MOUNTLIFE = rand()%20 + 10
	hatchLifeSpan  = 20
)

// Ids de efeito que o caminho de refino condiciona (ItemEffect.h).
const (
	efNoSanc    = 126 // EF_NOSANC: este item nunca pode ser refinado
	efIncubate  = 78  // EF_INCUBATE: o nível de refino em que um ovo choca
	efIncuDelay = 84  // EF_INCUDELAY: o cooldown de um ovo entre tentativas de refino
)

func isEgg(it world.Item) bool { return it.Index >= eggLo && it.Index < eggHi }

// Itens Tintura (3397-3406) não refinam — uma poeira os converte direto no
// Feijão Mágico correspondente (3407-3416 = tintura + 10, espelhando a ordem
// de cores que a matemática de cor do useMagicBean já depende, item.go:1072). Isto precisa
// rodar antes de refine.Bootstrap plantar um par EF_SANC falso nos slots de efeito
// zerados de uma tintura nova (issue #130).
const (
	tinturaLo = 3397 // Tintura_Azul
	tinturaHi = 3406 // Tintura_Azul_Claro
)

func isTintura(it world.Item) bool { return it.Index >= tinturaLo && it.Index <= tinturaHi }

// itemInstanceAbility soma SOMENTE os efeitos de instância de um item, que é o que
// BASE_GetBonusItemAbility faz (Basedef.cpp:2048) — diferentemente de itemAbility (=
// BASE_GetItemAbility) ele nunca lê o catálogo.
//
// Só é válido para os efeitos que BASE_GetBonusItemAbility exclui da sua
// escala de refino `value *= sanc+10` (Basedef.cpp:2084). EF_INCUBATE e
// EF_INCUDELAY, os dois que este caminho precisa, estão ambos nessa lista de exclusão.
//
// A cegueira ao catálogo é estrutural, não um descuido: o EF_INCUBATE de um ovo
// vive no ItemList.csv, então isto devolve 0 para um ovo novo e o gate de choco
// `level >= incubate` passa no primeiro refino bem-sucedido.
func itemInstanceAbility(it world.Item, eff uint8) int {
	var total int
	for _, ef := range it.Effects {
		if ef.Effect == eff {
			total += int(ef.Value)
		}
	}
	return total
}

// putShort escreve um valor cru de 16 bits nos dois bytes de um par de efeito, do modo
// como o legado atribui o membro de união `stEffect[i].sValue` de STRUCT_ITEM em
// x86 little-endian.
func putShort(ef *world.Effect, v uint16) {
	ef.Effect = uint8(v)
	ef.Value = uint8(v >> 8)
}

// refineItem é o caminho de refino com poeira de _MSG_UseItem (_MSG_UseItem.cpp:153-208
// e :802-976): arrastar uma Poeira de Ori (Vol 4) ou de Lac (Vol 5) sobre um item
// aumenta seu nível de refino. A poeira é consumida de qualquer modo; na falha o item
// em si nunca é destruído nem rebaixado, ele só acumula "piedade" que eleva
// as chances da próxima tentativa.
//
// Só o caminho de equipamento padrão está portado (issue #103). Os quatro
// outros ramos de poeira do legado — itens selados (:224), celestial/HC (:385), pedras arch
// (:514) e brincos +10..+14 (:716) — todos sorteiam rand()%115 com o desconto de -15 e
// destroem ou reiniciam o item na falha; estão adiados para as próprias issues.
func (d *Dispatcher) refineItem(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgUseItemBody, src, vol int) {
	dst := d.itemSlot(w, s, e, int(body.DestType), int(body.DestPos))
	if dst == nil || dst.Empty() {
		return // não há slot alvo assim — o legado só loga e descarta a mensagem
	}

	if isTintura(*dst) {
		d.refineTintura(w, s, e, dst, body, src)
		return
	}

	// Uma poeira só refina equipamento: um alvo que é ele mesmo consumível (o que
	// inclui arrastar uma poeira sobre outra poeira) é recusado (:178).
	//
	// O legado lê EF_VOLATILE com BASE_GetItemAbility (catálogo + instância);
	// aqui se lê o mapa de catálogo com que o useItem já classifica a poeira de origem.
	// EF_VOLATILE é propriedade de catálogo — nenhum item cunhado a carrega como efeito de
	// instância — então os dois concordam na prática.
	if d.itemVolatiles[int(dst.Index)] != 0 {
		d.refineDustReject(w, s, e, src, refineMessageOnlyEquipment)
		return
	}
	if d.itemAbility(*dst, efNoSanc) != 0 {
		d.refineDustReject(w, s, e, src, refineMessageCantMore)
		return
	}

	limits := d.refineLimits()
	level := refine.Level(*dst)
	// Ori não toca um item já no seu teto (-6 por padrão, :203).
	if vol == volDustOri && level >= limits.OriMax {
		d.refineDustReject(w, s, e, src, refineMessageCantMore)
		return
	}
	// Lac chega ao próprio teto (9) mas o passo +10 é permitido como o
	// único caso especial (vira +11); em HardCap e acima o caminho da poeira
	// recusa de imediato (:802, onde o legado compara com REF_11).
	if level == limits.PLMax || level >= limits.HardCap || (level >= limits.PLMax && dst.Index == item769) {
		d.refineDustReject(w, s, e, src, refineMessageCantMore)
		return
	}

	// Ler a piedade ANTES do bootstrap: plantar o par zera o slot, e
	// o legado também a lê primeiro (:809 antes de :814), então um refino falho continua
	// acumulando sobre a contagem anterior.
	pity := refine.Pity(*dst)

	// Um item nunca refinado não tem onde guardar um nível ainda. Quando os três slots
	// de efeito têm efeitos reais o legado recusa refinar o item de todo (:814).
	if level == 0 && !refine.Bootstrap(dst) {
		d.refineDustReject(w, s, e, src, refineMessageCantMore)
		return
	}

	// Um ovo que falhou um refino anterior precisa cumprir seu timer de incubação,
	// que o RegenMob decrementa enquanto o ovo está equipado (:842, regenPlayers).
	//
	// UNVERIFIED: o legado lê BaseScore.Level aqui; a Entity só carrega
	// CurrentScore.Level. Eles diferem só via equipamento EF_LEVEL, nunca perto de 999.
	if isEgg(*dst) && e.Level < incuLevelExempt && itemInstanceAbility(*dst, efIncuDelay) > 0 {
		d.refineDustReject(w, s, e, src, refineMessageIncubationWait)
		return
	}

	anvil := vol - volDustOri // 0 = Ori, 1 = Lac — também indexa g_pSancGrade
	// A bigorna de taxa pode diferir da bigorna de grade: Lactolerium_100 força a
	// linha do Âmago para um 100% garantido, mas ainda sobe o nível como um Lac (:850).
	rateAnvil := anvil
	if e.Carry[src].Index == itemLacto100 {
		rateAnvil = refine.Amago
	}

	// A ordem é crítica para paridade: a taxa é calculada ANTES do sorteio, e a
	// grade lida depois (:850-853). Um rand() no caminho quente.
	rate := refine.SuccessRate(d.sancRate, *dst, rateAnvil)
	roll := w.Rand().Intn(refineRollModulo)
	grade := d.itemAbility(*dst, efItemLevel)

	t := refineTarget{item: dst, place: int(body.DestType), slot: int(body.DestPos)}
	// `<=` é inclusivo, e uma taxa 0 sempre falha apesar do sorteio 0 passar (:855).
	if roll <= rate && rate != 0 {
		d.refineSucceed(w, s, e, t, src, vol, anvil, level, grade)
		return
	}
	d.refineFail(w, s, e, t, src, anvil, level, pity)
}

// refineTintura converte uma tintura no Feijão Mágico correspondente. Isto é uma
// conversão determinística, não um sorteio de taxa de sucesso — tinturas não carregam
// EF_SANC para sortear, e não existe fonte legada para este ramo (veja
// game-rules.md §3.6.1); é uma decisão de desenho da migração, não um port.
func (d *Dispatcher) refineTintura(w *world.World, s *world.Session, e *world.Entity, dst *world.Item, body protocol.MsgUseItemBody, src int) {
	dst.Index = int16(magicBeanBase + (int(dst.Index) - tinturaLo))
	d.sendRefineMessage(w, s, refineMessageSuccess)
	d.sendSlot(w, s, int(body.DestType), int(body.DestPos), *dst)
	d.sendRefineMotion(w, s, e.ID, refineMotionSuccess, refineMotionSuccessParm)
	consumeOneItem(&e.Carry[src])
}

// refineTarget é o item sendo refinado, junto com onde ele está para o
// resultado poder ser empurrado de volta ao cliente.
type refineTarget struct {
	item        *world.Item
	place, slot int
}

// refineSucceed aplica um sorteio ganho (_MSG_UseItem.cpp:856-927).
func (d *Dispatcher) refineSucceed(w *world.World, s *world.Session, e *world.Entity, t refineTarget, src, vol, anvil, level, grade int) {
	dst := t.item

	limits := d.refineLimits()
	if level == limits.GemLevel {
		// +10 → +11, carregando o índice da gema junto (:857).
		refine.Set(dst, limits.GemLevel+1, refine.Gem(*dst))
	} else {
		if grade >= 1 && grade <= 5 {
			level += refine.Step(anvil, grade)
			if level >= limits.OriMax && vol == volDustOri {
				level = limits.OriMax
			} else if level > limits.PLMax {
				level = limits.PLMax
			}
		} else {
			// Nenhuma grade utilizável → um +1 simples. Note que o legado pula OS DOIS tetos neste
			// ramo (:874), então é o único jeito de passar deles; reproduzido.
			level++
		}
		refine.Set(dst, level, 0) // um sucesso limpa a piedade
	}

	d.refreshScore(e)
	d.sendScore(w, s, e)
	d.sendRefineMessage(w, s, refineMessageSuccess)

	if isEgg(*dst) {
		d.hatchEgg(w, s, t, level)
	}

	d.sendSlot(w, s, t.place, t.slot, *dst)
	d.sendRefineMotion(w, s, e.ID, refineMotionSuccess, refineMotionSuccessParm)
	consumeOneItem(&e.Carry[src])
	// O slot da poeira deliberadamente NÃO é reenviado: o cliente já removeu o
	// item que arrastou, e o legado só ecoa a origem nos caminhos
	// de recusa, para desfazer aquela remoção.
}

// refineFail aplica um sorteio perdido (_MSG_UseItem.cpp:929-974). O item sobrevive
// intocado — só a poeira é gasta e o contador de piedade cresce.
func (d *Dispatcher) refineFail(w *world.World, s *world.Session, e *world.Entity, t refineTarget, src, anvil, level, pity int) {
	d.sendRefineMessage(w, s, refineMessageFailure)
	consumeOneItem(&e.Carry[src])

	if w.Rand().Intn(pityRollModulo) <= pityRollMax {
		// Lac sempre acumula piedade; Ori só até +5 (:943-949).
		if anvil == refine.Lac || (level <= pityOriMaxLevel && anvil == refine.Ori) {
			pity++
		}
	}

	dst := t.item
	if isEgg(*dst) {
		// Um refino de ovo falho inicia o cooldown de incubação (:951).
		dst.Effects[2] = world.Effect{Effect: efIncuDelay, Value: uint8(w.Rand().Intn(pityRollModulo))}
	}
	if limits := d.refineLimits(); level != limits.GemLevel {
		refine.Set(dst, level, pity)
	}
	d.sendSlot(w, s, t.place, t.slot, *dst)
	d.sendRefineFailureMotion(w, s, e)
}

// hatchEgg é o ramo do ovo de montaria embutido no sucesso do refino
// (_MSG_UseItem.cpp:890-917). Ele marca um cooldown de incubação novo e, quando o
// nível do ovo alcança seu limiar EF_INCUBATE, o transforma na montaria em
// sIndex+30.
func (d *Dispatcher) hatchEgg(w *world.World, s *world.Session, t refineTarget, level int) {
	dst := t.item
	dst.Effects[2] = world.Effect{
		Effect: efIncuDelay,
		Value:  uint8(w.Rand().Intn(pityRollModulo) + eggIncuDelayBase),
	}

	// Lê o EF_INCUBATE de INSTÂNCIA, que um ovo novo não tem (seu
	// limiar é efeito de catálogo) — então isto é 0 e o ovo choca no
	// primeiro refino bem-sucedido. Esse é o comportamento do legado, não um atalho:
	// veja itemInstanceAbility.
	if level < itemInstanceAbility(*dst, efIncubate) {
		return
	}

	dst.Index += eggHatchAdd
	putShort(&dst.Effects[0], hatchMountHP)
	dst.Effects[1] = world.Effect{
		Effect: hatchMountSanc,
		Value:  uint8(w.Rand().Intn(hatchLifeSpan) + hatchLifeBase),
	}
	dst.Effects[2] = world.Effect{Effect: hatchMountFeed, Value: hatchMountKill}
	d.sendRefineMessage(w, s, refineMessageIncubated)
	// MountProcess(conn, 0) é no-op: com uma montaria NULL o IsEqual dele fica 1 e
	// ele retorna de imediato (Server.cpp:4639-4643). Nada a portar.
	//
	// O legado envia o slot aqui E de novo depois do bloco do ovo (:916, :919);
	// a duplicata é inofensiva e o refineSucceed faz o segundo envio.
	d.sendSlot(w, s, t.place, t.slot, *dst)
}

// refineDustReject recusa um refino com poeira e reenvia o slot da POEIRA, para o cliente pôr
// de volta o item que arrastou onde ele estava (_MSG_UseItem.cpp:805).
func (d *Dispatcher) refineDustReject(w *world.World, s *world.Session, e *world.Entity, src int, message string) {
	d.sendRefineMessage(w, s, message)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
}

// refineReject é compartilhado por consumíveis parecidos com refino que ainda usam o
// sistema de Notice provisório e estão fora da issue #317.
func (d *Dispatcher) refineReject(w *world.World, s *world.Session, e *world.Entity, src int, n Notice) {
	d.notify(w, s, n)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
}

// sendSlot empurra um slot de item ao cliente (SendItem).
func (d *Dispatcher) sendSlot(w *world.World, s *world.Session, place, slot int, it world.Item) {
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(place, slot, itemToSel(it)))
}
