# FASE-06-4 — Catálogo de ameaças B1–B17 + §6 do `wyd2.md`

**Fase:** 6 · **Depende de:** FASE-06-2 ✅ · **Bloqueia:** FASE-15 (release) · **Esforço:** S

## Objetivo

Transformar o histórico de bugs do cliente real↔Go (L5 B1–B17) e a lista do §6 do
`wyd2.md` no catálogo de ameaças do nosso Rust: uma linha por ameaça com causa,
impacto, prevenção e validação.

## Contexto / evidência

Pesquisa `docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` item 8 e
`docs/wyd/pesquisa-wyd2-missao-fase4-2026-11-10.md` §2 (item 8: ❌ sem catálogo
próprio). Esqueleto pronto: `docs/migration/ingame-bugs.md` (B1–B17,
`Jean1dev/w2pp-OpenWYD@main`). Proteções já existentes a registrar: Argon2id
(senha/PIN), sessão única (`0030_sessao_unica.sql`), dano server-authoritative
(B12), `MissingRate` sem mutação, `delivery_queue.idempotency_key`, rate-limit
(FASE-06-2).

Casos especiais: B6 (posição exata não persiste) é comportamento esperado, não bug;
B1 (duplicação visual) não reproduzido no cliente real — observar, não mexer.

## Passos

1. Uma linha por ameaça do §6 do `wyd2.md` (dup item/moeda, exploits de
   troca/inventário/refino/combine/quest/XP/drop/PVP/skill/movimento/teleporte/
   canal, abuso de sessão, manipulação de pacotes, modificação de cliente, speed
   hack, bots, replay, race, timers, persistência, auth, concorrência) com os 10
   subitens do §6.
2. Matriz B1–B17 contra o Rust (passa/falha/aberto), um a um, sem declarar
   cobertura sem teste.
3. Registrar onde o Rust já protege e onde falta (trade/drop/canal/sessão como
   pontos críticos de dup, cf. `flows.md` §5).

## Arquivos

- Criados: `docs/wyd/sistemas/ameacas.md` (catálogo) ou seção em
  `docs/wyd/sistemas.md` (seguir o padrão do manual Kersef).
- Alterados: `docs/rust/08-checklist-funcionalidades.md` (§6 sai de ❌),
  `docs/wyd/README.md` (índice, se nova página).
- Proibido: executar exploits contra servidores de terceiros (pesquisa defensiva,
  reprodução só em ambiente próprio).

## Testes / verificação

```bash
cargo test -p wyd-protocol -p wyd-net
```

Sem código novo: a "prova" é a matriz publicada + fuzz de framing `Strict`
existente. Cada linha da matriz cita teste ou marca "aberto".

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs — hardening e telemetria.
- .agents/skills/wyd-rust-testing — harness, fuzz, regressões.
- .agents/skills/wyd-rust-net — framing `Strict`, rate-limit.

## Documentação de referencia

- docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md (item 8).
- `docs/migration/ingame-bugs.md` (B1–B17).
- `wyd2.md` §6.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Uma linha por ameaça do §6 com os 10 subitens.
- [ ] Matriz B1–B17 publicada (passa/falha/aberto).
- [ ] Nenhuma cobertura declarada sem teste.

## Riscos / rollback

- Risco: nenhum em runtime (só docs); risco de escopo é virar inventário morto →
  mitigação: cada linha aponta teste ou "aberto".
- Rollback: reverter o commit de docs.
