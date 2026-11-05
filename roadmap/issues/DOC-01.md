# DOC-01 — Esqueleto mkdocs, índice de sistemas e ADRs

**Fase:** revisão · **Depende de:** — · **Esforço:** L · **Estado:** ⏳ proposta

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
- `docs/adr/**` (novo), `docs/sistemas/**` (novo)
- `roadmap/doc_legacy/**` (novo), `README.md`, `AGENTS.md` §6

## Testes/verificação

```bash
docker compose up -d docs        # http://localhost:8000
mkdocs build --strict            # link quebrado falha
```

## Critérios de aceite

- [ ] `mkdocs build --strict` verde.
- [ ] Todo doc citado no `AGENTS.md` §6 alcançável em ≤2 cliques.
- [ ] Uma página por sistema de jogo, com handler e regra citados.
- [ ] ≥6 ADRs registradas (incluindo o arquivamento do legado C++).
- [ ] `doc_legacy/` com README de substituição por doc movido.

## Riscos/rollback

Risco: mover doc e quebrar link em `AGENTS.md`, skills e issues. Mitigação: `redirect_maps` e `grep`
de links antes de mover. Rollback: movimentação em commits por pasta.
