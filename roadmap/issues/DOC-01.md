# DOC-01 — Esqueleto mkdocs, índice de sistemas e ADRs

**Fase:** revisão · **Depende de:** — · **Esforço:** L · **Estado:** 🟢 concluída (06/11/2026)

## Objetivo

Uma porta de entrada para a documentação: site navegável, índice por sistema de jogo e ADRs das
decisões caras — sem depender de abrir 283 arquivos `.md` um por um.

## Contexto/evidência (05/11/2026)

```bash
git ls-files '*.md' | wc -l     # 283
ls mkdocs.yml                   # não existe
ls docs/adr                     # não existe
```

- `docs/` tem engenharia reversa boa (`docs/migration/`, 58/58 handlers), análise (`docs/analysis/`),
  integrações e relatórios de agente — mas nenhum índice executável.
- `AGENTS.md` §6 é o melhor mapa existente (“preciso de X → leio Y”) e serve de semente para o `nav`.
- Nesta sessão, arquivar as ferramentas C++ exigiu criar um README de `_legacy` porque não havia
  lugar canônico para “o que substituiu o quê”.

## Passos

1. `mkdocs.yml` (tema material, busca) com `docs_dir` apontando para `docs/`; container `docs` no
   compose (porta 8000) e job de build no CI (`mkdocs build --strict`).
2. `nav` derivado do `AGENTS.md` §6: Começando · Arquitetura · Serviços · Protocolo e formatos ·
   Regras de jogo · Handlers · Banco · Portal · Cliente · Operação · Roadmap · ADRs.
3. Índice de sistemas de jogo (lacuna real): uma página por sistema — personagem/evolução, combate,
   skills/afects, guildas, castelos/guerras, quests, drops, eventos, comércio, comandos — citando o
   handler e o doc de regra.
4. ADRs em `docs/adr/0001-…`: Go como linguagem do servidor, goroutine única do mundo, BFF do portal,
   banco como fonte de conteúdo, `delivery_queue`, cliente fechado + ClientPatch, arquivamento de
   `Source/Code/_legacy`.
5. `roadmap/doc_legacy/` para doc superado, com README de substituição (as issues REVIEW-06/10 já
   pedem; nada é apagado antes de o site cobrir).

## Arquivos

- `mkdocs.yml`, `docker-compose.yaml` (serviço `docs`), `.github/workflows/docs.yml`
- `docs/adr/**` (novo), `docs/sistemas/**` (índice + 12 páginas de sistema)
- `docs/assets/portal-theme.css`, `docs/guias/**`, `docs/lore/**` (integração portal, 06/11)
- `portal/scripts/build-guias.mjs`, `portal/src/content/guias.json`,
  `portal/src/app/(public)/guias/**`, `portal/src/server/guias.ts` (integração portal, 06/11)
- `roadmap/doc_legacy/**` (novo), `README.md`, `AGENTS.md` §6

## Testes/verificação

```bash
docker compose up -d docs        # http://localhost:8000
mkdocs build --strict            # link quebrado falha
```

## Critérios de aceite

- [x] `mkdocs build --strict` verde.
- [x] Todo doc citado no `AGENTS.md` §6 alcançável em ≤2 cliques.
- [x] Uma página por sistema de jogo, com handler e regra citados (12 páginas).
- [x] ≥6 ADRs registradas (incluindo o arquivamento do legado C++).
- [x] `doc_legacy/` com README de substituição por doc movido (criado; nada foi apagado).

## Resultado (06/11/2026)

| Critério | Estado |
|---|---|
| `mkdocs build --strict` verde | ✅ 46,06 s, exit 0 |
| Todo doc do `AGENTS.md` §6 em ≤2 cliques | ✅ (nav do `mkdocs.yml` é o §6 traduzido em seções) |
| Uma página por sistema de jogo | ✅ 12 páginas + índice em `docs/sistemas/` |
| Mesma identidade visual do portal | ✅ `docs/assets/portal-theme.css` (tokens de `globals.css`) |
| Docs alimentam áreas do portal | ✅ `/guias` publica **19** páginas de `docs/` via JSON versionado |
| ≥6 ADRs | ✅ 6 (0001–0006) |
| `doc_legacy/` com README de substituição | ✅ `roadmap/doc_legacy/README.md` |

**Build verificado de verdade**, em container (o host não tem Python):

```bash
docker run --rm -v "$PWD:/docs" squidfunk/mkdocs-material build --strict
# INFO - Documentation built in 48.65 seconds   exit 0
```

A primeira execução **reprovou**, e o motivo era útil: `launcher-plan.md`, `analysis/README.md` e
`analysis/migracao-legado.md` linkavam para `roadmap/` e `Source/` — destinos fora de `docs_dir`, que
existem no repositório mas são link morto num site com `docs_dir: docs`. Os três viraram caminho em
`code`, não link. **Esse é o custo real de adotar o site**: 119 links relativos em `docs/`, 3 deles
apontando para fora.

### Gate que roda sem mkdocs

`scripts/check-docs-links.sh` mede o mesmo em qualquer host (o Windows não tem Python):

```
check-docs-links: 119 link(s) relativos, 0 quebrado(s), 3 fora de docs/
```

### O que fechou em 06/11/2026

1. **Página por sistema** — `docs/sistemas/` passou a ter uma página por sistema catalogado
   (personagem, combate, skills, drops, refino, quests, comércio, guildas, castelos, eventos, canais,
   mapa). Cada uma segue o mesmo contrato: fluxo, tabela peça → arquivo, comando de verificação e
   bloco UNVERIFIED onde o dado depende de conteúdo do cliente ou de medição que não existe.
2. **`roadmap/doc_legacy/`** — README de substituição com contrato de arquivamento e o comando que
   mostra o que ficou fora do site (`git ls-files`), em vez de mover arquivo às cegas.
3. ~~Serviço `docs` no compose (porta 8000)~~ — **feito**: `docs/mkdocs.yml`, `docs/Dockerfile` e o
   serviço `mkdocs` no compose, sob `--profile docs`, para nada subir junto com o jogo:
   `docker compose --profile docs up mkdocs` (http://localhost:8000) e
   `docker compose --profile docs run --rm mkdocs build -f docs/mkdocs.yml --strict` como gate.

Nada disso é bloqueio técnico — eram conteúdo e uma decisão de arquivamento, agora fechados. Evidência
datada (06/11/2026, Windows + Git Bash):

```make guias             # build-guias: 19 página(s)
make guias-check       # exit 0 (sem diff em portal/src/content/guias.json)
bash scripts/check-docs-links.sh
                       # 139 link(s) relativos, 0 quebrado(s), 0 fora de docs/
docker compose --profile docs run --rm mkdocs build -f docs/mkdocs.yml --strict
                       # Documentation built in 46.06 seconds, exit 0
npx vitest run         # 13 arquivos, 101 testes, 0 falha
npx tsc --noEmit       # exit 0
```

O teste novo `portal/src/server/guias.test.ts` não checa só o JSON: ele varre as páginas de sistema,
extrai todo arquivo `.go` citado entre crases e reprova se ele não existir em
`tmserver/internal/handler`. Ao ser escrito, ele **achou quatro nomes mortos** no índice antigo
(`attack.go`, `characterlogin.go`, `npcshop*.go`, `worldevents.go`) e três citações minhas sem caminho
(`clan.go`, `event.go`, `nightmare_event.go` — este último nem existe, só o `_test.go`). Todos foram
corrigidos; a auditoria em shell sobre `docs/sistemas/*.md` hoje não acusa nada.

## Riscos/rollback

Risco: mover doc e quebrar link em `AGENTS.md`, skills e issues. Mitigação: `redirect_maps` e `grep`
de links antes de mover. Rollback: movimentação em commits por pasta.
