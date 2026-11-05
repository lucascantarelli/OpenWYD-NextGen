# Issue 56 — Limpeza do repositório e arquivamento do legado

**Fase:** 8 · **Depende de:** — · **Esforço:** S · **Estado:** ⏳ aberta

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

- [ ] Nenhum `serverlist.bin` versionado; `.gitignore` cobre a pasta
- [ ] `Source/` com status explícito (arquivo histórico) e sem referência de build
- [ ] `openspec` e `project-analizer` com decisão registrada (adotar ou arquivar)
- [ ] Uma única fonte de skills (`git ls-files .claude/skills .agents/skills` sem duplicata)
- [ ] Build e testes verdes depois da limpeza

## Riscos

- **Remover algo em uso**: `git rm --cached` não apaga o arquivo local; validar com `go build` e com o
  bring-up completo antes do commit.
- **Perder histórico**: mover com `git mv` (preserva blame), nunca copiar e apagar.
