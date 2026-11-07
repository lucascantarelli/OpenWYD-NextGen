# Issue 06-2 — Hardening: DoS, rate limiting, sanitização do parser

**Fase:** 6 · **Depende de:** FASE-08 · **Bloqueia:** FASE-12 · **Esforço:** M

## Objetivo

Endurecer o handshake (limite por IP, timeout, buffer máximo) com rate limiting
e sanitização estrita, tudo testado contra flood básico.

## Contexto / evidência

`pre-release-rust.md` FASE 6 §6.2. Base: `MAX_PACKET_BYTES`=4096 em
`crates/wyd-net/src/lib.rs`.

## Passos

1. Implementar limite de conexões por IP + timeout de handshake + buffer máximo.
2. Implementar rate limiting por conexão/IP com resposta documentada.
3. Sanitização estrita: tamanho→opcode→payload validados antes de qualquer
   parse (ver 08-2).
4. Teste de flood básico: atacante limitado, legítimo preservado (números anexados).
5. `make check`.

## Arquivos

- Criados: rate-limiter, testes de flood/rejeição em `wyd-net`.
- Alterados: `crates/wyd-net/src/lib.rs`.
- Proibido tocar: `.deprecated-go/`, `client/`.

## Testes / verificação

```bash
cargo test -p wyd-net
make check
```

Saída esperada: `test result: ok` incluindo teste de flood (rejeição acima do
limite, legítimo intacto).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - hardening e limites.
- .agents/skills/wyd-rust-net - parser e rejeição precoce.
- .agents/skills/rust-code-quality - sem panic sob input hostil.

## Documentação de referencia

- pre-release-rust.md (FASE 6 §6.2).
- roadmap-rust/issues/FASE-08-2-parsers.md - validação de limites.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Limites documentados e testados; flood básico contido.
- [ ] Nenhum panic com input truncado/hostil (fuzz mínimo anexado).

## Riscos / rollback

- Risco: limite agressivo derrubando legítimos (NAT) → mitigação: limites por
  IP + por conta, documentados e ajustáveis.
- Rollback: `git checkout -- crates/wyd-net/`.
