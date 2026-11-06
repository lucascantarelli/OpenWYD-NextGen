# CANAL-02 — Troca de canal em jogo (`/srv`) com token de crossover

**Fase:** 7 · **Depende de:** CANAL-01 · **Esforço:** M · **Estado:** 🟡 testada no e2e; falta teste no cliente gráfico

## Objetivo

Permitir que um jogador troque de canal sem sair do cliente e sem redigitar a senha, reproduzindo o
fluxo de crossover do legado.

## O que o legado faz (medido)

```
/srv <n>  (_MSG_MessageWhisper.cpp:75-95)
  tmsrv → db   MSG_DBServerChange { NewServerID, Slot }          id 20, FLAG_GAME2DB
  db    → tm   MSG_DBCNFServerChange { AccountName, Enc }        id 42, DB2GAME|GAME2CLIENT
               Enc = "*<srv> <Enc[0..7]> <slot>"   (CFileDB.cpp:1996-2011)
  db           grava TempKey = Enc                            (52 bytes)
  cliente      reconecta mandando Enc no campo Zero[52] do MSG_AccountLogin
  db           TempKey == Zero → ChangeServer = 1, login sem senha (CFileDB.cpp:662-812)
```

## Implementação nossa (sem RPC novo)

O token é **assinado** com `W2PP_CROSSOVER_KEY` em vez de persistido: evita coluna nova no Postgres e
RPC de troca em `api/db/v1/db.proto`.

| Peça | Arquivo |
|---|---|
| token `*<canal> ~x1:<canal>:<slot>:<exp>:<hmac>` (≤52 B, TTL 2 min, HMAC-SHA256 truncado) | `internal/crossover/crossover.go` |
| frame `MSG_DBCNFServerChange` (42) com `AccountName[16]+Enc[52]` | `tmserver/internal/protocol/serverchange.go` |
| comando `/srv <n>` (valida canal, recusa o próprio canal, emite o frame) | `tmserver/internal/handler/serverchange.go` |
| token aceito no `Zero[52]` do login e validado no dbServer | `handler/login.go`, `dbserver/internal/grpcsrv/server.go` (`WithCrossoverKey`) |
| chave compartilhada nos 3 serviços | `docker-compose.yaml` (`W2PP_CROSSOVER_KEY`) |

Endurecimento aplicado: `Build`/`Verify` recusam **chave vazia** (sem isso, HMAC de chave vazia é
determinístico e qualquer um forja um token) e o canal do prefixo `*` tem de concordar com o alvo
assinado.

## Evidência (06/11/2026)

```
go build ./...                                             exit 0
go vet ./...                                               exit 0
go test ./internal/crossover/                              PASS
go test -run 'ServerChange' ./tmserver/internal/handler/   PASS
TestE2EChannelChange        PASS  canal 1 → /srv 2 → token 38 B → canal 2
TestE2EChannelChangeBack    PASS  canal 2 → /srv 1 → token 38 B → canal 1
TestE2EServerChangeSameChannel PASS  mesmo canal não emite token
docker compose logs --since=10m db | grep -c 42P01          0
```

## O que falta

- [ ] Teste no cliente gráfico: `/srv 2` a partir do canal 1 e `/srv 1` a partir do canal 2, com o
      personagem entrando no outro servidor sem pedir senha.
- [ ] Confirmar no cliente que a janela do **ESC** não envia troca (o servidor não recebe pacote
      desse caminho); o caminho suportado é o comando.

## Riscos

- `W2PP_CROSSOVER_KEY` diferente entre canais faz o token ser recusado — o log do dbServer mostra
  `BAD_PASSWORD` sem indicar o motivo.
- A janela do ESC é client-side e não tem correção server-side; não confundir com falha do `/srv`.
