# TEST-01 — Triagem dos 43 `t.Skip`: justificar, converter em issue ou apagar

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Nenhum `t.Skip` escondendo regressão: cada skip é *conteúdo ausente* (mantém, com motivo no teste),
*feature não implementada* (vira issue + `TODO`) ou *teste morto* (apaga).

## Contexto/evidência (05/11/2026)

```bash
git grep -o 't.Skip' -- '*_test.go' | wc -l    # 43 ocorrências
git grep -l 't.Skip' -- '*_test.go' | wc -l    # 17 arquivos
```

43 skips em 17 arquivos é volume suficiente para esconder uma regressão real: um teste que salta é
reportado como `ok` no pacote. O `REVIEW-03` estimou parte legítima (conteúdo proprietário ausente,
ex.: `Release/` fora do runner) — mas a triagem nunca foi feita arquivo a arquivo.

## Passos

1. Inventariar: para cada ocorrência, registrar arquivo:linha, motivo declarado e classificação
   (`conteudo`, `feature`, `morto`).
2. `conteudo` (ex.: falta `HeightMap.dat`/`BaseMob`): manter, mas com mensagem que diga **o que**
   falta e como obter; se o conteúdo já é semeado no banco, o teste deve migrar para o caminho do
   banco em vez de pular.
3. `feature`: o skip sai e entra `t.Fatal("não implementado — ISSUE-XX")` **apenas** se a issue
   existir; senão o teste é removido com justificativa no commit.
4. `morto` (testa CSV/`NPCGener.txt`/arquivo que não existe mais): apagar; o caminho do banco já tem
   teste próprio.
5. Gate: teste novo com `t.Skip` exige comentário no arquivo com motivo + data; revisão periódica
   mensal (`grep` no PR).

## Arquivos

- `*_test.go` nos 17 arquivos afetados
- `docs/development-guidelines/testing.md` (regra do skip, criado na TEST-02/TEST-03)

## Testes/verificação

```bash
git grep -n "t.Skip" -- '*_test.go'
go test ./...                 # nenhum pacote "ok" por skip sem motivo
grep -rn "t.Skip" --include=*_test.go . | grep -v "motivo:" | wc -l   # 0
```

## Critérios de aceite

- [ ] Toda ocorrência classificada e com motivo no próprio arquivo.
- [ ] Zero skip testando arquivo que não existe mais no repositório.
- [ ] Nenhum skip de feature sem issue correspondente.
- [ ] Números antes/depois registrados na issue.

## Riscos/rollback

Risco: transformar skip em falha e deixar a suíte vermelha em CI sem conteúdo. Mitigação: os casos de
`conteudo` continuam skip, mas com detecção explícita (`if !fileExists { t.Skip(...) }` + log).
Rollback: a triagem é só edição de teste; reverter é barato.
