# QA-12 — Validação manual da entrada no mundo e das mensagens no 7662

> **ID canônico:** `QA-12`
> **ID histórico:** `QA-12`
> **Estado 2026-11-13:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Teste manual · **Esforço:** M · **Depende de:** stack no ar (`make stack-up`), conta de teste, cliente 7662 em `client/` · **Bloqueia:** fechamento de `BUG-01`, `BUG-02` e `GAME-17`

## Objetivo

Provar, no cliente real, o caminho **status → login → PIN → personagens → mundo** e a matriz
“mensagem de recusa × aparece na tela”. Nada aqui pode ser provado por teste automatizado: o
cliente é fechado e o render depende da máquina.

## Contexto / evidência

- Código e testes de fio verdes: `tests/sistema/{login,selchar,pin_*,sessao_unica,lotacao}.rs`.
- `0x0117 NewAccountFail` **não existe** no roteador do 7662 (zero `cmp` no desassemblage) —
  senha errada hoje é silêncio (`docs/audit/15-bugs-and-divergences.md` `DIV-01`).
- Tentativa anterior (2026-11-12): o cliente desenhou o *server select*, mas descartou mouse e
  teclado sintéticos; sem login no canal. Ver `docs/cliente/roteiro-prova-visual-7662.md`.
- Canal responde `/health` com `ok:true` e `serv00.htm` com contadores (verificado 2026-11-13).

## Pré-condições

1. Stack no ar: `docker compose ps` com `db`, `gateway`, `server-canal-1/2`, `portal` healthy.
2. `client/serverlist.bin` e `client/sn.bin` gerados para `127.0.0.1`/`192.168.100.3` (o
   `commerce` do launcher: `wyd-launcher serverlist tools/client/serverlist-servers.txt client`).
3. Conta de teste com personagem; **nunca** a conta do dono.
4. Backup de `client/Config.bin` antes de qualquer alteração.

## Passos (executar na ordem e capturar print + log)

| # | Caso | Ação | Esperado | Evidência a colar |
|---:|---|---|---|---|
| 1 | Servidor listado | abrir `WYD.exe` | canal aparece com nome e status | print da lista |
| 2 | Senha errada | digitar senha incorreta | **hoje:** nada aparece; registrar | print + `event=login result=refused reason=senha` |
| 3 | Conta em uso | logar a mesma conta em outro cliente | recusa visível ou silêncio? | log `result=conflict` |
| 4 | Primeiro PIN | conta sem PIN | cadastro do PIN e entrada | print |
| 5 | PIN errado ×5 | errar 5 vezes | bloqueio de 10 min | log `pin_locked` + print |
| 6 | Lista de personagens | após PIN | nome/classe/nível/ouro corretos | print + `tests/sistema/selchar.rs` como referência |
| 7 | Entrar no mundo | selecionar personagem | personagem aparece na cena | print + `character ok action=select client_id=…` |
| 8 | Sair e relogar | logout explícito e novo login | reloga sem esperar timeout | log `event=logout reason=explicit` |
| 9 | Canal cheio/manutenção | `WYD_SERVER_MAINTENANCE=1` no canal | recusa clara | print + métrica `wyd_packets_rejected_total` |

## Arquivos

- Atualizar: `docs/cliente/pacotes.md` (matriz mensagem × opcode × exibe/não-exibe),
  `docs/cliente/roteiro-prova-visual-7662.md` (resultado por caso), `BUG-01`/`BUG-02`/`GAME-17`
  (checkboxes e veredito).
- Proibido: inventar layout/texto; editar binário do cliente sem backup e sem ferramenta.

## Critérios de aceite

- [ ] Os 9 casos têm **print + trecho de log** (ou “não reproduzível” com motivo).
- [ ] A matriz de mensagens foi atualizada com evidência, não com hipótese.
- [ ] `BUG-01`, `BUG-02` e `GAME-17` receberam veredito A/B/C conforme o roteiro.
- [ ] Nenhuma conta de produção foi usada.

## Se falhar

Abra um comentário no próprio arquivo com: caso, print, linha de log, horário e o que divergiu.
Se o cliente não aceitar input, registre como bloqueio de ambiente (não como bug de servidor) e
siga para `QA-13`/`CLIENT-12`, que têm o mesmo requisito de GUI.
