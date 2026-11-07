package handler

import (
	"openwyd-nextgen/tmserver/internal/combine"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// odinRate é g_pOdinRate[12] (Basedef.cpp:95-99), indexado pelo id da receita.
var odinRate = [12]int{100, 35, 100, 40, 100, 100, 100, 100, 100, 100, 100, 100}

// índices do item resultante das receitas Odin (_MSG_CombineItemOdin.cpp). A receita
// Capa Celestial mira o mesmo slot de capa que reinoCapeSlot (chat.go).
const (
	odinPistaDeRunasResult     = 5134
	odinPedraDaFuriaResult     = 3020
	odinSecretaBase            = 5334 // água=5334, terra=+1, sol=+2, vento=+3
	odinSementeDeCristalResult = 4032
)

// combineItemOdin trata _MSG_CombineItemOdin (0x02D2): a família Odin de 11
// receitas (Composição de Sets / de armas, o nível "+12+" de refino +11→+15,
// Pista de runas, Destrave Lv40, Pedra da fúria, as 4 pedras Secreta,
// Semente de cristal, Capa celestial). Este é um handler dedicado, não parte
// do motor genérico CombineFamily (combine.go): ao contrário de Anct/Ehre/…, estas
// receitas não compartilham o formato "taxa entra, um resultado no slot 0" — os resultados entram
// em slots variados, uma receita muda equipamento equipado direto (Equip[15]),
// e uma define uma flag de quest sem item nenhum.
//
// combine.MatchOdin devolver combine.OdinNoMatch (0) NÃO é "inválido" do
// jeito que uma taxa 0 é para toda outra família de combine: o retorno padrão de
// GetMatchCombineOdin É "Composição de Sets", uma receita real (quase sempre bem-sucedida)
// ligada ao campo Extra do catálogo de Item[0] — o Exec sempre a tenta.
// Não há recusa de "nenhuma receita casou" em lugar nenhum do handler legado.
//
// Duas divergências deliberadas da origem (plano da issue #138): os reseeds de
// srand() por relógio espalhados pelo _MSG_CombineItemOdin.cpp são descartados —
// sorteia direto do w.Rand(), como todo outro caminho de combine/refino; e o
// ramo de falha da receita "+12+", byte a byte idêntico ao seu ramo de sucesso no
// código vivo (não comentado), é portado como ele de fato roda: sempre tem sucesso
// assim que as pré-checagens abaixo passam.
func (d *Dispatcher) combineItemOdin(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	d.combineOdin(w, s, e, payload)
}

// combineOdin é o corpo de combineItemOdin, separado (como useItem/useClasseItem)
// para testes poderem conduzi-lo com uma entidade explícita em vez de precisar de uma
// registrada na tabela de conexões do mundo.
func (d *Dispatcher) combineOdin(w *world.World, s *world.Session, e *world.Entity, payload []byte) {
	var body protocol.MsgCombineItemBody
	if err := body.Decode(payload); err != nil {
		return
	}

	items, slots, active, ok := d.resolveComboInputs(w, s, e, body)
	if !ok {
		return
	}

	id := combine.MatchOdin(d.odinCatalog, items[:])

	// Pré-gates específicos de receita (_MSG_CombineItemOdin.cpp:48-90): falham aqui
	// SEM consumir nada, ao contrário de todo outro desfecho abaixo. As duas
	// locais abaixo carregam um valor que cada gate já teve de obter até seu
	// ramo correspondente mais adiante, em vez de recalculá-lo lá.
	//
	// Divergência da origem, não entre as três sinalizadas no plano mas no mesmo
	// espírito: o id 0 (Composição de Sets, o padrão pega-tudo de "nada mais casou")
	// é a única receita cujo padrão não impõe restrição nenhuma a
	// Item[1]. A origem ainda escreve cegamente em Carry[m->InvenPos[1]]
	// mesmo quando Item[1] estava vazio e seu InvenPos nunca foi validado contra
	// nada que o jogador realmente ofereceu — um risco de sobrescrever slot arbitrário
	// que o invariante de dono único/sem duplicação deste fork (CLAUDE.md) não pode
	// permitir. Exigimos que o slot 1 tenha sido uma entrada real e validada.
	if (id == combine.OdinNoMatch || id == combine.OdinItemCelestial) && items[1].Empty() {
		sendCombineComplete(w, s, combineInvalid)
		return
	}
	var targetLevel, capeLevel int
	switch id {
	case combine.OdinPlus12:
		targetLevel = refine.Level(items[2])
		if odinSecretaUneven(items[:]) || targetLevel >= 15 || d.itemAbility(items[2], efMobType) == 3 {
			sendCombineComplete(w, s, combineInvalid)
			return
		}
	case combine.OdinDestraveLv40:
		if e.Level != 39 || e.CelLv40 != 0 || e.ClassMaster != classMasterCelestial {
			sendCombineComplete(w, s, combineInvalid)
			return
		}
	case combine.OdinCapaCelestial:
		capeLevel = itemSanc(e.Equip[reinoCapeSlot])
		if e.ClassMaster == classMasterMortal || e.ClassMaster == classMasterArch || capeLevel >= 9 {
			sendCombineComplete(w, s, combineInvalid)
			return
		}
	}

	// Consome todo slot de entrada ativo incondicionalmente — os insumos são
	// gastos quer a receita abaixo acabe produzindo algo ou não
	// (_MSG_CombineItemOdin.cpp:93-100), a mesma regra de "gasto de todo jeito" que o
	// refino por pó e o combine Anct já seguem.
	for _, i := range active {
		e.Carry[slots[i]] = world.Item{}
		sendCarrySlot(w, s, e, slots[i])
	}

	roll := combine.RollOdin(w.Rand())

	switch id {
	case combine.OdinNoMatch, combine.OdinItemCelestial:
		d.odinComposicao(w, s, e, items[0], items[1], slots[1], id, roll)
	case combine.OdinPlus12:
		d.odinPlus12(w, s, e, items[:], slots[2], targetLevel)
	case combine.OdinPistaDeRunas:
		d.odinFreshResult(w, s, e, slots[0], odinPistaDeRunasResult, roll, id)
	case combine.OdinDestraveLv40:
		d.odinDestraveLv40(w, s, e, roll)
	case combine.OdinPedraDaFuria:
		d.odinFreshResult(w, s, e, slots[0], odinPedraDaFuriaResult, roll, id)
	case combine.OdinSecretaAgua, combine.OdinSecretaTerra, combine.OdinSecretaSol, combine.OdinSecretaVento:
		d.odinFreshResult(w, s, e, slots[0], int16(odinSecretaBase+(id-combine.OdinSecretaAgua)), roll, id)
	case combine.OdinSementeDeCristal:
		d.odinFreshResult(w, s, e, slots[0], odinSementeDeCristalResult, roll, id)
	case combine.OdinCapaCelestial:
		d.odinCapaCelestial(w, s, e, roll, capeLevel)
	}
}

// odinSecretaUneven é a checagem cruzada das quatro pedras elementais da receita "+12+"
// (_MSG_CombineItemOdin.cpp:50-71): entre os 8 slots, as quatro pedras Secreta
// (água/terra/sol/vento, 5334-5337) precisam estar todas ausentes ou todas
// presentes — uma, duas ou três presentes sozinhas é recusado.
func odinSecretaUneven(items []world.Item) bool {
	var agua, terra, sol, vento bool
	for _, it := range items {
		switch it.Index {
		case 5334:
			agua = true
		case 5335:
			terra = true
		case 5336:
			sol = true
		case 5337:
			vento = true
		}
	}
	some := agua || terra || sol || vento
	all := agua && terra && sol && vento
	return some && !all
}

// odinComposicao é o id 0 ("Composição de Sets") e o id 1 ("Composição de
// armas") — os dois compartilham uma forma idêntica na origem, diferindo só na
// taxa de sucesso. No sucesso, item[1] é renomeado para o Extra do catalisador
// e seu sanc volta a 0; na falha item[1] é restaurado sem mudança. De um
// jeito ou de outro item[0] (o catalisador/selado) continua consumido.
func (d *Dispatcher) odinComposicao(w *world.World, s *world.Session, e *world.Entity, catalyst, base world.Item, baseSlot int, id, roll int) {
	rate := odinRate[id] + w.Rand().Intn(5)
	if roll > rate {
		e.Carry[baseSlot] = base
		sendCarrySlot(w, s, e, baseSlot)
		sendCombineComplete(w, s, combineFailed)
		return
	}
	result := base
	if extra := d.odinCatalog.Extra[int(catalyst.Index)]; extra > 0 {
		result.Index = int16(extra)
	}
	refine.Set(&result, 0, 0)
	e.Carry[baseSlot] = result
	sendCarrySlot(w, s, e, baseSlot)
	sendCombineComplete(w, s, combineSuccess)
}

// odinPlus12 é o id 2, o nível de refino "+11→+15". Conforme o plano da issue #138
// (o ramo de falha vivo é uma cópia byte a byte do ramo de sucesso,
// com a falha real condicionada à taxa toda comentada na origem), este
// sempre tem sucesso depois que o pré-gate de quem chama passou: a taxa acumulada
// ainda é calculada e logada (útil para ajuste de economia futuro) mas
// não condiciona o desfecho. level é o nível de refino atual de items[2], já
// calculado pelo pré-gate de quem chama.
func (d *Dispatcher) odinPlus12(w *world.World, s *world.Session, e *world.Entity, items []world.Item, targetSlot, level int) {
	rate, protect := combine.Plus12Rate(odinRate[combine.OdinPlus12], items)
	newLevel, ok := combine.Plus12NewLevel(level)
	if !ok {
		newLevel = level
	}
	result := items[2]
	refine.Set(&result, newLevel, refine.Gem(items[2]))
	e.Carry[targetSlot] = result
	sendCarrySlot(w, s, e, targetSlot)
	sendCombineComplete(w, s, combineSuccess)
	d.log.Info("odin +12+ refine", "conn", s.Conn, "from", level, "to", newLevel, "rate", rate, "protect", protect)
}

// odinFreshResult cobre as receitas cujo resultado é um item puro carimbado
// direto num slot já zerado (Pista de runas, Pedra da fúria, as
// 4 pedras Secreta, Semente de cristal): a origem atribui só .sIndex no
// slot que o laço de consumo já zerou, então — ao contrário de odinComposicao/odinPlus12
// — nenhum efeito/quantidade anterior sobrevive. Na falha o slot fica vazio; nenhuma
// destas receitas restaura nada (_MSG_CombineItemOdin.cpp:517-650).
func (d *Dispatcher) odinFreshResult(w *world.World, s *world.Session, e *world.Entity, slot int, result int16, roll, id int) {
	if roll > odinRate[id] {
		sendCombineComplete(w, s, combineFailed)
		return
	}
	e.Carry[slot] = world.Item{Index: result}
	sendCarrySlot(w, s, e, slot)
	sendCombineComplete(w, s, combineSuccess)
}

// odinDestraveLv40 é o id 4: nenhum item é produzido, só o gate de nível 40
// Celestial vira. Reusa destravarCelestial (chat.go), a mesma flag que /destravar40
// já define — o pré-gate de quem chama já confirmou Level==39,
// CelLv40==0 e ClassMaster==Celestial.
func (d *Dispatcher) odinDestraveLv40(w *world.World, s *world.Session, e *world.Entity, roll int) {
	if roll > odinRate[combine.OdinDestraveLv40] {
		sendCombineComplete(w, s, combineFailed)
		return
	}
	d.destravarCelestialFor(w, s, e, false) // envia seu próprio MsgCombineComplete + persiste
}

// odinCapaCelestial é o id 11: nenhum slot de entrada de combine é tocado aqui (eles
// já foram consumidos acima) — a capa equipada (Equip[15]) tem seu sanc
// aumentado em um, inicializando antes um par EF_SANC novo se ela ainda não tiver nenhum.
// O pré-gate de quem chama já confirmou os requisitos de classe/sanc da capa e
// calculou level, o nível de refino atual da capa.
func (d *Dispatcher) odinCapaCelestial(w *world.World, s *world.Session, e *world.Entity, roll, level int) {
	if roll > odinRate[combine.OdinCapaCelestial] {
		sendCombineComplete(w, s, combineFailed)
		return
	}
	cape := &e.Equip[reinoCapeSlot]
	if !refine.Bootstrap(cape) {
		sendCombineComplete(w, s, combineFailed)
		return
	}
	refine.Set(cape, level+1, 0)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceEquip, reinoCapeSlot, itemToSel(*cape)))
	sendCombineComplete(w, s, combineSuccess)
}
