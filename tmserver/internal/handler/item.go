package handler

import (
	"context"
	"encoding/binary"
	"time"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// maxItemList delimita um índice de item válido (1 <= sIndex < MAX_ITEMLIST).
//
// UNVERIFIED: MAX_ITEMLIST (tamanho de g_pItemList) não está documentado; provisório.
const maxItemList = 30000

// dropBlacklist contém os sIndex que não podem ser largados (itens de quest/vinculados
// a itens), exatamente como em _MSG_DropItem.cpp:110-111 (handlers/_MSG_DropItem.md).
var dropBlacklist = func() map[int16]bool {
	m := map[int16]bool{508: true, 509: true, 522: true, 446: true, 747: true, 3993: true, 3994: true}
	for i := int16(526); i <= 537; i++ {
		m[i] = true
	}
	return m
}()

// dropItem trata _MSG_DropItem (0x0272), handlers/_MSG_DropItem.md: mover um
// item do inventário para o chão. Criar no chão e depois limpar a origem é atômico
// (goroutine única do loop) — sem duplicação.
func (d *Dispatcher) dropItem(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 1, 14)
		return
	}
	if s.Trade.Active {
		d.removeTrade(w, s) // largar no meio de uma troca a cancela (anti-dup)
		return
	}
	if s.TradeMode != 0 {
		d.notify(w, s, NoticeCantAutoTrade)
		return
	}

	var body protocol.MsgDropItemBody
	if err := body.Decode(payload); err != nil {
		return
	}
	if int(body.GridX) >= w.GridDim() || int(body.GridY) >= w.GridDim() {
		d.notify(w, s, NoticeCantDropHere)
		return
	}
	if int(body.SourType) == world.ItemPlaceEquip || int(body.SourType) != world.ItemPlaceCarry {
		return // não dá para largar equipado direto; só CARRY neste lote
	}
	slot := int(body.SourPos)
	if !carrySlotAccessible(e, slot) {
		return
	}
	item := e.Carry[slot]
	if item.Empty() || item.Index < 1 || int(item.Index) >= maxItemList {
		return
	}
	if dropBlacklist[item.Index] {
		return // não largável
	}

	id := w.CreateGroundItem(item, int16(body.GridX), int16(body.GridY))
	if id < 0 {
		return // chão cheio
	}
	e.Carry[slot] = world.Item{} // limpar a origem
	w.Send(s, protocol.MsgCNFDropItem, slotPayload(slot))
	// UNVERIFIED: broadcast de _MSG_CreateItem (spawn no chão em visão) — adiado.
}

// getItem trata _MSG_GetItem (0x0270), handlers/_MSG_GetItem.md: pegar um
// item do chão para o inventário. O id de chão é ItemID-10000 no wire.
func (d *Dispatcher) getItem(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 1, 13)
		return
	}
	if s.Trade.Active {
		d.removeTrade(w, s) // pegar no meio de uma troca a cancela (anti-dup)
		return
	}
	if s.TradeMode != 0 {
		d.notify(w, s, NoticeCantAutoTrade)
		return
	}

	var body protocol.MsgGetItemBody
	if err := body.Decode(payload); err != nil {
		return
	}
	if int(body.DestType) != world.ItemPlaceCarry {
		return
	}
	id := int(body.ItemID) - world.GroundItemIDOffset
	if id <= 0 || id >= world.MaxItem {
		return
	}

	gi := w.GroundItem(id)
	if gi == nil || gi.Mode == 0 || gi.Static {
		w.Send(s, protocol.MsgDecayItem, uint32Payload(uint32(body.ItemID)))
		return
	}
	if abs(int(e.X)-int(gi.X)) > 3 || abs(int(e.Y)-int(gi.Y)) > 3 {
		return // longe demais (anti-teleporte-de-coleta)
	}
	if id == 1727 && e.Level < 1000 {
		return // restrição especial
	}
	if gi.Item.Index == itemPilulaOrc {
		d.pickUpPilulaOrc(w, s, e, gi, body.ItemID)
		return
	}

	slot := int(body.DestPos)
	if !carrySlotAccessible(e, slot) || !e.Carry[slot].Empty() {
		return // inventário cheio/travado/ocupado → deixar no chão
	}
	e.Carry[slot] = gi.Item
	w.RemoveGroundItem(id) // ponto de reivindicação atômica
	w.Send(s, protocol.MsgCNFGetItem, slotPayload(slot))
}

// deleteItem trata _MSG_DeleteItem (0x02E4, Basedef.h:2371): destruir um item do
// carry. Como no _MSG_DropItem, mover um item no meio de uma troca cancela a troca em vez de
// apagar (o caminho anti-dup). O legado limpa o slot incondicionalmente depois
// das guardas de slot/modo e não envia confirmação S→C; nós ainda empurramos um
// MsgSendItem para cliente e servidor nunca divergirem no slot. A remoção
// persiste no próximo save periódico/logout (paridade: sem escrita imediata). A loja de NPC
// é requisição/resposta sem estado, então não há estado de "loja aberta" a proteger.
//
// Diferente do legado, o slot só é limpo quando ainda contém o SIndex que o
// cliente nomeou. O cliente esvazia a própria célula antes de pedir (grade de lixo,
// caixa de mensagem TMFieldScene 740), então um cliente cuja grade divergiu do servidor
// destruiria um item que o jogador nunca viu. Numa divergência o slot
// real é reenviado, o que ressincroniza aquela célula.
func (d *Dispatcher) deleteItem(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 1, 15)
		return
	}
	if s.Trade.Active {
		d.removeTrade(w, s) // apagar no meio de uma troca a cancela (anti-dup)
		return
	}
	var body protocol.MsgDeleteItemBody
	if err := body.Decode(payload); err != nil {
		return
	}
	slot := int(body.Slot)
	if !carrySlotAccessible(e, slot) {
		return
	}
	if int32(e.Carry[slot].Index) != body.SIndex {
		if body.SIndex != 0 {
			d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])
		}
		return
	}
	if e.Carry[slot].Empty() {
		return
	}
	idx := e.Carry[slot].Index
	e.Carry[slot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])
	d.log.Info("item deleted", "conn", s.Conn, "slot", slot, "index", idx)
}

// isSplittable informa se um item pode ser dividido por _MSG_SplitItem
// (_MSG_SplitItem.cpp:45-52): um conjunto fixo de especiais de moeda/empilháveis mais a
// faixa 2390..2419 (poeiras Vol 4/5 — o caso quente, alimenta o refino, issue #103).
//
// itemPedraDoSabio é uma divergência deliberada da lista legada: a loja de NPC
// a vende em pacotes (npcconfig.go shopEffects grava EF_AMOUNT), então Shift+clique
// precisa poder destacar unidades da pilha (issue #268).
func isSplittable(index int16) bool {
	switch index {
	case 412, 413, 414, 416, 419, 420, itemPedraDoSabio:
		return true
	}
	return index >= 2390 && index <= 2419 || index >= itemQuestRewardBase && index <= itemQuestRewardLast
}

// itemPedraDoSabio é a Pedra do Sábio (ItemList.csv:2903), o catalisador de
// extração da Caçadora — veja combineExtracao.
const itemPedraDoSabio = 1774

const maxStackAmount = 120

// setItemAmount grava n no slot de efeito EF_AMOUNT do item (espelha
// BASE_SetItemAmount): reusa um efeito EF_AMOUNT existente, senão pega o primeiro
// slot de efeito vazio. É o inverso de itemAmount/consumeOneItem.
func setItemAmount(it *world.Item, n int) {
	for i := range it.Effects {
		if it.Effects[i].Effect == efAmount {
			it.Effects[i].Value = uint8(n)
			return
		}
	}
	for i := range it.Effects {
		if it.Effects[i].Effect == 0 {
			it.Effects[i].Effect = efAmount
			it.Effects[i].Value = uint8(n)
			return
		}
	}
}

func tryMergeItemStacks(src, dst *world.Item) bool {
	if !sameStackClass(*src, *dst) {
		return false
	}
	srcAmount := itemAmount(*src)
	dstAmount := itemAmount(*dst)
	if srcAmount <= 0 || dstAmount <= 0 {
		return true
	}
	if dstAmount >= maxStackAmount {
		return true
	}
	if !canWriteItemAmount(*dst) {
		return true
	}
	move := srcAmount
	if space := maxStackAmount - dstAmount; move > space {
		move = space
	}
	if move < srcAmount && !canWriteItemAmount(*src) {
		return true
	}
	setItemAmount(dst, dstAmount+move)
	if move == srcAmount {
		*src = world.Item{}
		return true
	}
	setItemAmount(src, srcAmount-move)
	return true
}

func sameStackClass(a, b world.Item) bool {
	if a.Empty() || b.Empty() || a.Index != b.Index || !isSplittable(a.Index) || a.ExpiresAt != b.ExpiresAt {
		return false
	}
	ae, an := nonAmountEffects(a)
	be, bn := nonAmountEffects(b)
	if an != bn {
		return false
	}
	for i := 0; i < an; i++ {
		if ae[i] != be[i] {
			return false
		}
	}
	return true
}

func canWriteItemAmount(it world.Item) bool {
	for _, ef := range it.Effects {
		if ef.Effect == 0 || ef.Effect == efAmount {
			return true
		}
	}
	return false
}

func nonAmountEffects(it world.Item) ([3]world.Effect, int) {
	var out [3]world.Effect
	n := 0
	for _, ef := range it.Effects {
		if ef.Effect == 0 || ef.Effect == efAmount {
			continue
		}
		out[n] = ef
		n++
	}
	return out, n
}

// splitItem trata _MSG_SplitItem (0x02E5, Basedef.h:2381): destacar Num unidades da
// pilha no Slot do carry para uma pilha nova. Só empilháveis da lista branca dividem; a
// origem precisa ter mais que Num (nunca dividir um item só ou a pilha inteira) e
// precisa haver célula livre no carry para a pilha nova. Persistência adiada (paridade).
func (d *Dispatcher) splitItem(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 1, 17)
		return
	}
	if s.Trade.Active {
		d.removeTrade(w, s) // dividir no meio de uma troca a cancela (anti-dup)
		return
	}
	var body protocol.MsgSplitItemBody
	if err := body.Decode(payload); err != nil {
		return
	}
	slot := int(body.Slot)
	num := int(body.Num)
	if !carrySlotAccessible(e, slot) {
		return
	}
	if num <= 0 || num >= maxStackAmount {
		return
	}
	src := &e.Carry[slot]
	if src.Empty() || !isSplittable(src.Index) {
		return
	}
	amount := itemAmount(*src)
	if amount <= 1 || amount <= num { // não dá para dividir um item só nem destacar a pilha inteira
		return
	}
	nItem := *src
	setItemAmount(&nItem, num)
	dst := firstEmptyAccessibleCarry(e)
	if dst < 0 {
		return // sem célula livre — deixar a pilha inteira
	}
	e.Carry[dst] = nItem
	setItemAmount(src, amount-num)
	d.sendSlot(w, s, world.ItemPlaceCarry, dst, e.Carry[dst])
	d.sendSlot(w, s, world.ItemPlaceCarry, slot, *src)
}

// Classes de consumível (valor EF_VOLATILE). O Affect 34 do Divino usa Entity.DivineEnd
// como prazo real; os outros buffs temporizados guardam a duração em Affect.Time.
const (
	volHpMpPotion        = 1
	volFairyDust         = 7
	volBuffKappa         = 10
	volRecallScroll      = 11 // Pergaminho do Retorno: retorna ao spawn da última cidade
	volGemaEstelar       = 12 // Gema Estelar: registra o ponto de save do warp
	volPortalScroll      = 13 // Pergaminho de Portal: teleporta para o ponto de save
	volBuffCombat        = 52
	volBuffMental        = 53
	volBuffKappa30       = 55
	volBuffCombat60      = 56
	volBuffMental60      = 57
	volVigor             = 58
	volFrango            = 63
	volDivine7           = 64
	volDivine30          = 66
	volGemDiamond        = 180
	volGemEmerald        = 181
	volGemCoral          = 182
	volGemGarnet         = 183
	volSilverBar         = 185
	volMagicBean         = 186
	volPaint             = volMagicBean
	volEntradaTerritorio = 188 // ticket Entrada do Território (LAN) (_MSG_UseItem.cpp:4202)
	volQuestReward       = 191 // itens de recompensa de quest 4117..4121 (_MSG_UseItem.cpp:2344)
	volHuntingScroll     = 195 // Pedido de Caça: destino escolhido por MSG_UseItem.WarpID
	volExpChest          = 198
	volIdealStone        = 211 // Pedra_Ideal 5338: Arch→Celestial (_MSG_UseItem.cpp:3003)
	volBuffKappa20h      = 200
	volBuffCombat20h     = 201
	volBuffMental20h     = 202
	// unidades de tick de affect (Basedef.h): um tick = 8s de tempo real.
	affect1H          = 450
	affect1D          = 10800
	affectExpChestInc = affect1H * 2
	affectTimeCap     = 324000
	// divineAffectTime é o Affect.Time "infinito" do original para o slot Divino —
	// a expiração real é DivineEnd (wall-clock), não este campo (captura §B).
	divineAffectTime = 2000000000
)

// issue #135: classes EF_VOLATILE que antes caíam no no-op padrão do
// useItem — o cliente mostrava um consumo fantasma que qualquer ressincronização
// de slot posterior (ex.: um move de _MSG_TradingItem) revertia, já que nada era
// decrementado do lado do servidor.
const (
	volAdamantita  = 9   // combine de upgrade lendário Adamantita/Beril/Tectita/Spinner (_MSG_UseItem.cpp:1097-1213)
	volChocolate   = 204 // Chocolate do Amor (_MSG_UseItem.cpp:6082-6131)
	volCoracaoDoce = 205 // Coração Doce (_MSG_UseItem.cpp:6030-6079)

	// Categorias dos pergaminhos da Água; ciclo de vida e tabelas de origem em water.go.
	volWaterMLo, volWaterMHi = 21, 30   // Pergaminho da Água (M), _MSG_UseItem.cpp:1726
	volWaterNLo, volWaterNHi = 131, 140 // Pergaminho da Água (N), _MSG_UseItem.cpp:1920
	volWaterALo, volWaterAHi = 161, 170 // Pergaminho da Água (A), _MSG_UseItem.cpp:2025

	// itemSeloDoGuerreiro e itemPedraMisteriosa não carregam EF_VOLATILE no
	// catálogo (BASE_GetItemAbility cai para 0), então o legado — e este
	// despachante — os identifica por sIndex (_MSG_UseItem.cpp:3184,3325).
	itemSeloDoGuerreiro = 4146
	itemPedraMisteriosa = 4148
	itemFlashPrateado   = 3451
	itemFlashDourado    = 3452
	itemFadaPrateada    = 3914
	itemFadaDourada     = 3915
	itemFadaVermelha3d  = 3902
	itemFadaVermelha5d  = 3905
	itemFadaVermelha7d  = 3908

	// tickets Entrada do Território (LAN) — três tiers compartilhando EF_VOLATILE 188,
	// distinguidos por sIndex: (N)=4111, (M)=4112, (A)=4113. Tier = sIndex - base
	// (_MSG_UseItem.cpp:4204).
	itemEntradaTerritorioBase = 4111
	itemQuestRewardBase       = 4117
	itemQuestRewardLast       = 4121

	itemHuntingScrollBase = 3432
	itemHuntingScrollLast = 3437
)

// huntingScrollDestinations preserva HuntingScrolls[6][10][2] do
// _MSG_UseItem.cpp legado. As linhas são índices de item 3432..3437 e as colunas são
// a seleção do cliente WarpID 1..10.
var huntingScrollDestinations = [6][10][2]int16{
	{{2370, 2106}, {2508, 2101}, {2526, 2109}, {2529, 1882}, {2126, 1600}, {2005, 1617}, {2241, 1474}, {1858, 1721}, {2250, 1316}, {1989, 1755}},
	{{290, 3799}, {724, 3781}, {481, 4062}, {876, 4058}, {855, 3922}, {808, 3876}, {959, 3813}, {926, 3750}, {1096, 3730}, {1132, 3800}},
	{{1242, 4035}, {1264, 4017}, {1333, 3994}, {1358, 4041}, {1462, 4033}, {1326, 3788}, {1493, 3777}, {1437, 3741}, {1389, 3740}, {1422, 3810}},
	{{1376, 1722}, {1426, 1686}, {1381, 1861}, {1326, 1896}, {1510, 1723}, {1543, 1726}, {1580, 1758}, {1182, 1714}, {1634, 1727}, {1237, 1764}},
	{{2367, 4024}, {2236, 4044}, {2236, 3993}, {2209, 3989}, {2453, 4067}, {2485, 4043}, {2537, 3897}, {2489, 3919}, {2269, 3910}, {2202, 3866}},
	{{3664, 3024}, {3582, 3007}, {3514, 3008}, {3819, 2977}, {3517, 2889}, {3745, 2977}, {3639, 2877}, {3650, 2727}, {3660, 2773}, {3746, 2879}},
}

// volClasses (Classes A-E, _MSG_UseItem.cpp:4959) é tratado por useClasseItem
// (classe.go), NÃO por rejectUnimplementedConsumable: sua lógica — o bloco
// #pragma region Classe, SetItemBonus2
// (Server.cpp:2719-2861), e as quatro tabelas g_pBonusValue2..5
// (Basedef.cpp:353-529) — está toda presente em Source/.
const volClasses = 190

// useFairyFlash transforma uma fada vermelha equipada mantendo seus efeitos
// e expiração absoluta. Os itens Flash não têm entrada EF_VOLATILE no catálogo.
func (d *Dispatcher) useFairyFlash(w *world.World, s *world.Session, e *world.Entity, src int) {
	target := &e.Equip[fairyEquipSlot]
	if target.Index != itemFadaVermelha3d && target.Index != itemFadaVermelha5d && target.Index != itemFadaVermelha7d {
		d.notify(w, s, NoticeReqNotMet)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	if target.ExpiresAt != 0 && time.Now().Unix() >= target.ExpiresAt {
		d.notify(w, s, NoticeReqNotMet)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	transformed, _ := fairyFlashTarget(e.Carry[src].Index)
	target.Index = transformed
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceEquip, fairyEquipSlot, *target)
	d.refreshEquip(w, s, e)
}

func fairyFlashTarget(index int16) (int16, bool) {
	switch index {
	case itemFlashPrateado:
		return itemFadaPrateada, true
	case itemFlashDourado:
		return itemFadaDourada, true
	default:
		return 0, false
	}
}

// potionDelay é o mínimo de ms entre usos de poção (_MSG_UseItem.cpp:105-115).
// O original usa 100 por padrão e o expõe na config de execução (Server.cpp:647,
// :1463); usamos o padrão — um botão de config pode vir se algum dia for ajustado.
const potionDelay = 100

// useItem trata _MSG_UseItem (0x0373), handlers/_MSG_UseItem.md. A ação é
// classificada pelo valor EF_VOLATILE do item de origem (BASE_GetItemAbility, captura §B):
// 0 = equipar (CARRY → EQUIP); 11 = Retorno (recall); 12/13 = Gema Estelar/Portal
// (ponto de save do warp, issue #140); 64-66 = Poção Divina. Consumíveis desconhecidos são
// recusados com ressincronização de slot em vez de no-op silencioso.
// Arrastar e soltar entre slots é outra mensagem (tradingItem).
func (d *Dispatcher) useItem(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	var body protocol.MsgUseItemBody
	if err := body.Decode(payload); err != nil {
		return
	}
	if int(body.SourType) != world.ItemPlaceCarry {
		return // itens consumidos/equipados vêm do inventário
	}
	src := int(body.SourPos)
	if !carrySlotAccessible(e, src) || e.Carry[src].Empty() {
		return
	}
	if e.Carry[src].Index == itemWandererBag {
		d.useWandererBag(w, s, e, src)
		return
	}
	if e.Carry[src].Index == itemFlashPrateado || e.Carry[src].Index == itemFlashDourado {
		d.useFairyFlash(w, s, e, src)
		return
	}
	if d.useQuest256Ticket(w, s, e, src) {
		return
	}
	if e.Carry[src].Index >= archCrystalFirstItem && e.Carry[src].Index <= archCrystalFirstItem+3 {
		d.useArchCrystal(w, s, e, src)
		return
	}
	// Selo do Guerreiro e Pedra Misteriosa não têm EF_VOLATILE, então d.itemVolatiles
	// fica 0 para eles — cheque sIndex primeiro para não caírem no caminho de
	// equipar do vol==0 (canEquipSlot simplesmente os recusaria em silêncio, o mesmo bug de
	// "consumo fantasma" que este bloco inteiro corrige para a issue #135).
	switch e.Carry[src].Index {
	case itemSeloDoGuerreiro:
		d.useSeloDoGuerreiro(w, s, e, src)
		return
	case itemPedraMisteriosa:
		d.usePedraMisteriosa(w, s, e, src)
		return
	case furyStoneItem: // issue #408: também sem EF_VOLATILE
		d.useFuryStone(w, s, e, src)
		return
	}
	switch vol := d.itemVolatiles[int(e.Carry[src].Index)]; {
	case vol == 0:
		d.equipItem(w, s, e, body, payload)
	case vol == volDustOri || vol == volDustLac:
		d.refineItem(w, s, e, body, src, vol)
	case vol == volHpMpPotion:
		d.useHealPotion(w, s, e, src)
	case vol >= 173 && vol <= 175:
		d.useNightmareScroll(w, s, e, src, vol-173)
	case vol == 212 && e.Carry[src].Index == 5137:
		d.useNightmareDeed(w, s, e, src)
	case vol == volRecallScroll:
		d.useRecallScroll(w, s, e, src)
	case vol == volGemaEstelar:
		d.useGemaEstelar(w, s, e, src)
	case vol == volPortalScroll:
		d.usePortalScroll(w, s, e, src)
	case vol >= volSephiraLo && vol <= volSephiraHi:
		d.useSkillBook(w, s, e, src, vol)
	case vol == volFairyDust:
		d.useFairyDust(w, s, e, src)
	case isLegacyBuffConsumableVol(vol):
		d.useLegacyBuffConsumable(w, s, e, src, vol)
	case vol == volExpChest:
		d.useExpChest(w, s, e, src)
	case vol >= volDivine7 && vol <= volDivine30:
		d.useDivine(w, s, e, src, vol)
	case vol == volVigor:
		d.useVigor(w, s, e, src, int(e.Carry[src].Index))
	case vol == volFrango:
		d.useFrangoAssado(w, s, e, src)
	case vol == volSilverBar:
		d.useSilverBar(w, s, e, src)
	case vol >= volGemDiamond && vol <= volGemGarnet:
		d.useBaseGem(w, s, e, body, src, vol)
	case vol == volJoiaPvP:
		d.useJoiaPvP(w, s, e, src)
	case vol == volJoiaRecovery:
		d.useJoiaRecovery(w, s, e, src)
	case vol == volMagicBean:
		d.useMagicBean(w, s, e, body, src)
	case vol == volAdamantita:
		d.useAdamantita(w, s, e, body, src)
	case vol == volChocolate:
		d.useChocolateDoAmor(w, s, e, src)
	case vol == volCoracaoDoce:
		d.useCoracaoDoce(w, s, e, src)
	case vol == volClasses:
		d.useClasseItem(w, s, e, body, src)
	case vol == volEntradaTerritorio:
		d.useEntradaTerritorio(w, s, e, src)
	case vol == volQuestReward:
		d.useQuestReward(w, s, e, src)
	case vol == volMolarGargula:
		d.useMolarGargula(w, s, e, src)
	case vol == volPilulaOrc:
		d.usePilulaOrc(w, s, e, src)
	case vol == volIdealStone:
		d.useIdealStone(w, s, e, src)
	case vol == volHuntingScroll:
		d.useHuntingScroll(w, s, e, src, body.WarpID)
	case vol >= volWaterMLo && vol <= volWaterMHi,
		vol >= volWaterNLo && vol <= volWaterNHi,
		vol >= volWaterALo && vol <= volWaterAHi:
		d.useWaterScroll(w, s, e, src, vol)
	default:
		// issue #204: não fazer no-op silencioso em consumíveis desconhecidos. O cliente pode
		// removê-los de forma otimista, e então a próxima ressincronização de slot (compra/move/save)
		// parece trazer o item de volta.
		d.rejectUnimplementedConsumable(w, s, e, src)
	}
}

// useHuntingScroll porta o bloco legado de Pedidos de Caça. O item seleciona
// uma de seis linhas de mapa e o WarpID seleciona um de dez destinos dentro dela.
func (d *Dispatcher) useHuntingScroll(w *world.World, s *world.Session, e *world.Entity, src int, warpID uint16) {
	itemIndex := int(e.Carry[src].Index)
	if itemIndex < itemHuntingScrollBase || itemIndex > itemHuntingScrollLast || warpID == 0 || warpID > 10 {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	dest := huntingScrollDestinations[itemIndex-itemHuntingScrollBase][int(warpID)-1]
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.doTeleport(w, s, dest[0], dest[1])
	d.log.Info("hunting scroll used", "conn", s.Conn, "item", itemIndex, "warp_id", warpID, "to_x", dest[0], "to_y", dest[1])
}

func isLegacyBuffConsumableVol(vol int) bool {
	switch vol {
	case volBuffKappa, volBuffKappa30, volBuffKappa20h,
		volBuffCombat, volBuffCombat60, volBuffCombat20h,
		volBuffMental, volBuffMental60, volBuffMental20h:
		return true
	default:
		return false
	}
}

func quest256StepForTicket(index int16) (quest256Step, bool) {
	stepIdx := int(index - 4038)
	if stepIdx < 0 || stepIdx >= len(quest256Steps) {
		return quest256Step{}, false
	}
	return quest256Steps[stepIdx], true
}

func (d *Dispatcher) useQuest256Ticket(w *world.World, s *world.Session, e *world.Entity, src int) bool {
	step, ok := quest256StepForTicket(e.Carry[src].Index)
	if !ok {
		return false
	}
	if s.Trade.Active {
		d.removeTrade(w, s)
		return true
	}
	if s.TradeMode != 0 {
		s.TradeMode = 0
		d.removeTrade(w, s)
		return true
	}
	if e.ClassMaster != classMasterMortal && e.ClassMaster != classMasterArch {
		d.notify(w, s, NoticeReqNotMet)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return true
	}
	if e.Level < step.minLevel || e.Level >= step.maxLevel {
		d.notify(w, s, NoticeReqNotMet)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return true
	}

	itemIdx := e.Carry[src].Index
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	// Teleporta imediatamente: o DoTeleport de _MSG_Quest.cpp para esses mesmos tickets
	// (COVEIRO/JARDINEIRO/KAIZEN/HIDRA/ELFOS, :332/:376/:420/:464/:508) é
	// síncrono, e também os outros teleportes disparados por item deste arquivo
	// (usePortalScroll, useGemaEstelar). O masterGriffTravelDelay de 9.5s que isto
	// pegava emprestado só se justifica para o opcode real _MSG_MasterGriff, que tem
	// animação de viagem correspondente no cliente; um clique direito puro não tem, então o
	// atraso só deixava o jogador olhando para o nada por ~10s depois do item sumir.
	d.teleportQuest256Step(w, s, e, step)
	d.log.Info("quest256 ticket teleport", "conn", s.Conn, "item", itemIdx, "level", e.Level, "quest_flag", step.flag)
	return true
}

// rejectUnimplementedConsumable recusa itens não suportados e restaura o slot no cliente.
func (d *Dispatcher) rejectUnimplementedConsumable(w *world.World, s *world.Session, e *world.Entity, src int) {
	d.notify(w, s, NoticeCantUseHere)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
}

const efAmount = 61

// efKeyID é EF_KEYID (ItemEffect.h:96): um portão e sua chave correspondente o carregam;
// um portão trancado só abre quando o EF_KEYID de um item carregado é igual ao do portão (gate.go).
const efKeyID = 39

// efQuest é EF_QUEST (ItemEffect.h:115), usado para vincular as chaves internas de Castle/Zakum
// ao nível de quest ativo no momento.
const efQuest = 58

func itemAmount(it world.Item) int {
	for _, ef := range it.Effects {
		if ef.Effect == efAmount {
			return int(ef.Value)
		}
	}
	return 1
}

func consumeOneItem(it *world.Item) {
	if itemAmount(*it) <= 1 {
		*it = world.Item{}
		return
	}
	for i := range it.Effects {
		if it.Effects[i].Effect == efAmount {
			it.Effects[i].Value--
			return
		}
	}
}

// faixa de voláteis dos livros Sephira: Vol 31-38 ensina o bit LearnedSkill Vol-7
// (bits 24-31, as skills de classe extra; _MSG_UseItem.cpp "Livros Sephira").
const (
	volSephiraLo = 31
	volSephiraHi = 38
)

// useSkillBook consome um livro Sephira: marca o bit aprendido, atualiza a
// janela de skills (Learn viaja no UpdateEtc) e gasta uma unidade. Já aprendido
// recusa e ressincroniza o slot. O legado também marca um Affect(44) cosmético
// de brilho — adiado até o motor de affect (M4) chegar.
func (d *Dispatcher) useSkillBook(w *world.World, s *world.Session, e *world.Entity, src, vol int) {
	bit := int32(1) << uint(vol-7)
	if e.LearnedSkill&bit != 0 {
		d.notify(w, s, NoticeAlreadyLearned)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	e.LearnedSkill |= bit
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendEtc(w, s, e)
	d.log.Info("sephira book learned", "conn", s.Conn, "bit", vol-7)
}

const (
	affectLegacyConsumableBuff = 4
	legacyBuffKappa            = 1
	legacyBuffCombat           = 2
	legacyBuffMental           = 3
	legacyBuffSephira          = 4
	legacyBuffAntidote         = 5
)

func legacyBuffConsumableEffect(index int16) (value uint8, ticks uint32, ok bool) {
	ticks = 80
	switch index {
	case 787, 3310, 3319:
		value = legacyBuffKappa
	case 1764, 3311, 3320:
		value = legacyBuffCombat
	case 1765, 3312, 3321:
		value = legacyBuffMental
	case 3361, 3362, 3363:
		value = legacyBuffSephira
	case 416:
		value = legacyBuffAntidote
	default:
		return 0, 0, false
	}

	switch index {
	case 3310:
		ticks = affect1H / 2
	case 3311, 3312:
		ticks = affect1H
	case 3319, 3320, 3321:
		ticks = affect1H * 20
	case 3361:
		ticks = affect1H * 168
	case 3362:
		ticks = affect1H * 360
	case 3363:
		ticks = affect1H * 720
	case 416:
		ticks = affect1H / 2
	}
	return value, ticks, true
}

func legacyBuffConsumableSlot(e *world.Entity, value uint8) int {
	for i := range e.Affect {
		if e.Affect[i].Type == affectLegacyConsumableBuff && e.Affect[i].Value == value {
			return i
		}
	}
	for i := range e.Affect {
		if e.Affect[i].Type == 0 {
			return i
		}
	}
	return -1
}

// useLegacyBuffConsumable porta _MSG_UseItem.cpp:1215-1341: Kappa,
// Combatente, Mental, Sephira e Antidoto compartilham o Affect tipo 4 e diferem por
// Value/Time. A busca de slot é mais restrita que GetEmptyAffect: só um slot
// tipo 4 existente com o mesmo Value pode ser reavivado.
func (d *Dispatcher) useLegacyBuffConsumable(w *world.World, s *world.Session, e *world.Entity, src, vol int) {
	itemIdx := e.Carry[src].Index
	value, ticks, ok := legacyBuffConsumableEffect(itemIdx)
	if !ok {
		d.rejectUnimplementedConsumable(w, s, e, src)
		return
	}
	slot := legacyBuffConsumableSlot(e, value)
	if slot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	e.Affect[slot] = world.Affect{Type: affectLegacyConsumableBuff, Value: value, Time: ticks}
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
	d.log.Info("legacy buff consumable used", "conn", s.Conn, "item", itemIdx, "vol", vol, "value", value, "time", ticks)
}

// equipItem é o caminho CARRY → EQUIP do _MSG_UseItem (itens com Vol==0).
func (d *Dispatcher) equipItem(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgUseItemBody, payload []byte) {
	if int(body.DestType) != world.ItemPlaceEquip {
		return
	}
	src, dst := int(body.SourPos), int(body.DestPos)
	if dst < 0 || dst >= world.MaxEquip {
		return
	}
	if !d.canEquipSlot(e.Carry[src].Index, dst) {
		return // slot errado para este item (ex.: um consumível no slot do corpo)
	}
	if !d.meetsEquipReq(e, e.Carry[src]) {
		d.notify(w, s, NoticeReqNotMet) // nível/atributos insuficientes para este item
		return
	}
	e.Carry[src], e.Equip[dst] = e.Equip[dst], e.Carry[src]
	w.Send(s, protocol.MsgUseItem, payload) // devolver o resultado
	d.refreshEquip(w, s, e)                 // atualizar o equipamento renderizado
	if dst == mountEquipSlot {
		d.refreshBabyMountSummon(w, s, e)
	}
}

func (d *Dispatcher) useExpChest(w *world.World, s *world.Session, e *world.Entity, src int) {
	slot := e.EmptyAffect(world.AffectExpChest)
	if slot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	af := &e.Affect[slot]
	af.Type = world.AffectExpChest
	af.Level = 0
	af.Value = 0
	af.Time += affectExpChestInc
	if af.Time > affectTimeCap {
		af.Time = affectTimeCap
	}
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// useFairyDust consome Poeira de Fada (EF_VOLATILE 7). O handler legado coloca
// a Exp direto no próximo limiar da curva e chama CheckGetLevel; os dois lados
// do seu ramo rand()%2 fazem o mesmo neste fork.
func (d *Dispatcher) useFairyDust(w *world.World, s *world.Session, e *world.Entity, src int) {
	if archExpLocked(e) {
		d.sendChatText(w, s, "Conclua a liberacao de nivel na Lindy antes de usar Poeira de Fada.")
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	if e.Level < 0 {
		return
	}
	if e.Level >= level.MaxLevel {
		e.Exp = level.MaxExp
	} else {
		e.Exp = level.NextLevelExp(e.Level)
	}
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	if !d.applyLevelUpsWithoutCirclet(w, s, e) {
		d.sendEtc(w, s, e)
	}
	d.log.Info("fairy dust used", "conn", s.Conn, "classmaster", e.ClassMaster, "level", e.Level)
}

// inPesadelo informa se (x,y) cai nas zonas de dungeon instanciada Pesadelo_A
// ou Pesadelo_M (linhas 5-6 de tmserver/content/TMsrv/run/Regions.txt), o mesmo
// teste de segmento de 128 unidades que _MSG_UseItem.cpp reusa para os
// tickets Vol 12/13/174/175 (Gema Estelar, Portal e os dois pergaminhos de entrada do Pesadelo).
func inPesadelo(x, y int16) bool {
	return (x/128 == 9 && y/128 == 1) || (x/128 == 8 && y/128 == 2)
}

// useRecallScroll consome um Pergaminho do Retorno (EF_VOLATILE 11): o handler
// legado chama DoRecall(conn) e depois decrementa o item (_MSG_UseItem.cpp:1344).
func (d *Dispatcher) useRecallScroll(w *world.World, s *world.Session, e *world.Entity, src int) {
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.recall(w, s, e)
}

// useGemaEstelar consome uma Gema Estelar (EF_VOLATILE 12): registra a posição
// atual do jogador como ponto de save do warp (_MSG_UseItem.cpp:1360-1419). Bloqueado
// quando se está dentro dos limites de uma cidade, a menos que o tile seja uma exceção
// de instância Pesadelo (goto CanSave legado) — espelha o gate Village/Arena legado, menos
// as checagens map_att&4 e BASE_GetArena: a grade bruta de atributo por tile não tem
// caminho de consulta em execução hoje e a tabela WarArea de BASE_GetArena não tem
// equivalente em Go ainda (plano da issue #140), então esses dois são divergência deliberada.
func (d *Dispatcher) useGemaEstelar(w *world.World, s *world.Session, e *world.Entity, src int) {
	if !inPesadelo(e.X, e.Y) && world.Village(e.X, e.Y) >= 0 {
		d.notify(w, s, NoticeCantUseHere)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	e.SaveX, e.SaveY = e.X, e.Y
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.notify(w, s, NoticeSetWarp)
}

// usePortalScroll consome um Pergaminho de Portal (EF_VOLATILE 13): teleporta
// o jogador ao ponto de Gema Estelar salvo (_MSG_UseItem.cpp:1421-1452).
// Bloqueado se o ponto salvo estiver dentro de uma instância Pesadelo (não dá para entrar
// nela de fora, espelhando a checagem legada), ou se nenhum ponto nunca foi
// salvo (SaveX/SaveY ainda zero — uma proteção do port em Go contra teleportar para a
// origem do mapa; o legado sempre tem algum padrão pela posição de save semeada do
// personagem).
func (d *Dispatcher) usePortalScroll(w *world.World, s *world.Session, e *world.Entity, src int) {
	if inPesadelo(e.SaveX, e.SaveY) {
		d.notify(w, s, NoticeCantUseHere)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	if e.SaveX == 0 && e.SaveY == 0 {
		return
	}
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.doTeleport(w, s, e.SaveX, e.SaveY)
}

// useEntradaTerritorio consome um ticket "Entrada do Território" (LAN)
// (EF_VOLATILE 188) e teleporta o jogador para a região Lan_N/M/A correspondente,
// portando _MSG_UseItem.cpp:4202-4236. O tier é o sIndex do item menos a
// base (N) (4111→0, 4112→1, 4113→2); cada tier cai numa coordenada fixa no
// mapa 0 com um espalhamento rand()%5-3 por eixo, dentro dos limites da região declarados em
// tmserver/content/TMsrv/run/Regions.txt:45-47 (Lan_N/M/A).
//
// O gate casa exatamente com o legado — só tier de classe, sem checagem de cidadania: tiers
// N/M (0,1) exigem ClassMaster Arch ou Mortal; tier A (>=2) exige um tier
// celestial. Numa divergência de classe o código legado reenvia o slot e retorna sem
// consumir; fazemos o mesmo (mais um NoticeReqNotMet, como os outros handlers de
// ticket). Na prática só N/M disparam hoje: nenhum personagem alcança um
// ClassMaster celestial até o sistema adiado de evolução celestial chegar
// (docs/migration/celestial-system-plan.md), mas o gate (A) é portado fielmente
// para funcionar sozinho quando ele existir.
func (d *Dispatcher) useEntradaTerritorio(w *world.World, s *world.Session, e *world.Entity, src int) {
	territorio := int(e.Carry[src].Index) - itemEntradaTerritorioBase

	reject := func() {
		d.notify(w, s, NoticeReqNotMet)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	}
	if territorio <= 1 {
		if e.ClassMaster != classMasterArch && e.ClassMaster != classMasterMortal {
			reject()
			return
		}
	} else if e.ClassMaster != classMasterCelestial && e.ClassMaster != classMasterCelestialCS && e.ClassMaster != classMasterSCelestial {
		reject()
		return
	}

	var bx, by int16
	switch territorio {
	case 0:
		bx, by = 3639, 3639
	case 1:
		bx, by = 3782, 3527
	default:
		bx, by = 3911, 3655
	}
	// rand()%5 - 3 por eixo, X depois Y, para preservar a ordem/espalhamento legado das chamadas.
	x := bx + int16(w.Rand().Intn(5)) - 3
	y := by + int16(w.Rand().Intn(5)) - 3

	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.doTeleport(w, s, x, y)
}

// useQuestReward trata os cinco troféus de quest Vol-191. Sua linha é o
// deslocamento contíguo de item, e QuestsRate.txt fornece recompensas por tier e
// faixas de nível semiabertas. Toda mutação fica no loop do mundo.
func (d *Dispatcher) useQuestReward(w *world.World, s *world.Session, e *world.Entity, src int) {
	tier := int(e.Carry[src].Index) - itemQuestRewardBase
	rate, ok := d.questRates.Tier(tier)
	if !ok {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	questExp := rate.ArchExp
	minLevel, maxLevel := rate.ArchMin, rate.ArchMax
	if e.ClassMaster == classMasterMortal {
		questExp = rate.MortalExp
		minLevel, maxLevel = rate.MortalMin, rate.MortalMax
	}
	if e.Level < minLevel || e.Level >= maxLevel {
		d.notify(w, s, NoticeReqNotMet)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	if int64(e.Coin)+int64(rate.Coin) > maxCoin {
		e.Coin = maxCoin
	} else {
		e.Coin += rate.Coin
	}
	d.grantDirectExp(w, s, e, questExp)
	d.grantQuestPartyExp(w, e, questExp/10)

	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.sendEtc(w, s, e) // moeda alterada mesmo com a EXP já no teto
}

// grantQuestPartyExp porta a divisão por grupo do troféu (_MSG_UseItem.cpp:2403-2480).
// O líder é consumer.Leader, ou o próprio consumidor quando isso é <= 0 ou
// >= MAX_USER, e sempre recebe a parte, mesmo quando é o consumidor:
// um consumidor solo ou líder recebe QuestExp mais QuestExp/10 (issue #396). Só
// os membros do grupo do líder pulam o consumidor e as sessões fora de jogo.
func (d *Dispatcher) grantQuestPartyExp(w *world.World, consumer *world.Entity, share int64) {
	if share <= 0 {
		return
	}
	leaderID := consumer.Leader
	if leaderID <= 0 || leaderID >= world.MaxUser {
		leaderID = consumer.ID
	}
	leader := w.Entity(leaderID)
	if leader == nil {
		return
	}
	// O grant legado ao líder não tem filtro USER_PLAY; uma sessão nil ainda
	// aumenta a Exp e só pula os pacotes do cliente.
	d.grantDirectExp(w, w.Session(leaderID), leader, share)

	seen := map[int]bool{consumer.ID: true, leaderID: true}
	for _, conn := range leader.PartyList {
		if conn <= 0 || conn >= world.MaxUser || seen[conn] {
			continue
		}
		seen[conn] = true
		rs, re := w.Session(conn), w.Entity(conn)
		if rs == nil || rs.Mode != world.UserPlay || re == nil {
			continue
		}
		d.grantDirectExp(w, rs, re, share)
	}
}

// useHealPotion consome uma poção de HP/MP (EF_VOLATILE 1), portando
// _MSG_UseItem.cpp:101-148 (docs/migration/handlers/_MSG_UseItem.md, catalog.go:102).
//
// A poção NÃO cura: ela eleva os alvos de pedido ReqHp/ReqMp pelo
// EF_HP/EF_MP do catálogo do item (limitado ao máximo efetivo atual) e o applyHp/applyMp do tick
// de 1s fecha as barras neles (regenPlayers, mobai.go). Essa rampa é o
// comportamento do original — dá para superar a cura com dano, não é preenchimento instantâneo.
//
// A resposta é SetHpMp e é deliberadamente só para si (SendSetHpMp, SendFunc.cpp:1722):
// ela carrega ReqHp/ReqMp para o cliente de quem bebe animar o preenchimento. Jogadores
// próximos são informados pelo sendScore multicast do tick quando o HP realmente muda —
// esse caminho, não este, é o que corrige a barra de HP velha da issue #99.
func (d *Dispatcher) useHealPotion(w *world.World, s *world.Session, e *world.Entity, src int) {
	// Anti-spam do PotionDelay (_MSG_UseItem.cpp:105-115): um uso dentro da janela é
	// descartado e o slot ressincronizado, então a pilha não é consumida. A guarda usa o
	// relógio do SERVIDOR — o original lê GetTickCount(), e um tick fornecido pelo cliente
	// seria falsificável. Aritmética int64 evita underflow de uint32 no tick 0.
	now := w.Now()
	if s.PotionTick != 0 && int64(now)-int64(s.PotionTick) < potionDelay {
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	s.PotionTick = now

	maxHP, maxMP := effectiveMaxHP(e), effectiveMaxMP(e)
	for _, be := range d.itemEffects[int(e.Carry[src].Index)] {
		switch be.Eff {
		case efHp:
			s.ReqHp += int32(be.Val)
			if s.ReqHp > maxHP {
				s.ReqHp = maxHP
			}
		case efMp:
			s.ReqMp += int32(be.Val)
			if s.ReqMp > maxMP {
				s.ReqMp = maxMP
			}
		}
	}
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendSetHpMp(w, s, e)
}

// useDivine consome uma Poção Divina: define o buff Divino (Affect 34) por 8/16/31
// dias e recalcula o score para o cliente ver +20% MaxHp/MaxMp/Damage. Espelha
// _MSG_UseItem.cpp:2128 (captura §B,C). Se o jogador já está divino, recusa
// (_NN_CantEatMore) e ressincroniza o slot do item.
func (d *Dispatcher) useDivine(w *world.World, s *world.Session, e *world.Entity, src, vol int) {
	slot := e.EmptyAffect(world.AffectDivine)
	if slot < 0 || e.Affect[slot].Type == world.AffectDivine {
		d.notify(w, s, NoticeCantEatMore)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	// Vol 64/65/66 → 8/16/31 dias (a folga de +1 casa com _MSG_UseItem.cpp:2142).
	days := int64(8)
	switch vol {
	case 65:
		days = 16
	case volDivine30:
		days = 31
	}
	e.DivineEnd = time.Now().Unix() + days*86400
	e.Affect[slot] = world.Affect{Type: world.AffectDivine, Level: 1, Time: divineAffectTime}
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e) // re-limite; os +20% são na leitura (getters efetivos)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// useVigor consome uma Poção de Vigor (EF_VOLATILE 58): o Affect 35 soma +10% MaxHp/MaxMp
// na leitura (_MSG_UseItem.cpp:2174, captura §B,C). Reaplicar reaviva o slot.
func (d *Dispatcher) useVigor(w *world.World, s *world.Session, e *world.Entity, src, itemIdx int) {
	slot := e.EmptyAffect(world.AffectVigor)
	if slot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	ticks := uint32(affect1H)
	switch itemIdx {
	case 3364:
		ticks = affect1D * 7
	case 3365:
		ticks = affect1D * 15
	case 3366:
		ticks = affect1D * 30
	}
	e.Affect[slot] = world.Affect{Type: world.AffectVigor, Level: 1, Time: ticks}
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// useFrangoAssado consome um Frango Assado (EF_VOLATILE 63): o Affect 30 soma
// +2000 fixos de ForceMobDamage por 4h (_MSG_UseItem.cpp:2308-2341, Basedef.cpp:4427). Diferente
// das poções de cura, isto não é cura — é um buff de dano contra criatura lido no score.
func (d *Dispatcher) useFrangoAssado(w *world.World, s *world.Session, e *world.Entity, src int) {
	slot := e.EmptyAffect(world.AffectForceMobDamage)
	if slot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	e.Affect[slot] = world.Affect{Type: world.AffectForceMobDamage, Level: 2000, Time: affect1H * 4}
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// silverBarGold devolve o ouro creditado pelos itens "Barra de Prata" Vol 185
// (_MSG_UseItem.cpp, "Barra de Prata"). O caminho 4011 é a barra de 1Bi da issue #56.
func silverBarGold(itemIdx int16) (int32, bool) {
	switch itemIdx {
	case 4026:
		return 1_000_000, true
	case 4027:
		return 5_000_000, true
	case 4028:
		return 10_000_000, true
	case 4029:
		return 50_000_000, true
	case 4010:
		return 100_000_000, true
	case 4011:
		return 1_000_000_000, true
	default:
		return 0, false
	}
}

// useSilverBar consome um item de barra de prata e credita seu valor fixo em ouro,
// preservando o teto legado de 2G de ouro do personagem.
func (d *Dispatcher) useSilverBar(w *world.World, s *world.Session, e *world.Entity, src int) {
	itemIdx := e.Carry[src].Index
	gold, ok := silverBarGold(itemIdx)
	if !ok {
		return
	}
	if int64(e.Coin)+int64(gold) > maxCoin {
		d.notify(w, s, NoticeCargoFull)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	e.Coin += gold
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendEtc(w, s, e)
	d.log.Info("silver bar used", "conn", s.Conn, "item", itemIdx, "gold", gold, "coin", e.Coin)
}

// baseGemVariant mapeia as quatro gemas de troca de base para o índice de gema
// empacotado +10..+15 usado por BASE_SetItemSanc (_MSG_UseItem.cpp:3890-4201).
func baseGemVariant(vol int) (int, bool) {
	switch vol {
	case volGemDiamond:
		return 0, true
	case volGemEmerald:
		return 1, true
	case volGemCoral:
		return 2, true
	case volGemGarnet:
		return 3, true
	default:
		return 0, false
	}
}

// useBaseGem trata Gema de Diamante/Esmeralda/Coral/Garnet (Vol 180..183).
// Elas trocam um item equipado ANCT grau 5..8 pela variante de base escolhida e,
// em alvos +10..+15, reescrevem o índice de gema de sanc empacotado. O legado aceita só
// equipamento equipado, exceto o slot de corpo 0 e os slots de acessório 8..15.
func (d *Dispatcher) useBaseGem(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgUseItemBody, src, vol int) {
	gem, ok := baseGemVariant(vol)
	if !ok {
		return
	}

	dstSlot := int(body.DestPos)
	if int(body.DestType) != world.ItemPlaceEquip || dstSlot == 0 || (dstSlot >= 8 && dstSlot < world.MaxEquip) {
		d.baseGemReject(w, s, e, src, NoticeOnlyToEquips)
		return
	}
	dst := d.itemSlot(w, s, e, int(body.DestType), dstSlot)
	if dst == nil {
		return
	}

	level := refine.Level(*dst)
	gemLevel := d.refineLimits().GemLevel
	grade := d.itemGrades[int(dst.Index)]
	if level < gemLevel && grade < 5 {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	if grade >= 5 && grade <= 8 {
		dst.Index += int16(gem - (grade - 5))
	}
	if level >= gemLevel {
		refine.Set(dst, level, gem)
	}

	d.refreshScore(e)
	d.sendScore(w, s, e)
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceEquip, dstSlot, *dst)
}

func (d *Dispatcher) baseGemReject(w *world.World, s *world.Session, e *world.Entity, src int, n Notice) {
	d.notify(w, s, n)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
}

// Voláteis de Jóia (joia de cash). Vol 242 é o conjunto de buff PvP (concede affect tipo
// 8, um bit de Level por joia); Vol 243 é o par recuperação/armazenagem.
const (
	volJoiaPvP      = 242
	volJoiaRecovery = 243
	affectPvP       = 8
)

// joiaPvPBit mapeia o sIndex de cada joia Vol-242 para seu bit de Level do affect 8
// (_MSG_UseItem.cpp:4287-4313). O bit escolhe qual bônus BASE_GetCurrentScore
// aplica (veja applyAffectScore case 8). Note que 3203/3207 são Vol 243, não aqui.
var joiaPvPBit = map[int16]uint{
	3200: 0, // Sagacidade (o bit 0 não tem efeito no score — peculiaridade do legado)
	3201: 1, // Resistência
	3202: 2, // Revelação
	3204: 3, // Absorção
	3205: 4, // Proteção
	3206: 5, // Poder
	3208: 6, // Precisão (Accuracy é um campo morto no legado — só ícone)
	3209: 7, // Magia
}

// joiaRecoveryCleanse são os tipos de affect de buff de skill que a Jóia da Recuperação
// (3203) remove (_MSG_UseItem.cpp:4356).
var joiaRecoveryCleanse = map[uint8]bool{1: true, 3: true, 5: true, 7: true, 10: true, 12: true, 20: true, 32: true}

// useJoiaPvP consome uma joia PvP Vol-242: define (ou reaviva por OR) o slot
// affect 8 compartilhado com o bit de Level da joia por uma hora, depois recalcula e
// empurra o snapshot de score/affect. Joias empilhadas acumulam seus bits no
// mesmo slot (GetEmptyAffect devolve o slot tipo 8 existente). Espelha a
// região "Jóias PvP" de _MSG_UseItem.cpp.
func (d *Dispatcher) useJoiaPvP(w *world.World, s *world.Session, e *world.Entity, src int) {
	bit, ok := joiaPvPBit[e.Carry[src].Index]
	if !ok {
		return
	}
	slot := e.EmptyAffect(affectPvP)
	if slot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
		return
	}
	if e.Affect[slot].Type != affectPvP {
		e.Affect[slot] = world.Affect{Type: affectPvP, Level: uint16(1) << bit, Value: 0}
	} else {
		e.Affect[slot].Level |= uint16(1) << bit
	}
	e.Affect[slot].Time = affect1H
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// useJoiaRecovery consome uma joia Vol-243. A Jóia da Recuperação (3203)
// remove os buffs/debuffs de skill do jogador (joiaRecoveryCleanse) antes de consumir;
// a Jóia da Armazenagem (3207) não tem atributo do lado do servidor, então é só consumida.
// Espelha a região "Armazenagem - Recuperação" de _MSG_UseItem.cpp.
func (d *Dispatcher) useJoiaRecovery(w *world.World, s *world.Session, e *world.Entity, src int) {
	if e.Carry[src].Index == 3203 {
		cleared := false
		for i := range e.Affect {
			if joiaRecoveryCleanse[e.Affect[i].Type] {
				e.Affect[i] = world.Affect{}
				cleared = true
			}
		}
		if cleared {
			d.refreshScore(e)
			d.sendScore(w, s, e)
			d.sendAffect(w, s, e)
		} else {
			// Nada a limpar: sem aviso, o item era consumido em silêncio.
			d.sendClientMessage(w, s, noCleanseText)
		}
	}
	consumeOneItem(&e.Carry[src])
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, src, itemToSel(e.Carry[src])))
}

// useCoracaoDoce consome Coração Doce (EF_VOLATILE 205): um buff curto de Velocidade
// (Affect 2) + Defesa (Affect 11), 1h/5 = 12min (_MSG_UseItem.cpp:6030-6079).
func (d *Dispatcher) useCoracaoDoce(w *world.World, s *world.Session, e *world.Entity, src int) {
	speedSlot := e.EmptyAffect(2) // Velocidade
	if speedSlot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	e.Affect[speedSlot] = world.Affect{Type: 2, Value: 2, Time: affect1H / 5}

	defSlot := e.EmptyAffect(11) // Defesa
	if defSlot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	e.Affect[defSlot] = world.Affect{Type: 11, Time: affect1H / 5}

	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// useChocolateDoAmor consome Chocolate do Amor (EF_VOLATILE 204): um buff curto de Dano
// (Affect 9) + Skill (Affect 15), 1h/5 = 12min (_MSG_UseItem.cpp:6082-6131).
func (d *Dispatcher) useChocolateDoAmor(w *world.World, s *world.Session, e *world.Entity, src int) {
	dmgSlot := e.EmptyAffect(9) // Dano
	if dmgSlot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	e.Affect[dmgSlot] = world.Affect{Type: 9, Time: affect1H / 5}

	skillSlot := e.EmptyAffect(15) // Skill
	if skillSlot < 0 {
		d.notify(w, s, NoticeCantEatMore)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	e.Affect[skillSlot] = world.Affect{Type: 15, Value: 55, Time: affect1H / 5}

	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}

// adamantitaUniqueType mapeia o nUnique de um item alvo para o tier de Adamantita a
// que pertence (0-3), ou -1 se não for atualizável a lendário — a tabela literal de
// baldes de _MSG_UseItem.cpp:1113-1126, incluindo o mapeamento duplicado
// 10/20/30/40→3 (peculiaridade do legado, preservada em vez de "corrigida").
func adamantitaUniqueType(nUnique int) int {
	switch nUnique {
	case 5, 14, 24, 34:
		return 0
	case 6, 15, 25, 35:
		return 1
	case 7, 16, 26, 36:
		return 2
	case 8, 17, 27, 37, 10, 20, 30, 40:
		return 3
	default:
		return -1
	}
}

// useAdamantita aplica o combine de upgrade lendário Vol 9 (Spinner/Beril/
// Tectita/Adamantita, índices de item 575-578) no alvo body.DestType/DestPos,
// casando com _MSG_UseItem.cpp:1097-1213. O índice do item de origem escolhe um tier
// (Type = index-575); o nUnique de catálogo do alvo precisa cair no mesmo
// tier e seu Grade precisa ser 1-3. Um sorteio de 51% (rand()%100<=50) troca o
// índice do alvo pelo resultado Extra do catálogo; de um jeito ou de outro a origem é gasta.
func (d *Dispatcher) useAdamantita(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgUseItemBody, src int) {
	dst := d.itemSlot(w, s, e, int(body.DestType), int(body.DestPos))
	if dst == nil || dst.Empty() {
		return // sem esse alvo — o legado só loga e descarta a mensagem
	}
	adamType := int(e.Carry[src].Index) - 575
	if adamType < 0 || adamType >= 4 {
		return
	}

	uniqueType := adamantitaUniqueType(d.itemUnique[int(dst.Index)])
	if uniqueType == -1 || uniqueType != adamType {
		d.refineReject(w, s, e, src, NoticeCantRefineMore)
		return
	}
	grade := d.itemGrades[int(dst.Index)]
	if grade <= 0 || grade >= 4 {
		d.refineReject(w, s, e, src, NoticeCantRefineMore)
		return
	}

	// A Adamantita arrastada já sumiu da visão de inventário do próprio cliente
	// quando esta resposta chega (o arrasto em si é otimista) — só o
	// slot alvo precisa de ressincronização, casando com a convenção de poeira de
	// refineSucceed/refineFail (refine.go:232,262), não a origem.
	if w.Rand().Intn(100) <= 50 {
		if extra := d.itemExtra[int(dst.Index)]; extra > 0 {
			dst.Index = int16(extra)
		}
		d.refreshScore(e)
		d.sendScore(w, s, e)
		d.notify(w, s, NoticeRefineSuccess)
	} else {
		d.notify(w, s, NoticeFailToRefine)
	}
	d.sendSlot(w, s, int(body.DestType), int(body.DestPos), *dst)
	consumeOneItem(&e.Carry[src])
}

// amuletSlot é o índice de Equip em que o amuleto de reino do Selo do Guerreiro cai
// (_MSG_UseItem.cpp:3325-3364, Equip[15]).
const amuletSlot = 15

// useSeloDoGuerreiro consome Selo do Guerreiro (sIndex 4146, sem EF_VOLATILE):
// concede Fama e, uma vez em nível mortal alto sem nenhum amuleto de reino
// já equipado, um de tier de entrada (_MSG_UseItem.cpp:3325-3364).
// SendEmotion(conn,14,3) é cosmético e não é portado, casando com o
// precedente existente em refine.go:233,263 (nenhum opcode de emoção existe neste fork ainda).
func (d *Dispatcher) useSeloDoGuerreiro(w *world.World, s *world.Session, e *world.Entity, src int) {
	const maxFame int32 = 2_000_000_000
	if e.Fame >= maxFame-10 {
		e.Fame = maxFame
	} else {
		e.Fame += 10
	}

	hasAmulet := false
	switch e.Equip[amuletSlot].Index {
	case 3191, 3192, 3193, 3194, 3195, 3196:
		hasAmulet = true
	}
	if e.ClassMaster == classMasterMortal && e.Level >= 354 && !hasAmulet {
		amulet := int16(3193) // Elite dos Aventureiros — reino padrão/outros
		switch e.Clan {
		case 7:
			amulet = 3191 // Elite de Hekalotia
		case 8:
			amulet = 3192 // Elite de Akelonia
		}
		e.Equip[amuletSlot] = world.Item{Index: amulet}
		// O legado não atualiza o score aqui, mas o amuleto carrega bônus reais de
		// EF_AC/EF_HP (ItemList.csv) — recalcular para valerem
		// imediatamente em vez de esperar o próximo refresh de score não relacionado.
		d.refreshScore(e)
		d.sendSlot(w, s, world.ItemPlaceEquip, amuletSlot, e.Equip[amuletSlot])
		d.sendScore(w, s, e)
	}

	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	// +10 de Fama sem retorno visual nenhum: confirmar evita "usei o selo e nada".
	d.sendClientMessage(w, s, fameGainedText)
}

const celestialArchLevelReq = 355

// useIdealStone porta o bloco EF_VOLATILE 211 (_MSG_UseItem.cpp:3003-3181),
// alcançado clicando com o botão direito na Pedra_Ideal 5338 que o Rei funde das quatro
// Pedras Secretas (kingPedraIdeal). A issue #407 reverteu a #222, que havia ligado
// isto à 1742: aquela é a Pedra_da_Imortalidade, não tem EF_VOLATILE
// e só importa equipada em Equip[10] para a criação de Arch do Rei.
//
// Um Arch vira Celestial (:3101-3181, abaixo); um Celestial com um Sephirot
// vira Sub-Celestial (:3026-3085, createSubCelestial, issue #410). Qualquer outro
// tier recebe a ressincronização silenciosa de slot de :3005-3009, e as recusas de Arch
// (:3011-3024) também, exceto pelo texto de armadura — sem aviso em nenhuma delas (issue #413).
func (d *Dispatcher) useIdealStone(w *world.World, s *world.Session, e *world.Entity, src int) {
	switch e.ClassMaster {
	case classMasterArch:
	case classMasterCelestial:
		d.createSubCelestial(w, s, e, src)
		return
	default:
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	// :3011-3015 — abaixo do teto de nível o legado só ressincroniza o slot.
	if e.Level < celestialArchLevelReq {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	// :3017-3024 — armadura e o nível do Mortal de origem dividem uma recusa cujo
	// único texto é _NN_Cant_with_armor, e só quando Equip[1] está ocupado: um
	// MortalLevel baixo sem armadura é silencioso, e com os dois o texto de armadura vence.
	if !e.Equip[1].Empty() || e.MortalLevel < 99 {
		if !e.Equip[1].Empty() {
			d.sendClientMessage(w, s, cantWithArmor)
		}
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	staged := *e
	d.buildCelestialSnapshot(&staged, src)
	save := w.CharacterSaveFor(s, &staged)
	p := w.Persistence()
	s.Mode = world.UserWaitDB
	w.Go(s, func() func(*world.World, *world.Session) {
		err := p.SaveOnShutdown(context.Background(), save)
		return func(w *world.World, s *world.Session) {
			if err != nil {
				s.Mode = world.UserPlay
				d.log.Warn("celestial save failed", "conn", s.Conn, "name", e.Name, "err", err)
				d.notify(w, s, NoticeDBError)
				d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
				return
			}
			*e = staged
			d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
			d.sendSlot(w, s, world.ItemPlaceEquip, 1, e.Equip[1])
			d.sendSlot(w, s, world.ItemPlaceEquip, capeEquipSlot, e.Equip[capeEquipSlot])
			d.sendScore(w, s, e)
			d.sendEtc(w, s, e)
			d.returnPersistedCharacterToSelection(w, s, func(w *world.World, s *world.Session) {
				w.SendTo(s, protocol.Header{Type: protocol.MsgSendArchEffect, ID: protocol.IDScene}, protocol.EncodeStandardParm(int32(s.Slot)))
			})
			d.log.Info("celestial created", "conn", s.Conn, "name", e.Name)
		}
	})
}

func (d *Dispatcher) buildCelestialSnapshot(e *world.Entity, src int) {
	archLevel := e.Level
	e.CelestialArchLevel = celestialArchBand(archLevel)
	e.ClassMaster, e.Level, e.Exp = classMasterCelestial, 0, 0
	bases := [4][6]int32{{8, 4, 7, 6, 80, 45}, {5, 8, 5, 5, 60, 65}, {6, 6, 9, 5, 70, 55}, {8, 9, 13, 6, 75, 60}}
	class := int(e.Class)
	if class < 0 || class >= len(bases) {
		class = 0
	}
	b := bases[class]
	e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon = int16(b[0]), int16(b[1]), int16(b[2]), int16(b[3])
	e.Str, e.Int, e.Dex, e.Con = e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon
	e.BaseSpecial, e.Special = [4]int16{}, [4]int16{}
	e.BaseAC, e.BaseDamage = 230, 0
	e.BaseMaxHP, e.BaseMaxMP = b[4], b[5]
	e.HP, e.MaxHP, e.MP, e.MaxMP = b[4], b[4], b[5], b[5]
	e.ScoreBonus, e.SkillBonus, e.SpecialBonus = 0, 0, 855
	e.LearnedSkill, e.SecLearnedSkill = 1<<30, 0
	e.SkillBar = [4]uint8{}
	e.CelLv40, e.CelLv90, e.CelCircle = 0, 0, 0
	body := int16(3500)
	if e.ArchCrystalStage == 4 && archLevel >= 399 {
		body = 3502
	} else if e.ArchCrystalStage == 4 && archLevel >= 380 {
		body = 3501
	}
	e.Equip[1] = world.Item{Index: body}
	cape := int16(3199)
	switch e.Clan {
	case clanHekalotia:
		cape = 3197
	case clanAkelonia:
		cape = 3198
	}
	e.Equip[capeEquipSlot] = world.Item{Index: cape}
	e.Equip[0].Effects[1] = world.Effect{Effect: 98, Value: 3}
	e.Equip[0].Effects[2] = world.Effect{Effect: 106, Value: uint8(e.Equip[0].Index)}
	consumeOneItem(&e.Carry[src])
	d.refreshScore(e)
}

func celestialArchBand(level int32) uint8 {
	switch {
	case level < 370:
		return 1
	case level < 380:
		return 2
	case level < 398:
		return 3
	case level == 398:
		return 4
	default:
		return 5
	}
}

const (
	magicBeanBase      = 3407
	magicBeanRemover   = 10
	magicBeanPaintLo   = 116
	magicBeanPaintHi   = 125
	magicBeanFirstSlot = 1
	magicBeanLastSlot  = 7
)

// useMagicBean consome um Feijao Magico / Removedor de tintura (EF_VOLATILE 186)
// carimbando só o byte do id do efeito de destino, preservando o cValue exatamente
// como _MSG_UseItem.cpp:3767-3861 faz. Efeitos de tinta reusam os slots de efeito de sanc:
// 116..125 são cores, enquanto EF_SANC (43) é o marcador removedor/neutro.
func (d *Dispatcher) useMagicBean(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgUseItemBody, src int) {
	dstSlot := int(body.DestPos)
	if int(body.DestType) != world.ItemPlaceEquip || dstSlot < magicBeanFirstSlot || dstSlot > magicBeanLastSlot {
		d.magicBeanReject(w, s, e, src, NoticeOnlyToEquips)
		return
	}
	if magicBeanWeaponSlot(dstSlot) && s.AccessLevel < world.AccessModerator {
		d.magicBeanReject(w, s, e, src, NoticeCantUseHere)
		return
	}
	dst := d.itemSlot(w, s, e, int(body.DestType), dstSlot)
	if dst == nil {
		return
	}
	if dst.Empty() {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	if refine.Level(*dst) < 1 {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	color := int(e.Carry[src].Index) - magicBeanBase
	if color < 0 || color > magicBeanRemover {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	effect := uint8(magicBeanPaintLo + color)
	if color == magicBeanRemover {
		effect = efSanc
	}

	i := magicBeanEffectSlot(*dst, color == magicBeanRemover)
	if i < 0 {
		d.magicBeanReject(w, s, e, src, NoticeCantRefineMore)
		return
	}
	dst.Effects[i].Effect = effect

	d.notify(w, s, NoticeRefineSuccess)
	// refreshEquip recalcula e.EquipVisual/EquipAnct (os códigos de cor das peças
	// vestidas em cache) e os retransmite; só refreshScore os deixa velhos, então num
	// teleporte posterior createMobFrom reenviaria a cor pré-tinta (#157). Ele também
	// embute refreshScore + sendScore internamente.
	d.refreshEquip(w, s, e)
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, int(body.DestType), dstSlot, *dst)
	d.sendEquipVisual(w, s, e)
}

func (d *Dispatcher) magicBeanReject(w *world.World, s *world.Session, e *world.Entity, src int, n Notice) {
	d.notify(w, s, n)
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
}

func magicBeanWeaponSlot(slot int) bool {
	return slot == weaponSlotR || slot == weaponSlotL
}

func magicBeanEffectSlot(it world.Item, remover bool) int {
	for i, ef := range it.Effects {
		if magicBeanSlotWritable(ef, remover) {
			return i
		}
	}
	return -1
}

func magicBeanSlotWritable(ef world.Effect, remover bool) bool {
	if ef.Effect == 0 || ef.Effect >= magicBeanPaintLo && ef.Effect <= magicBeanPaintHi {
		return true
	}
	if !remover && ef.Effect == efSanc {
		return true
	}
	return false
}

// sendAffect empurra MSG_SendAffect (0x03B9): o snapshot completo dos 32 slots de buff, para o
// cliente renderizar ícones/timers. O Time é em ticks de 8 segundos reais para todo
// slot (affect1H=450, affect1D=10800); o Affect.Time guardado do Divino é só o
// sentinela "infinito" de exibição (divineAffectTime) — o prazo real é DivineEnd
// (wall clock), então o Time exibido é recalculado aqui como os segundos restantes
// ÷ 8. Enviar os segundos restantes crus (ou, depois de DivineEnd ter passado, deixar
// o sentinela cru de 2e9 passar sem conversão) mostrava uma duração
// muito inflada — issue #202.
func (d *Dispatcher) sendAffect(w *world.World, s *world.Session, e *world.Entity) {
	now := time.Now().Unix()
	var a [protocol.MaxAffect]protocol.AffectData
	for i := range e.Affect {
		af := e.Affect[i]
		if af.Type == 0 {
			continue
		}
		ad := protocol.AffectData{Type: af.Type, Value: af.Value, Level: af.Level, Time: af.Time}
		if af.Type == world.AffectDivine && af.Time >= affectInfiniteTime {
			ad.Time = 0
			if remaining := e.DivineEnd - now; remaining > 0 {
				ad.Time = uint32(remaining / 8)
			}
		}
		a[i] = ad
	}
	w.Send(s, protocol.MsgSendAffect, protocol.EncodeSendAffect(a))
}

// canEquipSlot informa se um item pode ser equipado no slot dst. nPos
// (STRUCT_ITEMLIST.nPos) é uma MÁSCARA dos slots em que um item cabe — body=1<<0, hat=1<<1,
// armadura 1<<2/1<<3, armas 1<<6/1<<7, montaria 1<<14 — confirmada contra o equipamento dos templates
// (Garra nPos 64=slot6, Shire 16384=slot14, itens de corpo nPos 1=slot0). Um consumível ou
// material tem nPos 0 e não cabe em lugar nenhum, então isto recusa uma poção caindo num slot
// de equipamento. Itens ausentes do catálogo são permitidos (equipamento legado/desconhecido, ex.: testes).
func (d *Dispatcher) canEquipSlot(idx int16, dst int) bool {
	if idx == 0 {
		return true // vazio/desequipar é sempre válido
	}
	pos, ok := d.itemPos[int(idx)]
	if !ok {
		return true
	}
	return pos != 0 && pos&(1<<uint(dst)) != 0
}

// meetsEquipReq informa se a entidade satisfaz o requisito de equipar de um item
// (nível + Str/Int/Dex/Con, STRUCT_ITEMLIST Req*). É checado contra o
// CurrentScore vivo (atributos incluindo outros equipamentos), como o original faz.
// Itens ausentes do catálogo de requisitos (ou sem requisito) sempre passam.
//
// O requisito só condiciona personagens MORTAIS (Basedef.cpp:5024-5028,
// BASE_CanEquip): para qualquer outro tier de ClassMaster (ARCH, CELESTIAL*) o código
// legado força Lvl/Str/Int/Dex/Con para 0 antes de comparar, um bypass total —
// não uma base de nível diferente. Personagens Arch criados via kingArch
// falhariam nessas checagens no nível baixo de recém-criados (issue #209).
func (d *Dispatcher) meetsEquipReq(e *world.Entity, it world.Item) bool {
	if e.ClassMaster != classMasterMortal {
		return true
	}
	r, ok := d.itemReqs[int(it.Index)]
	if !ok {
		return true
	}
	return e.Level >= int32(r.Lvl) &&
		e.Str >= r.Str && e.Int >= r.Int && e.Dex >= r.Dex && e.Con >= r.Con
}

// equipVisual deriva os 16 códigos visíveis de equipamento e os bytes de
// sobreposição de refino/ancestral dos itens equipados da entidade, casando com BASE_VisualItemCode e
// BASE_VisualAnctCode.
//
// Uma transformação BM ativa sobrescreve o slot 0 (a malha do corpo/rosto) com o
// modelo da fera — na LEITURA, diferente do legado que muta MOB.Equip[0].sIndex e
// precisa reiniciá-lo em todo recálculo (Basedef.cpp:4106/3908). Manter e.Equip
// intocado significa que o item de corpo persistido não pode ser corrompido e a reversão na
// expiração é só esta sobreposição deixar de atuar. O brilho normal do item é carregado
// por EquipAnct; o EF_SANC sintético específico da transformação de Basedef.cpp:4166
// fica deliberadamente de fora até aquela regra cosmética separada ser capturada.
func equipVisual(e *world.Entity) ([16]uint16, [16]uint8) {
	var v [16]uint16
	var a [16]uint8
	for i := range e.Equip {
		v[i], a[i] = protocol.VisualEquip(itemToSel(e.Equip[i]), i)
	}
	if value, _, ok := activeTransform(e); ok {
		v[0] = transMesh(value)
		a[0] = 0
		// Figurinos competem com a malha da fera no cliente (issue #320).
		// A faixa/slot de figurino legado está em ProcessSecMinTimer.cpp:601.
		// Esconder só o visual: o equipamento real precisa manter seus bônus e
		// reaparecer sozinho quando a transformação termina.
		if costume := e.Equip[12].Index; costume >= 4150 && costume <= 4188 {
			v[12], a[12] = 0, 0
		}
	}
	return v, a
}

func (d *Dispatcher) sendEquipVisual(w *world.World, s *world.Session, e *world.Entity) {
	e.EquipVisual, e.EquipAnct = equipVisual(e)
	body := protocol.EncodeUpdateEquip(e.EquipVisual, e.EquipAnct)
	h := protocol.Header{Type: protocol.MsgUpdateEquip, ID: uint16(s.Conn)}
	w.SendTo(s, h, body)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, h, body)
	})
}

// refreshEquip recalcula o equipamento visível da entidade e empurra _MSG_UpdateEquip para
// o cliente do próprio jogador E para todo jogador em visão, para um equipar/desequipar ser
// renderizado no modelo do personagem em todo lugar (SendFunc.cpp:SendEquip). HEADER.ID
// é o id da entidade para o cliente aplicar na criatura certa. Ele também reenvia o
// score, já que equipamento muda os atributos do personagem.
func (d *Dispatcher) refreshEquip(w *world.World, s *world.Session, e *world.Entity) {
	d.sendEquipVisual(w, s, e)
	d.refreshScore(e) // somar o AC/atributos/HP/MP do equipamento novo no CurrentScore
	d.sendScore(w, s, e)
}

// Bytes de tipo de efeito de item (ItemEffect.h) somados no CurrentScore.
const (
	efDamage     = 2
	efAc         = 3
	efHp         = 4
	efMp         = 5
	efStr        = 7
	efInt        = 8
	efDex        = 9
	efCon        = 10
	efSpecial1   = 11 // EF_SPECIAL1..4 → CurrentScore.Special[0..3]
	efSpecial2   = 12
	efSpecial3   = 13
	efSpecial4   = 14
	efSpecialAll = 74 // EF_SPECIALALL → CurrentScore.Special[1..3], "Aprendizagem de Skill"
	efWType      = 21 // EF_WTYPE: animação/tipo de arma, usado pelos gates RSV da Caçadora
	efCritical   = 42 // EF_CRITICAL: fonte de crítico; somado no equipamento e depois /4 (Basedef.cpp:3209)
	efSanc       = 43 // EF_SANC: nível de refino ("anc"/joias) do item — condiciona o limiar +9, não é atributo fixo
	efHpAdd      = 45 // EF_HPADD: bônus % de MaxHp (MaxHp*(HPADD+HPADD2+100)/100), captura §E
	efMpAdd      = 46 // EF_MPADD: bônus % de MaxMp
	efResist1    = 49 // EF_RESIST1..4: resistência/imunidade por tipo, veja itemResist
	efResist2    = 50
	efResist3    = 51
	efResist4    = 52
	efAcAdd      = 53 // EF_ACADD: AC extra — FIXO (somado com EF_AC), captura §E
	efResistAll  = 54 // EF_RESISTALL: se soma aos quatro EF_RESISTi — veja itemResist
	efMagic      = 60 // EF_MAGIC: ataque mágico fixo — excluído só no slot de arma da mão secundária (Basedef.cpp:2447)
	efDamageAdd  = 67 // EF_DAMAGEADD: dano fixo extra — só conta para joias (nUnique 41-50)
	efMagicAdd   = 68 // EF_MAGICADD: ataque mágico fixo extra — condicionado a joia como EF_DAMAGEADD (Basedef.cpp:1699-1703)
	efHpAdd2     = 69 // EF_HPADD2/EF_MPADD2: também entram no multiplicador HPADD%/MPADD%
	efMpAdd2     = 70
	efCritical2  = 71 // EF_CRITICAL2: crítico encantado — SUBSTITUI EF_CRITICAL no mesmo item
	efItemLevel  = 87
	efMobType    = 112
	efRunSpeed   = 29 // EF_RUNSPEED: bônus das botas no nibble de velocidade de movimento (baixo) de AttackRun
	efDamage2    = 73 // EF_DAMAGE2: dano encantado — SUBSTITUI EF_DAMAGE num item nPos 32

	// Ids de efeito que BASE_GetItemAbility deixa SEM ESCALA pelo multiplicador de refino
	// (Basedef.cpp:1854). São metadados de identidade/requisito do item, não atributos, então
	// refinar um item não pode movê-los. efWType, efItemLevel, efMobType acima e
	// efNoSanc/efIncubate/efIncuDelay (refine.go) pertencem à mesma lista.
	efLevel    = 1
	efPos      = 17
	efClass    = 18
	efReqStr   = 22
	efReqInt   = 23
	efReqDex   = 24
	efReqCon   = 25
	efRange    = 27
	efGrid     = 33
	efVolatile = 38
	efDonate   = 91
	efItemType = 113
	efNoTrade  = 127

	// baseAttackRun é o byte de velocidade base dos templates de classe (run<<4 | move) = 82
	// (run 5, move 2). UNVERIFIED: as curvas de velocidade por estado não são reproduzidas.
	baseAttackRun = 82
	// Política de movimento da issue #64: jogador a pé = 2, botas = 4, montaria = 6.
	bootMoveSpeed    = 4
	mountedMoveSpeed = 6

	// Mãos de arma em STRUCT_MOB.Equip (direita/esquerda). GetCurrentScore (CMob.cpp:756)
	// deriva WeaponDamage do EF_DAMAGE desses dois slots.
	weaponSlotR = 6
	weaponSlotL = 7
)

// itemSanc lê o nível de refino ("anc") de um item pelos efeitos de instância, 0..15
// (BASE_GetItemSanc). 0 = sem refino.
//
// O cValue NÃO é o nível: ele empacota o nível junto com o contador de pena,
// então precisa ser desempacotado. Veja o doc do pacote refine.
func itemSanc(it world.Item) int { return refine.Level(it) }

// Classes de slot de equipamento nPos que recebem bônus de limiar de refino (+9) (captura §E).
const (
	nPosWeapon1     = 64  // mão de arma → +40 WeaponDamage com sanc>=9
	nPosWeapon2     = 192 // arma de classe dupla → +40
	nPosDef1        = 4   // armadura/elmo/botas → +25 AC
	nPosDef2        = 8
	nPosDef3        = 128 // escudo → +25 AC
	nPosDamage2     = 32  // a classe de slot cujo EF_DAMAGE2 substitui EF_DAMAGE
	refineThreshold = 9
)

// itemBaseDamage é BASE_GetItemAbility(item, EF_DAMAGE) para uma mão de arma: o
// EF_DAMAGE de catálogo+instância, escalado pelo refino. captura §E ("o dano cru do item, que já
// cresce com o refino, vem dos stEffect/catálogo via BASE_GetItemAbility") descreve esse
// crescimento como vindo DESTE multiplicador — o limiar separado de +40 em weaponDamage fica
// em cima dele, não o substitui.
//
// EF_DAMAGE2 (Basedef.cpp:1711-1715) substitui EF_DAMAGE em itens nPos 32, a mesma
// forma ou-um-ou-outro do EF_CRITICAL2 de itemCritical.
func (d *Dispatcher) itemBaseDamage(it world.Item) int32 {
	if it.Empty() {
		return 0
	}
	want := uint8(efDamage)
	if d.itemPos[int(it.Index)] == nPosDamage2 &&
		(it.Effects[1].Effect == efDamage2 || it.Effects[2].Effect == efDamage2) {
		want = efDamage2
	}
	return d.itemAbilityRefined(it, want)
}

// forEachEffectID chama fn uma vez por id de efeito DISTINTO carregado por um item, ao longo de suas
// entradas de catálogo e de seus efeitos de instância. BASE_GetItemAbility é consultado por id de efeito
// e soma toda entrada que o carrega, então um chamador que soma um item no score precisa visitar
// cada id exatamente uma vez — visitar por entrada contaria em dobro e, pior, truncaria o
// multiplicador de refino uma vez por entrada em vez de uma vez por efeito.
func (d *Dispatcher) forEachEffectID(it world.Item, fn func(eff uint8)) {
	var seen [4]uint64 // bitset sobre o espaço 0..255 de ids de efeito
	visit := func(eff uint8) {
		if eff == 0 { // EF_NONE: um slot vazio, não um efeito
			return
		}
		bit := uint64(1) << (eff % 64)
		if seen[eff/64]&bit != 0 {
			return
		}
		seen[eff/64] |= bit
		fn(eff)
	}
	for _, be := range d.itemEffects[int(it.Index)] {
		visit(be.Eff)
	}
	for _, ef := range it.Effects {
		visit(ef.Effect)
	}
}

func (d *Dispatcher) itemAbility(it world.Item, effect uint8) int {
	if it.Empty() {
		return 0
	}
	var total int
	for _, be := range d.itemEffects[int(it.Index)] {
		if be.Eff == effect {
			total += int(be.Val)
		}
	}
	for _, ef := range it.Effects {
		if ef.Effect == effect {
			total += int(ef.Value)
		}
	}
	return total
}

// itemScoreSanc converte o nível real 0..15 informado pelo pacote refine
// em Go de volta para os valores REF_* de BASE_GetItemSanc usados na aritmética de score.
func itemScoreSanc(level int) int {
	if level <= 10 {
		return level
	}
	return [...]int{11: 12, 12: 15, 13: 18, 14: 22, 15: 27}[level]
}

// accessoryPosMask marca os quatro slots de acessório (Equip[8..11]) na máscara nPos
// do catálogo de um item — os slots em que a Pedra_Amunra equipa, e é por isso que o jogador usa quatro.
// BASE_GetItemAbility condiciona a promoção de refino +9 a ela (Basedef.cpp:1850).
const accessoryPosMask = 0xF00

// refineScaled informa se o multiplicador de refino se aplica a eff, ou seja, se ele está
// ausente da lista de isenção de BASE_GetItemAbility (Basedef.cpp:1854).
func refineScaled(eff uint8) bool {
	switch eff {
	case efGrid, efClass, efPos, efWType, efRange, efLevel,
		efReqStr, efReqInt, efReqDex, efReqCon,
		efVolatile, efIncubate, efIncuDelay,
		efMobType, efItemType, efItemLevel, efNoTrade, efNoSanc, efDonate:
		return false
	}
	return true
}

// refineFactor é o numerador do multiplicador de refino de BASE_GetItemAbility (Basedef.cpp:1849-1852):
// o sanc de score REF_* do item + 10, então +11..+15 usam seus valores não lineares legados.
// Acessórios (nPos & 0xF00) que chegam a +9 são promovidos a sanc 10 primeiro, o que é o que
// transforma o bônus +9 num x2 limpo — a regra por trás de uma Pedra_Amunra +9 conceder 200
// por atributo em vez de 100 (#282).
func (d *Dispatcher) refineFactor(it world.Item) int {
	sanc := itemScoreSanc(itemSanc(it))
	if sanc == refineThreshold && d.itemPos[int(it.Index)]&accessoryPosMask != 0 {
		sanc = 10
	}
	return sanc + 10
}

// itemAbilityRefined é BASE_GetItemAbility (Basedef.cpp:1687-1867): a soma de
// catálogo+instância de um efeito, depois o multiplicador de refino para todo efeito não isento. O legado
// multiplica a SOMA, não cada entrada, e a divisão trunca — então os chamadores precisam pedir
// um efeito inteiro de uma vez em vez de escalar entrada por entrada.
//
// EF_RUNSPEED carrega um limite pós-multiplicador próprio (Basedef.cpp:1860-1867): botas
// nunca somam mais que 2 no nibble de movimento, e um par +9 ganha um ponto extra.
func (d *Dispatcher) itemAbilityRefined(it world.Item, eff uint8) int32 {
	v := int32(d.itemAbility(it, eff))
	if v == 0 || !refineScaled(eff) {
		return v
	}
	factor := d.refineFactor(it)
	v = v * int32(factor) / 10
	if eff == efRunSpeed {
		if v >= 3 {
			v = 2
		}
		// O legado testa o sanc PROMOVIDO, então um acessório em +9 (promovido a 10)
		// perde esse ponto extra enquanto botas +9 o ganham.
		if v > 0 && factor == refineThreshold+10 {
			v++
		}
	}
	return v
}

// itemCritical é BASE_GetItemAbility(item, EF_CRITICAL) com a única regra legada que
// itemAbilityRefined não consegue expressar, porque é propriedade do item inteiro e
// não de um efeito: a substituição por EF_CRITICAL2 (Basedef.cpp:1705-1709). 42 é o
// id de consulta e 71 o id de armazenamento do crítico encantado, então um item com EF_CRITICAL2 em
// stEffect[1] ou [2] informa esse valor EM VEZ do seu EF_CRITICAL — ou um ou outro, nunca
// somados. Só os slots 1 e 2 o ativam; o slot 0 não.
//
// O multiplicador de refino vem de itemAbilityRefined e herda a decodificação do byte
// EF_SANC feita por itemSanc (a issue #103 está corrigindo essa decodificação); o crítico escala com o nível
// que ela informar, então melhora junto.
func (d *Dispatcher) itemCritical(it world.Item) int32 {
	want := uint8(efCritical)
	if it.Effects[1].Effect == efCritical2 || it.Effects[2].Effect == efCritical2 {
		want = efCritical2
	}
	return d.itemAbilityRefined(it, want)
}

// resistEffects mapeia o índice de resistência [0..3] para seu id EF_RESISTn (CMob.cpp:640-643 atribui
// MOB.Resist[0..3] a partir de EF_RESIST1..4 nessa ordem literal).
var resistEffects = [4]uint8{efResist1, efResist2, efResist3, efResist4}

// itemResist é BASE_GetItemAbility(item, EF_RESISTn) (Basedef.cpp:1830-1858): a soma fixa
// de catálogo+instância para o índice de resistência consultado, MAIS qualquer EF_RESISTALL somado do
// mesmo item (catálogo+instância), E DEPOIS o multiplicador de refino. As duas somas são somadas
// ANTES do multiplicador, como o legado faz, então a truncagem acontece uma vez — e é por isso que
// isto não pode simplesmente chamar itemAbilityRefined duas vezes (issue #211).
func (d *Dispatcher) itemResist(it world.Item, i int) int32 {
	if it.Empty() {
		return 0
	}
	v := int32(d.itemAbility(it, resistEffects[i])) + int32(d.itemAbility(it, efResistAll))
	if v == 0 {
		return 0
	}
	return v * int32(d.refineFactor(it)) / 10
}

// weaponDamage é o WeaponDamage de GetCurrentScore (CMob.cpp:756-789): a mão de arma
// mais forte com dano cheio mais a mais fraca pela metade (dupla empunhadura), mais um limiar
// de refino +40 por mão de arma com sanc>=9 (captura §E). É um campo SEPARADO de
// CurrentScore.Damage, somado no momento do acerto/exibição, então não está em e.Damage.
func (d *Dispatcher) weaponDamage(e *world.Entity) int32 {
	w1 := d.itemBaseDamage(e.Equip[weaponSlotR])
	w2 := d.itemBaseDamage(e.Equip[weaponSlotL])
	if w1 < w2 {
		w1, w2 = w2, w1
	}
	offhandDivisor := int32(2)
	if e.Class == 0 && e.LearnedSkill&(1<<9) != 0 {
		offhandDivisor = 1 // TK Mestre das Armas: a mão secundária contribui com EF_DAMAGE cheio.
	}
	if e.Class == 3 && e.LearnedSkill&(1<<10) != 0 {
		offhandDivisor = 1 // HT Pericia do Cacador: a mão secundária contribui com EF_DAMAGE cheio.
	}
	dmg := w1 + w2/offhandDivisor
	for _, slot := range [2]int{weaponSlotR, weaponSlotL} {
		it := e.Equip[slot]
		if !it.Empty() && itemSanc(it) >= refineThreshold {
			if pos := d.itemPos[int(it.Index)]; pos == nPosWeapon1 || pos == nPosWeapon2 {
				dmg += 40
			}
		}
	}
	return dmg
}

// equipBonus é a contribuição FIXA somada de todos os itens equipados no CurrentScore
// (efeitos base do catálogo + refinos/divinos de instância por item). O EF_DAMAGE das duas
// mãos de arma fica FORA (é o weaponDamage separado). Os efeitos percentuais
// EF_HPADD/EF_MPADD são acumulados separadamente (hpAddPct/mpAddPct) e aplicados na
// LEITURA, nunca embutidos aqui — então o score guardado fica fixo (captura §E).
type equipBonus struct {
	str, intel, dex, con int16
	special              [4]int16
	ac, damage           int32
	maxHP, maxMP         int32
	hpAddPct, mpAddPct   int32
	runSpeed             int32
	// criticalRaw é a soma de EF_CRITICAL antes do /4 sobre o equipamento
	// (BASE_GetMobAbility(EF_CRITICAL), Basedef.cpp:3209) — veja itemCritical.
	criticalRaw int32
	// magicRaw é a soma EF_MAGIC+EF_MAGICADD antes da escala (veja mountMagicScore) sobre TODO
	// o equipamento: a contribuição derivada de g_pMountBonus do slot de montaria (Equip[14], não
	// efeitos genéricos de item) MAIS o EF_MAGIC/EF_MAGICADD fixo do equipamento comum, somados pela
	// closure add abaixo — Basedef.cpp:3194-3195 soma os dois em cada slot antes de
	// aplicar a única escala (x+1)/4. parry/resist continuam broadcasts só de montaria
	// (Equip[14]); o dano já se soma no campo acima.
	magicRaw, parry, resist int32
	// itemResist é a soma por tipo de EF_RESIST1..4/EF_RESISTALL do equipamento comum
	// (veja itemResist), escalada pelo refino — distinta do broadcast fixo de montaria acima.
	itemResist [4]int32
}

func (d *Dispatcher) equipBonus(e *world.Entity) equipBonus {
	var b equipBonus
	// add soma um par efeito/valor no bônus. weaponSlot exclui o EF_DAMAGE da mão de
	// arma; dmgJewel condiciona EF_DAMAGEADD/EF_MAGICADD a itens nUnique 41-50;
	// offHand exclui EF_MAGIC só no slot de arma da mão secundária.
	add := func(eff uint8, val int32, weaponSlot, dmgJewel, offHand bool) {
		switch eff {
		case efStr:
			b.str += int16(val)
		case efInt:
			b.intel += int16(val)
		case efDex:
			b.dex += int16(val)
		case efCon:
			b.con += int16(val)
		case efSpecial1:
			b.special[0] += int16(val)
		case efSpecial2:
			b.special[1] += int16(val)
		case efSpecial3:
			b.special[2] += int16(val)
		case efSpecial4:
			b.special[3] += int16(val)
		case efSpecialAll:
			for i := 1; i <= 3; i++ {
				b.special[i] += int16(val)
			}
		case efAc, efAcAdd: // EF_AC + EF_ACADD, ambos somados direto no AC (captura §E)
			b.ac += val
		case efDamage:
			if !weaponSlot { // o dano da mão de arma é o WeaponDamage separado
				b.damage += val
			}
		case efDamageAdd:
			if dmgJewel { // só joias (nUnique 41-50) somam EF_DAMAGEADD
				b.damage += val
			}
		case efMagic:
			if !offHand { // BASE_GetMobAbility exclui só o slot de arma da mão secundária
				b.magicRaw += val
			}
		case efMagicAdd:
			if dmgJewel { // só joias (nUnique 41-50) somam EF_MAGICADD
				b.magicRaw += val
			}
		case efHp:
			b.maxHP += val
		case efMp:
			b.maxMP += val
		case efHpAdd, efHpAdd2:
			b.hpAddPct += val
		case efMpAdd, efMpAdd2:
			b.mpAddPct += val
		case efRunSpeed:
			b.runSpeed += val
		}
	}
	for slot := range e.Equip {
		it := e.Equip[slot]
		if it.Empty() {
			continue
		}
		// O slot de montaria tira seus atributos das tabelas g_pMountBonus, não de
		// efeitos genéricos de item (os Effects de uma montaria guardam HP/nível/alimentação, não atributos) —
		// espelhar o retorno antecipado de montaria de BASE_GetItemAbility e pular o laço de efeitos.
		if slot == mountEquipSlot {
			if mb, ok := mountBonusFor(it); ok {
				b.damage += mb.damage
				b.magicRaw += mb.magicRaw
				b.parry += mb.parry
				b.resist += mb.resist
			}
			continue
		}
		// O crítico é lido por item, não por efeito (a substituição EF_CRITICAL2 e
		// o multiplicador de refino são do item inteiro) — depois do retorno antecipado do slot de montaria,
		// o que casa com BASE_GetItemAbility não retornar crítico para montarias.
		b.criticalRaw += d.itemCritical(it)
		for i := range b.itemResist {
			b.itemResist[i] += d.itemResist(it, i)
		}
		weaponSlot := slot == weaponSlotR || slot == weaponSlotL
		offHand := slot == weaponSlotL
		nUnique := d.itemUnique[int(it.Index)]
		dmgJewel := nUnique >= 41 && nUnique <= 50
		// Um add por id de efeito DISTINTO, não por entrada: BASE_GetItemAbility soma as
		// entradas de catálogo e de instância de um efeito e só então aplica o multiplicador
		// de refino, então a divisão inteira precisa truncar a soma inteira uma vez.
		d.forEachEffectID(it, func(eff uint8) {
			add(eff, d.itemAbilityRefined(it, eff), weaponSlot, dmgJewel, offHand)
		})
		// Limiar de refino (+9): peças de defesa ganham +25 AC (o +40 das armas está em
		// weaponDamage). captura §E.
		if itemSanc(it) >= refineThreshold {
			switch d.itemPos[int(it.Index)] {
			case nPosDef1, nPosDef2, nPosDef3:
				b.ac += 25
			}
		}
	}
	return b
}

// deriveBaseScore captura o BaseScore sem equipamento no login. Os atributos
// persistidos e MaxHP/MaxMP recuperam sua base subtraindo o equipamento;
// os máximos salvos excluem recursos de CON/INT do equipamento (issue #321). refreshScore
// soma essas contribuições só de execução depois desta derivação histórica da base.
//
// AC e Damage são reconstruídos porque o contrato do banco os omite.
// WeaponDamage continua separado.
func (d *Dispatcher) deriveBaseScore(e *world.Entity) {
	b := d.equipBonus(e)
	e.BaseStr = e.Str - b.str
	e.BaseInt = e.Int - b.intel
	e.BaseDex = e.Dex - b.dex
	e.BaseCon = e.Con - b.con
	if e.BaseStr < 0 || e.BaseInt < 0 || e.BaseDex < 0 || e.BaseCon < 0 {
		// O CurrentScore guardado não casa com o bônus de equipamento de hoje: as
		// regras mudaram depois que ele foi salvo. Logado para uma divergência nunca ser silenciosa.
		d.log.Warn("derived base score is negative", "name", e.Name,
			"str", e.BaseStr, "int", e.BaseInt, "dex", e.BaseDex, "con", e.BaseCon)
	}
	e.BaseAC = playerBaseAC(e)
	e.BaseDamage = playerBaseDamage(e)
	e.BaseMaxHP = e.MaxHP - b.maxHP
	e.BaseMaxMP = e.MaxMP - b.maxMP
	// Magic/Parry/Resist não têm termo de BaseScore no legado. refreshScore os deriva
	// inteiramente do equipamento atual em vez de subtrair o valor do login.
}

// loadBaseScore pega o BaseScore como salvo (migration 0027) em vez de derivá-lo;
// o refreshScore do login então reconstrói o CurrentScore com as regras de equipamento
// de hoje, então uma mudança nessas regras pode mover o CurrentScore mas nunca a base que o
// jogador construiu.
func (d *Dispatcher) loadBaseScore(e *world.Entity, st world.CharacterState) {
	e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon = st.BaseStr, st.BaseInt, st.BaseDex, st.BaseCon
	e.BaseAC = playerBaseAC(e)
	e.BaseDamage = playerBaseDamage(e)
	e.BaseMaxHP, e.BaseMaxMP = st.BaseMaxHP, st.BaseMaxMP
}

// refreshScore recalcula o CurrentScore vivo = BaseScore + equipamento FIXO, após qualquer
// mudança de equipamento ou atributo, e faz cache dos bônus percentuais EF_HPADD/EF_MPADD. Os
// efeitos multiplicativos (HPADD%/MPADD% e os buffs Divino/Vigor) NÃO são embutidos aqui —
// são aplicados em camadas na leitura (effectiveMaxHP/MP, effectiveDamage), então o score
// guardado fica fixo e a derivação da base por subtração se mantém. HP/MP são limitados aos
// máximos vivos (efetivos).
func (d *Dispatcher) refreshScore(e *world.Entity) {
	b := d.equipBonus(e)
	e.Str = e.BaseStr + b.str
	e.Int = e.BaseInt + b.intel
	e.Dex = e.BaseDex + b.dex
	e.Con = e.BaseCon + b.con
	for i := range e.Special { // maestria alocada + equipamento (EF_SPECIAL1..4)
		e.Special[i] = e.BaseSpecial[i] + b.special[i]
	}
	flatAC := e.BaseAC + b.ac
	e.AC = flatAC + skillDerivedACBonus(e, flatAC)
	e.Damage = e.BaseDamage + b.damage + d.derivedDamageBeforeAffects(e)
	e.EquipmentAttributeHP, e.EquipmentAttributeMP = 0, 0
	if world.IsPlayer(e.ID) {
		// Basedef.cpp:3152-3156 converte CON/INT só de equipamento em 2 recursos
		// por ponto. Não copie o dobramento adicional do bloco inteiro do
		// máximo, nem conte de novo atributos alocados/buffados (issue #321).
		e.EquipmentAttributeHP = 2 * int32(b.con)
		e.EquipmentAttributeMP = 2 * int32(b.intel)
	}
	e.MaxHP = e.BaseMaxHP + b.maxHP + e.EquipmentAttributeHP
	e.MaxMP = e.BaseMaxMP + b.maxMP + e.EquipmentAttributeMP
	e.HpAddPct = b.hpAddPct
	e.MpAddPct = b.mpAddPct
	e.RunSpeedBonus = b.runSpeed
	// O crítico é derivado INTEIRAMENTE do equipamento em todo refresh — não há
	// BaseCritical a somar, e nenhuma subtração em deriveBaseScore, porque o legado também
	// não tem termo de base (Basedef.cpp:3209). O /4 divide a SOMA, não cada item. Criaturas
	// também o derivam do equipamento (BASE_GetCurrentScore roda para elas, CMob.cpp:709) e
	// não carregam Critical de template, então diferente do bloco de montaria abaixo isto não precisa de
	// guarda IsPlayer. O limite é reaplicado em effectiveCritical depois dos termos de skill/affect.
	// (O limite inferior é redundância: nenhuma linha de catálogo carrega crítico negativo, mas uma
	// conversão uint8 crua transformaria um em crítico quase garantido.)
	e.Critical = uint8(min(max(b.criticalRaw/4, 0), 255))
	// Magic recebe a escala (sum+1)/4 do equipamento mais o termo Dex+Int de classe/arma.
	// Magic, Parry (esquiva) e cada Resist são derivados INTEIRAMENTE do equipamento em
	// todo refresh, como o Critical acima: o legado não tem campos de BaseScore para eles.
	// Tratar um Magic persistido de valor zero como base cancelaria o bônus do equipamento
	// no login (issue #231). Resist é limitado
	// ao teto legado (CMob.cpp:640-643,692). Montarias vivem só no Equip[14] de um
	// jogador; criaturas carregam Magic/Parry/Resist direto do template
	// (world.SpawnMob) e nunca derivam um BaseScore, então recalculá-los aqui zeraria
	// esses valores de template — restrinja a jogadores.
	if world.IsPlayer(e.ID) {
		e.Magic = clampInt16(mountMagicScore(b.magicRaw) + d.classWeaponMagic(e))
		e.Parry = int(b.parry)
		for i := range e.Resist {
			e.Resist[i] = clampResist(int16(b.resist) + int16(b.itemResist[i]))
		}
	}
	d.applyAffectScore(e)

	e.EquipExpBonus = d.equipExpBonus(e)
	if isPlayerMob(e) {
		e.Damage += attributeDamageBonus(e, true)
	}
	if m := effectiveMaxHP(e); e.HP > m {
		e.HP = m
	}
	if m := effectiveMaxMP(e); e.MP > m {
		e.MP = m
	}
}

// affectMul devolve o multiplicador de buff (×100) em MaxHp/MaxMp dos buffs ativos:
// Divino (+20%) ou Vigor (+10%). 100 = sem buff (captura §C).
func affectMul(e *world.Entity) int32 {
	switch {
	case e.HasAffect(world.AffectDivine):
		return 120
	case e.HasAffect(world.AffectVigor):
		return 110
	}
	return 100
}

// effectiveMaxHP é o HP máximo real do jogador: (MaxHP fixo + deltas de affect) ×
// EF_HPADD% × buff. Aplicado na leitura (exibição/combate/regen), nunca guardado
// (captura §C,E).
func effectiveMaxHP(e *world.Entity) int32 {
	return (e.MaxHP + e.AffMaxHP) * (e.HpAddPct + 100) / 100 * affectMul(e) / 100
}

// effectiveMaxMP é o MP máximo real do jogador: (MaxMP fixo + deltas de affect) ×
// EF_MPADD% × buff.
func effectiveMaxMP(e *world.Entity) int32 {
	return (e.MaxMP + e.AffMaxMP) * (e.MpAddPct + 100) / 100 * affectMul(e) / 100
}

// effectiveDamage é o poder de ataque que cliente/combate veem: o CurrentScore.Damage fixo
// mais os deltas de affect (Buff Loop), escalado pela porcentagem DAMAGEMULTI (transformação BM;
// aplicada onde Basedef.cpp:4654 multiplica CurrentScore.Damage), aumentado +20% pelo
// buff Divino, mais o WeaponDamage separado (que nenhum dos multiplicadores toca — é um
// campo separado somado depois, captura §C). UNVERIFIED: o legado embute o Divino no
// DAMAGEMULTI de forma aditiva; compor os dois aqui diverge em alguns pontos quando ambos estão ativos.
func (d *Dispatcher) effectiveDamage(e *world.Entity) int32 {
	dmg := e.Damage + e.AffDamage
	if e.AffDamageMultiPct != 100 && e.AffDamageMultiPct > 0 {
		dmg = dmg * e.AffDamageMultiPct / 100
	}
	if e.HasAffect(world.AffectDivine) {
		dmg += dmg * 20 / 100
	}
	return dmg + d.weaponDamage(e)
}

// computeScore monta o CurrentScore que o cliente mostra. Efeitos multiplicativos
// (EF_HPADD%/MPADD% e os buffs Divino/Vigor) e o dano de arma separado são
// embutidos aqui pelos getters efetivos; o incremento de velocidade da montaria também.
func (d *Dispatcher) computeScore(e *world.Entity) protocol.ScoreData {
	var special [4]int16
	for i := range special {
		special[i] = int16(effectiveSpecial(e, i))
	}
	sc := protocol.ScoreData{
		Level: e.Level, Ac: effectiveAC(e), Damage: d.effectiveDamage(e),
		MaxHp: effectiveMaxHP(e), Hp: e.HP, MaxMp: effectiveMaxMP(e), Mp: e.MP,
		Str: effectiveStr(e), Int: effectiveInt(e), Dex: effectiveDex(e), Con: e.Con + e.AffCon,
		Special:    special,
		AttackRun:  attackRunOf(e),
		Critical:   effectiveCritical(e),
		SaveMana:   uint8(e.SaveMana),
		Guild:      e.Guild,
		GuildLevel: uint16(e.GuildLevel),
		Magic:      effectiveMagic(e),
	}
	// Vetor de ícones de buff (SendScore → GetAffect): o cliente renderiza/atualiza a
	// barra de buff a partir disto, então todo push de score mantém os ícones em sincronia.
	for i := range e.Affect {
		if e.Affect[i].Type == 0 {
			continue
		}
		sc.Affect[i] = protocol.PackAffect(protocol.AffectData{
			Type: e.Affect[i].Type, Time: e.Affect[i].Time,
		})
	}
	for i := range sc.Resist {
		sc.Resist[i] = int8(effectiveResist(e, i))
	}
	return sc
}

// attackRunOf é o byte de velocidade vivo da entidade (run<<4 | move). Jogadores recebem os
// tiers de velocidade de movimento da issue #64 (a pé=2, botas=4, montaria=6); criaturas carregam o
// valor de CurrentScore do template (definido no spawn). Todo score/snapshot S→C
// (UpdateScore, CreateMob, o blob MOB do login) precisa carregá-lo: o cliente anima as
// caminhadas — as próprias e as de entidades remotas — nessa velocidade, e um 0 aqui renderiza um
// avatar rastejante e elástico.
func attackRunOf(e *world.Entity) uint8 {
	if !world.IsPlayer(e.ID) {
		return e.AttackRun
	}
	attack := int32(baseAttackRun>>4)*10 + e.AffAttackSpeed
	if attack < 0 {
		attack = 0
	}
	if attack > 150 {
		attack = 150
	}
	attack /= 10
	// Uma montaria tem precedência sobre botas e usa o tier máximo de movimento do cliente.
	if !e.Equip[mountEquipSlot].Empty() {
		return uint8(attack<<4) | mountedMoveSpeed
	}
	move := int32(baseAttackRun & 0x0F)
	if e.RunSpeedBonus > 0 {
		move = bootMoveSpeed
	}
	move += e.AffRunSpeed
	if move < 1 {
		move = 1
	}
	if move > 6 {
		move = 6
	}
	return uint8(attack<<4) | uint8(move)
}

// sendScore publica o CurrentScore recalculado (_MSG_UpdateScore) para o jogador
// E para todos em visão, espelhando o SendScore legado — cuja única saída é
// GridMulticast(TargetX, TargetY, ..., skip=0), ou seja, ele mesmo mais a janela
// de visão inteira (SendFunc.cpp:1298; todos os 113 pontos de chamada de SendScore passam por ali).
//
// O multicast é o que ressincroniza uma mudança de HP fora de banda (poção, regen,
// cura, ressurreição) nos OUTROS clientes: o MSG_Attack carrega só deltas de STRUCT_DAM que
// o cliente SUBTRAI da própria cópia do HP do alvo, então sem um push
// explícito um atacante renderiza uma barra de HP velha para quem bebeu (issue #99).
//
// HEADER.ID é o conn do SUJEITO para todo destinatário — é assim que o cliente sabe
// qual entidade o score descreve. Nada aqui é privado: ouro/exp/pontos livres
// ficam em MSG_UpdateEtc (sendEtc), que continua unicast como o SendEtc legado.
func (d *Dispatcher) sendScore(w *world.World, s *world.Session, e *world.Entity) {
	body := protocol.EncodeUpdateScore(d.computeScore(e))
	w.SendTo(s, protocol.Header{Type: protocol.MsgUpdateScore, ID: uint16(s.Conn)}, body)
	w.BroadcastInView(s.Conn, protocol.MsgUpdateScore, body) // exclui a origem
}

// sendScoreSelf é o push unicast de score, para o único caminho em que o sujeito ainda
// não é visível aos jogadores ao redor — enterWorldView empurra o score ANTES
// de transmitir o CreateMob do recém-chegado, e o legado não tem SendScore ali
// (ProcessDBMessage.cpp:1017-1037 envia o blob de login por unicast e depois faz GridMulticast
// do CreateMob, que já carrega Hp/MaxHp para os observadores).
func (d *Dispatcher) sendScoreSelf(w *world.World, s *world.Session, e *world.Entity) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgUpdateScore, ID: uint16(s.Conn)}, protocol.EncodeUpdateScore(d.computeScore(e)))
}

// sendEtc empurra o MSG_UpdateEtc do jogador (SendFunc.cpp SendEtc): ouro, exp e —
// crucialmente — os pontos livres (ScoreBonus/SpecialBonus/SkillBonus) e a
// máscara de skills aprendidas (Learn), que é o que popula a janela de skills do cliente.
// STRUCT_SCORE/UpdateScore NÃO carrega esses, então este pacote é o único caminho
// de atualização. É a struct inteira (não só moeda) porque o original
// sempre envia todos os campos; uma atualização parcial zeraria o estado do cliente.
// Hold ainda não é modelado (0).
func (d *Dispatcher) sendEtc(w *world.World, s *world.Session, e *world.Entity) {
	w.Send(s, protocol.MsgUpdateEtc, protocol.EncodeUpdateEtc(protocol.UpdateEtcData{
		Exp:          e.Exp,
		Learn:        int64(e.LearnedSkill),
		ScoreBonus:   e.ScoreBonus,
		SpecialBonus: e.SpecialBonus,
		SkillBonus:   e.SkillBonus,
		Magic:        uint16(e.Magic),
		Coin:         e.Coin,
	}))
}

// tradingItem trata _MSG_TradingItem (0x0376): a troca universal de item por
// arrastar e soltar do cliente entre dois slots — dentro do inventário, entre
// inventário e equipamento, e de/para o armazém da conta (cargo). Apesar do
// nome "Trading" isto NÃO é a troca P2P entre jogadores (essa é _MSG_Trade, 0x0383); é
// a troca de slot que o cliente envia sempre que um item é arrastado
// (Source/Code/TMSrv/_MSG_TradingItem.cpp). Mover um item durante uma troca P2P
// cancela a troca (anti-dup).
//
// A troca permuta o conteúdo dos dois slots (então arrastar sobre um slot ocupado
// os troca; sobre um slot vazio move). Roda na goroutine única do loop, então
// trocas concorrentes não podem duplicar um item.
func (d *Dispatcher) tradingItem(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 1, 19)
		return
	}
	if s.Trade.Active {
		d.removeTrade(w, s) // mover um item no meio de uma troca a cancela
		return
	}
	if s.TradeMode != 0 {
		d.notify(w, s, NoticeCantAutoTrade)
		return
	}
	var body protocol.MsgTradingItemBody
	if err := body.Decode(payload); err != nil {
		return
	}
	srcPlace, srcSlot := int(body.SrcPlace), int(body.SrcSlot)
	dstPlace, dstSlot := int(body.DestPlace), int(body.DestSlot)

	// O cargo é compartilhado pela conta e só acessível perto do NPC guarda do cargo
	// (WarpID o identifica). Moves só de inventário/equipamento pulam este gate.
	if (srcPlace == world.ItemPlaceCargo || dstPlace == world.ItemPlaceCargo) && !d.nearCargoGuard(w, e, int(body.WarpID)) {
		return
	}

	src := d.itemSlot(w, s, e, srcPlace, srcSlot)
	dst := d.itemSlot(w, s, e, dstPlace, dstSlot)
	if src == nil || dst == nil {
		return
	}
	if srcPlace == dstPlace && srcSlot == dstSlot {
		return
	}
	if src.Empty() && dst.Empty() {
		return // nada a mover
	}
	// Regras de equipamento: o item que cairia num slot de equipamento precisa caber nele (nPos)
	// E cumprir o requisito de nível/atributo. Numa troca o item de origem vai para o
	// slot de destino (e vice-versa).
	if dstPlace == world.ItemPlaceEquip && !src.Empty() && (!d.canEquipSlot(src.Index, dstSlot) || !d.meetsEquipReq(e, *src)) ||
		srcPlace == world.ItemPlaceEquip && !dst.Empty() && (!d.canEquipSlot(dst.Index, srcSlot) || !d.meetsEquipReq(e, *dst)) {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	if srcPlace == world.ItemPlaceCarry && dstPlace == world.ItemPlaceCarry && tryMergeItemStacks(src, dst) {
		w.Send(s, protocol.MsgTradingItem, payload) // devolver o move
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(srcPlace, srcSlot, itemToSel(*src)))
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(dstPlace, dstSlot, itemToSel(*dst)))
		return
	}
	*src, *dst = *dst, *src
	w.Send(s, protocol.MsgTradingItem, payload) // devolver o move
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(srcPlace, srcSlot, itemToSel(*src)))
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(dstPlace, dstSlot, itemToSel(*dst)))
	d.shiftWeaponToRightHand(w, s, e)
	// Um equipar/desequipar muda o equipamento renderizado: atualize o modelo em todo lugar.
	if srcPlace == world.ItemPlaceEquip || dstPlace == world.ItemPlaceEquip {
		d.refreshEquip(w, s, e)
	}
	if (srcPlace == world.ItemPlaceEquip && srcSlot == mountEquipSlot) || (dstPlace == world.ItemPlaceEquip && dstSlot == mountEquipSlot) {
		d.refreshBabyMountSummon(w, s, e)
	}
}

// shiftWeaponToRightHand espelha _MSG_TradingItem.cpp:394-414: depois de qualquer
// troca de item, se a mão de arma principal (slot 6) fica vazia enquanto
// a mão secundária (slot 7) tem um item que não é escudo, o servidor o move à força
// para o slot 6 e devolve uma troca sintética para o cliente ficar em sincronia. Isto
// é o que faz o equipar rápido (clique duplo) de uma arma de uma mão cair na
// mão principal em vez do slot de escudo, já que o próprio cliente
// escolhe o slot de destino e às vezes mira direto no slot 7.
func (d *Dispatcher) shiftWeaponToRightHand(w *world.World, s *world.Session, e *world.Entity) {
	if !e.Equip[weaponSlotR].Empty() {
		return
	}
	off := &e.Equip[weaponSlotL]
	if off.Empty() {
		return
	}
	if pos, ok := d.itemPos[int(off.Index)]; ok && pos == nPosDef3 {
		return // um escudo de verdade fica na mão secundária
	}
	e.Equip[weaponSlotR], *off = *off, world.Item{}
	shiftBody := protocol.MsgTradingItemBody{
		DestPlace: world.ItemPlaceEquip, DestSlot: weaponSlotR,
		SrcPlace: world.ItemPlaceEquip, SrcSlot: weaponSlotL,
	}
	w.Send(s, protocol.MsgTradingItem, shiftBody.Encode())
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(world.ItemPlaceEquip, weaponSlotR, itemToSel(e.Equip[weaponSlotR])))
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(world.ItemPlaceEquip, weaponSlotL, itemToSel(e.Equip[weaponSlotL])))
}

// itemSlot devolve um ponteiro para o slot de item vivo de um par lugar/slot, ou nil
// se o lugar é desconhecido ou o slot está fora dos limites. Moves de carry são limitados
// pela faixa de Carry desbloqueada no momento. O slot de cargo é nil a menos que o
// armazém da conta esteja carregado.
func (d *Dispatcher) itemSlot(w *world.World, s *world.Session, e *world.Entity, place, slot int) *world.Item {
	switch place {
	case world.ItemPlaceEquip:
		if slot < 0 || slot >= world.MaxEquip {
			return nil
		}
		return &e.Equip[slot]
	case world.ItemPlaceCarry:
		if !carrySlotAccessible(e, slot) {
			return nil
		}
		return &e.Carry[slot]
	case world.ItemPlaceCargo:
		cargo := w.Cargo(s.AccountID)
		if cargo == nil || slot < 0 || slot >= world.MaxCargo {
			return nil
		}
		return &cargo.Items[slot]
	}
	return nil
}

// nearCargoGuard informa se warpID é um NPC guarda do cargo (Merchant==2) em
// visão do jogador — o gate de proximidade para qualquer acesso a slot de cargo.
func (d *Dispatcher) nearCargoGuard(w *world.World, e *world.Entity, warpID int) bool {
	npc := w.Entity(warpID)
	if npc == nil || npc.Mode == world.MobEmpty || npc.Merchant != 2 {
		return false
	}
	return abs(int(e.X)-int(npc.X)) <= world.ViewRange && abs(int(e.Y)-int(npc.Y)) <= world.ViewRange
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// slotPayload é um corpo S→C provisório carregando um índice de slot afetado.
// UNVERIFIED: layouts reais de _MSG_CNFDropItem/_MSG_CNFGetItem (adiados até a captura).
func slotPayload(slot int) []byte { return uint32Payload(uint32(slot)) }

func uint32Payload(v uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return b[:]
}
