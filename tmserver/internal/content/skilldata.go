package content

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Dimensões da tabela de skills (Basedef.h:171/200). MaxSkill é a contagem de skills
// por classe: índice de skill / MaxSkill é a classe dona, e o espaço de índices vai
// até MaxSkillIndex (as skills de Sephira vivem em 96+).
const (
	MaxSkill      = 24
	MaxSkillIndex = 248
)

// affectTimeDivisor escala o AffectTime cru da coluna 12 no carregamento, exatamente
// como o loader legado faz (Basedef.cpp:6708).
//
// Isto já foi 8 — a tentativa da issue #92 de encurtar durações de buff. Esse
// botão estava no lugar errado: a divisão é INTEIRA, então custava à cauda curta
// da tabela muito mais que a metade pretendida (30 cru ia de 30/4=7 → 30/8=3,
// −57%, que é o que a issue #236 relatou), e nunca tocava o termo que
// de fato domina a duração, o multiplicador de maestria (100+Special)/100. O
// ajuste agora vive num lugar só, world.AffectDuration, aplicado quando uma
// CONJURAÇÃO instala o affect — ver issue #229 e world/affect.go.
const affectTimeDivisor = 4

// Spell é uma linha STRUCT_SPELL do SkillData.csv (Basedef.h, carregada por
// BASE_InitializeSkill, Basedef.cpp:6657). Os nomes de campo seguem a struct; os
// bytes de animação de Act[] são descartados (só do lado do cliente). Name é a
// coluna final do CSV, que o servidor legado nunca lia — mantida aqui para logs.
type Spell struct {
	Index             int
	SkillPoint        int // learn cost in skill points
	TargetType        int
	ManaSpent         int // base MP cost (scaled by BASE_GetManaSpent)
	Delay             int
	Range             int
	InstanceType      int // 0 none, 1-5 elemental damage, 6 heal, 7-12 specials
	InstanceValue     int
	TickType          int // affect type applied by SetTick (17 HoT, 20 DoT, …)
	TickValue         int
	AffectType        int // affect type applied by SetAffect
	AffectValue       int
	AffectTime        int // stored ÷affectTimeDivisor, as the legacy loader does post-parse
	InstanceAttribute int
	TickAttribute     int
	Aggressive        int // 1 = hostile (resist roll applies)
	MaxTarget         int
	BParty            int // 1 = party-wide
	AffectResist      int
	Passive           int // 1 = not castable
	Name              string
}

// SkillKind seleciona qual CurrentScore.Special[1..3] conduz uma skill:
// (index % MaxSkill) / 8 + 1 (_MSG_Attack.cpp).
func SkillKind(index int) int { return index%MaxSkill/8 + 1 }

// SkillClass é a classe dona de um índice de skill (index / MaxSkill).
func SkillClass(index int) int { return index / MaxSkill }

// SkillData é o catálogo de skills indexado pelo índice de STRUCT_SPELL (g_pSpell).
type SkillData struct {
	skills map[int]Spell
}

// NewSkillData monta um catálogo a partir de spells explícitas (testes/ferramentas);
// linhas com Index fora da faixa são puladas, como no loader de arquivo.
func NewSkillData(spells []Spell) *SkillData {
	s := &SkillData{skills: make(map[int]Spell, len(spells))}
	for _, sp := range spells {
		if sp.Index < 0 || sp.Index >= MaxSkillIndex {
			continue
		}
		s.skills[sp.Index] = sp
	}
	return s
}

// Get devolve a spell de um índice de skill.
func (s *SkillData) Get(index int) (Spell, bool) { e, ok := s.skills[index]; return e, ok }

// Len devolve o número de skills carregadas.
func (s *SkillData) Len() int { return len(s.skills) }

// LoadSkillData lê SkillData.csv exatamente como BASE_InitializeSkill: o índice
// vem da coluna 0 (NÃO do número da linha — os índices são esparsos além de 149),
// linhas com índice fora de [0,MaxSkillIndex) são puladas, e AffectTime é
// escalado após o parse (÷4, igual ao loader legado).
func LoadSkillData(path string) (*SkillData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open SkillData: %w", err)
	}
	defer f.Close()
	return parseSkillData(f)
}

func parseSkillData(r io.Reader) (*SkillData, error) {
	s := &SkillData{skills: make(map[int]Spell)}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		// O sscanf legado consome exatamente 22 tokens: 13 ints, as 2 strings
		// Act pontuadas, depois 7 ints. As linhas têm 23 ou 24 colunas; o que estiver
		// entre Passive e Name nas linhas de 24 colunas é ignorado pelo
		// loader original, então é ignorado aqui também (paridade).
		if len(fields) < 22 {
			continue
		}
		var v [20]int // the 20 numeric conversions, in sscanf order
		ok := true
		for i := 0; i < 20; i++ {
			col := i
			if i >= 13 {
				col = i + 2 // skip the two Act columns
			}
			n, err := strconv.Atoi(strings.TrimSpace(fields[col]))
			if err != nil {
				ok = false
				break
			}
			v[i] = n
		}
		if !ok || v[0] < 0 || v[0] >= MaxSkillIndex {
			continue
		}
		s.skills[v[0]] = Spell{
			Index: v[0], SkillPoint: v[1], TargetType: v[2], ManaSpent: v[3],
			Delay: v[4], Range: v[5], InstanceType: v[6], InstanceValue: v[7],
			TickType: v[8], TickValue: v[9], AffectType: v[10], AffectValue: v[11],
			AffectTime:        v[12] / affectTimeDivisor,
			InstanceAttribute: v[13], TickAttribute: v[14], Aggressive: v[15],
			MaxTarget: v[16], BParty: v[17], AffectResist: v[18], Passive: v[19],
			Name: strings.TrimSpace(fields[len(fields)-1]),
		}
	}
	return s, sc.Err()
}
