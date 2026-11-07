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
`.deprecated-go/`). Exceções encontradas e **reparadas** (§3).

## 3. Reparos aplicados nesta revisão (implementação iniciada, com prova)

O arquivamento do Go (commit `b0a0394`) esvaziou diretórios que o portal e o
compose pressupõem. Nada disso exigia recriar o portal — bastou restaurar do
histórico git do próprio repo:

1. **`api/web/v1/web.proto` restaurado** (`b0a0394^`, 2046 linhas — o número
   "1294" na skill está desatualizado). Quebrava 3 coisas: teste
   `route.test.ts`, `COPY` do `wyd-portal/Dockerfile` e o runtime gRPC.
   `api/web/v1/README.md` documenta o contrato e como validar.
2. **63 handlers Go citados pelos guias restaurados** no arquivo
   (`.deprecated-go/tmserver/internal/handler/`, todos existiam em
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
