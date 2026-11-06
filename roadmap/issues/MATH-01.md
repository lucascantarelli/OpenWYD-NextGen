# MATH-01 — Catálogo único de atributos e fórmulas com nível de confiança

**Fase:** matemática · **Depende de:** — · **Bloqueia:** MATH-02 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Ter um documento (e, onde der, testes) que diga, para cada atributo e fórmula do jogo, de onde o número
vem, em que unidade está, qual o limite, como arredonda e **qual o nível de confiança** — para que
nenhuma fórmula de hipótese entre em código como se fosse fato.

## Contexto / evidência

- As regras hoje estão espalhadas: `docs/migration/game-rules.md` (fórmulas), `docs/sistemas/`
  (`combate.md`, `refino.md`, `drops.md`, `skills.md`, `personagem.md`), `Source/Code/TMSrv`
  (referência) e o código (`tmserver/internal/handler`, `world`).
- Não existe índice que responda "qual o multiplicador de crítico e com que evidência?".
- O golden case já é prática do projeto (`protocol/*_test.go`, testes de dano/EXP), mas não há
  catálogo que diga o que ainda **não** tem golden.
- Medições recentes de QA in-game dão números reais (defesa 149→189 com +9, ataque 69→118) que hoje
  só existem no `roadmap/index.md`.

## Passos

1. Levantar, por leitura de código, todos os atributos lidos/escritos (`STRUCT_MOB`, `score`, `affect`).
2. Para cada fórmula encontrada, registrar: expressão, arquivo:linha, unidade, arredondamento, nível
   A/B/C e teste que a trava (ou "sem teste").
3. Consolidar em `docs/sistemas/matematica.md` (página do sistema) + entrada em `docs/migration/game-rules.md`.
4. Abrir issue filha para cada fórmula de nível C que precise de medição.
5. Acrescentar ao teste de guias a checagem de que os arquivos citados existem (já existe para `.go`).

## Arquivos

- `docs/sistemas/matematica.md` (novo)
- `docs/migration/game-rules.md` (índice e links)
- `docs/mkdocs.yml` (nav)

## Testes / verificação

```
bash scripts/check-docs-links.sh
mkdocs build --strict
bash scripts/check-docs-links.sh | tail -3
```

Saída esperada: página publicada, zero link quebrado, e toda fórmula com nível declarado.

## Critérios de aceite

- [ ] Todo atributo usado no combate tem linha no catálogo
- [ ] Toda fórmula tem nível A/B/C explícito
- [ ] Toda fórmula de nível A aponta o teste que a trava
- [ ] Lacunas de nível C viraram issue filha

## Riscos / rollback

Risco: catálogo virar cópia da documentação antiga sem medição. Mitigação: campo "evidência" obrigatório
com arquivo:linha ou teste. Rollback: página nova, remover do nav.
