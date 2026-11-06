# TEST-01 — Triagem dos `t.Skip`: justificar, converter em issue ou apagar

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ✅ concluída (06/11/2026)

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
ex.: `tmserver/content/` fora do runner) — mas a triagem nunca foi feita arquivo a arquivo.

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

## Triagem concluída (06/11/2026)

Antes: **43** ocorrências em 17 arquivos, nenhuma classificada (hoje 44, ver recontagem abaixo).
Depois da triagem: **42**, todas com
`motivo: <classe> — <razão>` no próprio texto do skip.

**Recontagem de 06/11/2026: 47 ocorrências em 19 arquivos** — 43 na onda 1 mais 4 skips novos e
justificados: um da paridade de conteúdo (`tmserver/internal/content/settings_test.go`) e três da
paridade de regiões do worldatlas (dois de conteúdo e um de banco ausente);
um skip já tinha entrado depois da triagem
(na classe `ambiente`) e já nasceu com `motivo:`. Medição e prova:

```bash
bash scripts/check-skips.sh     # check-skips: 47 skip(s), todos com motivo declarado
grep -rIn 't.Skip' --include='*_test.go' . | grep -vc 'motivo:'   # 0
```

| Classe | Qtde | O que significa |
|---|---:|---|
| `conteúdo` | 36 | falta arquivo/árvore (CSV que saiu do repo, mapa, `ItemList.bin`, captura de transporte, árvore `tmserver/content` fora do checkout) |
| `ambiente` | 9 | exige DSN, segunda conta, segunda instância, FS case-sensitive ou serviço externo (recontagem de 06/11) |
| `feature` | 2 | comportamento UNVERIFIED que precisa de captura real (layout 4294, `_AUTH_GAME`) |
| `morto` | 0 restantes | 1 removido |

```bash
bash scripts/check-skips.sh          # 47 skip(s), todos com motivo declarado (06/11)
grep -rn 't.Skip' --include=*_test.go | grep -vc 'motivo:'   # 0
```

### O que foi efetivamente tratado

| Item | Ação |
|---|---|
| `TestTakeInventoryRealContentTree` (`internal/npctemplate`) | **removido** — o diretório `tmserver/content/TMsrv/run/npc/` saiu do repositório na fase 5 do conteúdo; o teste não podia mais rodar em árvore nenhuma deste repo, e `TakeInventory` continua coberto por teste de unidade |
| `TestResolveTrimsLegacyTrailingDot`, `TestResolveCaseInsensitive`, `TestResolveAmbiguousLegacyName` | **estavam vermelhos** no Windows (não por código: o FS normaliza caixa e ponto final, então `Reiners`/`reiners` é o mesmo arquivo). Passaram a pular com `motivo: ambiente` via `requireCaseSensitiveFS`; no Linux/CI os três rodam de verdade |
| CSVs que saíram do repo (`ItemList.csv`, `SkillData.csv`, `NPCGener.txt`) | mensagem passou a dizer **o que** falta e **por que** o caminho do banco cobre o caso; eram 20 ocorrências de "X unavailable: err" sem contexto |
| Skips de e2e (`W2PP_E2E_*`, `W2PP_TEST_DSN`, `E2E_ADMIN_*`) | mensagem pt-BR nomeando a variável de ambiente que liga o teste |
| UNVERIFIED (`4294`, `_AUTH_GAME`, vetor só de schema) | classe `feature`, com o que precisa ser capturado |

### Regra que passa a valer

`scripts/check-skips.sh` roda em `make ci` e no CI: um `t.Skip` novo sem `motivo:` **reprova o job**.
Sem isso, o número volta a crescer e a triagem de hoje vira fotografia.

### Pendências registradas (não fecháveis agora)

- Os 36 de `conteúdo` só deixam de pular quando a árvore externa estiver montada no runner (CI-02/
  DB-01 para o caminho do banco; fixtures reais para o transporte).
- `tmserver/internal/handler/combat_special_test.go` precisa de alvo vivo→morto na mesma sessão; é
  reescrita de teste, não triagem.

## Riscos/rollback

Risco: transformar skip em falha e deixar a suíte vermelha em CI sem conteúdo. Mitigação: os casos de
`conteudo` continuam skip, mas com detecção explícita (`if !fileExists { t.Skip(...) }` + log).
Rollback: a triagem é só edição de teste; reverter é barato.
