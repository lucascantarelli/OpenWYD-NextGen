package world

// MaxAffect é o número de slots de affect (buff/debuff) por entidade
// (STRUCT_AFFECT Affect[MAX_AFFECT], captura-wyd-affect-divina.md §A).
const MaxAffect = 32

// Tipos de affect aos quais o modelo de score reage (BASE_GetCurrentScore, captura §C).
const (
	AffectExpChest       = 39 // Baú de Experiência: +100 ExpBonus (Basedef.cpp:4451)
	AffectDivine         = 34 // Poção Divina: +20% MaxHp/MaxMp/Damage
	AffectVigor          = 35 // Poção de Vigor: +10% MaxHp/MaxMp
	AffectForceMobDamage = 30 // Frango Assado: +2000 de dano fixo em criatura, 4h (Basedef.cpp:4427)
)

// Affect espelha STRUCT_AFFECT (8 bytes): um buff/debuff temporizado. Type==0 é um
// slot vazio. Para o buff Divino o prazo real é Entity.DivineEnd (wall-clock); o
// campo Affect.Time é só o timer do ícone no cliente.
type Affect struct {
	Type  uint8
	Value uint8
	Level uint16
	Time  uint32
}

// EmptyAffect devolve o slot que já contém o affect do tipo t, senão o primeiro slot
// livre, senão -1 quando cheio — espelha GetEmptyAffect (GetFunc.cpp:734).
func (e *Entity) EmptyAffect(t uint8) int {
	for i := range e.Affect {
		if e.Affect[i].Type == t {
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

// ResetAffects limpa todo slot de buff/debuff e o estado derivado de affect
// (caches + Rsv + DivineEnd). A Entity do jogador vive enquanto a
// CONEXÃO, não o personagem (alocada uma vez por conexão TCP, reutilizada
// entre trocas de seleção de personagem), então tanto o login de personagem (antes de
// reidratar os slots persistidos) quanto o retorno à tela de seleção precisam chamar isto;
// caso contrário um buff lançado num personagem sobrevive à troca, aplica-se a um
// personagem diferente e ainda é persistido nele pelo próximo save de logout.
func (e *Entity) ResetAffects() {
	e.Affect = [MaxAffect]Affect{}
	e.DivineEnd = 0
	e.Rsv = 0
	e.AffDamage, e.AffAC, e.AffMaxHP, e.AffMaxMP, e.AffRunSpeed, e.AffAttackSpeed, e.AffExpBonus = 0, 0, 0, 0, 0, 0, 0
	e.AffStr, e.AffInt, e.AffDex, e.AffCon, e.AffCritical = 0, 0, 0, 0, 0
	e.AffForceDamage, e.AffForceMobDamage = 0, 0
	e.AffHpAbs, e.AffMagic = 0, 0
	e.AffSpecial = [4]int16{}
	e.AffResist = [4]int16{}
	e.AffDamageMultiPct = 100
}

// HasAnyAffect informa se algum slot contém um buff/debuff (Type != 0).
func (e *Entity) HasAnyAffect() bool {
	for i := range e.Affect {
		if e.Affect[i].Type != 0 {
			return true
		}
	}
	return false
}

// HasAffect informa se algum slot contém o affect do tipo t.
func (e *Entity) HasAffect(t uint8) bool {
	for i := range e.Affect {
		if e.Affect[i].Type == t {
			return true
		}
	}
	return false
}

// ClearAffect remove todo slot que contém o affect do tipo t.
func (e *Entity) ClearAffect(t uint8) {
	for i := range e.Affect {
		if e.Affect[i].Type == t {
			e.Affect[i] = Affect{}
		}
	}
}

// ClearFirstAffect remove o primeiro slot que contém o affect do tipo t.
func (e *Entity) ClearFirstAffect(t uint8) bool {
	for i := range e.Affect {
		if e.Affect[i].Type == t {
			e.Affect[i] = Affect{}
			return true
		}
	}
	return false
}

// Flags de estado Rsv (Basedef.h:244-251) — recalculados a partir dos affects ativos
// pelo refresh de score; o combate e o motor de affect os leem.
const (
	RsvFrost uint8 = 0x01
	RsvDrain uint8 = 0x02
	RsvParry uint8 = 0x08
	RsvHide  uint8 = 0x10
	RsvHaste uint8 = 0x20
	RsvCast  uint8 = 0x40
	RsvBlock uint8 = 0x80
)

// AffectDuration é o ajuste deliberado do lado do servidor de quanto tempo dura um affect
// dura (issue #229) — a mesma classe de divergência da correção do índice do affect 25
// da issue #233. Seu valor zero é fiel ao legado: nada é escalado nem limitado, então
// os testes de paridade ainda exercitam a fórmula original.
//
// Ele substitui a tentativa da issue #92 para o mesmo objetivo, que dividia pela metade o
// divisor de AffectTime do loader. Aquele botão estava no lugar errado e era o botão errado:
// a divisão em tempo de carga é inteira, então truncava a cauda curta da tabela
// muito além da metade pretendida (issue #236), e deixava intacto o termo que
// de fato domina a duração — o multiplicador de maestria (100+Special)/100,
// que vale até 5x no teto de Special, e é por isso que os personagens nível 400 que
// abriram #92, #202 e #229 continuavam vendo buffs de 30-100 min.
//
// Só os affects instalados por um cast passam por aqui. Affects de item, montaria,
// joia de cash e Divino escrevem direto em Entity.Affect com suas próprias
// duras intencionais (7/30 dias, 1 h) e não são tocados.
//
// A escala é deliberadamente NÃO uniforme: ela só toca o que a política existe para
// cortar. Veja withinTargetBand — um primeiro corte aplicava ScalePct a tudo e
// esmagava a cauda curta do SkillData.csv em MinTicks, que é o que a issue #236
// seguia relatando pelo outro lado.
type AffectDuration struct {
	// ScalePct escala a contagem legada de ticks. 0 ou 100 a deixam como está. Aplica-se
	// só FORA da banda alvo (veja withinTargetBand).
	ScalePct int
	// MinTicks impõe um piso ao resultado só para affects NÃO agressivos, para que os
	// buffs amistosos mais curtos (AffectTime bruto 1-15) continuem utilizáveis. Affects
	// hostis ficam de fora para o piso não ALONGAR um debuff curto como Enfraquecer
	// (bruto 5).
	MinTicks int
	// MaxTicks limita o resultado, cortando a cauda de maestria que nenhum ajuste do lado
	// da base alcançaria. Ele também serve como borda da banda alvo.
	MaxTicks int
}

// withinTargetBand informa se a duração BASE de uma linha — seu AffectTime+1, a
// contagem de ticks sem maestria — já cabe na banda alvo da política, caso em que
// não há nada a cortar e a curva de maestria legada é mantida como está.
//
// A pertinência é julgada na base, NÃO na contagem inflada pela maestria, para que ela
// seja uma propriedade da linha de SkillData e constante ao longo de Special. Julgar a
// contagem inflada em vez disso torna a curva não monotônica: Teleporte (bruto 60 → base
// 16) cruza MaxTicks em maestria alta e sai do ramo de identidade para o
// ramo de 15%, o que o fazia cair em 2m08 sem maestria, 6m24 com Special 200 e voltar
// a 1m36 com Special 400 — um buff que fica PIOR quanto mais você investe.
// Não há limiar a ajustar aqui: as durações base no SkillData.csv são 1-16
// ticks e depois 151, nada no meio, então qualquer corte dentro dessa lacuna é o
// mesmo corte.
//
// Só affects amistosos se qualificam. A duração hostil de controle de grupo (Perseguição,
// Lâmina Congelante, Nevasca, Conexão de Gelo — todas base 1-2) é uma questão de
// balanceamento separada que ninguém abriu, e deixar o bypass alcançá-la
// multiplicaria em silêncio as durações de congelamento por até 5x. Mesma restrição que o piso
// de MinTicks já carrega, pelo mesmo motivo.
//
// Com MaxTicks não definido não há banda na qual estar, então nada fica isento.
func (d AffectDuration) withinTargetBand(baseTicks, aggressive int) bool {
	return aggressive == 0 && d.MaxTicks > 0 && baseTicks <= d.MaxTicks
}

// scale aplica a política a uma contagem legada de ticks. baseTicks é a contagem do mesmo
// affect sem maestria, que seleciona o ramo (veja withinTargetBand). Um affect
// aplicado nunca vira 0 ticks — isso instalaria um buff que a próxima varredura
// apagaria.
func (d AffectDuration) scale(ticks, baseTicks, aggressive int) int {
	if d.ScalePct > 0 && d.ScalePct != 100 && !d.withinTargetBand(baseTicks, aggressive) {
		ticks = ticks * d.ScalePct / 100
	}
	if d.MaxTicks > 0 && ticks > d.MaxTicks {
		ticks = d.MaxTicks
	}
	if aggressive == 0 && ticks < d.MinTicks {
		ticks = d.MinTicks
	}
	if ticks < 1 {
		ticks = 1
	}
	return ticks
}

// SetAffect é o SetAffect legado (Server.cpp:9209): instala um affect temporizado
// a partir de um cast. Só JOGADORES recebem affects (`conn > MAX_USER` devolve FALSE);
// casts agressivos batem em RSV_BLOCK. time é o Delay do cast (100+Special);
// o Time guardado é (affectTime+1)*time/100 ticks (um tick = 8s de tempo real),
// depois ajustado por dur. Um slot reutilizado cujo tipo ANTERIOR é 1/3/10 recebe o
// timer curto de 4 ticks (o clamp estranho de sType do legado) e pula o ajuste, assim como
// o sentinela "infinito". Devolve false quando nada foi aplicado.
func (e *Entity) SetAffect(affectType, affectValue, affectTime, aggressive, time, level int, dur AffectDuration) bool {
	if !IsPlayer(e.ID) || e.Merchant == 1 {
		return false
	}
	if e.Rsv&RsvBlock != 0 && aggressive != 0 {
		return false
	}
	if affectType <= 0 {
		return false
	}
	slot := e.EmptyAffect(uint8(affectType))
	if slot < 0 {
		return false
	}
	prev := e.Affect[slot].Type
	t, fixed := (affectTime+1)*time/100, false
	if prev == 1 || prev == 3 || prev == 10 {
		t, fixed = 4, true
	}
	if time >= 2139062143 {
		t, fixed = 2139062143, true
	}
	if !fixed {
		t = dur.scale(t, affectTime+1, aggressive)
	}
	e.Affect[slot] = Affect{Type: uint8(affectType), Value: uint8(affectValue), Level: uint16(level), Time: uint32(t)}
	return true
}

// SetTick é o SetTick legado (Server.cpp:9256): instala um affect periódico
// (HoT 17 / DoT 20 / …). Diferente de SetAffect, CRIATURAS podem carregar ticks (só NPCs
// mercadores ficam de fora); tipos 1/3/10 são limitados a 2 ticks. Affects periódicos compartilham a
// coluna AffectTime com os buffs de cast, então recebem o mesmo ajuste de dur.
func (e *Entity) SetTick(tickType, tickValue, affectTime, aggressive, delay, level int, dur AffectDuration) bool {
	if e.Merchant == 1 && !IsPlayer(e.ID) {
		return false
	}
	if e.Rsv&RsvBlock != 0 && aggressive != 0 {
		return false
	}
	if tickType <= 0 {
		return false
	}
	slot := e.EmptyAffect(uint8(tickType))
	if slot < 0 {
		return false
	}
	t, fixed := delay*(affectTime+1)/100, false
	if delay >= 500000000 {
		t, fixed = 500000000, true
	}
	if t >= 3 && (tickType == 1 || tickType == 3 || tickType == 10) {
		t, fixed = 2, true
	}
	if !fixed {
		t = dur.scale(t, affectTime+1, aggressive)
	}
	e.Affect[slot] = Affect{Type: uint8(tickType), Value: uint8(tickValue), Level: uint16(level), Time: uint32(t)}
	return true
}
