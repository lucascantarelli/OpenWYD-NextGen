# GAME-05 — Autoridade total do servidor (física, dano, skills, inventário)

> **ID canônico:** `GAME-05`
> **ID histórico:** `FASE-03-5`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Fase:** 3 · **Depende de:** GAME-02, GAME-03, GAME-04 · **Bloqueia:** FASE-10 · **Esforço:** M

## Objetivo

Garantir que física, dano, skills e inventários sejam validados
server-authoritative com paridade funcional do jogo original, sem suas falhas.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.3: autoridade total + paridade funcional sem
falhas de segurança. Cliente nunca decide resultado (base para FASE-10 e
anti-exploit).

## Passos

1. Listar as ações que exigem validação rated (movimento, dano, skill, item).
2. Implementar os pontos de validação no core (rejeitar + logar, nunca confiar
   no cliente).
3. Teste: pacote forjado (dano/posição/item impossível) é rejeitado sem alterar
   estado.
4. `make check`.

## Arquivos

- Criados: validadores + testes de rejeição em `wyd-core`.
- Alterados: `crates/wyd-core/src/lib.rs` (pontos de validação).
- Proibido tocar: fórmulas finais (FASE-10).

## Testes / verificação

```bash
cargo test -p wyd-core
make check
```

Saída esperada: `test result: ok` incluindo testes de rejeição (estado
inalterado após ação forjada).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - pontos de autoridade.
- .agents/skills/wyd-rust-net - detecção no parse (rejeição precoce).
- .agents/skills/rust-code-quality - invariantes testadas.

## Documentação de referencia

- pre-release-rust.md (FASE 3 §3.3).
- docs/cliente/pacotes.md + crates/wyd-net/tests/ - fluxos ponta a ponta legados.

## Execução (2026-11-08)

- Passos 1–3 entregues para **movimento, combate e inventário**: mapa das ações em
  `docs/arquitetura/autoridade-servidor.md`; `systems/movement.rs` e `systems/combat.rs`
  (ADR-0014); `tests/autoridade.rs` prova recusa de pacote forjado sem mutação.
- Fórmula de dano continua fora do core (`DamageModel`); `AttackRules`/`MoveRules` são
  configuração explícita — números de jogo ficam para a FASE-10.
- Eixo **skill** fechado em `systems/skill.rs` (`SkillBook` + lista `known` do servidor) e eixo
  **inventário** em `systems/inventory.rs` (ADR-0015): `equip`/`equip_into`/`unequip`/`move_item`
  com slot derivado de `item_kind`, `req_level` de `item_catalog`, política de troca explícita e
  mochila cheia como recusa. `tests/inventario.rs` prova pacote forjado e o ciclo
  equipar → trocar → desequipar → reequipar sem duplicar nem perder item.
- Pendente (fora do core hoje): `req_str/int/dex/con` (o `Stats` não carrega os quatro), layout
  de rede do inventário (`UNVERIFIED` até a CLIENT-06) e baú (`vault_items`).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [ ] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Toda ação crítica validada no servidor; forjadas rejeitadas com log.
- [ ] Paridade funcional documentada por ação (tabela fato vs legado).

## Revisão de aceite (2026-11-12)

**Decisão:** manter aberta (🚧).

- **Evidência favorável:** movimento, combate, skill e inventário têm autoridade no servidor e testes de pacote forjado sem mutação.
- **Residual que impede o fechamento:** faltam `req_str/int/dex/con`, o layout de rede do inventário (UNVERIFIED até CLIENT-06) e o baú (`vault_items`).

## Riscos / rollback

- Risco: validação rígida demais quebrando gameplay legítimo → mitigação:
  tolerâncias documentadas + teste contra capturas reais (FASE-08).
- Rollback: `git checkout -- crates/wyd-core/`.
