# CLIENT-11 — Launcher completo + integração com o portal + autoupdate

> **ID canônico:** `CLIENT-11`
> **ID histórico:** `LAUNCHER-01`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** Código (launcher+portal) · **Esforço:** L · **Depende de:** PROC-24, CLIENT-03 ✅, CLIENT-08 🚧 · **Bloqueia:** —

## Objetivo

Fechar o launcher jogável (config do jogo, patch, download, `check/launch/serverlist/update`) com login unificado, notícias/eventos do portal, geração de patches no admin e autoupdate do próprio launcher.

## Contexto / evidência

- Hoje: `wyd-launcher` (manifest/patch/bootstrap/sha256, `check` real 6/6 OK, `update` atômico com backup, 19/19 testes) + GUI Tauri mínima entregue em 2026-11-12 (CLIENT-08 🚧; faltam barra de progresso e instalador NSIS).
- Portal já tem identidade única (`docs/arquitetura/identidade.tokens` → launcher + portal).
- `prompt.md` pede: notícias/eventos/suporte no launcher; login unificado; sincronização de dados; personalização; patches gerados no admin (versão, arquivos); monitoramento de desempenho do launcher.

## Passos

1. Núcleo: `launch` valida manifesto → `serverlist.bin` atômico com backup → abre o jogo; `update` aplica pacote assinado; `--dry-run` antes de tocar disco; `serverlist` só para troca de canal.
2. Integração portal: launcher lê `/api/launcher/{noticias,eventos,versao}` (público, cache curto) + login unificado via sessão do portal (mesma conta, sem segunda senha; PIN continua no cliente 7662) + sincronização de versão/config.
3. Admin gera patch: `/admin/patches` (versão, arquivos, notas) → pacote assinado + manifesto → download; launcher aplica com backup e rollback.
4. Autoupdate do launcher: canal de versão do próprio binário + assinatura + troca atômica (nunca se auto-sobrescrever em execução sem backup).
5. Telemetria: tempo de patch, falhas por arquivo, versão em uso — agregado no monitoramento (sem PII).

## Arquivos

- Criados: `crates/wyd-launcher/src/{gui,portal_client,autoupdate}.rs`, `wyd-portal/src/app/(admin)/patches/*`, `wyd-portal/src/app/api/launcher/*/route.ts`, testes.
- Alterados: `crates/wyd-launcher/src/{distribuicao,update,manifest}.rs`, `docs/cliente/{launcher,customizacao}.md`, `wyd-portal/README.md`.
- Proibido: tocar em `client/` além do manifesto; patch sem assinatura; credencial no launcher em claro.

## Testes / verificação

```bash
cargo test -p wyd-launcher
cd wyd-portal && npx tsc --noEmit && npx vitest run
```

Saída esperada: `check` real OK; `update` adulterado recusado; patch gerado no admin instala no launcher de teste.

## Skills relacionadas

- wyd-launcher — distribuição, patch atômico, bootstrap.
- wyd-rust-obs — telemetria de patch.

## Documentação de referência

- `docs/cliente/launcher.md`, `docs/cliente/customizacao.md` (ADR-0003)
- `docs/arquitetura/adr/0007-gui-launcher-tauri.md`, `docs/arquitetura/adr/0003-estrategia-patch-cliente.md`

## Subtasks

- [x] Núcleo `check/launch/serverlist/update` com testes reais (pré-existente: 60+4+3 testes verdes)
- [x] Leitura de versão/eventos do portal (`portal_client.rs` std-puro + `/api/launcher/{versao,eventos}` públicos com cache de 60 s; 8 testes contra HTTP falso)
- [x] Configuração remota assinada (2026-11-11): `configuracao.rs` + `GET /api/launcher/configuracao` + `tools/launcher/sign-config.mjs` — marca, canais e URLs sem recompilar
- [x] `/api/launcher/noticias` (2026-11-11) — vitrine pública do JSON de `docs/noticias` (mesma fonte do site), cache 60 s + `stale-while-revalidate`, sem corpo markdown; consumida pelo `gui` desde 2026-11-12 (painel `Notícias`, timeout 700 ms, fallback com motivo no stderr e na janela)
- [x] `POST /api/launcher/telemetria` (2026-11-11) — contrato fechado (sem PII, corpo 4 KiB, `no-store`), agregação por `versao+operacao` no processo e log estruturado; enviada por `check/config/update/launch` desde 2026-11-12 (best-effort, falha só no stderr); exposição no painel de monitoramento pendente
- [x] Publicação da config assinada no portal (2026-11-12): `/configuracoes/launcher` revisa forma, recusa revisão repetida/regressiva, preserva os bytes assinados e grava histórico append-only com autor e SHA-256 (`server/launcher-dist.ts`)
- [x] Painel de telemetria (2026-11-12): `/monitoramento/launcher` + `GET /api/admin/launcher/telemetria` (staff), agregado por `versao + operacao` com eventos, falhas, taxa, média e pior duração
- [ ] Geração de patch no admin (`/admin/patches`) — BLOQUEADO: exige canal de versão publicável (migration + `ConfigStore` + rotas de patch assinado, escopo backend próprio)
- [x] Assinatura assimétrica do pacote (2026-11-11): Ed25519 com chave pública embutida (`distribuicao::CHAVE_PUBLICA_HEX`), `update` sem `--key-file`, emissor `tools/launcher/sign-package.mjs`, golden Node↔Rust + rollback no meio do patch
- [ ] Autoupdate assinado + GUI Tauri mínima jogável (CLIENT-08 🚧)

## Critérios de aceite

- [ ] Jogador instala, atualiza e entra sem download manual.
- [ ] Patch adulterado recusado (teste dedicado).
- [ ] Manifesto inacessível = nada escrito, jogo não abre.

## Riscos / rollback

- Risco: patch corrompe jogo → mitigação: backup + rollback atômico.
- Rollback: manifesto anterior re-assinado (versão pinada).
