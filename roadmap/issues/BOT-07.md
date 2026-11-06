# BOT-07 — Comportamento social e LLM opcional

**Fase:** BOT · **Depende de:** BOT-03 · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

População com comportamento social (chat, convite de grupo, negociação) e camada **opcional** de LLM
local para diálogo, sem que a ausência dela impeça qualquer cenário.

## Contexto / evidência

- `profile.go` já tem `ChatChance` e perfis `social`/`comerciante`; falta o envio real de chat.
- A recusa genérica do servidor hoje vira texto visível no cliente (`notice.go`, 42 códigos com texto
  em `Language.txt`) — o bot precisa ler o painel de texto para reagir, não o número.
- Modelos locais candidatos (Phi mini, Qwen 3 compacto, Gemma, Llama quantizado) exigem validação de
  licença, memória e suporte real a saída estruturada; nenhum foi avaliado ainda.
- Regra do documento de missão: núcleo determinístico primeiro; LLM é complementar.

## Passos

1. Implementar chat real com limite de frequência por perfil (mensagens por minuto, não por segundo).
2. Implementar negociação determinística (oferta, contraproposta, aceite) entre dois bots.
3. Definir a interface `Talker { Say(ctx, msg) (string, error) }` com implementação nula (padrão) e
   implementação HTTP apontando para um serviço local — desligada por padrão.
4. Avaliar **um** modelo compacto local em container, medindo latência e memória, antes de aceitar.
5. Garantir que todo cenário rode com `Talker` nulo e produza o mesmo veredito.

## Arquivos

- `tmserver/internal/bot/social.go` (novo)
- `tmserver/internal/bot/talker.go` (novo, interface + nulo)
- `tmserver/internal/bot/scenarios/social.json`

## Testes / verificação

```
go test -race ./tmserver/internal/bot/ -run TestSocial
```

Saída esperada: dois bots trocam oferta e fecham negociação com seed fixa; chat respeita o limite por
minuto; cenário com `Talker` nulo passa igual.

## Critérios de aceite

- [ ] Negociação determinística reproduzível por semente
- [ ] Chat com limite de frequência e conteúdo configurável
- [ ] LLM atrás de interface e desligado por padrão
- [ ] Cenário idêntico com e sem LLM (veredito igual)

## Riscos / rollback

Risco: LLM inundar o servidor com mensagens artificiais. Mitigação: limite por minuto e canal de log.
Rollback: `Talker` nulo é o padrão; nada quebra.
