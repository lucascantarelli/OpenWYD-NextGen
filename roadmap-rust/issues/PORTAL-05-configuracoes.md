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

- [ ] Mudança no portal reflete no jogo sem restart (log `config_reload`) — exige stack viva (E2E `multiplier:exp` 1→3 existe, roda com gateway+canal)
- [x] Valor inválido nunca derruba snapshot bom (`config_gameplay` 9/9: SancRate 101 recusado, revisão 3 segue valendo; `config_recarga` 6/6: `escrita_invalida_e_falha_de_leitura_mantem_o_ultimo_snapshot_bom`)
- [x] Toda alteração auditada com autor real (`taxa_valida_grava_e_audita` agora confere `actor_id == 7` da sessão; `moderator_id` nunca vem do corpo — `adminCall.withoutModerator`)

## Riscos / rollback

- Risco: taxa absurda quebra economia → mitigação: tetos + confirmação + rollback.
- Rollback: revisão anterior (dado, não deploy).

## Verificação on-line (2026-11-11)

- **A tela deixou de ser um rolo único**: virou sete assuntos com parâmetro
  `?aba=` (Refino, EXP e multiplicadores, Drops, NPCs, Teleportes, Preços de
  loja, Eventos); só a seção ativa é carregada e renderizada.
- **Removidos da UI**: campo "Fonte desta edição", todas as menções a módulo
  Rust/migration/tabela/"snapshot" e o bloco "Restaurar default" que exigia
  domínio + chave digitados à mão (agora cada linha editada tem "Voltar ao
  padrão").
- **Procedência**: derivada da sessão (`portal#<conta>`) nas server actions —
  a trilha de auditoria continua com autor e valor anterior.
- **Rótulos/unidades**: "Chance (10.000 = 100%)" + coluna "Equivale",
  "Preço de compra/venda", "Aparece no mapa", "Peso do item no sorteio".
- **Controles compactos**: `min-height` do tema vencia o input de linha; a
  regra foi reforçada (`.ui-table .ui-input--compact`) e `grant-form` ganhou
  layout de linha.
- **Pendente conhecido**: o editor grava configuração que o servidor relê na
  revisão, mas composição, teleporte e spawn de NPC ainda não têm consumidor no
  jogo — a tela não promete o contrário.
