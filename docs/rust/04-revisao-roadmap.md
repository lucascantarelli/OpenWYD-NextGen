# Revisão do roadmap-rust — cobertura, conformidade e reparos (2026-11-07)

Revisão completa do planejado com as skills vigentes como base
(`wyd-rust-*`, `rust-code-quality`, `software-architecture`,
`solid-design-patterns`, `skill-creator`, `wyd-web-admin-ui`,
`wyd-client-reverso`, `wyd-rust-toolchain`).

## 1. Inventário: 100% das issues planejadas

- `roadmap-rust/issues/`: **60 arquivos** (45 planejados nas FASES 00–15 +
  15 subissues de execução).
- `roadmap-rust/index.md`: referencia os 60 (15 pais + 45 filhos) — **100%**.
- Todas têm: Skills, Subtasks, Riscos/rollback, Arquivos (criados/alterados/
  proibido tocar) e Critérios de aceite. 4 usam "Resultados" em vez de
  "Passos" (02-5, 08-5, 08-6, 08-7) porque já foram **executadas** — ok.
- Painel: **23 ✅ / 5 🚧**, idêntico em `roadmap-rust/index.md` e
  `docs/rust/03-progresso.md` (verificado por diff).

## 2. Conformidade com as skills (amostra verificada)

| Issue | Base | Veredito |
|---|---|---|
| 05-2 (compose/docker) | `wyd-rust-toolchain` | ✅ após reparo (nota Dockerfile por serviço já registrada na issue) |
| 06-2 (handshake/flood) | `wyd-rust-net` | ✅ balde+porte, 10 testes |
| 07-núcleo (admin/billing) | `wyd-rust-db`, `solid` (DIP) | ✅ `BillingPort`, 15 testes |
| 09-1 (auditoria skills) | `skill-creator` | ✅ plano executivo, sem conflito |
| 10-1/10-3 (equações/lab) | `wyd-rust-sim`, `rust-code-quality` | ✅ funções puras, sem float indevido |
| 12-1 (dashboards) | `wyd-rust-obs` | ✅ contrato 06-1 como única fonte |
| 00-2/00-3, 01-2, 11-1, 14-1, 15-x | skills próprias | ✅ estrutura conforme; conteúdo diferido p/ execução (dependem de fases 🚧) |

Caminhos citados nas issues existem no arquivo: `combat/`, `monitoring/`,
`game-rules.md`, `config-ops.md`, `webserver/` (todos OK em
`arquivo Go removido (2026-11)`). Exceções encontradas e **reparadas** (§3).

## 3. Reparos aplicados nesta revisão (implementação iniciada, com prova)

O arquivamento do Go (commit `b0a0394`) esvaziou diretórios que o portal e o
compose pressupõem. Nada disso exigia recriar o portal — bastou restaurar do
histórico git do próprio repo:

1. **`api/web/v1/web.proto` restaurado** (`b0a0394^`, 2046 linhas — o número
   "1294" na skill está desatualizado). Quebrava 3 coisas: teste
   `route.test.ts`, `COPY` do `wyd-portal/Dockerfile` e o runtime gRPC.
   `api/web/v1/README.md` documenta o contrato e como validar.
2. **63 handlers Go citados pelos guias restaurados** no arquivo
   (`crates/wyd-protocol/src/payload/ + crates/wyd-net/src/dispatch/`, todos existiam em
   `b0a0394^:tmserver/internal/handler/`). Preservação de conhecimento
   (`pre-release.md` §1.2): os guias do portal citam esses arquivos.
3. **`wyd-portal/src/server/guias.test.ts` adaptado**: `handlerDir` agora aponta
   para o arquivo (pós-arquivamento `tmserver/` na raiz é conteúdo Rust).
   Adaptação de caminho, não enfraquecimento — a asserção continua exigindo
   que toda citação exista.
4. **`wyd-portal/Dockerfile` resiliente**: copia o proto só se presente (aviso em
   vez de `COPY` quebrado); `WEB_API_PROTO` documentado. Comentários de path
   obsoletos (path de exemplo com prefixo morto) corrigidos em `.env.example` e
   `wyd-portal/.env.example`.

## 4. Prova observável (este turno)

- `portal`: `tsc --noEmit` exit 0; `vitest run` **16 arquivos, 118 testes,
  100% verde** (antes: 114/115 com 2 arquivos falhando).
- Rust: `cargo fmt --check` OK; `clippy -D warnings` OK;
  `cargo test --workspace` verde em todas as suites.
- Compose: `docker compose config` verde (base+dev e prod).
- Sem commit/push (aguarda ordem do dono); sem dependência nova; sem
  recriação do portal.

## 5. Bloqueadores que permanecem (fora do escopo deste turno)

- Transporte async/tokio + rede/vendor (04-3 pg, 06-1, 07-2/07-3/07-4):
  exigem ADR + ambiente com rede.
- Captura ao vivo 08-6: sem servidor executável + VM.
- Desinstalar Go da máquina: exige elevação do dono.
- Imagens Docker não puxadas aqui (`node:22-alpine` sem rede): `build`
  do portal validado por `tsc` + `next build` prévio (`.next/` presente),
  não por build de imagem.

## Adendo 2026-11-07 (pós-revisão): rename + GUI
- Projeto renomeado para **WYD** (decisão do dono; repo `lucascantarelli/wyd`).
  49 arquivos atualizados, zero restos fora de paths locais reais.
- Nova issue **FASE-08-8** (GUI Tauri, ADR-0007 aceita) + skill `wyd-launcher` +
  `docs/cliente/launcher.md`. Inventário passa a **61/61, 23✅/6🚧**.

---

# §6 — Mega revisão de 2026-11-09 (código × roadmap × documentação)

Revisão pedida pelo dono: validar **todas** as issues entregues contra o código, ler a
documentação e as skills, achar inconsistências/gaps/violações e registrar o que falta em
issues, ordenadas por tamanho. Base normativa usada: `AGENTS.md` §3, `docs/rust/02-execucao.md`,
`docs/arquitetura/estrutura-e-padroes.md` (ADR-0002), as 27 skills de `.agents/skills/`,
`docs/wyd/` e `docs/cliente/`.

## 6.1 Método e o que foi realmente verificado

1. **Inventário do código**: 8 crates, 170 arquivos `.rs`, ~39 900 linhas; 31 migrations;
   24 ADRs; 37 arquivos de teste de integração; 15 páginas + 5 route handlers no portal.
2. **Gate executado**: `cargo test --workspace --all-targets` (38 alvos). **Falhou** — ver 6.2.
3. **Checagem de claim por claim** das issues ✅: arquivo citado existe? teste citado existe?
   constante/rota/migration citada existe? Quando a prova exigia Docker/banco/VM, está dito
   abaixo como **não reexecutada** — nunca como “passou”.
4. **Leitura integral** de `docs/rust/*`, `docs/arquitetura/*` (estrutura, protocolo,
   contratos, modelo de banco), `docs/wyd/*` e `docs/cliente/*`; skills Rust, de domínio e
   legadas.
5. **Banco medido** na stack viva (2026-11-09): 70 tabelas com ledger, `item_catalog` 3.220,
   `skill_catalog` 154 (72 classificadas), `region` 52, `level_item` 161, `npc_shop_item` 6.706.

## 6.2 Defeitos e violações encontrados (e o que foi feito)

| # | Achado | Evidência | Ação |
|---|---|---|---|
| D-1 | **Gate vermelho**: **quatro** alvos de `wyd-server` exigiam `DATABASE_URL` e **panicavam** sem ela (`pin_persistencia`, `pin_troca`, `portal_pin`, `selchar`), derrubando `cargo test --workspace` em máquina sem stack | `expect("WYD_TEST_DATABASE_URL…")` nos 4 arquivos; log `EXIT=101` | corrigido: os quatro **pulam com aviso**, mesmo contrato de `wyd-db/tests/sistema/pg.rs` e do macro de `login.rs` |
| D-2 | **Crate fora da árvore normativa**: `wyd-server` existe no workspace (e roda em produção) mas não estava em `AGENTS.md` §0/§1, `README.md`, DAG/árvore de `estrutura-e-padroes.md` nem na tabela do `05-stack-docker.md` §1 | `Cargo.toml` members vs docs | corrigido: linha em AGENTS/README/stack, aresta no DAG e **§16** com a árvore real do crate |
| D-3 | **Status divergente em 8 issues** entre índice, painel e cabeçalho (03-2, 03-4, 04-2, 07-1, 08-7, 08-8, 09-2, 11-1, 11-2, 15-3) | tabelas de `index.md` × `03-progresso.md` | reconciliado nos três lugares; issue 02-6 cria gate para não voltar |
| D-4 | **Número errado de schema**: painel dizia “24 migrations / 59 tabelas” com 31/69 no repo e 70 no banco | `ls migrations`, `information_schema` | corrigido para 31/69 (+ledger) em 2026-11-09; **reaberto em 2026-11-10**: o código tem 36 migrations / 74 tabelas (`check-migrations`, `cargo test -p wyd-db --test contrato_catalogo`, `SCHEMA_VERSION = 36`), e `docs/wyd/README.md` e `modelo-banco.md` foram alinhados a esse número |
| D-5 | **Contagem de skills contraditória**: “69 de 151” e “63 de 154” no mesmo parágrafo, com 82 `unmapped` | `decisoes-servidor.md` §6 | medido: **72 de 154**, 82 unmapped; texto corrigido com a query |
| D-6 | **Doc de operação mentindo**: `AGENTS.md` §2 prometia 502 do `/api/health` “esperado sem gateway”, e `05-stack-docker.md` dizia `tmserver` bloqueado — gateway e dois canais existem e rodam | `docker ps`, `/health` | corrigido nos dois docs |
| D-7 | **Issue 03-4 com status 🚧 apesar de “Concluído” na mesma página** (loader + 2 testes entregues) | `FASE-03-4` §§“Concluído” × “Status” | status ✅ + nota do que falta é o processo (03-7) |
| D-8 | **Issue 03-2 com “ainda aberto” obsoleto** (`systems/`, ADR, bench já entregues) | `FASE-03-2` notas finais | fechamento registrado |
| D-9 | **Issue 03-5 sem teste dedicado?** — falso: `cast` tem 3 testes de recusa/custo em `tests/integracao/autoridade.rs` | checklist 4.3 | checklist corrigido (🚧 só pelo fio) |
| D-10 | **`docs/rust/00-visao-estrutura.md`** descreve 7 crates, 15 skills e Go no PATH como estado “efetivo” | doc de 07/11 | marcado como **snapshot histórico**; estado vigente é o §16 + `03-progresso.md` |

## 6.3 Gaps de implementação (o que o roadmap não tinha registrado)

O achado estrutural da revisão: **o motor não roda em processo nenhum**. `wyd-server` não
depende de `wyd-core`; as regras de 03-5/03-6 passam em teste, mas não há tick, nem spawn, nem
cena no serviço. O checklist já marcava 1.11/4.8/10.4 como ⬜, mas **o roadmap não tinha issue
para isso** — era o maior buraco entre “entregue” e “jogável”.

| Gap | Onde estava escondido | Issue criada |
|---|---|---|
| tick real no canal, métricas de motor reais | implicitamente “FASE-03/04/06” na ADR-0022 | **03-7** |
| personagem como entidade + save na saída | checklist 1.11/3.5 | **03-8** |
| `ESCENE_FIELD` nunca emitido | invariante 4 e checklist 4.8/10.4 | **03-9** |
| cidadania e canal verde sem regra | checklist 2.3 | **03-10** |
| lotação/manutenção de canal | checklist 1.14 | **06-3** |
| 13 serviços do gateway em `501` | “ainda aberto” da 07-1, sem issue | **07-5** (leitura) e **07-6** (operação) |
| logout explícito | checklist 1.13 | **08-9** |
| status divergente crônico | revisão 04 (2026-11-07) já apontava | **02-6** |

## 6.4 Revisão dos sistemas de jogo (itens, skills, cidadania, canais)

**Itens/inventário.** O que existe é sólido no core: `ItemKind`→slot derivado, `req_level`,
política de troca, `max_stack`, gema como transação com evento único (ADR-0012) e paridade com
o seed real (4 gemas, refino 10..=15). O que **não** existe: item no fio (equipar/usar), drop,
loja, armazém, durabilidade e trade — e o `MSG_Trade` segue UNVERIFIED, corretamente fora de
tipos. Gaps de dado: `grade` numérico sem mapeamento provado para N/M/A/Le; `price` sem
semântica de moeda; 22 itens de loja fora do catálogo (FK proibida, decisão certa).

**Skills.** O catálogo tem 154 linhas e **comportamento** (MP, delay, alcance, efeito,
`aggressive`, `passive`); a classificação por classe/linhagem é 72/154. O risco caro apontado
em `skills.md` §7 continua válido e não tem teste: `aggressive` errado transforma cura em PK.
Isso merece teste de dados quando a curadoria avançar — hoje é lacuna de fonte, não de código.

**Cidadania.** Documentada com evidência do legado (`citizen = ServerIndex + 1`, Kibita
4.000.000, `/getout`, bônus de canal), implementada **só como campo** do `STRUCT_SELCHAR`. É o
item 2.3: sem bônus aplicado, cidadania é decoração. Issue 03-10 assume o escopo seguro
(bônus por `game_setting` + papel verde), deixando compra por diálogo de NPC para quando houver
sistema de NPC no fio.

**Canais.** O desenho está correto e provado: 2 processos, mesmo Postgres, sessão única por
conta (0030), `server_channel` como dado, IP próprio porque a porta é fixa, papéis
normal/verde. O que falta: papel verde **não muda comportamento** e lotação/manutenção não
existem. Trânsito entre canais é relogar — e o `_MSG_DBServerChange` do legado **não** deve ser
portado (cada canal tinha DB próprio; o nosso compartilha), decisão já registrada em
`canais-e-cidadania.md` §5.2.

## 6.5 Conformidade com as skills (amostra revisada)

| Área | Veredito |
|---|---|
| `wyd-rust-core` (sem I/O no tick, DOD) | ✅ no core; ⬜ no processo (03-7) |
| `wyd-rust-net` (validar antes de parsear, sem panic) | ✅ `cpsock`/`handshake` sem `unwrap` de entrada; testes de flood/limite |
| `wyd-rust-db` (migrations reversíveis, integridade) | ✅ 31 migrations + catálogo + verify-schema |
| `rust-code-quality` (sem `unwrap` fora de teste) | ✅ nos caminhos de rede; os 351 hits são majoritariamente `#[cfg(test)]`/locks de fake — revisão pontual recomendada no `inventory.rs` (26) |
| `software-architecture` (árvore/ADR) | ❌→✅ após D-2 (`wyd-server` agora no DAG e no §16) |
| `solid-design-patterns` (porta por domínio) | ✅ `ContentSource`, `GrantQueue`, `Accounts`/`Characters` |
| `wyd-rust-testing` (teste pula, golden, sem `#[ignore]`) | ❌→✅ após D-1 |
| `wyd-client-reverso` / `wyd-launcher` | ✅ (hash fixado, patches por ferramenta, GUI só chama o núcleo) |

## 6.6 O que **não** foi reexecutado (limitações declaradas)

- E2E Playwright (`3/3`), `vitest`, `tsc`, imagens Docker, prova em Loki e captura do cliente:
  exigem stack/portal/VM e não foram repetidos nesta passada. Continuam como evidência
  **histórica** das issues, não como verificação desta revisão.
- Gate final desta revisão: `cargo fmt --check` **0**, `cargo clippy --workspace --all-targets
  -- -D warnings` **0**, `cargo test --workspace --all-targets` **0** — **49 alvos
  `test result: ok`, 0 falhas** (os testes de banco imprimem `SKIP` e retornam).

## 6.7 Perguntas abertas para o dono

Consolidadas em `docs/wyd/referencia-externa-2026-11.md` §9 (P-1 a P-8): pontos de skill por
nível, custo do +10 (50kk × 5kk), 8 canais × 2 canais, bônus de cidadania, teto de nível,
escopo do Arch, uso de layout W2PP como provisório e escopo da loja de donate.

## 6.8 Fechamento

- **Issues novas:** 9 (02-6, 03-7, 03-8, 03-9, 03-10, 06-3, 07-5, 07-6, 08-9), com fila por
tamanho em `roadmap-rust/index.md`. Nenhuma depende de captura, de decisão do dono ou de
serviço externo.
- **Issues reabertas/corrigidas:** 03-2 (fechamento), 03-4 (status), 04-2 (números/runner),
  07-1 (split dos handlers).
- **Docs corrigidos:** `AGENTS.md`, `README.md`, `estrutura-e-padroes.md`, `05-stack-docker.md`,
  `03-progresso.md`, `08-checklist-funcionalidades.md`, `docs/wyd/README.md`,
  `docs/wyd/decisoes-servidor.md`, `docs/wyd/fontes.md` + nova `referencia-externa-2026-11.md`.
- **Código:** quatro arquivos de teste (`pin_persistencia.rs`, `pin_troca.rs`, `portal_pin.rs`,
  `selchar.rs`) para restaurar o contrato “sem banco = skip”, sem tocar em asserção alguma.
