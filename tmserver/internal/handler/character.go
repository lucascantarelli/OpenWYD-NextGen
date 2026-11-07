package handler

import (
	"context"
	"encoding/binary"
	"time"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// dropExpired remove qualquer item temporizado cujo prazo passou (now = segundos Unix).
func dropExpired(items []world.Item, now int64) {
	for i := range items {
		if items[i].ExpiresAt != 0 && now >= items[i].ExpiresAt {
			items[i] = world.Item{}
		}
	}
}

// createCharacter trata _MSG_CreateCharacter (0x020F),
// handlers/_MSG_CreateCharacter.md. Exige USER_SELCHAR e um nome válido, depois
// repassa a criação ao dbServer.
func (d *Dispatcher) createCharacter(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	var body protocol.MsgCreateCharacterBody
	if err := body.Decode(payload); err != nil {
		d.log.Warn("create char: decode failed", "conn", s.Conn, "len", len(payload), "err", err)
		w.Send(s, protocol.MsgNewCharacterFail, nil)
		return
	}
	name := cstr(body.MobName[:])
	d.log.Info("create char", "conn", s.Conn, "slot", body.Slot, "class", body.MobClass,
		"name", name, "mode", s.Mode, "want_mode", world.UserSelChar)
	if s.Mode != world.UserSelChar {
		w.Send(s, protocol.MsgNewCharacterFail, nil)
		return
	}
	if !validCharName(name) {
		d.log.Warn("create char: invalid name", "conn", s.Conn, "name", name)
		w.Send(s, protocol.MsgNewCharacterFail, nil)
		return
	}

	slot := int(body.Slot)
	class := int(body.MobClass)
	accID := s.AccountID
	s.Mode = world.UserWaitDB

	p := w.Persistence()
	w.Go(s, func() func(*world.World, *world.Session) {
		ok, err := p.CreateCharacter(context.Background(), accID, slot, name, class)
		if err != nil || !ok {
			return func(w *world.World, s *world.Session) {
				s.Mode = world.UserSelChar
				d.log.Warn("create char: dbServer rejected", "conn", s.Conn, "ok", ok, "err", err)
				w.Send(s, protocol.MsgNewCharacterFail, nil)
			}
		}
		// Sucesso: buscar a lista de novo e reenviar o SELCHAR completo (o original
		// responde MSG_CNFNewCharacter com a seleção inteira, agora com o personagem novo).
		chars, lerr := p.ListCharacters(context.Background(), accID)
		return func(w *world.World, s *world.Session) {
			s.Mode = world.UserSelChar
			if lerr != nil {
				d.log.Warn("create char: list after create failed", "conn", s.Conn, "err", lerr)
				w.Send(s, protocol.MsgNewCharacterFail, nil)
				return
			}
			d.log.Info("create char: OK", "conn", s.Conn, "name", name, "slot", slot, "total", len(chars))
			body := protocol.EncodeCNFNewCharacterBody(d.selCharsFrom(chars))
			w.SendTo(s, protocol.Header{Type: protocol.MsgCNFNewCharacter, ID: protocol.IDNewCharacter}, body)
		}
	})
}

// deleteCharacter trata _MSG_DeleteCharacter (0x0211),
// handlers/lote2-sessao-conta.md. Exige USER_SELCHAR; a senha é verificada pelo
// dbServer.
func (d *Dispatcher) deleteCharacter(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	var body protocol.MsgDeleteCharacterBody
	if err := body.Decode(payload); err != nil {
		return
	}
	if s.Mode != world.UserSelChar {
		d.notify(w, s, NoticeDeletingWait)
		return
	}
	slot := int(body.Slot)
	name := cstr(body.MobName[:])
	pass := cstr(body.Password[:])
	accID := s.AccountID
	s.Mode = world.UserWaitDB

	p := w.Persistence()
	w.Go(s, func() func(*world.World, *world.Session) {
		ok, err := p.DeleteCharacter(context.Background(), accID, slot, name, pass)
		if err != nil || !ok {
			// Legado _MSG_DBDeleteCharacterFail -> _MSG_DeleteCharacterFail
			// (ProcessDBMessage.cpp:641-649). O cliente mostra o próprio
			// aviso de "delete failed" para 0x011B; 0x011A diria que o
			// personagem não pôde ser criado.
			return func(w *world.World, s *world.Session) {
				s.Mode = world.UserSelChar
				w.Send(s, protocol.MsgDeleteCharacterFail, nil)
			}
		}
		// Sucesso: buscar a lista de novo e reenviar o SELCHAR completo, igual a
		// createCharacter. MSG_CNFDeleteCharacter (Basedef.h:1646-1652) carrega o
		// mesmo corpo `STRUCT_SELCHAR sel` que MSG_CNFNewCharacter — sem
		// reenviar a lista atualizada, o cliente mantém o array de slots obsoleto de antes
		// do delete, o que dessincroniza os personagens restantes na tela até relogar.
		chars, lerr := p.ListCharacters(context.Background(), accID)
		return func(w *world.World, s *world.Session) {
			s.Mode = world.UserSelChar
			if lerr != nil {
				d.log.Warn("delete char: list after delete failed", "conn", s.Conn, "err", lerr)
				w.Send(s, protocol.MsgNewCharacterFail, nil)
				return
			}
			respBody := protocol.EncodeCNFNewCharacterBody(d.selCharsFrom(chars))
			w.SendTo(s, protocol.Header{Type: protocol.MsgCNFDeleteCharacter, ID: protocol.IDNewCharacter}, respBody)
		}
	})
}

// characterLogin trata _MSG_CharacterLogin (0x0213),
// handlers/_MSG_CharacterLogin.md. Exige USER_SELCHAR e um slot válido; carrega
// o personagem do dbServer e injeta o jogador no mundo.
//
// O gate de billing é a fronteira NOVA (binServer por gRPC, Fase 6): antes de
// carregar o personagem perguntamos ao World.Billing se a conta pode entrar. O
// gate legado fixo de free-exp (Unk_*, BILLING, FREEEXP, g_Hour) NÃO é
// reproduzido — ele é UNVERIFIED. O gate padrão (AllowAllBilling) é
// free-to-play, então isto não quebra nada até um binServer ser ligado.
func (d *Dispatcher) characterLogin(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	var body protocol.MsgCharacterLoginBody
	if err := body.Decode(payload); err != nil {
		return
	}
	slot := int(body.Slot)
	if slot < 0 || slot >= world.MobPerAccount {
		d.notify(w, s, NoticeSelectCharacter)
		return
	}
	if s.Mode != world.UserSelChar {
		d.notify(w, s, NoticeSelectCharacter)
		return
	}
	d.log.Info("character login request", "conn", s.Conn, "slot", slot, "mode", s.Mode)
	s.Slot = slot
	s.Mode = world.UserCharWait
	accID := s.AccountID
	accName := s.AccountName

	p := w.Persistence()
	b := w.Billing()
	// As duas chamadas são I/O bloqueante; rodá-las em sequência fora do loop. O resultado
	// volta ao loop pelo callback devolvido.
	w.Go(s, func() func(*world.World, *world.Session) {
		allowed, berr := b.Check(context.Background(), accName)
		if berr != nil {
			return func(w *world.World, s *world.Session) { d.billingFailed(w, s, berr) }
		}
		if !allowed {
			return func(w *world.World, s *world.Session) { d.billingDenied(w, s) }
		}
		st, err := p.LoadCharacter(context.Background(), accID, slot)
		return func(w *world.World, s *world.Session) { d.completeCharacterLogin(w, s, st, err) }
	})
}

// billingDenied devolve o jogador à seleção de personagem depois que o binServer
// recusa a entrada (expirado/bloqueado). UNVERIFIED: a mensagem real S→C de recusa não foi
// capturada; um aviso provisório ocupa o lugar (parity-tests.md §5).
func (d *Dispatcher) billingDenied(w *world.World, s *world.Session) {
	s.Mode = world.UserSelChar
	d.notify(w, s, NoticeBillingDenied)
}

// billingFailed trata um erro do serviço de billing (tratado como "recusar, tente de novo").
func (d *Dispatcher) billingFailed(w *world.World, s *world.Session, err error) {
	d.log.Error("billing check failed", "conn", s.Conn, "account", s.AccountName, "err", err)
	s.Mode = world.UserSelChar
	d.notify(w, s, NoticeDBError)
}

func (d *Dispatcher) completeCharacterLogin(w *world.World, s *world.Session, st world.CharacterState, err error) {
	if err != nil {
		d.log.Error("load character failed", "conn", s.Conn, "slot", s.Slot, "err", err)
		s.Mode = world.UserSelChar
		w.Send(s, protocol.MsgCharacterLoginFail, nil)
		return
	}
	// Descartar itens temporizados (ex.: uma montaria Perzen de 30 dias expirada) antes de injetar
	// o personagem — a expiração é aplicada aqui no carregamento.
	now := time.Now().Unix()
	dropExpired(st.Equip[:], now)
	dropExpired(st.Carry[:], now)
	// Semear o equipamento inicial para personagens que ainda não têm nenhum (recém-criados ou
	// criados antes de a semeadura existir). Isto restaura a aparência da classe (o item de corpo
	// no slot 0 de equipamento é o que dá aparência a TK/FM/BM/HT), entrega o
	// equipamento de classe (armadura/arma), E o inventário inicial de classe (poções de HP/MP etc. do
	// Carry do template). Persiste no próximo save. Um equipamento vazio é o
	// marcador de "personagem novo", então o inventário é concedido só uma vez (não
	// concedido de novo depois que o jogador gasta as poções).
	if equipEmpty(st.Equip) {
		st.Equip = d.starterEquip(st.Class)
		d.grantStarterCarry(&st.Carry, st.Class)
	}
	// Curar personagens cujos slots de equipamento foram corrompidos por um bug antigo que deixava
	// itens que não são equipamento (uma poção, uma montaria na mão errada) serem equipados — mais visivelmente um
	// consumível no slot do corpo, que deixa o corpo invisível/errado. Itens em slot errado
	// voltam ao inventário e o item de corpo da classe é restaurado.
	d.repairEquip(&st)
	// Um personagem nunca pode entrar no mundo morto. Agora que criaturas podem matar jogadores
	// (mobai.go), um que foi morto e depois desconectou sem reiniciar fica
	// persistido com HP 0 — revivê-lo no login (HP/MP cheios) o põe de volta em jogo na
	// sua cidade em vez de logar preso/morto (a regeneração passiva exclui HP 0).
	if st.HP <= 0 {
		st.HP = st.MaxHP
		st.MP = st.MaxMP
		if st.HP <= 0 {
			st.HP = 1 // guardar um MaxHP quebrado/zero para o jogador ainda poder agir
		}
	}
	// A posição de login segue a divisão legada: STRUCT_MOB.SPX/SPY é o ponto de
	// save do warp da Gema Estelar (st.SaveX/SaveY, reidratado na entidade abaixo),
	// enquanto MSG_CNFCharacterLogin.PosX/PosY é o tile real de entrada no mundo
	// (st.X/Y). Uma posição carregada explícita (testes/capturas) é respeitada quando
	// presente; cargas do banco ao vivo caem hoje na regra de spawn da última cidade.
	saveX, saveY := st.SaveX, st.SaveY
	loginX, loginY := st.X, st.Y
	if (loginX == 0 && loginY == 0) || world.NightmareMap(loginX, loginY) >= 0 {
		loginX, loginY = world.CitySpawn(int(st.LastCity))
		if x, y, ok := w.EmptyCellNear(loginX, loginY); ok {
			loginX, loginY = x, y
		}
	}
	s.LoginSpawnX, s.LoginSpawnY = loginX, loginY
	s.LoginTick = w.Now()
	s.LoggedFirstAction = false
	// Injetar a entidade do jogador no mundo (o slot foi atracado na conexão).
	if e := w.Entity(s.Conn); e != nil {
		e.Mode = world.MobUser
		// A entidade é por conexão: limpar qualquer affect deixado por um personagem
		// jogado antes, antes de reidratar os slots persistidos deste (a
		// reidratação abaixo só SOMA slots, o que causou os vazamentos
		// entre personagens das issues #21/#47).
		e.ResetAffects()
		e.Name = st.Name
		e.Class = uint8(st.Class)
		e.LastCity = st.LastCity
		e.SaveX, e.SaveY = st.SaveX, st.SaveY
		// Registrar o jogador na grade espacial, não só nos campos da entidade —
		// o aggro de criaturas (FindEnemyFromView) e a reconciliação de visão varrem
		// a grade, então um jogador parado desde o login precisa estar lá.
		w.SetEntityPos(s.Conn, loginX, loginY)
		e.HP, e.MaxHP = st.HP, st.MaxHP
		e.MP, e.MaxMP = st.MP, st.MaxMP
		e.EquipmentAttributeHP, e.EquipmentAttributeMP = 0, 0
		// Critical é só cache do lado do save: refreshScore abaixo o rederiva do
		// equipamento (Basedef.cpp:3209), então o valor guardado nunca sobrevive ao login.
		// Damage e AC NÃO são lidos do estado de jeito nenhum: o contrato do banco não
		// carrega nenhum dos dois, então st.Damage/st.AC são sempre 0 — deriveBaseScore reconstrói
		// sua base pelas regras legadas e refreshScore preenche os campos vivos
		// (issue #232).
		e.Master, e.Critical = st.Master, st.Critical
		e.Level, e.Coin, e.Exp = int32(st.Level), st.Coin, st.Exp
		e.Clan, e.Guild, e.GuildLevel, e.Citizen, e.ClassMaster, e.Soul = st.Clan, st.GuildID, st.GuildLevel, st.Citizen, st.ClassMaster, st.Soul
		e.Fame = st.Fame
		e.KefraTicket = st.KefraTicket
		e.NightmareEntries = st.NightmareEntries
		e.LastNightmareUse = st.LastNightmareUse
		// Linhas antigas criadas antes de ClassMaster ser persistido ainda podem vir com 0.
		// Tratar isso como MORTAL (=2, Basedef.h:238) para o EXP não passar pelo
		// caminho do divisor celestial (issue #43).
		if e.ClassMaster == 0 {
			e.ClassMaster = classMasterMortal
		}
		// Estado de PK/karma (issue #210). PKPoint == 0 significa "nunca persistido"
		// (SetPKPoint nunca escreve 0 legitimamente; a coluna do banco tem default 75)
		// — linhas pré-migração e fixtures de teste zeradas leem 0, então
		// tratar isso como neutro, a mesma convenção de ClassMaster == 0 acima.
		e.PKPoint, e.Guilty, e.CurKill, e.TotKill = st.PKPoint, st.Guilty, st.CurKill, st.TotKill
		if e.PKPoint == 0 {
			e.PKPoint = pkPointNeutral
		}
		// Gates de quest celestial (definidos por /destravar40/90 e /arcana; CheckGetLevel
		// lê Lv40/Lv90 para destravar os limites 40/90).
		e.CelLv40, e.CelLv90, e.CelCircle = st.CelLv40, st.CelLv90, st.CelCircle
		e.ArchLv355, e.ArchLv370 = st.ArchLv355, st.ArchLv370
		e.MortalLevel, e.CelestialArchLevel = st.MortalLevel, st.CelestialArchLevel
		e.ArchCrystalStage = st.ArchCrystalStage
		e.TerraMistica = st.TerraMistica
		e.NewbieQuest = st.NewbieQuest
		e.MolarGargula = st.MolarGargula
		e.PilulaOrc = st.PilulaOrc
		e.CelReset = st.CelReset
		e.CelAdd120, e.CelAdd150, e.CelAdd180, e.CelAdd200 = st.CelAdd120, st.CelAdd150, st.CelAdd180, st.CelAdd200
		e.CelestialLevel, e.SubCelestialLevel, e.SaveCelestial = st.CelestialLevel, st.SubCelestialLevel, st.SaveCelestial
		e.Str, e.Int, e.Dex, e.Con, e.ScoreBonus = st.Str, st.Int, st.Dex, st.Con, st.ScoreBonus
		// Estado de skills: a máscara de aprendidas, a maestria alocada e a hotbar vêm
		// direto do banco; SkillBonus é rederivado do nível + custos
		// aprendidos (BASE_GetBonusSkillPoint ao carregar o personagem, ProcessDBMessage.cpp:816).
		e.LearnedSkill, e.SecLearnedSkill, e.SpecialBonus = st.LearnedSkill, st.SecLearnedSkill, st.SpecialBonus
		e.BaseSpecial, e.SkillBar = st.BaseSpecial, st.SkillBar
		s.ShortSkill = st.ShortSkill
		d.deriveSkillBonus(e)
		e.Equip = st.Equip
		e.Carry = st.Carry
		// O BaseScore sem equipamento é salvo à parte (migration 0027). Um
		// personagem salvo antes disso ainda carrega só o CurrentScore, então sua
		// base é capturada uma vez dele, como antes; o próximo save a guarda.
		if st.HasBase {
			d.loadBaseScore(e, st)
		} else {
			d.deriveBaseScore(e)
		}
		// Reaplicar um buff Divino ainda ativo a partir do prazo persistido (o buff é
		// de tempo de leitura, então isto não afeta a base recém-derivada). Expirado → descartado.
		if st.DivineEnd > time.Now().Unix() {
			if slot := e.EmptyAffect(world.AffectDivine); slot >= 0 {
				e.DivineEnd = st.DivineEnd
				e.Affect[slot] = world.Affect{Type: world.AffectDivine, Level: 1, Time: divineAffectTime}
			}
		}
		// Reidratar os outros slots de buff persistidos (Time em ticks de affect de 8s; a
		// varredura de tick retoma a contagem regressiva).
		for _, a := range st.Affects {
			if a.Type == 0 || a.Type == world.AffectDivine || a.Time == 0 {
				continue
			}
			if slot := e.EmptyAffect(a.Type); slot >= 0 {
				e.Affect[slot] = a
			}
		}
		// Recalcular recursos de equipamento em runtime, Special vivo e caches de affect.
		// Os máximos salvos omitem recursos de CON/INT do equipamento, então até um save antigo ganha
		// a correção da issue #321 aqui sem acumulá-la em logins seguintes.
		d.refreshScore(e)
		s.ReqHp, s.ReqMp = e.HP, e.MP
		s.CriticalProgress = 0
		// Códigos visuais de equipamento do equipamento REAL do personagem, para outros (e o
		// próprio cliente, via UpdateEquip) verem o que está realmente equipado — não o
		// conjunto inicial. Slots vazios → 0 (sem item). DEPOIS da reidratação de affect +
		// refreshScore, para uma transformação persistida (affect 16) renderizar sua malha de fera.
		e.EquipVisual, e.EquipAnct = equipVisual(e)
	}
	s.Mode = world.UserPlay
	// O bloco de skills persistido vai no snapshot de login (mask/points/bar/Special);
	// o Special vivo (base + equip) está na Entity depois que o refreshScore-no-login
	// ainda não rodou, então enviar base+equip pela entidade quando disponível.
	var skill protocol.SkillState
	loginPKPoint := pkPointNeutral // MobName[12] para o próprio nick: 75 = branco (personagem novo nunca é culpado)
	if e := w.Entity(s.Conn); e != nil {
		skill = protocol.SkillState{
			LearnedSkill: e.LearnedSkill,
			ScoreBonus:   e.ScoreBonus, SpecialBonus: e.SpecialBonus, SkillBonus: e.SkillBonus,
			Special: e.Special, BaseSpecial: e.BaseSpecial, SkillBar: e.SkillBar,
		}
		loginPKPoint = pkPoint(e)
	}
	shortSkill := s.ShortSkill
	// Preferir o template BaseMob por classe (STRUCT_MOB real com equipamento inicial
	// → modelo de classe correto e sem crash no cliente); ajustar nome + posição.
	if tmpl, ok := d.baseMobs[st.Class]; ok && len(tmpl) == content.BaseMobSize {
		var equip [16]protocol.SelItem
		for i := range st.Equip {
			equip[i] = itemToSel(st.Equip[i])
		}
		// Uma transformação BM persistida aparece no próprio modelo: o runtime monta seu
		// herói a partir deste snapshot e ignora o próprio CreateMob assim que ele existe
		// (TMFieldScene::OnPacketCreateMob, ramo de nó vivo). O legado obtém o
		// mesmo enviando MOB depois de GetCurrentScore, que escrevia a malha de fera
		// em Equip[0].sIndex (Basedef.cpp:4106, ProcessDBMessage.cpp). Só a
		// cópia do pacote muda; o item de corpo salvo não.
		if e := w.Entity(s.Conn); e != nil {
			if value, _, ok := activeTransform(e); ok {
				equip[0].Index = transMesh(value)
			}
			// O 7662 lê a cidadania do personagem do terceiro efeito do item de
			// corpo, não de um campo do pacote: o legado o sobrescreve logo antes de
			// entregar o MOB (ProcessDBMessage.cpp:763 — cEffect 28,
			// cValue = mobExtra.Citizen). Sem isso a janela do personagem diz que a
			// cidadania é inexistente mesmo depois da venda da Kibita.
			equip[0].Eff[2] = [2]uint8{28, e.Citizen}
		}
		var carry [64]protocol.SelItem
		for i := range st.Carry {
			if i >= 64 {
				break
			}
			carry[i] = itemToSel(st.Carry[i])
		}
		// O clima vai no snapshot de login em vez de um pacote separado, exatamente
		// como o legado faz (sm.Weather = CurrentWeather, ProcessDBMessage.cpp:834).
		body := protocol.EncodeCNFCharacterLoginRaw(tmpl, st.Name, st.Coin, st.Exp, equip, carry, loginX, loginY, saveX, saveY, s.Slot, s.Conn, uint16(d.currentWeather()), shortSkill, skill, loginPKPoint)
		// O template traz o score inicial da classe; sobrepor os scores
		// do próprio personagem, como o legado envia pMob[conn].MOB depois de GetCurrentScore.
		if e := w.Entity(s.Conn); e != nil {
			protocol.OverlayLoginScores(body, loginBaseScore(st, skill), d.computeScore(e))
		}
		d.logCNFCharacterLogin("template", s, st, loginX, loginY, body)
		w.SendTo(s, protocol.Header{Type: protocol.MsgCNFCharacterLogin, ID: protocol.IDScene}, body)
		d.enterWorldView(w, s)
		if e := w.Entity(s.Conn); e != nil {
			d.refreshBabyMountSummon(w, s, e)
		}
		d.sendLoginAffects(w, s)
		return
	}
	d.log.Info("char login: sending CNFCharacterLogin (fallback, no template)",
		"conn", s.Conn, "class", st.Class)
	// Fallback: montar o snapshot do estado relacional guardado (sem equipamento).
	// MSG_CNFCharacterLogin byte a byte (STRUCT_MOB + pos + skillbar), ID=30000.
	if saveX == 0 && saveY == 0 {
		saveX, saveY = loginX, loginY
	}
	m := protocol.MobSnapshot{
		Name:  st.Name,
		Clan:  st.Clan,
		Guild: st.GuildID,
		Class: uint8(st.Class),
		Coin:  st.Coin,
		Exp:   st.Exp,
		SPX:   saveX, SPY: saveY,
		Level: int32(st.Level), Ac: st.AC, Damage: st.Damage,
		MaxHp: st.MaxHP, MaxMp: st.MaxMP, Hp: st.HP, Mp: st.MP,
		Str: st.Str, Int: st.Int, Dex: st.Dex, Con: st.Con,
		AttackRun:  baseAttackRun,
		PKPoint:    loginPKPoint, // MobName[12]: 75 = nick branco (issue #59)
		ScoreBonus: st.ScoreBonus, GuildLevel: st.GuildLevel,
		LearnedSkill: skill.LearnedSkill, SpecialBonus: skill.SpecialBonus,
		SkillBonus: skill.SkillBonus, Special: skill.Special, SkillBar: skill.SkillBar,
	}
	for i := range st.Carry {
		if i >= len(m.Carry) {
			break
		}
		m.Carry[i] = protocol.SelItem{
			Index: uint16(st.Carry[i].Index),
			Eff: [3][2]uint8{
				{st.Carry[i].Effects[0].Effect, st.Carry[i].Effects[0].Value},
				{st.Carry[i].Effects[1].Effect, st.Carry[i].Effects[1].Value},
				{st.Carry[i].Effects[2].Effect, st.Carry[i].Effects[2].Value},
			},
		}
	}
	body := protocol.EncodeCNFCharacterLoginBody(s.Slot, s.Conn, uint16(d.currentWeather()), loginX, loginY, m, shortSkill)
	d.logCNFCharacterLogin("fallback", s, st, loginX, loginY, body)
	w.SendTo(s, protocol.Header{Type: protocol.MsgCNFCharacterLogin, ID: protocol.IDScene}, body)
	d.enterWorldView(w, s)
	if e := w.Entity(s.Conn); e != nil {
		d.refreshBabyMountSummon(w, s, e)
	}
	d.sendLoginAffects(w, s)
}

// loginBaseScore é o STRUCT_SCORE persistido que vai no snapshot de login como
// BaseScore, os mesmos valores que o fallback sem template escreve.
func loginBaseScore(st world.CharacterState, skill protocol.SkillState) protocol.ScoreData {
	return protocol.ScoreData{
		Level: int32(st.Level), Ac: st.AC, Damage: st.Damage,
		MaxHp: st.MaxHP, MaxMp: st.MaxMP, Hp: st.HP, Mp: st.MP,
		Str: st.Str, Int: st.Int, Dex: st.Dex, Con: st.Con,
		Special: skill.BaseSpecial, AttackRun: baseAttackRun,
	}
}

func (d *Dispatcher) logCNFCharacterLogin(path string, s *world.Session, st world.CharacterState, spawnX, spawnY int16, body []byte) {
	var msgX, msgY, mobSPX, mobSPY uint16
	if len(body) >= 4+44 {
		msgX = binary.LittleEndian.Uint16(body[0:])
		msgY = binary.LittleEndian.Uint16(body[2:])
		mobSPX = binary.LittleEndian.Uint16(body[4+40:])
		mobSPY = binary.LittleEndian.Uint16(body[4+42:])
	}
	d.log.Info("char login: sending CNFCharacterLogin",
		"path", path,
		"conn", s.Conn,
		"slot", s.Slot,
		"client_id", s.Conn,
		"class", st.Class,
		"name", st.Name,
		"last_city", st.LastCity,
		"spawn_x", spawnX,
		"spawn_y", spawnY,
		"cnf_pos_x", int16(msgX),
		"cnf_pos_y", int16(msgY),
		"mob_spx", int16(mobSPX),
		"mob_spy", int16(mobSPY),
		"body", len(body))
}

// sendLoginAffects empurra o snapshot de buffs reidratado logo após a
// entrada no mundo, para buffs persistidos mostrarem seus ícones sem esperar um evento de conjuração/score
// (o blob CNFCharacterLogin não carrega array de affect).
func (d *Dispatcher) sendLoginAffects(w *world.World, s *world.Session) {
	if e := w.Entity(s.Conn); e != nil && e.HasAnyAffect() {
		d.sendAffect(w, s, e)
	}
}

// enterWorldView liga a visibilidade de entidades depois que um jogador entra no mundo
// (ProcessDBMessage.cpp:1021): transmitir o MSG_CreateMob do recém-chegado a todo
// jogador em visão (CreateType=2), e enviar o MSG_CreateMob de cada jogador em visão ao
// recém-chegado. Sem isto o cliente inventa um avatar duplicado a partir de cada
// _MSG_Action de uma entidade desconhecida (B1). HEADER.ID é sempre IDScene (30000); o
// id da entidade vai no MobID.
func (d *Dispatcher) enterWorldView(w *world.World, s *world.Session) {
	self := w.Entity(s.Conn)
	if self == nil {
		return
	}
	w.ClearSeen(s) // conjunto de visão novo ao (re)entrar no mundo
	// Unicast, NÃO o sendScore multicast de sempre: isto roda antes de o CreateMob do
	// recém-chegado ser transmitido abaixo, então clientes em visão receberiam um UpdateScore de
	// uma entidade que ainda não criaram (a confusão do B1). O legado não tem
	// SendScore nenhum neste caminho (ProcessDBMessage.cpp:1017-1037) — observadores
	// aprendem o HP do recém-chegado pelo CreateMob.
	d.sendScoreSelf(w, s, self) // CurrentScore (atributos após equipamento + buffs ativos)
	if self.HasAnyAffect() {
		d.sendAffect(w, s, self) // ícones/timers de buff (ex.: um Divino reaplicado)
	}
	selfMob := protocol.EncodeCreateMobBody(d.createMobFrom(self, 2))
	// Enviar ao recém-chegado o PRÓPRIO CreateMob (o GridMulticast legado tem skip=0, então
	// a conn — já na grade — recebe o seu, ProcessDBMessage.cpp:1029).
	// Isto é o que colore o nick do PRÓPRIO jogador via MobName[12] (PKPoint): sem
	// ele o próprio nick renderiza para sempre do blob de login e não recolore. PKInfo
	// (flag de atacável) vai junto por paridade (SendPKInfo, SendFunc.cpp:1869).
	w.SendTo(s, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, selfMob)
	w.SendTo(s, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(s.Conn)}, protocol.EncodeStandardParm(pkInfoParm(self)))
	w.ForEachInView(s.Conn, func(vs *world.Session, ve *world.Entity) {
		// (A) outros jogadores veem o recém-chegado
		w.MarkSeen(vs, s.Conn)
		w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, selfMob)
		w.SendTo(vs, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(s.Conn)}, protocol.EncodeStandardParm(pkInfoParm(self)))
		// (B) o recém-chegado vê cada jogador já em visão
		w.MarkSeen(s, ve.ID)
		ty, body := d.createMobViewPacket(w, ve, 0)
		w.SendTo(s, protocol.Header{Type: ty, ID: protocol.IDScene}, body)
		w.SendTo(s, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(ve.ID)}, protocol.EncodeStandardParm(pkInfoParm(ve)))
	})
	// (C) o recém-chegado vê os NPCs/criaturas em visão.
	d.revealMobsInView(w, s)
}

// revealMobsInView envia um MSG_CreateMob para cada NPC/criatura agora na
// visão do jogador que o cliente ainda não viu (uma vez por entidade). Chamado na entrada e a cada
// movimento, para NPCs aparecerem conforme o jogador explora.
func (d *Dispatcher) revealMobsInView(w *world.World, s *world.Session) {
	w.ForEachMobInView(s.Conn, func(me *world.Entity) {
		if w.MarkSeen(s, me.ID) {
			w.SendTo(s, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene},
				protocol.EncodeCreateMobBody(d.createMobFrom(me, 0)))
		}
	})
}

// createMobFrom monta os dados de MSG_CreateMob a partir de uma entidade do mundo (jogador ou NPC).
// Os códigos visuais de equipamento e sobreposições de brilho vêm do
// EquipVisual/EquipAnct da entidade, definidos no login/spawn a partir dos dados relevantes de STRUCT_MOB.
// createType: 0 normal, 2 "acabou de entrar".
func (d *Dispatcher) createMobFrom(e *world.Entity, createType uint16) protocol.CreateMobData {
	data := protocol.CreateMobData{
		MobID:           e.ID,
		Name:            e.Name,
		PosX:            e.X,
		PosY:            e.Y,
		Guild:           e.Guild,
		GuildMemberType: e.GuildLevel,
		Level:           e.Level,
		Ac:              e.AC,
		Damage:          e.Damage,
		// HP/MP precisam casar com o score autoritativo (máximo efetivo incl. equipamento de HP/MP%):
		// o self-CreateMob agora dirige as próprias barras do jogador, então um MaxMp/Mp faltante
		// zerava o MP do cliente (regressão por adicionar o self-CreateMob de login).
		MaxHp: effectiveMaxHP(e), Hp: e.HP,
		MaxMp: effectiveMaxMP(e), Mp: e.MP,
		Str: e.Str, Int: e.Int, Dex: e.Dex, Con: e.Con,
		Merchant:   e.Merchant,
		AttackRun:  attackRunOf(e),
		Equip:      e.EquipVisual,
		AnctCode:   e.EquipAnct,
		CreateType: createType,
		// Jogadores empacotam PKPoint/CurKill/TotKill em MobName[12..15] para colorir o nick
		// (75 neutro/branco, 0 caos/vermelho) e mostrar os bytes de sequência de mortes (issue #210);
		// criaturas enviam o nome cru sem coloração de PK.
		IsPlayer: world.IsPlayer(e.ID),
		PKPoint:  playerPKPoint(e),
		CurKill:  e.CurKill,
		TotKill:  e.TotKill,
	}
	for i := range e.Affect {
		if e.Affect[i].Type == 0 {
			continue
		}
		data.Affect[i] = protocol.PackAffect(protocol.AffectData{
			Type: e.Affect[i].Type,
			Time: e.Affect[i].Time,
		})
	}
	// O legado envia MOB.CurrentScore (GetCreateMob, GetFunc.cpp:1111), o
	// mesmo score que o SendScore empurra. Para um jogador isso é computeScore — o
	// dano da arma e os buffs vão nos getters efetivos, não nos campos crus —
	// e o próprio cliente aplica seu self-CreateMob depois do UpdateScore de login.
	if world.IsPlayer(e.ID) {
		sc := d.computeScore(e)
		data.Level, data.Ac, data.Damage = sc.Level, sc.Ac, sc.Damage
		data.Str, data.Int, data.Dex, data.Con = sc.Str, sc.Int, sc.Dex, sc.Con
	}
	return data
}

// createMobViewPacket espelha o SendCreateMob legado: um jogador com loja
// pessoal aberta precisa ser revelado com MSG_CreateMobTrade, não com o pacote normal
// de avatar, para clientes que entram em visão depois de a loja abrir ainda verem a banca.
func (d *Dispatcher) createMobViewPacket(w *world.World, e *world.Entity, createType uint16) (protocol.Type, []byte) {
	data := d.createMobFrom(e, createType)
	if world.IsPlayer(e.ID) {
		s := w.Session(e.ID)
		if s != nil && s.Mode == world.UserPlay && s.TradeMode == 1 && s.AutoTrade != nil {
			data.Con = 0 // Paridade GetCreateMobTrade: a pose de loja esconde o campo Con.
			return protocol.MsgCreateMobTrade, protocol.EncodeCreateMobTradeBody(data, nil, s.AutoTrade.Title)
		}
	}
	return protocol.MsgCreateMob, protocol.EncodeCreateMobBody(data)
}

// playerPKPoint é pkPoint(e) para um jogador, ou 0 para uma criatura (criaturas nunca carregam
// coloração de PK e seu MobName é enviado cru, então o valor é ignorado de todo modo).
func playerPKPoint(e *world.Entity) uint8 {
	if !world.IsPlayer(e.ID) {
		return 0
	}
	return pkPoint(e)
}

// characterLogout trata _MSG_CharacterLogout (0x0215): voltar à tela de
// seleção. (Salvar o personagem em jogo é um lote posterior, quando SaveCharacter estiver no
// port.)
func (d *Dispatcher) characterLogout(w *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	d.returnToCharacterSelection(w, s, nil)
}

// returnToCharacterSelection faz a limpeza de mundo e o save durável compartilhados
// pelo logout comum e por transições de quest legadas que forçam recarregar o personagem.
// after roda no loop do mundo depois que o cliente recebeu a confirmação
// de logout, para efeitos seguintes da tela de seleção não correrem contra o save.
func (d *Dispatcher) returnToCharacterSelection(w *world.World, s *world.Session, after func(*world.World, *world.Session)) {
	// Sair do grupo (e recolher as invocações deste personagem) primeiro, enquanto a sessão
	// ainda é UserPlay — sendRemoveParty/sendAddParty só alcançam destinatários
	// UserPlay (_MSG_CharacterLogout.cpp:23 chama RemoveParty pelo mesmo
	// motivo). O gancho de desconexão é no-op depois disto.
	d.SessionEnd(w, s)
	// Despawnar esta entidade para jogadores em visão (volta à seleção de personagem).
	body := protocol.EncodeRemoveMobBody(2)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgRemoveMob, ID: uint16(s.Conn)}, body)
	})
	// Persistir primeiro, depois confirmar: o cliente relê o personagem do banco
	// quando reseleciona (e pode reconectar/relogar a conta), então o save precisa
	// commitar antes de devolvermos a tela de seleção (senão o reload
	// corre contra a escrita — last_city/coin). A carga compartilhada da conta é salva no
	// mesmo fluxo: depósitos/saques trocam itens entre o carry do personagem e
	// a carga, então persistir o personagem sem a carga duplicaria um
	// item sacado (salvo na linha do personagem enquanto a linha account_cargo obsoleta
	// ainda o guarda) no próximo carregamento.
	w.SaveCharacterThen(s, func(w *world.World, s *world.Session) {
		w.SaveCargoThen(s, func(w *world.World, s *world.Session) {
			if e := w.Entity(s.Conn); e != nil {
				e.Mode = world.MobUserDock
				// O save acima já capturou os buffs deste personagem; remover
				// eles da entidade por conexão para não vazarem para
				// o próximo personagem selecionado nesta sessão (issue #21/#47).
				e.ResetAffects()
			}
			// Descartar qualquer loja pessoal aberta (issue #115): o RemoveMob acima já
			// limpou a pose de banca para os espectadores, então só limpar o
			// estado de sessão para não vazar para o próximo personagem desta conexão.
			s.AutoTrade = nil
			s.TradeMode = 0
			s.Mode = world.UserSelChar
			w.Send(s, protocol.MsgCNFCharacterLogout, nil)
			if after != nil {
				after(w, s)
			}
		})
	})
}

// returnPersistedCharacterToSelection completa uma transição cujo snapshot
// de personagem já foi commitado fora do loop. Deliberadamente não salva o
// personagem de novo, evitando um segundo ponto de falha depois de publicar o snapshot.
func (d *Dispatcher) returnPersistedCharacterToSelection(w *world.World, s *world.Session, after func(*world.World, *world.Session)) {
	d.SessionEnd(w, s)
	body := protocol.EncodeRemoveMobBody(2)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgRemoveMob, ID: uint16(s.Conn)}, body)
	})
	w.SaveCargoThen(s, func(w *world.World, s *world.Session) {
		if e := w.Entity(s.Conn); e != nil {
			e.Mode = world.MobUserDock
			e.ResetAffects()
		}
		s.AutoTrade = nil
		s.TradeMode = 0
		s.Mode = world.UserSelChar
		w.Send(s, protocol.MsgCNFCharacterLogout, nil)
		if after != nil {
			after(w, s)
		}
	})
}

// restart trata _MSG_Restart (0x0289): o botão de respawn pós-morte / recall para a cidade
// (TMSrv/_MSG_Restart.cpp). Ele revive o personagem com 2 HP (NÃO uma cura total —
// o recall sempre custa descer a 2 HP, mesmo vivo) e o traz de volta ao
// spawn da última cidade. É assim que um jogador se levanta depois que uma criatura o mata (mobai.go).
//
// UNVERIFIED / adiado: os destinos por região capital de cada clã do original
// (caixas de coordenadas dos clãs 7/8) e a lógica exata de ponto de save do DoRecall não são
// reproduzidas — trazemos para o spawn padrão da última cidade. O
// _MSG_SetHpMp (0x0181) dedicado carrega os alvos de pedido de HP/MP; enviá-lo depois do
// refresh de score para as barras do cliente saltarem para o estado pós-restart.
func (d *Dispatcher) restart(w *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	e.HP = 2         // reviver (CurrentScore.Hp = 2)
	s.CrackError = 0 // NumError = 0
	d.sendScore(w, s, e)
	s.ReqHp = e.HP
	setReqMp(s, e)
	d.sendSetHpMp(w, s, e)

	d.recall(w, s, e)
	d.sendEtc(w, s, e) // SendEtc (gold + ScoreBonus)
}

func (d *Dispatcher) recall(w *world.World, s *world.Session, e *world.Entity) {
	e.QuestFlag = 0
	rx, ry := world.CitySpawn(int(e.LastCity))
	d.doTeleport(w, s, rx, ry)
}

// mountEquipSlot é o slot de equipamento de montaria (Equip[14]). Montarias são adquiridas no jogo
// (captura/loja, recompensa de quest, a troca da esfera Perzen — docs/migration/handlers/
// npc-map.md, _MSG_Quest-npcs.md) — nenhuma classe começa com uma, então starterEquip nunca
// popula este slot.
const mountEquipSlot = 14

// equipEmpty informa se um personagem não tem nenhum item equipado (um novo ou
// nunca semeado), então o equipamento inicial deve ser concedido.
func equipEmpty(equip [world.MaxEquip]world.Item) bool {
	for _, it := range equip {
		if !it.Empty() {
			return false
		}
	}
	return true
}

// repairEquip move qualquer item parado num slot de equipamento ao qual não pertence (nPos
// incompatível — um consumível no slot do corpo, uma montaria na mão de arma) de volta ao
// inventário, e restaura o item de corpo da classe se o slot 0 ficou vazio. Isto cura
// personagens corrompidos antes de existir validação de slot de equipamento; para personagens limpos é
// no-op.
func (d *Dispatcher) repairEquip(st *world.CharacterState) {
	for s := 0; s < world.MaxEquip; s++ {
		it := st.Equip[s]
		if it.Empty() || d.canEquipSlot(it.Index, s) {
			continue
		}
		if dst := firstEmptyCarry(&st.Carry); dst >= 0 {
			st.Carry[dst] = it // preservar o item deslocado
		}
		st.Equip[s] = world.Item{}
	}
	if st.Equip[0].Empty() {
		st.Equip[0] = d.classBody(st.Class) // restaurar a aparência da classe
	}
}

// classBody devolve o item de corpo da classe (Equip slot 0 do template), o item que dá
// aparência a TK/FM/BM/HT. Vazio quando o template da classe não está disponível.
func (d *Dispatcher) classBody(class int) world.Item {
	return d.starterEquip(class)[0]
}

// starterEquip monta o equipamento inicial de personagem novo para uma classe: o
// equipamento do template (o item de corpo @slot 0 — que dirige a aparência da classe —
// mais armadura e arma). Devolve um conjunto vazio se o template da classe
// não estiver disponível. O slot de montaria (14) fica vazio de propósito — igual a todo
// template BaseMob entregue — já que montarias são adquiridas no jogo, não concedidas.
func (d *Dispatcher) starterEquip(class int) [world.MaxEquip]world.Item {
	var eq [world.MaxEquip]world.Item
	if tmpl, ok := d.baseMobs[class]; ok && len(tmpl) == content.BaseMobSize {
		for i, it := range protocol.MobEquip(tmpl) {
			eq[i] = world.Item{
				Index: int16(it.Index),
				Effects: [3]world.Effect{
					{Effect: it.Eff[0][0], Value: it.Eff[0][1]},
					{Effect: it.Eff[1][0], Value: it.Eff[1][1]},
					{Effect: it.Eff[2][0], Value: it.Eff[2][1]},
				},
			}
		}
	}
	return eq
}

// grantStarterCarry coloca o inventário inicial do template da classe (poções de HP/MP,
// esfera da sorte, baú de exp — STRUCT_MOB.Carry@268) nos primeiros slots
// vazios do carry do personagem, preservando o que já está lá. No-op se o template da classe
// não estiver disponível.
func (d *Dispatcher) grantStarterCarry(carry *[world.MaxCarry]world.Item, class int) {
	tmpl, ok := d.baseMobs[class]
	if !ok || len(tmpl) != content.BaseMobSize {
		return
	}
	for _, it := range protocol.MobCarry(tmpl) {
		if it.Index == 0 {
			continue
		}
		dst := firstEmptyCarry(carry)
		if dst < 0 {
			return // inventário cheio
		}
		carry[dst] = world.Item{
			Index: int16(it.Index),
			Effects: [3]world.Effect{
				{Effect: it.Eff[0][0], Value: it.Eff[0][1]},
				{Effect: it.Eff[1][0], Value: it.Eff[1][1]},
				{Effect: it.Eff[2][0], Value: it.Eff[2][1]},
			},
		}
	}
}

// firstEmptyCarry devolve o índice do primeiro slot vazio do inventário, ou -1.
func firstEmptyCarry(carry *[world.MaxCarry]world.Item) int {
	return firstEmptyCarrySlot(carry[:], carryLimit(carry[:]))
}

// validCharName aproxima BASE_CheckValidString.
//
// UNVERIFIED: o conjunto exato de caracteres permitidos / regras de palavrões não estão na
// fonte (handlers/_MSG_CreateCharacter.md). Isto aceita 1..15 caracteres de
// letras ASCII, dígitos e sublinhado como espaço reservado conservador.
func validCharName(name string) bool {
	if len(name) == 0 || len(name) > 15 {
		return false
	}
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '_' || r == '-'
		if !ok {
			return false
		}
	}
	return true
}
