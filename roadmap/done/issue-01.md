# Issue 01 — Restaurar e validar a árvore de trabalho

**Fase:** 0 · **Depende de:** — · **Bloqueia:** 02–17 · **Esforço:** S · **Estado:** ✅ concluída (2026-11-02)

## Objetivo

Garantir que o working tree contém **todos** os diretórios versionados — em especial `Release/`,
`scripts/`, `internal/` e `api/` — para que o servidor possa compilar e carregar conteúdo.

## Contexto / evidência

O clone em `D:\dolorestec\wyd2\OpenWYD` estava com **sparse checkout** ativo:

```
$ git config core.sparseCheckout      → true
$ git sparse-checkout list            → Source/Code, docs, tmserver
$ ls                                   → Source, docs, tmserver (só)
$ git ls-files Release | wc -l         → centenas de arquivos versionados
$ test -f Release/Common/ItemList.csv  → MISSING
```

Efeitos observáveis: `go build ./...` impossível (imports de `internal/`, `api/` ausentes),
`make run-local` sem `scripts/run-local.sh`, e um tmServer que subiria sem conteúdo — sintoma
clássico "aceita TCP e nenhum login funciona" (`logins report no account`).

## Passos

```bash
# (comandos rodam na raiz do repositório)
git sparse-checkout list            # se imprime algo, o checkout é parcial
git status --short                  # deve estar limpo antes de mexer
git sparse-checkout disable         # restaura toda a árvore versionada
ls                                  # esperado: Release, api, binserver, dbserver, internal, scripts, webserver…
```

Depois, validar a integridade do conteúdo:

```bash
test -f Release/Common/ItemList.csv && echo "ItemList ok"
test -f Release/Common/SkillData.csv && echo "SkillData ok"
ls Release/Common/Settings          # CompRate SancRate QuestsRate CastleQuest MobMerc
ls Release/TMsrv/run/npc | wc -l    # ~1991 templates
ls -l Release/TMsrv/run/HeightMap.dat Release/TMsrv/run/AttributeMap.dat
ls -l Release/DBsrv/run/BaseMob     # TK FM BM HT, 816 bytes cada
```

## Arquivos

- Alterados: nenhum arquivo de conteúdo (só estado do checkout git).
- Documentação: `AGENTS.md` §3.8 e skill `wyd-server-ops` §1 registram a armadilha.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
git sparse-checkout list 2>&1        # esperado: erro/vazio (sem sparse)
ls | wc -l                           # esperado ~20 entradas na raiz
test -d Release && du -sh Release     # esperado ~67M
```

Saída obtida: `Release` com `Common/`, `DBsrv/`, `TMsrv/`; `npc/` com **1991** arquivos;
`HeightMap.dat` 16.777.216 B; `AttributeMap.dat` 1.048.576 B; `BaseMob/{BM,FM,HT,TK}` 816 B.

## Critérios de aceite

- [x] `git sparse-checkout list` vazio
- [x] `Release/` presente e com as três subárvores
- [x] `scripts/run-local.sh`, `internal/`, `api/`, `dbserver/`, `binserver/`, `webserver/` presentes
- [x] Contagem de templates de NPC > 1900 e mapas com tamanho esperado

## Riscos / rollback

- **Risco:** alguém rodar `git sparse-checkout set` de novo (economia de disco) e reintroduzir o
  problema. **Mitigação:** o preflight da issue 02 falha explicitamente quando `Release/` falta.
- **Risco:** `Release/` é ignorado pelo padrão `[Rr]elease/` do `.gitignore` herdado; arquivos
  *novos* ali podem não aparecer no `git status`. **Mitigação:** use `git status --ignored Release`
  ao adicionar conteúdo.
- **Rollback:** nenhum necessário (operação não destrutiva). Reativar sparse checkout só com pedido
  explícito e ciente de que o servidor fica inutilizável.
