package handler

import (
	"openwyd-nextgen/tmserver/internal/combine"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// Códigos de resultado de combine para _MSG_CombineComplete.Parm (game-rules.md §3.1).
const (
	combineInvalid = 0 // a receita não casou (entradas NÃO consumidas)
	combineSuccess = 1
	combineFailed  = 2
)

// Constantes mágicas do combine Anct base (game-rules.md §3.1/§7).
const (
	jewelBase  = 2441 // joia = Item[1].sIndex - 2441 (0..3)
	resultSanc = 7    // BASE_SetItemSanc(result, 7, 0)
)

// CombineFamily parametriza um sistema de combine (Anct/Ehre/Tiny/…). As ~9
// variantes diferem SÓ em Rate (GetMatchCombine<X>) e Apply (o resultado), então
// uma engine cuida de todas (consolidação, guidelines §19.3).
type CombineFamily struct {
	Name  string
	Rate  func(items []world.Item) int        // taxa da receita 0..100; 0 = sem casamento
	Apply func(items []world.Item) world.Item // item de resultado no sucesso
}

// combineItemTypes são as variantes de combine baseadas em Item[] que compartilham a engine.
// MsgCombineItemOdin NÃO está aqui — suas receitas não cabem na forma genérica
// CombineFamily{Rate,Apply}, então ele ganha um handler dedicado
// (combine_odin.go) em vez disso.
var combineItemTypes = []protocol.Type{protocol.MsgCombineItem}

// defaultCombineFamily é o placeholder UNVERIFIED usado até as tabelas de
// receita/taxa (Common/Settings/CompRate.txt) e o ItemList serem carregados (Fase 5): toda
// receita fica "sem casamento" (Rate 0), então os combines reportam inválido em vez de adivinhar.
func defaultCombineFamily(name string) CombineFamily {
	return CombineFamily{Name: name, Rate: func([]world.Item) int { return 0 }, Apply: anctApply}
}

// anctApply é o fallback do resultado Anct base quando nenhum catálogo de conteúdo está montado.
//
// Como combine.AnctResult, o sanc só entra se o item base já carregar um
// par de sanc — BASE_SetItemSanc não aloca nada e devolve FALSE caso contrário
// (Basedef.cpp:2312). Isto resolve a antiga UNVERIFIED "em que slot o sanc
// mora?": é o slot que já contém EF_SANC ou um efeito [116,125],
// varrendo 0→1→2, nunca um Effects[2] fixo.
func anctApply(items []world.Item) world.Item {
	result := items[0]
	if len(items) >= 2 {
		if joia := int(items[1].Index) - jewelBase; joia >= 0 && joia <= 3 {
			result.Index = int16(joia)
		}
	}
	refine.Set(&result, resultSanc, 0)
	return result
}

// resolveComboInputs valida as entradas alegadas por um pacote de combine contra os
// slots de carry acessíveis vivos (limites + sameItem anti-cheat) — o esqueleto que toda
// variante de combine baseada em Item[] compartilha, genérica e Odin iguais. Ele devolve os
// itens e seus slots de carry ainda indexados pela posição ORIGINAL da
// mensagem de combine (Item/slot zero onde o cliente deixou aquela posição vazia), mais
// a lista de posições ativas em ordem crescente. ok é falso quando o
// quem chamou já enviou uma resposta (RemoveTrade ou _NN_Wrong_Combination)
// e precisa retornar na hora sem consumir nada.
func (d *Dispatcher) resolveComboInputs(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgCombineItemBody) (items [protocol.MaxCombine]world.Item, slots [protocol.MaxCombine]int, active []int, ok bool) {
	active = make([]int, 0, protocol.MaxCombine)
	for i := 0; i < protocol.MaxCombine; i++ {
		if body.Item[i].Index == 0 {
			continue
		}
		pos := int(body.InvenPos[i])
		if !carrySlotAccessible(e, pos) {
			d.removeTrade(w, s) // fora da faixa → RemoveTrade (anti-cheat)
			return items, slots, active, false
		}
		if !sameItem(body.Item[i], e.Carry[pos]) {
			sendCombineComplete(w, s, combineInvalid) // mudou/removeu
			return items, slots, active, false
		}
		items[i] = e.Carry[pos]
		slots[i] = pos
		active = append(active, i)
	}
	return items, slots, active, true
}

// combineItem é o handler da engine compartilhado pelas variantes baseadas em Item[]. Ele
// segue exatamente a ORDEM original: valida a receita PRIMEIRO (inválida ⇒ entradas
// mantidas), depois consome as entradas, depois sorteia — então um sorteio que falha ainda consome as
// entradas (o comportamento WYD pretendido, game-rules.md §3.1).
func (d *Dispatcher) combineItem(w *world.World, s *world.Session, h protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	fam, ok := d.combineFamilies[h.Type]
	if !ok {
		return
	}
	var body protocol.MsgCombineItemBody
	if err := body.Decode(payload); err != nil {
		return
	}

	byPos, slotByPos, active, ok := d.resolveComboInputs(w, s, e, body)
	if !ok {
		return
	}
	items := byPos[:]

	rate := 0
	if len(items) > 0 {
		rate = fam.Rate(items)
	}
	if rate == 0 {
		// _NN_Wrong_Combination — as entradas NÃO são consumidas.
		sendCombineComplete(w, s, combineInvalid)
		return
	}

	// Consome as entradas ANTES do sorteio (perdidas na falha, por projeto).
	for _, pos := range active {
		sl := slotByPos[pos]
		e.Carry[sl] = world.Item{}
		sendCarrySlot(w, s, e, sl)
	}

	if _, success := combine.Roll(w.Rand(), rate); !success {
		sendCombineComplete(w, s, combineFailed)
		return
	}

	ipos := slotByPos[active[0]]
	e.Carry[ipos] = fam.Apply(items)
	sendCombineComplete(w, s, combineSuccess)
	sendCarrySlot(w, s, e, ipos)
}

// combineExtracao trata _MSG_CombineItemExtracao (0x02D4): a extração da Huntress
// usa MSG_STANDARDPARM2.Parm2 como slot de carry e consome uma Pedra do Sábio
// como catalisador — uma unidade, não a pilha inteira: a loja do NPC vende em pacotes.
func (d *Dispatcher) combineExtracao(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	_, p2, ok := protocol.StandardParm2(payload)
	if !ok {
		return
	}
	slot := int(p2)
	if !carrySlotAccessible(e, slot) {
		return
	}
	it := e.Carry[slot]
	if it.Empty() || int(it.Index) >= world.MaxItem {
		return
	}
	itemLevel := d.itemAbility(it, efItemLevel)
	if itemLevel >= 5 || itemSanc(it) < 9 || d.itemAbility(it, efMobType) != 0 {
		return
	}
	switch d.itemPos[int(it.Index)] {
	case 2, 4, 8, 16, 32:
	default:
		return
	}
	catalyst := -1
	for i := 0; i < activeCarryLimit(e); i++ {
		if e.Carry[i].Index == itemPedraDoSabio {
			catalyst = i
			break
		}
	}
	if catalyst < 0 {
		return
	}
	consumeOneItem(&e.Carry[catalyst])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, catalyst, itemToSel(e.Carry[catalyst])))

	roll := w.Rand().Intn(115)
	if roll > 100 {
		roll -= 15
	}
	rate := (effectiveSpecial(e, 2) + 1) / 6
	if roll < rate {
		if it.Effects[1].Effect == efDamage {
			it.Effects[1].Value = addEffectByte(it.Effects[1].Value, d.itemBaseDamage(it))
		}
		if it.Effects[2].Effect == efDamage {
			it.Effects[2].Value = addEffectByte(it.Effects[2].Value, d.itemBaseDamage(it))
		}
		it.Effects[0] = world.Effect{Effect: efItemLevel, Value: uint8(itemLevel)}
		it.Index = extractionResultIndex(d.itemPos[int(it.Index)])
		e.Carry[slot] = it
	} else {
		e.Carry[slot] = world.Item{}
	}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, slot, itemToSel(e.Carry[slot])))
}

func addEffectByte(v uint8, add int32) uint8 {
	sum := int32(v) + add
	if sum > 255 {
		return 255
	}
	if sum < 0 {
		return 0
	}
	return uint8(sum)
}

func extractionResultIndex(pos int) int16 {
	switch pos {
	case 4:
		return 3022
	case 8:
		return 3023
	case 16:
		return 3024
	case 32:
		return 3025
	default:
		return 3021
	}
}

// sendCombineComplete responde a um combine com _MSG_CombineComplete (0x03A7).
//
// O corpo é MSG_STANDARDPARM — um int Parm de 4 bytes (Basedef.h:1254-1258), não um
// short: um corpo de 2 bytes deixava o cliente lendo a metade alta de Parm do próximo
// frame. HEADER.ID é ESCENE_FIELD, como SendClientSignalParm define
// (SendFunc.cpp:300-310), não a conn do remetente.
func sendCombineComplete(w *world.World, s *world.Session, parm int32) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgCombineComplete, ID: protocol.IDScene}, protocol.EncodeStandardParm(parm))
}

// sendCarrySlot empurra o conteúdo atual de um slot de carry para o cliente.
//
// MSG_SendItem is {short invType; short Slot; STRUCT_ITEM item} (Basedef.h:2037-2046)
// — um corpo de 12 bytes. Os caminhos de combine enviavam um índice de slot cru, então o
// cliente lê invType do número do slot e o item do que viesse em seguida.
func sendCarrySlot(w *world.World, s *world.Session, e *world.Entity, slot int) {
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, slot, itemToSel(e.Carry[slot])))
}
