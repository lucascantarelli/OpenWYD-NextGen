# BUG-01 — Mensagens do cliente ausentes + personagem não entra no mundo

> **ID canônico:** `BUG-01`
> **ID histórico:** `BUG-01`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Bug · **Esforço:** M · **Depende de:** CLIENT-06 🚧 (captura/prova visual), GAME-17 🚧 (mesma entrada) · **Bloqueia:** onda 1 (jogável)

## Objetivo

(1) Listar todas as mensagens do 7662 que não são exibidas e corrigir sem inventar pacote; (2) fechar a entrada no mundo ponta a ponta com conta admin no cliente real.

## Contexto / evidência

- Relato `prompt.md`: senha incorreta não mostra mensagem (só não loga); char não entra no mundo.
- Código atual: senha errada → `reply_recusa(NewAccountFail, CodigoRecusa::Senha)` com trailer 1 B (`Size=13`); PIN errado → `AccountSecureFail` + fecha; `CharacterLogin` monta `STRUCT_MOB` do banco, entra via `Engine::enter` com `scene_id < MAX_USER`, responde `CNFCharacterLogin` + `UpdateScore` + cena `ESCENE_FIELD` (GAME-17). Testes de fio passam (`login`, `selchar`, `cena_mobs`), mas prova visual no 7662 segue pendente (resíduo GAME-17/CLIENT-06).
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
cargo test -p wyd-server --test sistema_login --test sistema_selchar -- --nocapture
WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=admin cargo test -p wyd-server --test sistema_cena_mobs -- --nocapture
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
- [x] Nenhum `panic!` em byte de rede (Strict).

## Revisão de prova (2026-11-12)

- **Provado:** as recusas têm motivo distinto no log e os testes de fio (`login`, `selchar`,
  `cena_mobs`) passam; a entrada no mundo entrega `ClientID` de jogador, `UpdateScore` e cena.
- **Pendente:** a matriz mensagem × exibe/não-exibe no 7662 e a reprodução de cada recusa com
  conta admin. Sem captura, o `Size=12` puro continua **UNVERIFIED** e não vira correção.
- **Dependência:** cliente/VM (`CLIENT-06`); a prova visual de entrada é a mesma da `GAME-17`.

## Seções de prova no 7662 (roteiro 2026-11-12)

> Roteiro consolidado de execução: [`docs/cliente/roteiro-prova-visual-7662.md`](../../docs/cliente/roteiro-prova-visual-7662.md).
>
> Executar na VM com o cliente 7662 e **conta de teste**, nunca na conta do dono. Cada item só
> vira ✅ com **print da tela + trecho do log do canal**; o que não for reproduzível fica `[ ]`
> e entra como bloqueio, não como conclusão. Preparação de captura: `docs/cliente/depuracao.md`
> §1–§5; o hash do `WYD.exe` e o serverlist têm de ser os do build de teste.

### A. Matriz de recusas visíveis

| # | Ação no cliente | O que o 7662 deve mostrar | O que o log deve comprovar | Evidência | [ ] |
|---:|---|---|---|---|---|
| A1 | Senha errada no login | Recusa visível; o texto atual é `Senha ou conta incorreta` | `NewAccountFail` + `_MSG_MessagePanel` 140 B (0x0101), motivo `Senha` | print + frame/log | |
| A2 | PIN errado depois do login | Recusa visível, sem derrubar a conexão antes da mensagem | `AccountSecureFail` + motivo `PinIncorreto` | print + frame/log | |
| A3 | PIN errado até bloquear | Aviso de bloqueio; nova tentativa recusada | motivo `PinBloqueado` e contador de tentativas | print + log | |
| A4 | Segunda conexão com a mesma conta | Aviso de conta em uso/sessão viva | `event=login result=conflict` + `origin_channel` | print + log dos dois canais | |
| A5 | Seleção de personagem inválida/removida | Recusa visível, sem entrar no mundo | motivo `SelecaoInvalida`/`PersonagemIndisponivel` | print + log | |
| A6 | Canal/zona indisponível | Recusa visível quando aplicável | motivo `ZonaIndisponivel` | print + log | |

Se o 7662 não exibir A1, registrar **qual frame saiu** (`Size`, opcode, corpo) antes de propor
`Size=12` puro; essa hipótese continua **UNVERIFIED** até a captura. O mesmo vale para qualquer
texto: não inventar layout nem tradução que o cliente não tenha recebido.

- [ ] A1–A6 executados, cada um com print e linha de log
- [ ] Tabela `mensagem × opcode × Size × exibe/não exibe × evidência` preenchida
- [ ] `docs/cliente/pacotes.md` e `docs/rust/08-checklist-funcionalidades.md` §1 atualizados
- [ ] Se um caso não puder ser reproduzido, motivo e bloqueio registrados na issue

### B. Entrada com conta admin

1. Abrir o 7662 e autenticar com a conta de teste que tem personagem no slot conhecido.
2. Escolher o personagem e confirmar que a tela **não** mostra “Falha ao conectar”.
3. Capturar a cena com o personagem, NPCs e HUD (HP/MP/EXP) visíveis.
4. Sair pelo caminho normal e entrar de novo imediatamente, sem aguardar timeout.

O log deve mostrar `character ok action=select`, `CNFCharacterLogin`, `UpdateScore` e a cena
`ESCENE_FIELD`; o `client_id` do jogador deve estar na faixa `< MAX_USER`.

- [ ] Personagem entra no mundo e a cena renderiza
- [ ] `client_id`/`MobID` do dono na faixa de jogador e `CreateType=2` no self
- [ ] `UpdateScore` (e `UpdateEtc`, quando aplicável) chegam antes da cena
- [ ] Logout + relogin imediato sem erro
- [ ] Print da cena e tail do log anexados

### C. Gate de evidência

- [ ] Nenhuma alteração de `reply_recusa` feita só com base em “não apareceu”; a decisão cita o
  frame capturado
- [ ] Cada critério de aceite visual tem link para o print/log correspondente
- [ ] O que continuar sem captura permanece aberto e marcado como bloqueado por cliente/VM

## Execução da Sessão A (2026-11-12)

**Veredito: bloqueada por entrada sintética, mesmo com geometria saudável; nenhum critério visual marcado.**

- O cliente abriu e desenhou o server select (`WYD`, `WYD-1`, `WYD-2`, `Conectar`, `Fechar`).
- `client/Config.bin` selecionava o modo 8 (`1024×768×32`); na tela 1366×768 a janela ficou
  `1030×788` com cliente `1024×759`, e o log registrava `Error in Init Render Target Texture`.
- Corrigido para o modo 7 (`800×600×32`): cliente `800×600` exato e menu renderizado. O erro
  de render target **persistiu**; o modo 2 (`800×600×16`) caiu para `1366×768` e também falhou;
  preferência pela GPU NVIDIA foi testada sem efeito.
- Com a geometria saudável, cliques elevados (`mouse_event`, `SendMessage` e `SendInput`) e
  teclas (`SendKeys`) continuaram sem alterar a tela; `Fechar` também não reagiu. O bloqueio
  não é resolução, formato de cor ou GPU.
- O canal 1 não registrou `event=login`/`handshake`/`session` e não houve TCP em `8281`: o
  cliente não chegou a autenticar.
- Evidência local em `tmp/prova-visual-7662/` (prints `19`/`23`/`24`/`25`, `canal.log`,
  `wyd-client.log`, `Config.bin.bak` e `resumo.md`), sem conta/senha/PIN. Runbook §9.5–§10 de
  [`docs/cliente/roteiro-prova-visual-7662.md`](../../docs/cliente/roteiro-prova-visual-7662.md).
- Nada foi alterado em `reply_recusa`; a hipótese `Size=12` continua **UNVERIFIED**. A issue
  segue aberta e depende de operador humano/VM com input aceito.

## Riscos / rollback

- Risco: mudar `Size` quebra cliente novo → mitigação: teste de ida-volta do frame + golden.
- Rollback: reverter `reply_recusa` (ponto único).
