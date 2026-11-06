# BOT-01 — Arquitetura do laboratório de bots (ADR e limites)

**Fase:** BOT · **Depende de:** 07 (login headless) · **Bloqueia:** BOT-02..08 · **Esforço:** S · **Estado:** ✅ concluída em 06/11/2026

## Objetivo

Decidir onde o laboratório de bots mora, o que ele reusa e o que ele **não** pode inventar, com a
decisão registrada em ADR antes de qualquer código.

## Contexto / evidência

- O codec do wire é `tmserver/internal/protocol` (4.621 linhas, 40 arquivos de teste) e é a fonte de
  verdade do protocolo CPSock neste repositório.
- A regra de `internal` do Go impede que um pacote na raiz do módulo importe
  `openwyd-nextgen/tmserver/internal/protocol`: o primeiro esboço em `botlab/` foi reprovado pelo
  compilador (`use of internal package … not allowed`), o que fixou a localização.
- Bots externos (`kevinkouketsu/WYD2Bot`, `Wyd2Client`) exigem offsets próprios — divergência
  silenciosa do binário 7662.
- O limite de mensagens do servidor é 200/s com burst 400 (`-max-msg-per-sec`, `-msg-burst`).

## Decisão

Registrada em [ADR 0007](../../docs/adr/0007-laboratorio-de-bots.md): biblioteca em
`tmserver/internal/bot`, CLI em `tmserver/cmd/botlab`, núcleo determinístico (FSM + perfis + sementes
+ cenários versionados), sem LLM no laço, recusando alvo não loopback por padrão.

## Arquivos

- `docs/adr/0007-laboratorio-de-bots.md` (novo)
- `tmserver/internal/bot/` (novo pacote)
- `tmserver/cmd/botlab/` (nova CLI)
- `Makefile` (`bot-test`, `bot-validate`, `botlab`)

## Testes / verificação

```
go build ./...                                          exit 0
go vet ./tmserver/internal/bot/...                      exit 0
```

Reprovação registrada: `botlab/bot/client.go:19:2: use of internal package … not allowed` (build
vermelho antes de mover o pacote para dentro de `tmserver/`).

## Critérios de aceite

- [x] ADR com contexto, alternativas e consequências
- [x] Localização imposta e justificada (não por preferência)
- [x] Limites declarados: o que o núcleo não faz ainda (`sim_*` em vez de pacotes de combate)
- [x] Recusa de alvo remoto como padrão de segurança

## Riscos / rollback

Risco: o pacote crescer e entrar no caminho de build do servidor sem necessidade. Mitigação: ele é
biblioteca separada, sem import de `world`/`handler`; rollback é remover `tmserver/internal/bot`,
`tmserver/cmd/botlab` e os alvos do `Makefile`.
