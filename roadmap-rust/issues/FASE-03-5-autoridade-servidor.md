# Issue 03-5 — Autoridade total do servidor (física, dano, skills, inventário)

**Fase:** 3 · **Depende de:** 03-2, 03-3, 03-4 · **Bloqueia:** FASE-10 · **Esforço:** M

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
- Proibido tocar: fórmulas finais (FASE-10), `.deprecated-go/`.

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
- .deprecated-go/docs/migration/flows.md - fluxos ponta a ponta legados.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Toda ação crítica validada no servidor; forjadas rejeitadas com log.
- [ ] Paridade funcional documentada por ação (tabela fato vs legado).

## Riscos / rollback

- Risco: validação rígida demais quebrando gameplay legítimo → mitigação:
  tolerâncias documentadas + teste contra capturas reais (FASE-08).
- Rollback: `git checkout -- crates/wyd-core/`.
