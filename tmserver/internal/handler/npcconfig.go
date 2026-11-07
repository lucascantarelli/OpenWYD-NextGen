package handler

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"openwyd-nextgen/tmserver/internal/npccfg"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Overlay de NPC editado pelo moderador (npc-editing-plan.md). A web escreve config fria
// no Postgres; o tmServer a lê (via dbServer) e MATERIALIZA as entidades
// vivas dentro do loop de dono único — a web nunca toca estado vivo.
//
// O banco é a fonte única dos blocos de spawn (issue 57): o tmServer não
// analisa mais o NPCGener.txt. No boot ele registra um slot de gerador por definição de
// conteúdo e materializa as entidades do mesmo snapshot, então o arquivo e
// a tabela não podem mais discordar. Blocos de propriedade de evento (salas de água, Kefra,
// Nightmare, quest de runas) são registrados mas nunca materializados aqui — o código de evento
// deles os gera.

const (
	// npcPollPeriod é quantos ticks de IA de criaturas (1s cada) entre polls de versão
	// de config. Edições de moderador são raras, então ~15s mantém a carga no banco desprezível
	// e ainda parece ao vivo.
	npcPollPeriod = 15
	// npcFetchTimeout limita as chamadas gRPC de versão/snapshot fora do loop.
	npcFetchTimeout = 5 * time.Second
)

// Orçamento de retry do boot. Vars, não consts, para testes poderem encolhê-los.
var (
	// npcBootRetryBudget limita quanto tempo o boot espera o dbServer ficar
	// alcançável e terminar de reconciliar o catálogo de conteúdo. Os dois serviços são
	// implantados juntos e o dbServer semeia o catálogo ANTES de escutar, então
	// o tmServer legitimamente ganha a corrida; sem essa espera o deploy
	// entra em crash-loop (incidente de produção, 2026-08-13).
	npcBootRetryBudget = 90 * time.Second
	// npcBootRetryInitial e npcBootRetryMax delimitam o backoff (que dobra a cada tentativa).
	npcBootRetryInitial = 1 * time.Second
	npcBootRetryMax     = 5 * time.Second
)

// FetchNPCConfig carrega o catálogo de spawn antes de o loop começar. Falhas
// transitórias são repetidas até o npcBootRetryBudget expirar — o dbServer pode ainda estar
// inalcançável, ou ainda semeando e a responder com catálogo vazio (ele semeia
// o catálogo inteiro ANTES de escutar: incidente de produção, 2026-08-13). Dados
// corrompidos (índice de gerador duplicado) nunca se curam sozinhos, então isto falha na hora
// em vez de queimar o orçamento.
func (d *Dispatcher) FetchNPCConfig(ctx context.Context) (npccfg.Snapshot, error) {
	if d.npcSource == nil {
		return npccfg.Snapshot{}, nil
	}
	deadline := time.Now().Add(npcBootRetryBudget)
	delay := npcBootRetryInitial
	for attempt := 1; ; attempt++ {
		snap, retriable, err := d.fetchNPCConfigAttempt(ctx)
		if err == nil {
			return snap, nil
		}
		if !retriable {
			return npccfg.Snapshot{}, err
		}
		if time.Now().After(deadline) {
			return npccfg.Snapshot{}, fmt.Errorf("gave up after %d attempts in %s: %w", attempt, npcBootRetryBudget, err)
		}
		d.log.Warn("npc config boot load failed, retrying", "attempt", attempt, "retry_in", delay, "err", err)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return npccfg.Snapshot{}, fmt.Errorf("npc config boot load: %w", ctx.Err())
		case <-timer.C:
		}
		delay = min(delay*2, npcBootRetryMax)
	}
}

// fetchNPCConfigAttempt roda uma tentativa de carga de boot. retriable informa se a
// falha é do tipo que se resolve quando o dbServer termina de subir.
func (d *Dispatcher) fetchNPCConfigAttempt(ctx context.Context) (snap npccfg.Snapshot, retriable bool, err error) {
	fetchCtx, cancel := context.WithTimeout(ctx, npcFetchTimeout)
	defer cancel()
	snap, err = d.npcSource.Snapshot(fetchCtx)
	if err != nil {
		return npccfg.Snapshot{}, true, fmt.Errorf("npc config boot load: %w", err)
	}
	seen := make(map[int]struct{}, len(snap.Defs))
	for _, def := range snap.Defs {
		if def.Origin != "content" || def.GeneratorIndex < 0 || world.EventOwnedGenerator(def.GeneratorIndex) {
			continue
		}
		if _, exists := seen[def.GeneratorIndex]; exists {
			return npccfg.Snapshot{}, false, fmt.Errorf("npc config duplicate generator index %d", def.GeneratorIndex)
		}
		seen[def.GeneratorIndex] = struct{}{}
	}
	if len(seen) == 0 {
		// Um catálogo vazio significa que o lemos antes do seed commitar — não que
		// este shard deva ficar sem criaturas.
		return npccfg.Snapshot{}, true, errors.New("npc config catalogue is empty")
	}
	return snap, false, nil
}

// ApplyNPCConfigBoot materializa o snapshot carregado antes de o loop começar (sem
// reveal — não há jogadores conectados ainda). Os slots de gerador foram registrados a partir deste
// mesmo snapshot, então uma definição sem slot é bug de ligação: falhar fechado em vez de
// subir um mundo com buracos.
func (d *Dispatcher) ApplyNPCConfigBoot(w *world.World, snap npccfg.Snapshot) error {
	for _, def := range snap.Defs {
		if !def.Enabled || def.Origin != "content" || def.GeneratorIndex < 0 {
			continue
		}
		if w.GeneratorAt(def.GeneratorIndex) == nil {
			return fmt.Errorf("npc config content generator index %d (%s) has no registered slot", def.GeneratorIndex, def.Slug)
		}
	}
	d.applyNPCConfig(w, snap, false)
	d.log.Info("npc config applied at boot", "version", snap.Version, "npcs", len(d.managedNPCs))
	return nil
}

// pollNPCConfig checa a versão de config fora do loop a cada npcPollPeriod ticks
// e, quando mudou, recarrega o snapshot e aplica dentro do loop. Só
// um poll está em andamento por vez (npcPolling). Chamado do Tick de IA de criaturas.
func (d *Dispatcher) pollNPCConfig(w *world.World) {
	if d.npcSource == nil || d.npcPolling {
		return
	}
	d.npcPollTick++
	if d.npcPollTick%npcPollPeriod != 0 {
		return
	}
	known := d.npcVersion // capturado dentro do loop; a goroutine de fora nunca lê estado do loop
	d.npcPolling = true
	w.GoDetached(func() func(*world.World) {
		ctx, cancel := context.WithTimeout(context.Background(), npcFetchTimeout)
		defer cancel()
		version, err := d.npcSource.Version(ctx)
		if err != nil {
			return func(*world.World) { d.npcPolling = false; d.log.Warn("npc config version poll failed", "err", err) }
		}
		if version == known {
			return func(*world.World) { d.npcPolling = false }
		}
		snap, err := d.npcSource.Snapshot(ctx)
		if err != nil {
			return func(*world.World) { d.npcPolling = false; d.log.Warn("npc config reload failed", "err", err) }
		}
		return func(w *world.World) {
			d.applyNPCConfig(w, snap, true)
			d.npcPolling = false
			d.log.Info("npc config reloaded", "version", snap.Version, "npcs", len(d.managedNPCs))
		}
	})
}

// applyNPCConfig reconcilia o mundo vivo com um snapshot de config, dentro do
// loop. Reconcilia por DIFF: uma definição cuja chave materializada não mudou
// mantém sua entidade viva, então uma edição de moderador só toca o NPC que mudou.
//
// A primeira implementação apagava e reconstruía todo o conjunto gerenciado em cada
// apply. É semanticamente mais simples mas errado para um shard vivo: o cliente de
// um jogador parado na cidade então segura a entidade VELHA ao lado da NOVA, então
// editar um único NPC duplicava todo mercador na tela (relato in-game,
// Armia, 2026-11-04 — três bumps de config com o jogador online). O reveal ainda
// anuncia os poucos spawns que um diff real produz (true no hot-reload, false no
// boot), e o overlay global de preço de item é atualizado em todo apply.
func (d *Dispatcher) applyNPCConfig(w *world.World, snap npccfg.Snapshot, reveal bool) {
	d.rebuildItemPrices(snap.PriceOverrides)

	// Uma definição de conteúdo desativada precisa parar de gerar a partir do seu bloco NPCGener.
	for _, def := range snap.Defs {
		if def.Enabled || def.Origin != "content" || def.GeneratorIndex < 0 || world.EventOwnedGenerator(def.GeneratorIndex) {
			continue
		}
		if g := w.GeneratorAt(def.GeneratorIndex); g != nil {
			w.ClearGenerator(def.GeneratorIndex)
			g.LeaderTmpl, g.FollowerTmpl = nil, nil
		}
	}

	live := make(map[string]struct{}, len(snap.Defs))
	for _, def := range snap.Defs {
		if !def.Enabled {
			continue
		}
		live[def.Slug] = struct{}{}
		key := npcDefKey(def)
		if m, ok := d.managedNPCs[def.Slug]; ok {
			if m.key == key && w.Entity(m.id) != nil {
				continue // sem mudança: deixar a entidade e sua receita em paz
			}
			d.teardownNPC(w, def.Slug, m)
		}
		if def.Origin == "content" && def.GeneratorIndex >= 0 {
			if world.EventOwnedGenerator(def.GeneratorIndex) {
				continue // o sistema de eventos gera este bloco; o overlay não pode
			}
			g := w.GeneratorAt(def.GeneratorIndex)
			if g == nil {
				d.log.Warn("npc content generator index has no registered slot", "slug", def.Slug, "index", def.GeneratorIndex)
				continue
			}
			w.ClearGenerator(def.GeneratorIndex)
			if !def.Enabled {
				g.LeaderTmpl = nil
				g.FollowerTmpl = nil
				continue
			}
			if def.Template == nil {
				d.log.Warn("npc definition has no template - skipped", "slug", def.Slug)
				continue
			}
			g.MinuteGenerate, g.MinGroup, g.MaxGroup, g.MaxNumMob = def.MinuteGenerate, def.MinGroup, def.MaxGroup, def.MaxNumMob
			g.RouteType, g.Formation = def.RouteType, def.Formation
			g.SegX, g.SegY, g.SegRange, g.SegWait = def.SegX, def.SegY, def.SegRange, def.SegWait
			g.FightAction, g.DieAction = def.FightAction, def.DieAction
			g.LeaderTmpl = npcTemplateWithDisplayName(def.Template, def.DisplayName)
			g.FollowerTmpl = def.FollowerTemplate
			if world.IsWaterGenerator(def.GeneratorIndex) {
				continue // salas de água são donas destes blocos; não as gerar pelo hot reload de NPC
			}
			ids := w.GenerateMob(def.GeneratorIndex)
			if len(ids) == 0 {
				d.log.Warn("npc content generator produced no entities", "slug", def.Slug, "index", def.GeneratorIndex)
				continue
			}
			if def.Merchant != 0 {
				applyShop(w.Entity(ids[0]), def.Shop)
			}
			d.managedNPCs[def.Slug] = managedNPC{id: ids[0], key: key, gen: def.GeneratorIndex}
			if reveal {
				d.revealSpawned(w, ids)
			}
			continue
		}
		if !def.Enabled {
			continue
		}
		if def.Template == nil {
			// Template falhou em resolver (arquivo ausente/corrompido) — logado também na
			// fonte do dbclient, mas avisado aqui para o pulo ficar visível junto
			// dos outros ramos de pulo (posição inválida, sem slot livre).
			d.log.Warn("npc definition has no template - skipped", "slug", def.Slug)
			continue
		}
		if def.X <= 0 || def.Y <= 0 {
			d.log.Warn("npc definition has no valid position - skipped", "slug", def.Slug, "x", def.X, "y", def.Y)
			continue
		}
		template := npcTemplateWithDisplayName(def.Template, def.DisplayName)
		id := w.SpawnMobAt(world.MobSpawn{
			Template:  template,
			X:         def.X,
			Y:         def.Y,
			RouteType: def.RouteType,
			GenIndex:  -1, // não pertence a um bloco NPCGener (sem contabilidade de CurrentNumMob)
		})
		if id < 0 {
			d.log.Warn("npc spawn failed (no free slot)", "slug", def.Slug)
			continue
		}
		if def.Merchant != 0 {
			applyShop(w.Entity(id), def.Shop)
		}
		d.managedNPCs[def.Slug] = managedNPC{id: id, key: key, gen: -1}
		if reveal {
			d.revealSpawned(w, []int{id})
		}
	}
	for slug, m := range d.managedNPCs {
		if _, ok := live[slug]; !ok {
			d.teardownNPC(w, slug, m) // definição apagada da config
		}
	}
	d.npcVersion = snap.Version
}

// managedNPC é uma definição materializada: sua entidade líder viva, a chave que
// resume o que foi aplicado, e o bloco NPCGener que ela possui (-1 para uma definição
// gerada na própria posição). A chave é o que transforma um reload em diff.
type managedNPC struct {
	id  int
	key string
	gen int
}

// teardownNPC remove uma definição gerenciada. Um bloco de conteúdo é limpo inteiro
// (líder, seguidores e qualquer respawn na fila); um NPC autônomo é despawnado.
func (d *Dispatcher) teardownNPC(w *world.World, slug string, m managedNPC) {
	if m.gen >= 0 {
		w.ClearGenerator(m.gen)
	} else {
		w.DespawnMob(m.id, 0) // removeType 0 = remoção fora de visão, nunca enfileira respawn
	}
	delete(d.managedNPCs, slug)
}

// npcDefKey resume tudo a que uma definição se aplica no mundo vivo, para um
// apply conseguir distinguir "nada mudou" de "respawnar este". Os bytes do template
// fazem parte: um STRUCT_MOB recarimbado precisa rematerializar mesmo quando só um
// stat mudou.
func npcDefKey(def npccfg.Definition) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%d|%s|%d,%d|%d|%d|%d,%d,%d,%d,%d|",
		def.Origin, def.GeneratorIndex, def.DisplayName, def.X, def.Y, def.RouteType,
		def.Merchant, def.MinuteGenerate, def.MinGroup, def.MaxGroup, def.MaxNumMob, def.Formation)
	for i := range def.SegX {
		fmt.Fprintf(h, "%d,%d,%d,%d;", def.SegX[i], def.SegY[i], def.SegRange[i], def.SegWait[i])
	}
	for i := range def.FightAction {
		fmt.Fprintf(h, "%s;%s;", def.FightAction[i], def.DieAction[i])
	}
	for _, it := range def.Shop {
		fmt.Fprintf(h, "%d:%d:%d:%v;", it.Slot, it.Index, it.Quantity, it.Eff)
	}
	h.Write(def.Template)
	h.Write(def.FollowerTemplate)
	return strconv.FormatUint(h.Sum64(), 16)
}

// npcTemplateWithDisplayName sobrepõe o nome de exibição do moderador numa cópia do
// STRUCT_MOB cru. Templates são compartilhados pelo cache do dbclient, então mutar a
// entrada vazaria o nome traduzido de um NPC para todo NPC que usa aquele template.
func npcTemplateWithDisplayName(template []byte, displayName string) []byte {
	if strings.TrimSpace(displayName) == "" {
		return template
	}
	out := append([]byte(nil), template...)
	for i := 0; i < 16 && i < len(out); i++ {
		out[i] = 0
	}
	copy(out[0:min(16, len(out))], displayName)
	return out
}

// applyShop sobrescreve o estoque de loja de uma entidade mercador (seu Carry[]) com os
// slots definidos pelo moderador. Slot é o índice de MSG_ShopList (0..26); ele é mapeado
// para o índice real de Carry via protocol.ShopSlot (3 abas de 9: Carry[0..8],
// [27..35], [54..62]) — o mesmo mapeamento que reqShopList lê de volta, então um item na
// aba 2/3 (slot >= 9) de fato chega ao cliente. A loja é autoritativa:
// slots não listados são limpos, então esvaziar a loja na web não vende nada.
func applyShop(e *world.Entity, shop []npccfg.ShopItem) {
	if e == nil {
		return
	}
	for i := range e.Carry {
		e.Carry[i] = world.Item{}
	}
	for _, it := range shop {
		if it.Slot < 0 || it.Slot >= maxShopSlots {
			continue
		}
		e.Carry[protocol.ShopSlot(it.Slot)] = world.Item{
			Index:   int16(it.Index),
			Effects: shopEffects(it),
		}
	}
}

const amountEffect = 61

func shopEffects(it npccfg.ShopItem) [3]world.Effect {
	eff := [3]world.Effect{
		{Effect: it.Eff[0][0], Value: it.Eff[0][1]},
		{Effect: it.Eff[1][0], Value: it.Eff[1][1]},
		{Effect: it.Eff[2][0], Value: it.Eff[2][1]},
	}
	if it.Quantity <= 1 {
		return eff
	}
	for i := range eff {
		if eff[i].Effect == 0 {
			eff[i] = world.Effect{Effect: amountEffect, Value: it.Quantity}
			return eff
		}
	}
	return eff
}

// maxShopSlots é o número de slots de MSG_ShopList (3 abas de 9); o Slot de um item de
// loja é um índice de exibição em [0, maxShopSlots).
const maxShopSlots = 27

// rebuildItemPrices reseta o mapa de preço efetivo para o catálogo de conteúdo, depois
// aplica os overrides globais do moderador por cima. Só do loop, então handlers que leem
// d.itemPrices nunca correm contra ele.
func (d *Dispatcher) rebuildItemPrices(overrides map[int]int32) {
	if len(overrides) == 0 && d.baseItemPrices == nil {
		return
	}
	d.itemPrices = cloneInt32Map(d.baseItemPrices)
	if d.itemPrices == nil {
		d.itemPrices = make(map[int]int32, len(overrides))
	}
	for idx, price := range overrides {
		d.itemPrices[idx] = price
	}
}

// cloneInt32Map devolve uma cópia rasa de m (nil para entrada nil).
func cloneInt32Map(m map[int]int32) map[int]int32 {
	if m == nil {
		return nil
	}
	out := make(map[int]int32, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
