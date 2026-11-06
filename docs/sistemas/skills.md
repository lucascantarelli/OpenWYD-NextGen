# Skills e afects

Skill é o que o jogador conjura; **afect** é o efeito que sobra no alvo (buff, debuff, dano por
tempo). São dois mecanismos: `skill.go` executa a conjuração, `affect_score.go` e `affect_tick.go`
mantêm o efeito vivo a cada tick.

## Fluxo

1. Cliente envia a conjuração; `skill.go` valida custo de MP, alcance, alvo e cooldown.
2. Efeito imediato é aplicado (dano, cura) seguindo `docs/migration/game-rules.md` §5.
3. Efeito com duração vira **afect** com contador em ticks. A duração vem do catálogo de skill no
   banco (`skill_catalog`, vindo do dbserver — `docs/analysis/persistencia-conteudo.md`).
4. A cada tick, `affect_tick.go` decrementa e aplica o efeito periódico; ao chegar a zero, remove e
   reenvia os stats do alvo.

## Onde está

| Peça | Arquivo |
|---|---|
| Conjuração e custo | `skill.go` |
| Aplicação e remoção de efeito | `affect_score.go` |
| Efeito por tempo | `affect_tick.go` |
| Transformação e subcelestial | `transform.go`, `subcelestial.go` |
| Regra | `docs/migration/game-rules.md` §5 |
| Catálogo | `skill_catalog` no Postgres |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Skill|Affect|Transform|Subcelestial'
```

No jogo: usar uma skill de buff e conferir que o ícone some **sozinho** no fim da duração — se ele
ficar preso, o defeito está no tick, não na conjuração.

!!! warning "UNVERIFIED"
    A lista de skills por classe e os valores por nível **não** estão em tabela consultável no
    repositório: vêm do conteúdo do cliente e do `skill_catalog` importado. Dizer "a skill X cura N"
    exige ler o catálogo do ambiente, não este documento.

## Limite atual

`/skills` no portal está marcado como "em breve" (`soon: true` em `portal/src/components/nav.ts`): a
tela e a RPC de administração de skill ainda não existem. A rota está protegida mesmo assim.
