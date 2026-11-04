# Issue 08 — Criação e seleção de personagem com equip inicial

**Fase:** 2 · **Depende de:** 07 · **Bloqueia:** 09, 11 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Criar personagens nas **quatro classes** (TK, FM, BM, HT) e provar que entram no mundo com aparência,
equipamento inicial e atributos corretos na tela de seleção e dentro do jogo.

## Contexto / evidência

- `handler/character.go` cria o personagem e copia o template de classe de
  `content.LoadBaseMobs` (`Release/DBsrv/run/BaseMob/{TK,FM,BM,HT}`, 816 B cada) para `STRUCT_MOB`.
- Regras de partida já implementadas: **1.000.000 de gold**, equip da classe (o item de corpo no slot
  0 dá a aparência da classe), montaria **Shire (item 342) no slot 14**, inventário inicial com
  poções/Esfera da Sorte/Baú de EXP.
- `SELCHAR` envia `level-1` **apenas no preview** para compensar o display one-based do cliente
  (bug B5, resolvido) — não "corrija" isso.
- Atributos usam o modelo Base↔Current (`deriveBaseScore`, `refreshEquip`): o valor carregado é o
  baseline e o equipamento soma por cima.
- Sem BaseMob, o login cai num snapshot relacional sem equipamento (personagem "pelado").

## Passos

1. Validar o pré-requisito: log `base mob templates loaded classes=4` (issue 04).
2. No cliente, criar um personagem de cada classe. Nomes distintos, sem acento.
3. Para cada um, conferir na tela de seleção: nome, classe visual, nível exibido (1), HP/MP e
   STR/INT/DEX/CON plausíveis.
4. Entrar com cada personagem e conferir: aparência da classe, montaria, inventário inicial, gold.
5. Repetir de forma automatizada (opcional, recomendado): harness com
   `MsgCreateCharacter` → `MsgCharacterLogin`, reaproveitando o codec de `internal/protocol`.
6. Conferir no banco que os quatro existem e que o equip persistiu.

## Arquivos

- Consulta: `tmserver/internal/handler/character.go`, `handler/login.go` (`selChars`),
  `handler/score_derive.go`, `protocol/selchar.go`.
- Dados: `Release/DBsrv/run/BaseMob/*` (não editar sem issue de balanceamento).
- Proibido: alterar `SELCHAR` para "corrigir" o nível; o off-by-one é intencional.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
# 1. personagens criados
docker compose exec -T db psql -U postgres -c \
  "select name, class, level, exp, coin from character order by name;"

# 2. equip inicial (slot 0 = corpo da classe, slot 14 = montaria 342)
docker compose exec -T db psql -U postgres -c \
  "select c.name, count(*) from item i join character c on c.id=i.character_id where i.slot < 16 group by c.name order by c.name;"

# 3. servidor serviu o login de personagem
docker compose logs --since 10m tmserver | grep -E "recv packet.*0x0213"
```

Saída esperada: 4 linhas em `character` (uma por classe), com `gold = 1000000`; equip com item de
corpo e item 342; log com `0x0213` para cada personagem.

## Critérios de aceite

- [ ] 4 personagens criados (TK, FM, BM, HT) sem erro no cliente
- [ ] Aparência de classe correta na seleção **e** no mundo
- [ ] Equip inicial presente (corpo + montaria) e persistido
- [ ] Gold inicial 1.000.000 em todos
- [ ] Preview de atributos coerente (sem nível "2" ou nível 0)
- [ ] Evidência (log + consulta SQL) anexada

## Riscos / rollback

- **Risco:** BaseMob não carregado → char sem visual e reclamação de "classe errada". **Mitigação:**
  checar log antes; se faltar, é issue 04.
- **Risco:** nome inválido/duplicado silencioso no cliente. **Mitigação:** usar nomes simples e
  conferir resposta do servidor no log.
- **Rollback:** deletar personagem pelo cliente (`MsgDeleteCharacter`) — não apagar linha no banco.
