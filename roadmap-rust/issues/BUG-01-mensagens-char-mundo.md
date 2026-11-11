# Issue BUG-01 — Mensagens do cliente ausentes + personagem não entra no mundo

**Tipo:** Bug · **Esforço:** M · **Depende de:** ROAD-01 · **Bloqueia:** onda 1 (jogável)

## Objetivo

(1) Listar todas as mensagens do 7662 que não são exibidas e corrigir sem inventar pacote; (2) fechar a entrada no mundo ponta a ponta com conta admin no cliente real.

## Contexto / evidência

- Relato `prompt.md`: senha incorreta não mostra mensagem (só não loga); char não entra no mundo.
- Código atual: senha errada → `reply_recusa(NewAccountFail, CodigoRecusa::Senha)` com trailer 1 B (`Size=13`); PIN errado → `AccountSecureFail` + fecha; `CharacterLogin` monta `STRUCT_MOB` do banco, entra via `Engine::enter` com `scene_id < MAX_USER`, responde `CNFCharacterLogin` + `UpdateScore` + cena `ESCENE_FIELD` (FASE-03-17). Testes de fio passam (`login`, `selchar`, `cena_mobs`), mas prova visual no 7662 segue pendente (resíduo FASE-03-17/08-6).
- Hipótese principal (não conclusão): o 7662 pode esperar `Size=12` sem trailer ou outro `*Fail`/painel para exibir texto — `reply_recusa` documenta "cliente antigo lê só HEADER", mas sem captura isso é UNVERIFIED.

## Passos

1. Reproduzir com conta admin: senha errada, PIN errado, PIN bloqueado, conta em uso, zona indisponível, seleção inválida — gravar log `result=refused/conflict` + comportamento visível no 7662 (print/log).
2. Auditar `CodigoRecusa` (Senha/PinIncorreto/PinBloqueado/SelecaoInvalida/PersonagemIndisponivel/ContaEmUso/ZonaIndisponivel) × `opcode_table.rs` × `docs/cliente/pacotes.md`: tabela mensagem × opcode × exibe/não-exibe × evidência (RE estático `tools/client/`, nunca chute).
3. Entrada no mundo: `CharacterLogin` 20/36 B → `enter_world` → `CNFCharacterLogin` → `UpdateScore` → cena; checar `client_id`, `CreateType`, `ESCENE_FIELD`, posição salva; registrar ponto exato de falha no log (`character ok action=select client_id=…`).
4. Correção só com prova: se o cliente ignora trailer, responder `Size=12` puro nos `*Fail` do login (mantendo motivo só no log); se espera `MessagePanel`, mapear sem inventar texto.
5. Atualizar `docs/rust/08-checklist-funcionalidades.md` §1 + `docs/cliente/pacotes.md` com a lista completa.

## Arquivos

- Criados: `crates/wyd-server/tests/bug_mensagens.rs` (recusas por motivo, sem banco quando possível).
- Alterados: `crates/wyd-server/src/cpsock.rs` (`reply_recusa`/`reply`), `crates/wyd-protocol/src/lib.rs` (`motivo`), `docs/cliente/pacotes.md`, `docs/rust/08-checklist-funcionalidades.md`.
- Proibido: inventar layout/texto de pacote; mudar cerca de sessão (0030).

## Testes / verificação

```bash
cargo test -p wyd-server --test login --test selchar -- --nocapture
WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=admin cargo test -p wyd-server --test cena_mobs -- --nocapture
```

Saída esperada: cada recusa com motivo distinto no log; admin entra, aparece na cena e reloga sem esperar timeout.

## Skills relacionadas

- wyd-rust-net — framing, sem `panic!` em byte de rede.
- wyd-client-reverso — confirmar opcode/layout no 7662.
- wyd-rust-testing — harness + goldens sem cliente gráfico.

## Documentação de referência

- `docs/cliente/pacotes.md` §4, `docs/arquitetura/protocolo.md` §1–§5
- `crates/wyd-server/src/cpsock.rs` (handshake, `authorize_pin`, `dispatch`), `crates/wyd-server/src/auth.rs`

## Subtasks

- [ ] Matriz mensagem × exibe/não-exibe com evidência
- [ ] Reprodução com conta admin no 7662
- [ ] Correção com prova (sem pacote inventado)
- [ ] Checklist + pacotes atualizados

## Critérios de aceite

- [ ] Senha/PIN errados exibem recusa visível ou motivo documentado como limitação do 7662 com evidência.
- [ ] Admin entra no mundo, vê cena, sai e reloga imediatamente.
- [ ] Nenhum `panic!` em byte de rede (Strict).

## Riscos / rollback

- Risco: mudar `Size` quebra cliente novo → mitigação: teste de ida-volta do frame + golden.
- Rollback: reverter `reply_recusa` (ponto único).
