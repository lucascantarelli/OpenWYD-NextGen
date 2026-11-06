# REVIEW-06 — Documentação: mkdocs, reorganização e ADRs

**Fase:** revisão · **Depende de:** 58 · **Esforço:** L · **Estado:** ⏳ proposta

## Objetivo

Uma documentação com porta de entrada: site navegável (mkdocs), índice de sistemas de jogo, ADRs das
decisões caras e separação entre **documentação de produto** (jogador/admin) e **engenharia reversa**
(o que já existe e é bom).

## Contexto/evidência (05/11/2026)

- 112 arquivos `.md` em `docs/`, distribuídos em `analysis/` (7), `audits/`, `agents/`,
  `integrations/`, `migration/` (handlers 58/58, flows, protocol-spec, data-formats, game-rules,
  config-ops, ingame-bugs), além de planos soltos na raiz (`celestial-system-plan`,
  `daily-reward-frontend`, `donate-shop-frontend`, `item-icons-plan`, `gm-commands-plan`,
  `investigacao-freeze-cliente`, `captura-wyd-*`).
- **Não existe `mkdocs.yml`** nem container de docs; a leitura depende de abrir arquivo por arquivo no
  GitHub.
- Não há ADR (`docs/adr/`) nem índice de sistemas de jogo (evolução de personagem, guerras, skills,
  guildas, quests, drops, eventos, comandos).
- `AGENTS.md` §6 já é um bom mapa "preciso de X → leio Y": é a semente do índice do site.

## Alanisar tudo
- mapear,documentar e organizar toda a documentação do projeto de forma coerente e acessível, incluindo descrições de funcionalidades, implementações, features a serem implementadas, mapeamentos, regras de negócio, etc. para que possamos ter um conhecimento mais profundo sobre o funcionamento do projeto e suas funcionalidades.
- caso precise pode mudar as pastas das docuemntações, recriar tudo novamente, atualizar as documentações existentes, organizar em indices, melhorar os recursos da documentação com recursos ricos, emojis, cores, imagens, links, etc. para que possamos ter uma documentação mais completa, acessível e fácil de entender.
- Crie uma documentação completa sobre o projeto, incluindo documentação técnica para desenvolvimento, documentação administrativa para gestão do sistema, documentação de usuário para utilização da plataforma, documentação de todos os sistemas do jogo, incluindo evolução de personagem, guerras, skills, guilds, quests, drops de mobs, eventos,comandos de usuário, comandos administrativos etc. para que possamos ter um conhecimento mais profundo sobre o funcionamento do projeto e suas funcionalidades.
- Faça uma pesquisa extensa na internet sobre todos os sistemas, itens, como é executado o sistema, prerequisitos, tudo, documente em docs/sistemas/
quero que vc entenda como tudo funciona, evolução, guerras, eventos, uso dos itens, para que cada coisa serve e documente tudo, para podermos implementar corretamente todos os comportamentos.
pesquise referencias em sites de servidores privates/oficiais para referencias de eventos, uso de itens, para que cada item serve, bugs, skills, novos sistemas diferentes dos originais/padrões
- analisar  o que pode ser melhorado após a análise completa, o que deve ser atualizar, reaorganizado,e demais ajustes, proponha melhorias, modernizações, avanços.


## Passos

1. **Esqueleto mkdocs**: `mkdocs.yml` (tema material, busca, navegação por seções), `docs-site/` como
   origem, container `docs` no compose (porta 8000) e job de build no CI.
2. **Mapa de navegação** derivado do `AGENTS.md` §6: Começando · Arquitetura · Serviços (tm/db/bin/web)
   · Protocolo e formatos · Regras de jogo · Handlers · Banco · Portal · Cliente · Operação ·
   Roadmap · ADRs.
3. **Índice de sistemas de jogo** (novo, o maior lacuna): uma página por sistema — personagem/evolução,
   combate, skills/afects, guildas, castelos e guerras, quests, drops, eventos, comércio, comandos de
   jogador e de GM — cada uma citando o handler e o doc de regra.
4. **ADR**: `docs/adr/0001-…` para as decisões que custam reverter: Go como linguagem do servidor,
   goroutine única do mundo, BFF do portal, banco como fonte de conteúdo, `delivery_queue`, `logpipe`,
   cliente fechado + ClientPatch.
5. **Legado**: mover documentação superada para `roadmap/doc_legacy/` (com README explicando o que
   substituiu) — nada é apagado antes de o site cobrir o assunto.
6. **Doc de produto**: guia do jogador (comandos, sistemas) e guia do administrador (papéis, telas,
   auditoria, GM), escritos para quem opera — não para quem programa.

## Arquivos

- `mkdocs.yml`, `docs-site/**` (ou `docs/` como origem com `docs_dir`), `docker-compose.yaml`
- `docs/adr/**` (novo)
- `docs/sistemas/**` (novo)
- `roadmap/doc_legacy/**` (novo)
- `README.md`, `AGENTS.md` §6 (apontar para o site)

## Testes/verificação

```bash
docker compose up -d docs        # http://localhost:8000
mkdocs build --strict            # links quebrados e nav inconsistente falham
```

## Critérios de aceite

- [ ] `mkdocs build --strict` verde (sem link quebrado).
- [ ] Todo doc de `AGENTS.md` §6 alcançável em ≤2 cliques no site.
- [ ] Uma página por sistema de jogo, com handler e regra citados.
- [ ] ≥6 ADRs registradas.
- [ ] `doc_legacy/` com README de substituição para cada doc movido.

## Riscos/rollback

Risco: mover arquivo e quebrar link em `AGENTS.md`, skills e issues. Mitigação: redirects do mkdocs
(`redirect_maps`) e um `grep` de links antes de mover. Rollback: movimentação em commits por pasta.

## Issues que este review abre

- `DOC-01` esqueleto mkdocs + container + CI
- `DOC-02` índices de sistemas de jogo
- `DOC-03` ADRs
- `DOC-04` `doc_legacy/` e redirects
- `DOC-05` guia do jogador e guia do administrador
