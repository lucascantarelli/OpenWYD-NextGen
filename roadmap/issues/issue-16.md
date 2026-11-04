# Issue 16 — Bots de teste e harness de carga

**Fase:** 4 · **Depende de:** 07, 11 · **Bloqueia:** 17 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Ter personagens controlados por IA que jogam o servidor sozinhos — login, andar, matar, lootear,
comprar, usar quest — servindo de teste de fumaça contínuo e de gerador de carga realista.

## Contexto / evidência

- O projeto já tem login headless (`-tags=e2e`, `TestE2ESmokeLogin`) e codecs reutilizáveis em
  `tmserver/internal/protocol`; é a base natural de um cliente de teste.
- Existem clientes/bots externos catalogados em
  `docs/agents/CLIENT-RESEARCH-2026-06-27.md`: `kevinkouketsu/WYD2Bot` (C/C++, headless) e
  `kevinkouketsu/Wyd2Client` (C#, UI). Ambos exigem ajuste de offset/versão e têm risco 🟡 (são bots).
- O servidor limita entrada a 200 msg/s com burst 400 (`-max-msg-per-sec`, `-msg-burst`).
- O loop de jogo é único: carga de bot afeta **todo** o servidor, então bots de carga não devem rodar
  na stack que alguém está testando à mão.

## Passos

1. Decidir o caminho (recomendado: bot interno em Go, usando `internal/protocol`) e registrar a
   decisão na issue.
2. Implementar o roteiro mínimo: login → listar/criar personagem → entrar → andar até coordenada →
   atacar mob → pegar loot → abrir loja/Comprar → deslogar, com asserts a cada passo.
3. Parametrizar por conta/char/roteiro (`bot1..botN`) e por ritmo, respeitando o rate limit.
4. Cobrir os cenários de aceitação da skill `wyd-testing-bots` §6 como roteiros nomeados.
5. Adicionar modo "smoke": 1 bot, 60 s, veredito binário (usar em CI/preflight).
6. Adicionar modo "carga": N bots com perfil de movimento/combate, medindo tick do servidor
   (`BenchmarkTickIdle10k` como referência) e latência de pacote.
7. Registrar evidências: quantos bots, por quanto tempo, com quais erros.

## Arquivos

- Criar: `cmd/wydbot/main.go` (ou `tmserver/cmd/bot`) + lógica em `tmserver/internal/bot/`.
- Testes: `tmserver/internal/bot/bot_test.go` (roteiro contra servidor de teste em memória).
- Alterar: skill `wyd-testing-bots` (comandos reais), `Makefile` (alvo `bot-smoke`),
  `.github/workflows/` se o smoke entrar no CI.
- Proibido: apontar bot para servidor de terceiros; commitar binário de bot de origem duvidosa.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
# roteiro determinístico (sem rede) 
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -race ./tmserver/internal/bot/
# smoke contra a stack local
docker run --rm --network host -v "$PWD":/src -w /src golang:1.26-alpine \
  go run ./cmd/wydbot -addr 127.0.0.1:8281 -account bot1 -pass bot123 -script smoke -duration 60s
# carga
go run ./cmd/wydbot -bots 10 -script grind -duration 5m
```

Saída esperada: smoke com `PASS` e relatório de passos; carga sem desconexão por rate limit e com
latência de tick estável (comparar com o benchmark).

## Critérios de aceite

- [ ] Bot completa o roteiro mínimo com asserts (não só "não caiu")
- [ ] Conta/char dedicados; nunca a conta de teste humano
- [ ] Respeita rate limit (sem disconnect em cascata)
- [ ] Modo smoke com veredito binário e duração curta
- [ ] Modo carga com relatório (bots, duração, erros, tick/latência)
- [ ] Roteiros cobrem login, movimento, combate, loot, loja e quest

## Riscos / rollback

- **Risco:** bot mascarar bug por ser tolerante. **Mitigação:** asserts estritos e log de divergência.
- **Risco:** carga derrubar o servidor em teste alheio. **Mitigação:** stack separada para carga;
  avisar antes; nunca durante validação humana.
- **Risco:** dependência de bot externo trazer binário não auditado. **Mitigação:** preferir interno;
  se usar externo, compilar da fonte.
- **Rollback:** bots usam contas próprias; encerrar processos e remover contas de teste.
