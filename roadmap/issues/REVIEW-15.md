# REVIEW-15 — Revisão permanente: segurança e superfície de ataque

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle de autenticação, autorização, segredos, transporte, auditoria de operação
administrativa e resistência a abuso — incluindo o fato de que o cliente é fechado e modificável.

## Escopo

- Senhas e PINs: argon2id obrigatório, nenhum texto plano em log ou dump.
- Sessão do portal: `SESSION_SECRET` fora de dev, cookie httpOnly, middleware default-deny.
- Autorização por papel: `admin` × `moderator` × `player`, com teste de 401/403 por rota.
- Segredos: `.env` fora do git, `gitleaks`, certificados mTLS, chaves do compose.
- Superfície do jogo: cliente pode mentir em pacote; toda decisão sensível é do servidor.
- Rate limit de conexão e de mensagem; banimento e auditoria de ação administrativa.
- Dependências: `govulncheck`, `npm audit`, Dependabot.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Segredo versionado | `bash scripts/check-secrets.sh` (deve reprovar valor sintético) |
| Vulnerabilidade Go | `make vuln` |
| Vulnerabilidade portal | `cd portal && npm audit --omit=dev` |
| Rotas sem guarda | `npx vitest run middleware.test.ts` |
| Texto plano de credencial | `git grep -n 'password' -- '*.go' \| grep -v _test` |
| Achados novos | severidade, impacto, reprodutibilidade, correção proposta |

## Critérios de aceite (por rodada)

Rodada de 06/11/2026 (onda 2), com a evidência de cada linha:

- [x] `check-secrets` verde e prova de vermelho registrada — RC 0 na árvore; RC 1 com `.pem`
  versionado e com token `ghp_` plantado (acima)
- [x] Nenhuma rota do portal fora do `matcher` sem justificativa — `npx vitest run
  src/middleware.test.ts`: 11 testes, incluindo "toda página de (admin) está no matcher", "nenhum padrão
  do matcher aponta para rota inexistente" e a regressão de 06/11 (`/precos`, `/atlas`, `/guildas`,
  `/castelo`)
- [x] `moderator_id` vindo da sessão — ponto único em `portal/src/server/admin.ts:83`
  (`moderator_id: session.accountId`, com `Omit` no tipo do request); **sem teste dedicado** para essa
  injeção, o que fica registrado como lacuna menor
- [x] Achado de severidade alta com issue própria no mesmo dia — não houve achado alto nesta rodada
  (o mais grave é média); nada a abrir

## Rodada 06/11/2026 (onda 2) — medido, não estimado

Ferramentas instaladas no host para não ficar em "não medido": `gitleaks v8.28.0` e
`govulncheck v1.9.0` em `.tools/bin` (`GOBIN=$PWD/.tools/bin go install …`).

| Item (tabela de medição) | Comando | Resultado |
|---|---|---|
| Segredo versionado | `bash scripts/check-secrets.sh` | **verde**; vermelho provado 2×: `.pem` versionado → RC 1, token `ghp_` plantado → RC 1, árvore limpa → RC 0 |
| Scan da árvore | `.tools/bin/gitleaks dir . --config .gitleaks.toml --redact` | `no leaks found`, 42,27 MB em 14,8 s |
| Vulnerabilidade Go | `govulncheck ./...` | 0 afetando o código, 0 em pacotes importados, 3 em módulos exigidos **não chamados** |
| Vulnerabilidade portal | `cd portal && npm audit --omit=dev` | `found 0 vulnerabilities` |
| Rotas sem guarda | `npx vitest run src/middleware.test.ts` | 11 testes passando |
| Texto plano de credencial | `git grep -n 'password' -- '*.go' \| grep -v _test` | 88 ocorrências, **todas** nome de campo protobuf/comentário; nenhum literal de senha |

### Achados e destino

| # | Achado | Severidade | Destino |
|---|---|---|---|
| 1 | `check-secrets` não olhava certificado/chave versionada nem permitia detectar allowlist cega | média | **corrigido**: 3 checagens novas (chave versionada, allowlist × índice, tokens GitLab/Stripe/npm/PAT) + cabeçalho documentando cobertura |
| 2 | gitleaks ignora arquivo > 5 MB: `seed_npc_definition.up.sql` (7 MB) e `HeightMap.dat` (16 MB) não são escaneados | baixa | **aceito**: o primeiro é blob de template NPC gerado do conteúdo, o segundo é mapa binário; nenhum dos dois recebe valor de segredo. O CI usa o mesmo `--max-target-megabytes 5`, então o limite é conhecido, não silencioso |
| 3 | `importcontent` com `-sets all` quebrava: `ItemList.csv`, `SkillData.csv`, `InitItem.csv` e `npc/` já saíram da árvore (issue 58) | média | **corrigido**: default passa a ser os 5 conjuntos que existem; `all` fica documentado como "árvore externa de backup" |
| 4 | `LevelItem.txt` tem blocos repetidos (318 linhas, 159 pares classe/seq, linha 13 = linha 256) | média | **corrigido**: chave de import vira o ordinal do arquivo (lossless e idempotente) |
| 5 | matriz de drop do `gameconfig` é 4×16 (64 células), não 8×8 como a issue supunha | baixa | **corrigido**: tabela guarda `row/col` medidos; teste fixa 64 e 4×16 |
| 6 | `MobMerc.txt` mora em `Common/Settings/`, não em `TMsrv/run/` como a issue 59 dizia | baixa | **corrigido**: caminho real no importador |

Nenhum achado de severidade alta nesta rodada, então nenhuma issue nova de segurança foi aberta.

### Reverificação no mesmo dia (após a onda 2)

| Item | Resultado medido | Tempo |
|---|---|---|
| `bash scripts/check-secrets.sh` | RC 0, 5 checagens `OK` | 23 s |
| `gitleaks dir .` | `no leaks found`, 40,55 MB | 30 s |
| `govulncheck ./...` | `No vulnerabilities found` / 0 afetando o código | 22 s |
| `cd portal && npm audit --omit=dev` | `found 0 vulnerabilities` | 4 s |
| `npx vitest run src/middleware.test.ts` | 1 arquivo / 11 testes passando | 5 s |
| `git grep -n 'password' -- '*.go' \| grep -v _test` | 88 ocorrências, nenhum literal | — |

O total varrido pelo gitleaks varia entre rodadas (42,27 MB na primeira, 40,55 MB nesta) porque o scan
direto do worktree vê arquivos gerados que o checkout do CI não tem; o veredito (`no leaks found`) e os
arquivos grandes ignorados são os mesmos.

## Histórico

| Data | Achados | Destino |
|---|---|---|
| 06/11/2026 | rodada de plataforma: 6 achados, 5 corrigidos e 1 aceito com justificativa (acima) | este documento + issue 59 |
| 06/11/2026 | `/precos` e `/atlas` fora do `matcher` (corrigido, PORTAL-01); lista `GUARDED` morta removida | done/PORTAL-01 |
| 06/11/2026 | cliente pode mentir no `MSG_Attack`/movimento; validação é toda server-side | coberto por `docs/sistemas/combate.md`; sem issue aberta |
