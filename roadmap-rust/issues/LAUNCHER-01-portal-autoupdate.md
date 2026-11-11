# Issue LAUNCHER-01 — Launcher completo + integração com o portal + autoupdate

**Tipo:** Código (launcher+portal) · **Esforço:** L · **Depende de:** ROAD-01, FASE-08-3 ✅, FASE-08-8 🚧 · **Bloqueia:** —

## Objetivo

Fechar o launcher jogável (config do jogo, patch, download, `check/launch/serverlist/update`) com login unificado, notícias/eventos do portal, geração de patches no admin e autoupdate do próprio launcher.

## Contexto / evidência

- Hoje: `wyd-launcher` (manifest/patch/bootstrap/sha256, `check` real 6/6 OK, `update` atômico com backup, 19/19 testes) + GUI Tauri incompleta (só `LauncherGUI.html` gerado da identidade; shell Tauri inexistente — FASE-08-8 🚧).
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
- frontend-design — GUI Tauri (notícias + Jogar).
- wyd-rust-obs — telemetria de patch.

## Documentação de referência

- `docs/cliente/launcher.md`, `docs/cliente/customizacao.md` (ADR-0003)
- `docs/arquitetura/adr/0007-gui-launcher-tauri.md`, `docs/arquitetura/adr/0003-estrategia-patch-cliente.md`

## Subtasks

- [ ] Núcleo `check/launch/serverlist/update` com testes reais
- [ ] Leitura de notícias/versão do portal
- [ ] Geração de patch no admin
- [ ] Autoupdate assinado + GUI Tauri mínima jogável

## Critérios de aceite

- [ ] Jogador instala, atualiza e entra sem download manual.
- [ ] Patch adulterado recusado (teste dedicado).
- [ ] Manifesto inacessível = nada escrito, jogo não abre.

## Riscos / rollback

- Risco: patch corrompe jogo → mitigação: backup + rollback atômico.
- Rollback: manifesto anterior re-assinado (versão pinada).
