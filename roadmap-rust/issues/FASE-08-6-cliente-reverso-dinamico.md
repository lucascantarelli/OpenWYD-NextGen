# Issue 08-6 — Engenharia reversa dinâmica e correlação

**Fase:** 8 · **Depende de:** 08-5 (mapa de funções), 08-2 (parsers) · **Bloqueia:** 08-7 · **Esforço:** L · **Status:** 🚧

## Objetivo

Confirmar em execução o que o desassemble estático propôs: capturar pacotes reais, observar o cliente processando cada opcode e produzir goldens que o servidor Rust reproduza byte a byte.

## Contexto / evidência

O servidor emite o dialeto 7662, mas não há prova de ida-e-volta contra o binário em todos os pacotes. `docs/cliente/depuracao.md` §4 registra freeze com o cliente ACKando TCP mas descartando tudo — `session last sends` como post-mortem. T2 (structs de jogo) foi transferido da 08-5 para cá por limitação de ferramenta.

## Resultados desta rodada (2026-11-07)

| Entrega | Evidência |
|---|---|
| Roteiro de trace | `docs/cliente/depuracao.md`: VM, breakpoints (VAs da 08-5), fases 0–6 login→campo, protocolo de reprodução do freeze, higiene de fixtures |
| Goldens gerados C→S | `fixtures/cliente/*.json` — 5/5 válidos (`captura.mjs --check`) |
| Validador | `tools/client/captura.mjs` (`--gen`/`--check`); opcode cruzado com a tabela gerada de `opcodes.rs` |
| Catálogo | `docs/cliente/pacotes.md`: 5 pacotes C→S **GERADO** + 5 S→C **PENDENTE** |
| Determinismo | duas execuções, artefatos byte-idênticos |

Os goldens provam o **codec**, não o cliente: bytes sintéticos (conta `GOLDEN`) com a matemática exata do CPSock.

## Resultados desta rodada (2026-11-08)

**A captura ao vivo continua NÃO executada** — e isso é medição, não opinião: o roteiro exige
o cliente `client/` rodando contra um servidor que complete login → seleção → campo. O
`wyd-net` existe, mas ainda não fecha o fluxo de jogo (FASE-07/08 e o gateway FASE-07-1 em
andamento); sem servidor atendendo o handshake completo, o cliente não chega ao campo e o
trace capturaria só a tentativa de conexão. Nenhum `.pcap` foi gerado nesta rodada.

O que foi possível provar sem a VM, e agora está congelado por teste:

| Prova | Onde | O que garante |
|---|---|---|
| Round-trip byte-exato dos 5 goldens C→S | `crates/wyd-protocol/tests/golden.rs` (9 testes) | corte → descortinado → checksum `Strict` → opcode → payload → **reselagem idêntica ao `wireHex`** |
| Offsets do handshake × ADR-0020 | `golden_handshake_offsets_confirmam_a_adr_0020` | `Size` 116 B, `ClientVersion` i32 em **offset 92** com valor **12 000**, senha em 12, conta em 24, zero em 40, `db_need_save` em 96, `adapter_name` em 100 — se o layout ou o valor mudar, o gate da ADR deixa de valer e o teste acusa |
| Gate do login sobre o golden | `crates/wyd-net/src/session/handshake.rs` (4 testes) | 12 000 aceito, 7640 recusado com erro tipado e métrica por versão |

Ou seja: do handshake, **o que o servidor precisa reproduzir está provado byte a byte**;
o que falta é a confirmação de que o **cliente** produz esses bytes ao vivo — exatamente o
item bloqueado abaixo.

## Checklist de captura — aba de cidadania/atributos (2026-11-09)

A segunda aba da seleção do cliente de referência mostra mais do que as quatro linhas do 7662:
cidadania, atributos e estado do personagem. Para o servidor Rust preencher isso, faltam **bytes**,
não decisão: cada campo abaixo já tem verdade no nosso banco (`crates/wyd-db/migrations/0002_personagens.sql`)
ou na base de conhecimento, mas nenhum tem offset confirmado no corpo de `CNFAccountLogin` (2008 B)
ou de `CNFCharacterLogin`. Nada aqui vira tipo antes da captura (invariante 5).

### `STRUCT_SELCHAR` (840 B no 7662; 904 B no fork v769 — `superficie-conhecida.md` §6)

| # | Campo da aba | Nossa verdade hoje | O que a captura precisa provar |
|---|---|---|---|
| 1 | nome | `characters.name` (≤ 16 B no `Create/Delete`) | offset, encoding e tamanho do campo no struct (não no pacote C→S) |
| 2 | classe (4 classes) | `characters.class` 0..=3 | offset e largura (byte vs short); índice de cada classe |
| 3 | nível | `characters.level` | offset/largura; `STRUCT_SCORE.Level` é int no 7662 e short no v769 |
| 4 | EXP atual e próximo nível | `characters.exp` | offset/largura (i64 no banco, provavelmente 32 bits no fio) |
| 5 | HP/MP atuais e máximos | `characters.hp`/`mp` | se os máximos vêm no selchar ou só no `STRUCT_MOB` |
| 6 | tier / camada (Mortal→Nightmare) | `characters.tier` + `arch_of_id` | se o cliente 7662 conhece o campo ou se é extensão nossa (não inventar) |
| 7 | cidadania (`Citizen = ServerIndex + 1`) | derivada do canal, `docs/wyd/canais-e-cidadania.md` §2 | offset, tipo e se o cliente desenha reino pelo id ou por tabela |
| 8 | guilda (id + nome) | `0004_guildas.sql` | layout do par id/nome e limite de bytes do nome |
| 9 | posição (mapa/x/y) | `characters.pos_x/pos_y` | se o selchar traz posição (provável: só `STRUCT_MOB`) |
| 10 | PvP/PK e kills | fora do banco ainda | existência do campo e faixa (o cliente pinta o nome por ele) |
| 11 | equipamento visível (arma/armadura/escudo/capa) | `0003_inventario.sql` (`STRUCT_ITEM` 4 B + efeitos) | quais slots o selchar carrega e em que ordem |
| 12 | atributos base (str/dex/int/con) e pontos livres | fora do banco (progressão é FASE-10) | quantos bytes por atributo e se vêm no selchar |
| 13 | bônus/efeitos ativos (buff, drop, EXP) | fora do banco | se existe máscara de flags e onde |
| 14 | data de criação / último acesso | `characters.created_at`/`updated_at` | se o cliente mostra e de que campo lê |

### `STRUCT_MOB` (corpo de `CNFCharacterLogin`, tamanho desconhecido no 7662)

| # | Campo | Nossa verdade hoje | O que a captura precisa provar |
|---|---|---|---|
| 1 | tamanho total do struct e do pacote | `pacotes.md`: `STRUCT_MOB + cauda` **PENDENTE** | `Size` real do frame e se há cauda variável |
| 2 | `STRUCT_SCORE` (nível, str/dex/int/con, HP/MP, resistências) | fórmula é FASE-10 | ordem e largura de cada membro |
| 3 | inventário/equipamento | `0003_inventario.sql` | `MAX_EQUIP` 16 (7662) vs 18 (v769) — não misturar |
| 4 | moedas (gold) | `0010_economia.sql` | posição do campo e tipo |
| 5 | guilda/cidadania no mob | `0004_guildas.sql` | duplicação em relação ao selchar (qual o cliente usa para desenhar) |
| 6 | posição e direção iniciais | `characters.pos_x/pos_y` | offsets e ordem (x,y vs y,x) |

**Runbook operacional:** [`docs/cliente/captura-selchar.md`](../../docs/cliente/captura-selchar.md) —
apontar o cliente para o proxy, capturar dois estados, rodar `tools/client/diff-captura.mjs` e
congelar o achado (corpo do PIN mascarado na conversão).

### Protocolo de captura (ordem mínima, sem inventar)

1. Subir canal com um personagem de cada classe/tier e **variar um campo por vez** no banco
   (nível, exp, cidadania, classe) entre dois logins.
2. Capturar `CNFAccountLogin` (0x010A) e `CNFCharacterLogin` (0x0114) nos dois estados e fazer
   **diff de bytes** — o campo que mudou junto com o valor é o campo (correlação, não palpite).
3. Congelar cada achado como golden em `fixtures/cliente/` com `captura.mjs --check`, marcando
   CONFIRMADO só o que o diff provou; o resto continua UNVERIFIED.
4. Só então abrir issue de corpo de pacote (`payload/`) com offsets e teste de ida-e-volta.

Aceite da captura: os 14 campos do selchar classificados (confirmado/ausente naquele build) e o
tamanho exato de `STRUCT_MOB`; qualquer campo que o cliente 7662 não tenha fica registrado como
"não existe neste build", não como "não implementado".

## O que falta (bloqueio declarado)

Captura ao vivo (§6 de `depuracao.md`): exige servidor executável falando 7662 (crate `wyd-net`/`wyd-core` ainda não existem) + VM Windows com D3D9 + executor humano. Passos concretos quando desbloquear:

1. Fases 0–6: INITCODE, AccountLogin, CNFAccountLogin (`Size` exato), CharacterLogin, burst CreateMob/ESCENE_FIELD.
2. Comparar cada frame com o golden gerado; divergência no C→S = servidor errado; em S→C = fechar struct.
3. Reproduzir o freeze (§4): tail de pacotes com `Size` anômalo = encoder culpado; stream íntegro com descarte = `RecvQueue`/NPC da área.
4. Higienizar (conta/senha → placeholders), versionar, `--check`.

## Arquivos

- Criados: `docs/cliente/depuracao.md`, `docs/cliente/pacotes.md`, `tools/client/captura.mjs`, `fixtures/cliente/*.json`, `docs/cliente/gerado/goldens.json`.
- Alterados: `docs/cliente/README.md`, `superficie-conhecida.md`, `engenharia-reversa.md`, `tools/client/README.md`, skill `wyd-client-reverso`, `roadmap-rust/index.md`, `docs/rust/03-progresso.md`.
- Proibido tocar: `client/`, tabelas de personagem vivo, `.pcap`/dumps na árvore.

## Testes / verificação

```bash
node tools/client/captura.mjs --gen && node tools/client/captura.mjs --check
# esperado: 5/5 válidos; exit 0; exit 1 em divergência
cargo test --workspace && cargo clippy --workspace -- -D warnings && cargo fmt --all -- --check
```

## Skills Relacionadas do freebuff

- .agents/skills/wyd-client-reverso - método dinâmico e higiene de fixture.
- .agents/skills/wyd-rust-net - golden e limites de pacote (quando o net existir).
- .agents/skills/wyd-testing-bots - harness e captura automatizada.

## Documentação de referencia

- docs/cliente/depuracao.md - roteiro e bloqueios.
- docs/cliente/pacotes.md - catálogo GERADO/PENDENTE.
- docs/cliente/pacotes.md + crates/wyd-net/tests/ - fluxos esperados login→campo.
- docs/cliente/depuracao.md - sintoma e hipóteses a confirmar.

## Subtasks

- [x] Codificar issue (roteiro + pipeline de goldens)
- [x] Testar issue (`--gen`/`--check` + gate)
- [x] Documentar issue
- [ ] Captura ao vivo (bloqueada: servidor que complete o login + VM; reavaliar após FASE-07)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Roteiro executável por terceiros sem dedução (VAs, fases, comandos, higiene).
- [x] Goldens gerados do C→S com validação automática e determinismo.
- [ ] Login → seleção → campo capturado e reproduzido pelo servidor Rust (ao vivo).
- [ ] Cada pacote S→C com golden e status (CONFIRMADO/UNVERIFIED).
- [ ] Causa do freeze identificada ou reduzida a hipótese com trace.
- [ ] Nenhuma fixture com dado de conta versionada.

## Riscos / rollback

- Risco: XTrap ativo derrubar a depuração → mitigação: diagnóstico T7 antes; VM descartável.
- Risco: captura com dado pessoal vazar → mitigação: sanitização obrigatória; `.pcap` fora do git.
- Rollback: `git checkout -- tools/client/captura.mjs fixtures/cliente docs/cliente`.
