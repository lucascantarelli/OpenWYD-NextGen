# CHANNEL-01 — `/srv` e login entre instâncias de ponta a ponta

Fase 6 · Depende de DB-04/CI-02 · Esforço M

## Objetivo

`/srv 1` e `/srv 2` levam o personagem ao canal pedido (recusando só quando ele já está nele) e o
login no canal de destino funciona com o token, sem redigitar a senha. O caminho de conexão do
NextGen-2 fica diagnosticado com evidência.

## Contexto/evidência

- Sintomas: `/srv 1` respondia "você já está neste canal" indevidamente, `/srv 2` não produzia
  efeito, e o NextGen-5 falhava ao conectar.
- Referência C++: `ProcessDBMessage.cpp:1471-1478` repassa o `MSG_DBCNFServerChange` e chama
  `CloseUser`; o cliente só reconecta quando a sessão atual termina. O Go enviava o frame mas
  **mantinha a conexão aberta**, então o comando parecia um no-op no cliente gráfico.
- Token: `crossover.Build` assina `target/slot/exp/conta` e o cliente devolve em `Zero[52]`; o
  dbServer valida em `grpcsrv/server.go` (`strings.HasPrefix(tok, crossover.Prefix)`).

## Correções

- `tmserver/internal/handler/serverchange.go`: após `SendTo` do `MsgCNFServerChange`, fecha a
  sessão com `w.Close(s)` — `Session.close` descarrega a fila antes de fechar, então o token chega.
- `tmserver/internal/world/e2e_channel_change_test.go`: harness real (login → seleção → `/srv 2` →
  token → reconexão no canal 2 → seleção), drenando o snapshot inicial e provando o fechamento.
- `Makefile`: alvo `test-channel`.
- NextGen-2: `tmserver2` no ar em `:8282`, status HTTP 200 em `192.168.100.4` e login/e2e OK em
  `192.168.100.4:8281`. Desde o boot às 12:59 o tmserver2 não registrou nenhuma tentativa do
  cliente gráfico — a falha era a instância parada, não o wire.

## Testes/verificação

```bash
MSYS_NO_PATHCONV=1 docker run --rm --network openwyd_default -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src \
  -e W2PP_E2E_ADDR=tmserver:8281 -e W2PP_E2E_ADDR2=tmserver2:8282 \
  -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 -e W2PP_E2E_SLOT=0 -e W2PP_E2E_VERSION=12000 \
  golang:1.26-alpine sh -c "go test -tags=e2e -run 'TestE2EChannelChange|TestE2EServerChangeSameChannel|TestE2ESmokeLogin' ./tmserver/internal/world/ -v -count=1"
```

Saída (13:33):

```
--- PASS: TestE2EChannelChange (0.12s)
    channel 1: logged in and selected slot 0
    channel 1: /srv 2 issued token (35 bytes)
    channel 2: token login OK, selected slot 0
--- PASS: TestE2EServerChangeSameChannel (3.12s)
--- PASS: TestE2ESmokeLogin (0.07s)
```

Logs dos dois tmservers no mesmo instante:

```
tmserver  | msg="server change: token issued" conn=1 account=test slot=0 from=1 to=2
tmserver  | msg="connection closed" conn=1
tmserver2 | msg="account login: server-change token" conn=1 account=test
tmserver2 | msg="account login: OK" conn=1 account=test id=1 role=admin chars=1
tmserver2 | msg="character login request" conn=1 slot=0
```

## Critérios de aceite

- [x] `/srv 2` emite token e a sessão fecha (paridade com `CloseUser`).
- [x] Login com token no outro canal chega a `CNFAccountLogin` e seleciona o personagem.
- [x] Pedir o canal atual responde aviso, nunca token.
- [x] Testes de protocolo/handler verdes (`TestCommandServerChange*`).
- [x] `test-channel` documentado no Makefile.

## Riscos/rollback

O fechamento é o mesmo do legado; se um cliente antigo não reconectar, o rollback é remover o
`w.Close(s)` — mas isso reintroduz o no-op. O token continua válido por 2 minutos.
