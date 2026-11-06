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

Peça central: `tmserver/internal/crossover`. Os dois tmServer e o dbServer precisam da **mesma**
`W2PP_CROSSOVER_KEY`; chave diferente aparece como "senha inválida" na reconexão.

## O que não funciona

O menu de canais do ESC **lista** canais, mas não envia troca ao servidor — só o comando funciona. Não
é bug de UI: o pacote de troca é o do comando.

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
go test ./tmserver/internal/crossover/ ./tmserver/internal/handler/ -run 'ServerChange|NewbieChannel'
```

No jogo: `/srv 2` e conferir que o personagem entra no outro canal sem redigitar senha; repetir o mesmo
token deve falhar (uso único).

!!! warning "UNVERIFIED"
    O limite de canais que o cliente desenha sem atualizar o `sn.bin` foi medido até dois canais. Para
    quatro, `docs/migration/canais.md` §4 descreve o plano, não um resultado verificado.

## Segurança

O token é de uso único e expira em 2 minutos. Não aumente a validade "para facilitar o teste": é o
mesmo token que autoriza assumption de personagem em outro servidor.
