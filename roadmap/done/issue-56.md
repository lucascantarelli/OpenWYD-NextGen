# Issue 56 — Limpeza do repositório e arquivamento do legado

**Fase:** 8 · **Depende de:** — · **Esforço:** S · **Estado:** ✅ fechada (05/11/2026)

## Objetivo

Remover do versionamento o que não deve estar lá (segredos de topologia, binários, artefatos), decidir
o destino de `Source/`, `openspec/`, `project-analizer/` e `.claude/`, e deixar o repositório com uma
única cópia de cada coisa.

## Contexto

- **`serverlist bin/`**: 4 pares de `serverlist.bin` com IPs privados + editor `.exe` **versionados**,
  o que viola o `AGENTS.md` (nunca versionar `serverlist.bin` gerado com IP privado).
- `Source/` (198 arquivos) é referência histórica; nenhum build o usa.
- `openspec/`: 5 mudanças paradas + 3 specs, **zero** referência no código (ver
  [`docs/analysis/estado-atual.md`](../../docs/analysis/estado-atual.md) §7).
- `project-analizer/`: plugin de terceiro, produto em `docs/agents/` (arquivar na issue 54).
- `.claude/skills` e `.agents/skills` se sobrepõem (20 skills, 41 arquivos).

## Decisões tomadas (05/11/2026)

| Item | Decisão | Motivo |
|---|---|---|
| `serverlist.bin` | **nunca versionado** + regra `serverlist bin/**/*.bin` no `.gitignore` | já não havia arquivo rastreado; a regra impede a volta |
| `Source/` | **fica no lugar**, read-only, `DropTool` movido para `Source/Code/_legacy/` | é a referência de wire/formato citada pelos docs; nenhum build o usa |
| `openspec/` | **arquivado** em `docs/archive/openspec/` com README de substituição | 0 referências em `.go`, 5 mudanças paradas desde 2026-09 e o plano real é o `roadmap/` |
| skills `openspec-*` | **removidas** (6) | sem o fluxo, viram ruído; o caminho é proposta → issue |
| `.claude/skills` | **removido** — era só a cópia dos 6 `openspec-*` | fonte única passa a ser `.agents/skills` |
| `project-analizer/` | já removido em `f39aed3` | produto vive em `docs/agents/` |
| artefatos soltos | **removidos**: `Conversor.rar`, `Source/enc_temp_folder/`, `*.tlog` do ClientPatch | lixo de build/editor, nada os lê |

## Passos

1. `git rm --cached` nos `.bin` de `serverlist bin/**`; `.gitignore` para `serverlist bin/**/*.bin`;
   manter o editor local (ou remover também, quando a issue 51 gerar o bin).
2. Mover `Source/` para `archive/Source/` (ou manter no lugar com nota e sparse-checkout documentado);
   README/AGENTS explicam que é referência read-only.
3. **Decidir openspec**: adotar (preencher `config.yaml` com contexto do projeto e usar em mudanças
   grandes) **ou** arquivar em `docs/archive/openspec/` e remover as 6 skills `openspec-*`.
4. **Decidir project-analizer**: mover para `tools/project-analyzer/` com README ou remover.
5. **Unificar skills**: `.agents/skills` canônico; `.claude/skills` vira symlink/espelho gerado (ou sai).
6. Remover artefatos soltos: `Conversor.rar`, `enc_temp_folder/`, `*.tlog`, `logpipe.exe` se aparecer.

## Arquivos

- `.gitignore`, `README.md`, `AGENTS.md`, `openspec/**` ou `docs/archive/openspec/**`,
  `project-analizer/**` → `tools/`, `Source/` (movido), `.claude/skills` (link/remoção).

## Verificação

```bash
git status --short --ignored "serverlist bin" | head
git ls-files | grep -Ei 'serverlist.*\.bin$' || echo "nenhum serverlist.bin versionado"
git grep -n "Source/" -- ':!docs' ':!*.md' || echo "nenhuma referencia de build ao Source"
go build ./... && echo BUILD_OK
```

## Critérios de aceite

- [x] Nenhum `serverlist.bin` versionado; `.gitignore` cobre a pasta
- [x] `Source/` com status explícito (arquivo histórico) e sem referência de build
- [x] `openspec` e `project-analizer` com decisão registrada (adotar ou arquivar)
- [x] Uma única fonte de skills (`git ls-files .claude/skills .agents/skills` sem duplicata)
- [x] Build e testes verdes depois da limpeza

## Riscos

- **Remover algo em uso**: `git rm --cached` não apaga o arquivo local; validar com `go build` e com o
  bring-up completo antes do commit.
- **Perder histórico**: mover com `git mv` (preserva blame), nunca copiar e apagar.

## Evidência de execução (05/11/2026)

```text
$ git ls-files | grep -Ei 'serverlist.*\.bin$'
(nenhum)                            # a pasta "serverlist bin" nem existe mais no repo

$ git status --porcelain | awk '{print $1}' | sort | uniq -c
      1 ??   (docs/archive/openspec/README.md)
     21 D    (Conversor.rar, Source/enc_temp_folder/**, *.tlog do ClientPatch)
      2 M
     73 R    (openspec/** → docs/archive/openspec/**)

$ ls .agents/skills
documentation-writer finishing-a-development-branch frontend-design github-actions-docs
skill-creator ui-ux-practices writing-plans wyd-client-{connect,database}
wyd-economy wyd-game-admin wyd-server-{dev,ops} wyd-testing-bots wyd-web-admin-ui
$ ls .claude
commands settings.json               # skills saiu: era só a cópia dos openspec-*

$ go build ./api/... ./internal/... ./dbserver/... ./tmserver/... ./webserver/... ./binserver/...
BUILD_OK   (container golang:1.26-alpine)
```

Documentação atualizada na mesma entrega: `AGENTS.md` (§1 e §7, sem `openspec-*` e sem
`ItemDropList.txt`), `docs/migration/config-ops.md`, `docs/migration/data-formats.md`,
`docs/analysis/migracao-legado.md` (DropTool aponta para `_legacy`) e
`docs/archive/openspec/README.md` (mapa do que substituiu cada parte do fluxo).
