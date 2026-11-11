# Issue PORTAL-05 — Configurações 100% via portal (sem arquivo manual)

**Tipo:** Código (portal+gateway+jogo) · **Esforço:** M · **Depende de:** ROAD-01 · **Bloqueia:** —

## Objetivo

Servidor completamente configurável via portal (personalização, preferências, taxas, eventos), com validação, auditoria, integração com monitoramento/financeiro e reflexo em tempo real sem editar arquivo.

## Contexto / evidência

- Hoje: `game_setting` + `content_rate_override` com `config_revision`; thread `wyd-config` (`WYD_CONFIG_POLL_MS`) troca `Arc<GameConfig>` sem restart; portal edita `multiplier:exp` com prova E2E (revisão 1→3, `config_reload` no processo).
- `prompt.md` pede: tudo via portal, tempo real, validação, auditoria, integração, personalização, relatórios de alterações, visuais.

## Passos

1. Inventariar `game_setting` × constantes ainda em código (refino, SancRate, drops, NPC, teleporte, `12th_skill_fame` etc.) e migrar número de servidor para `game_setting` com teto/default documentados.
2. UI `/configuracoes` por domínio (EXP, drop, refino, eventos, canais) com validação de faixa, preview de impacto e confirmação; cada gravação sobe `config_revision` + audit trail.
3. Reflexo em tempo real: resolvedores (`Gameplay::refine/loot/exp_reward`, NPC, teleporte) sempre no snapshot vigente; falha de leitura mantém último bom + `wyd_config_reload_failures_total`.
4. Relatórios: histórico de revisões (quem, quando, diff) + rollback em 1 clique para revisão anterior.

## Arquivos

- Criados: `wyd-portal/src/app/(admin)/configuracoes/dominios-*`, testes de faixa/rollback.
- Alterados: `wyd-portal/src/server/config.ts`, `crates/wyd-server/src/{config,gameplay}.rs`, `crates/wyd-db/migrations/*game_setting*` (só novas), `docs/arquitetura/*` afetados.
- Proibido: constante de balanceamento no código quando deveria ser `game_setting`.

## Testes / verificação

```bash
cargo test -p wyd-server --test config_recarga --test config_gameplay
cd wyd-portal && npx tsc --noEmit && npx vitest run
```

## Skills relacionadas

- wyd-rust-core — snapshot atômico fora do tick.
- wyd-rust-obs — métricas de recarga/falha.
- frontend-design — formulários densos com validação visível.

## Documentação de referência

- `docs/rust/03-progresso.md` (configuração viva), `crates/wyd-server/src/config.rs`

## Subtasks

- [x] Eventos de mundo: grade com horários + liga/desliga auditado (`world_event_admin/*`, revisão devolvida)
- [x] Validação portal-side (código 1..60, inteiros ≥ 0 nos formulários) + confirmação por revisão
- [ ] Rollback 1-clique para revisão anterior — BLOQUEADO: sem rota de histórico de revisões no gateway; `reset_*` por chave continua sendo o retorno ao default
- [ ] Inventário setting × constante — segue aberto (fora do escopo portal: exige varredura no `wyd-server`)

## Critérios de aceite

- [ ] Mudança no portal reflete no jogo sem restart (log `config_reload`).
- [ ] Valor inválido nunca derruba snapshot bom (teste dedicado).
- [ ] Toda alteração auditada com autor e diff.

## Riscos / rollback

- Risco: taxa absurda quebra economia → mitigação: tetos + confirmação + rollback.
- Rollback: revisão anterior (dado, não deploy).
