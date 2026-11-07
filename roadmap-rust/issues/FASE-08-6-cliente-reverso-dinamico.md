# Issue 08-6 — Engenharia reversa dinâmica e correlação

**Fase:** 8 · **Depende de:** 08-5 (mapa de funções), 08-2 (parsers) · **Bloqueia:** 08-7 · **Esforço:** L · **Status:** 🚧

## Objetivo

Confirmar em execução o que o desassemble estático propôs: capturar pacotes reais, observar o cliente processando cada opcode e produzir goldens que o servidor Rust reproduza byte a byte.

## Contexto / evidência

O servidor emite o dialeto 7662, mas não há prova de ida-e-volta contra o binário em todos os pacotes. `investigacao-freeze-cliente.md` (arquivo Go) registra freeze com o cliente ACKando TCP mas descartando tudo — `session last sends` como post-mortem. T2 (structs de jogo) foi transferido da 08-5 para cá por limitação de ferramenta.

## Resultados desta rodada (2026-11-07)

| Entrega | Evidência |
|---|---|
| Roteiro de trace | `docs/cliente/depuracao.md`: VM, breakpoints (VAs da 08-5), fases 0–6 login→campo, protocolo de reprodução do freeze, higiene de fixtures |
| Goldens gerados C→S | `fixtures/cliente/*.json` — 5/5 válidos (`captura.mjs --check`) |
| Validador | `tools/client/captura.mjs` (`--gen`/`--check`); opcode cruzado com a tabela gerada de `opcodes.rs` |
| Catálogo | `docs/cliente/pacotes.md`: 5 pacotes C→S **GERADO** + 5 S→C **PENDENTE** |
| Determinismo | duas execuções, artefatos byte-idênticos |

Os goldens provam o **codec**, não o cliente: bytes sintéticos (conta `GOLDEN`) com a matemática exata do CPSock.

## O que falta (bloqueio declarado)

Captura ao vivo (§6 de `depuracao.md`): exige servidor executável falando 7662 (crate `wyd-net`/`wyd-core` ainda não existem) + VM Windows com D3D9 + executor humano. Passos concretos quando desbloquear:

1. Fases 0–6: INITCODE, AccountLogin, CNFAccountLogin (`Size` exato), CharacterLogin, burst CreateMob/ESCENE_FIELD.
2. Comparar cada frame com o golden gerado; divergência no C→S = servidor errado; em S→C = fechar struct.
3. Reproduzir o freeze (§4): tail de pacotes com `Size` anômalo = encoder culpado; stream íntegro com descarte = `RecvQueue`/NPC da área.
4. Higienizar (conta/senha → placeholders), versionar, `--check`.

## Arquivos

- Criados: `docs/cliente/depuracao.md`, `docs/cliente/pacotes.md`, `tools/client/captura.mjs`, `fixtures/cliente/*.json`, `docs/cliente/gerado/goldens.json`.
- Alterados: `docs/cliente/README.md`, `superficie-conhecida.md`, `engenharia-reversa.md`, `tools/client/README.md`, skill `wyd-client-reverso`, `roadmap-rust/index.md`, `docs/rust/03-progresso.md`.
- Proibido tocar: `client/`, tabelas de personagem vivo, `.pcap`/dumps na árvore.

## Testes / verificação

```bash
node tools/client/captura.mjs --gen && node tools/client/captura.mjs --check
# esperado: 5/5 válidos; exit 0; exit 1 em divergência
cargo test --workspace && cargo clippy --workspace -- -D warnings && cargo fmt --all -- --check
```

## Skills Relacionadas do freebuff

- .agents/skills/wyd-client-reverso - método dinâmico e higiene de fixture.
- .agents/skills/wyd-rust-net - golden e limites de pacote (quando o net existir).
- .agents/skills/wyd-testing-bots - harness e captura automatizada.

## Documentação de referencia

- docs/cliente/depuracao.md - roteiro e bloqueios.
- docs/cliente/pacotes.md - catálogo GERADO/PENDENTE.
- .deprecated-go/docs/migration/flows.md - fluxos esperados login→campo.
- .deprecated-go/docs/migration/investigacao-freeze-cliente.md - sintoma e hipóteses a confirmar.

## Subtasks

- [x] Codificar issue (roteiro + pipeline de goldens)
- [x] Testar issue (`--gen`/`--check` + gate)
- [x] Documentar issue
- [ ] Captura ao vivo (bloqueada: servidor + VM)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Roteiro executável por terceiros sem dedução (VAs, fases, comandos, higiene).
- [x] Goldens gerados do C→S com validação automática e determinismo.
- [ ] Login → seleção → campo capturado e reproduzido pelo servidor Rust (ao vivo).
- [ ] Cada pacote S→C com golden e status (CONFIRMADO/UNVERIFIED).
- [ ] Causa do freeze identificada ou reduzida a hipótese com trace.
- [ ] Nenhuma fixture com dado de conta versionada.

## Riscos / rollback

- Risco: XTrap ativo derrubar a depuração → mitigação: diagnóstico T7 antes; VM descartável.
- Risco: captura com dado pessoal vazar → mitigação: sanitização obrigatória; `.pcap` fora do git.
- Rollback: `git checkout -- tools/client/captura.mjs fixtures/cliente docs/cliente`.
