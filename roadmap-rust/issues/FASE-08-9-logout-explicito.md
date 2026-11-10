# Issue 08-9 — Logout explícito: sair do mundo sem esperar timeout

**Fase:** 8 · **Depende de:** 08-2 ✅ · **Bloqueia:** — · **Esforço:** S

## Objetivo

Tratar o pedido de saída do jogador (voltar à seleção ou fechar o cliente) de forma explícita:
liberar `character_session`, sair do mundo e salvar — em vez de depender só do timeout de
120 s.

## Contexto / evidência

Checklist 1.13: “libera a sessão por ociosidade (120 s) em `session.rs`; pacote de saída não
tratado. **Sem teste dedicado**”. O legado tinha `CloseUser` no evento de socket; aqui o
`Drop` da conexão libera a vaga de IP, mas a sessão de personagem só cai no timeout
(`WYD_SERVER_SESSION_IDLE_SECS`).

Efeito prático: jogador que fecha o cliente fica “online” por até 2 min; relogar no outro canal
pode encontrar a própria sessão viva (recusa correta, mas por motivo falso).

## Passos

1. Mapear o opcode de saída em `crates/wyd-protocol/src/wire/generated/opcode_table.rs`
   (procurar `Exit`/`Logout`/`Quit`/`Close`); **não inventar** nome nem layout.
2. Se o pacote estiver provado: tratar no laço de seleção/sessão → liberar sessão + sair do
   mundo + save, responder confirmação quando houver.
3. Se **não** estiver provado: tratar a queda de TCP como saída (já ocorre) e reduzir a janela
   de ociosidade para o menor valor seguro; registrar a lacuna em `docs/cliente/`.
4. Testes: enviar o pacote de saída (ou fechar o socket) libera `character_session` na hora;
   segunda conexão da mesma conta entra sem esperar timeout; nenhuma entidade órfã no mundo.
5. Log `event=logout result=ok reason=explicit|socket` (sem conta em claro além do já praticado).

## Arquivos

- Criados: `crates/wyd-server/tests/logout.rs`.
- Alterados: `crates/wyd-server/src/{cpsock,session,characters}.rs`,
  `crates/wyd-protocol/src/payload/*` (só se o layout for provado),
  `docs/rust/08-checklist-funcionalidades.md` (1.13), `docs/cliente/pacotes.md`.
- Proibido tocar: cerca de sessão única (0030) — ela é a autoridade; `client/`.

## Testes / verificação```bash
WYD_TEST_DATABASE_URL=... cargo test -p wyd-server --test selchar logout_explicito
WYD_TEST_DATABASE_URL=... cargo test -p wyd-server --test sessao_unica
```
Saída esperada: `character_session.released_at` preenchido no logout explícito e na queda;
relogin imediato aceito. Sem DSN os testes **pulam** com aviso (`SKIP:`), sem derrubar a suíte.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net — pacote, sessão, sem `panic!`.
- .agents/skills/wyd-client-reverso — confirmar o opcode no build 7662.
- .agents/skills/wyd-rust-testing — teste headless de sessão.

## Documentação de referencia

- `docs/wyd/canais-e-cidadania.md` §5.2 (sessão única e trânsito).
- `crates/wyd-db/migrations/0030_sessao_unica.sql`.
- `docs/arquitetura/contrato-transporte-cpsock.md` §8.

## Subtasks

- [x] Codificar issue (handler `CharacterLogout` + log `reason=socket` na queda)
- [x] Testar issue (`selchar::logout_explicito_fecha_a_conexao_e_libera_a_sessao` + `sessao_unica`; pulam com `SKIP:` sem DSN)
- [x] Revisar issue (opcode provado `0x0215`/`0x0116` na tabela congelada; nenhum pacote inventado)
- [x] Documentar issue (checklist 1.13 ✅, `docs/cliente/pacotes.md` 4b, `protocolo.md` §5)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Logout explícito libera a sessão na hora (opcode provado; `live_session_of=None` em ≤ 5 s).
- [x] Queda de socket continua liberando (não regride; `sessao_unica` prova a transição de canal).
- [x] Teste dedicado com banco real, pulando com aviso sem DSN.
- [x] Nenhum pacote novo inventado sem prova.

## Evidência (2026-11-10)

- `crates/wyd-server/src/cpsock.rs`: ramo `CharacterLogout` responde `CNFCharacterLogout` e
  encerra; queda de socket loga `event=logout result=ok reason=socket` (passo 5 da issue).
- `crates/wyd-server/tests/selchar.rs::logout_explicito_fecha_a_conexao_e_libera_a_sessao`:
  login + PIN + seleção + logout pelo fio real, CNF, EOF e sessão liberada.
- Resíduo consciente: golden do frame real do cliente 7662 segue pendente da captura 08-6
  (bloqueada em VM + cliente em execução); nada foi inventado para cobri-lo.

## Riscos / rollback

- Risco: tratar um opcode errado como logout e derrubar jogador → mitigação: ação só após
  evidência de layout; até lá, queda de socket.
- Rollback: remover o ramo do opcode; timeout continua sendo a rede de segurança.
