# BOT-03 — Ações reais de combate, movimento e comércio

**Fase:** BOT · **Depende de:** BOT-02 · **Bloqueia:** BOT-04, BOT-06 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Substituir as ações `sim_*` do executor por pacotes reais, de modo que o bot ande, ataque, recolha loot,
compre e venda **no servidor**, com invariantes conferidos a cada passo.

## Contexto / evidência

- Hoje (`BOT-02`) o bot só fala conexão, INITCODE, login, seleção de personagem e leitura de frames.
  As ações de caça mutam apenas o modelo local.
- Formatos já implementados e testados em `tmserver/internal/protocol`: `MsgActionBody` (movimento,
  `messages.go:159`), `MsgAttack` (0x0367/0x039D/0x039E), `EncodeShopListBody`, `MsgWhisperBody`,
  `MsgMessageChat`.
- A regra do projeto proíbe inventar formato de pacote: cada ação só entra com vetor de bytes
  conferido contra `Source/` ou contra captura do cliente.

## Passos

1. Para cada ação (andar até coordenada, atacar, lootar, abrir loja, comprar, vender), escrever o
   teste de vetor de bytes no pacote `protocol` **antes** da ação no bot.
2. Implementar as ações como funções `Conn`-agnósticas: elas recebem o payload já codificado.
3. Trocar cada `sim_*` por envio + espera de confirmação (eco `MSG_Action`, `MSG_Attack`, `MSG_ShopList`).
4. Conferir invariantes reais a cada passo: HP, MP, EXP, gold e item no inventário vêm do snapshot do
   servidor, não do modelo.
5. Medir o custo: quantos pacotes por segundo um bot de caça gera e qual o tick do mundo sob 1/10/50
   bots, comparando com `BenchmarkTickIdle10k`.

## Arquivos

- `tmserver/internal/bot/actions.go` (novo)
- `tmserver/internal/protocol/*_test.go` (vetores novos)
- `tmserver/internal/bot/scenarios/*.json` (roteiros de caça e loja)

## Testes / verificação

```
go test -race ./tmserver/internal/bot/...
go test -run TestE2EBotHunt -tags=e2e ./tmserver/internal/bot/ -v   # stack local
```

Saída esperada: bot mata N mobs, EXP e gold sobem no relatório **lidos do servidor**, inventário
confere, sem desconexão por rate limit.

## Critérios de aceite

- [ ] Cada ação nova tem vetor de bytes no teste do `protocol`
- [ ] Nenhuma métrica `sim_*` de combate/comércio permanece sem pacote correspondente
- [ ] Invariantes lidas do servidor, não do modelo local
- [ ] Nenhuma desconexão por rate limit em 60 s de caça
- [ ] Custo medido e registrado (pacotes/s e tick do mundo)

## Riscos / rollback

Risco: o bot mascarar bug por ser tolerante. Mitigação: asserção estrita e log de divergência.
Risco: carga derrubar a stack de alguém. Mitigação: ambiente separado e aviso antes.
Rollback: manter as ações atrás de flag e voltar ao modo `sim_*`.
