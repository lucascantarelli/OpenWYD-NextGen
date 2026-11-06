# Issue 08 — Criação e seleção de personagem com equip inicial

**Fase:** 2 · **Depende de:** 07 · **Bloqueia:** 09, 11 · **Esforço:** M · **Estado:** 🟡 código provado; falta a conferência visual das quatro classes no cliente (regularizada em 06/11/2026)

## Objetivo

Criar personagens nas **quatro classes** (TK, FM, BM, HT) e provar que entram no mundo com aparência,
equipamento inicial e atributos corretos na tela de seleção e dentro do jogo.

## Evidência medida (05/11/2026, sessão de QA in-game)

Passo **B — criação** da sessão de 05/11 no cliente real, conta `test`:

> 4 personagens (HT/BM/FM/TK) criados com equip inicial e 1M de gold — `✅ issue-08`

Registro na fonte: [`roadmap/index.md`](../index.md) §"Sessão de QA in-game (05/11/2026)", linha
"B — criação". O caminho de código é `tmserver/internal/handler/character.go` copiando o template da
classe de `BaseMob/{TK,FM,BM,HT}` (816 B), com pré-requisito de boot
`base mob templates loaded classes=4`.

Reconfirmação por comando (stack local; nomes de tabela conforme a migração
`0003_rename_character_to_characters`):

```bash
docker compose exec -T db psql -U openwyd_app -c \
  "select name, class, level, coin from characters order by name;"
```

Saída esperada: 4 linhas (uma por classe), `coin = 1000000`.

## O que falta — critério humano, sem código novo

O código não mudou desde 05/11. Falta apenas o roteiro no cliente, com a evidência anexada aqui:

- [ ] conferir na seleção: nome, classe visual, nível exibido (1), HP/MP e STR/INT/DEX/CON plausíveis
- [ ] entrar com cada uma das quatro classes e conferir aparência, montaria, inventário inicial e gold
- [ ] anexar a saída do SQL acima e as linhas `0x0213` do log do tmServer
- [ ] nenhum preview de atributo exibindo nível 0 ou 2

## Contexto / invariantes que continuam valendo

- `SELCHAR` envia `level-1` **apenas no preview** (bug B5, resolvido): não "corrigir".
- Atributos usam o modelo Base↔Current (`deriveBaseScore`, `refreshEquip`).
- Sem BaseMob, o login cai num snapshot relacional sem equipamento (personagem "pelado") — se aparecer,
  é conteúdo/issue 04, não esta issue.
- Proibido alterar `SELCHAR` para "consertar" o nível; o off-by-one é intencional.

## Testes / verificação

```bash
docker compose exec -T db psql -U openwyd_app -c \
  "select name, class, level, coin from characters order by name;"
docker compose logs --since 10m openwyd-nextgen1-tmsrv | grep -E "recv packet.*0x0213"
```

## Critérios de aceite

- [x] 4 personagens criados (TK, FM, BM, HT) sem erro no cliente — 05/11/2026
- [x] Gold inicial 1.000.000 em todos — 05/11/2026
- [x] Equip inicial aplicado (corpo + montaria) — 05/11/2026
- [ ] Aparência de classe correta na seleção **e** no mundo, nas quatro classes
- [ ] Preview de atributos coerente (sem nível "2" ou nível 0)
- [ ] Evidência (log + consulta SQL) anexada nesta issue

## Riscos / rollback

- **Risco:** BaseMob não carregado → char sem visual. **Mitigação:** conferir o log antes; se faltar, é
  issue 04.
- **Risco:** concluir "classe errada" sem olhar a seleção de outra classe. **Mitigação:** conferir as
  quatro na mesma sessão.
- **Rollback:** deletar personagem pelo cliente (`MsgDeleteCharacter`) — não apagar linha no banco.
