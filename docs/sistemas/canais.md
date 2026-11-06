# Canais e troca de servidor

"Servidor" na lista do cliente é uma **linha** da `serverlist.bin`, e o que muda entre canais é um
índice (`ServerIndex`), não uma instalação diferente. O detalhe completo está em
[`../migration/canais.md`](../migration/canais.md); esta página cobre o mecanismo e o que o jogador vê.

## Dois canais no ambiente local

O compose sobe `tmserver` e `tmserver2` (novato e regular). O canal novato é fixo por configuração —
`W2PP_NOVICE_CHANNEL` —, não gira por calendário do cliente (`newbie_channel.go`).

## Troca em jogo (`/srv`)

1. `serverchange.go` recebe o pedido com o número do canal.
2. O servidor emite um token de uso único, válido por 2 minutos, assinado com a chave de crossover.
3. O cliente reconecta no outro canal e devolve o token; o login valida sem pedir senha de novo.

O `Enc` do frame é **a string do legado**: `*<canal>` seguido de nove inteiros decimais
(`CFileDB.cpp:2009`), porque é o único formato que o parser do cliente aceita. O token assinado
vai empacotado nesses inteiros (canal, slot, expiração em dois pedaços e o HMAC em quatro), e
`crossover.ParseWire` recusa qualquer campo fora da gramática — foi assim que o `~x1:` e o
`*2 ~x1:` das versões anteriores deixaram de circular: o servidor aceitava, o cliente travava.

Peça central: `tmserver/internal/crossover`. Os dois tmServer e o dbServer precisam da **mesma**
`W2PP_CROSSOVER_KEY`; chave diferente aparece como "senha inválida" na reconexão.

## O que não funciona

O menu de canais do ESC **lista** canais, mas não envia troca ao servidor — só o comando funciona. Não
é bug de UI: o dispatcher loga todo frame recebido (inclusive sem rota), e nenhum chega nesse
caminho. Fazer o ESC trocar de canal exige patch no binário do cliente: `roadmap/issues/CANAL-03.md`.

Consequência prática: quando o jogador escolhe um canal no ESC, o cliente fecha a própria conexão
e fica sem mundo, sem skill e sem NPC — o servidor não participou de nada. Oriente a usar `/srv <n>`
até o patch existir.

## Onde está

| Peça | Arquivo |
|---|---|
| Troca de canal | `serverchange.go` |
| Canal novato | `newbie_channel.go` |
| Token de crossover | `tmserver/internal/crossover` |
| Regras e layout do `serverlist.bin` | `docs/migration/canais.md` |
| Geração do binário | `client-tools/serverlist` (`make client-serverlist`) |

## Como verificar

```bash
go test ./internal/crossover/ ./tmserver/internal/handler/ -run 'ServerChange|NewbieChannel'
```

No jogo: `/srv 2` e conferir que o personagem entra no outro canal sem redigitar senha; repetir o mesmo
token deve falhar (uso único).

!!! warning "UNVERIFIED"
    O limite de canais que o cliente desenha sem atualizar o `sn.bin` foi medido até dois canais. Para
    quatro, `docs/migration/canais.md` §4 descreve o plano, não um resultado verificado.

## Segurança

O token é de uso único e expira em 2 minutos. Não aumente a validade "para facilitar o teste": é o
mesmo token que autoriza assumption de personagem em outro servidor.
