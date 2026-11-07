# Issue 08-4 — Dossiê, inventário e baseline do cliente

**Fase:** 8 · **Depende de:** 01-2 · **Bloqueia:** 08-5, 08-6, 08-7, 08-3 · **Esforço:** M

## Objetivo

Publicar o dossiê do cliente fechado (`client/`) como fonte canônica no repositório:
inventário verificado, hashes do build de referência, superfície já conhecida e o plano de
engenharia reversa — antes de qualquer trabalho de desassemble.

## Contexto / evidência

`client/` tem ~344 MB de binários proprietários, é **ignorado pelo git** e era citado apenas
como "ignorado" em `AGENTS.md` e na skill `wyd-client-connect`. O conhecimento real estava
enterrado em `.deprecated-go/docs/migration/client-patch-hooks.md` (mapa de 11 patches e 7 hooks
por endereço) e `openwyd-client-feasibility.md` (paridade de transporte e divergência de
structs). O `wyd-launcher` (08-3) precisa de manifesto de integridade que só existe se o
cliente estiver inventariado.

## Passos

1. Levantar binários, tamanhos, SHA-256 e arquitetura PE do build 7662.
2. Catalogar dados, catálogos e assets por pasta, com status CONHECIDO/UNVERIFIED.
3. Consolidar CPSock, patches, hooks, pontos de extensão e formatos opacos, com proveniência.
4. Publicar o plano de RE (alvos T1–T8, ferramentas, marcos M0–M6, riscos, DoD).
5. Ligar o dossiê ao `AGENTS.md`, README, roadmap e painel de progresso.

## Arquivos

- Criados: `docs/cliente/README.md`, `inventario.md`, `superficie-conhecida.md`,
  `engenharia-reversa.md`.
- Alterados: `AGENTS.md` (§1, §6, §7), `README.md` (tabela de docs),
  `roadmap-rust/index.md`, `docs/rust/03-progresso.md`, `docs/manual-técnico/skills.md`,
  `.agents/skills/wyd-client-connect/SKILL.md`.
- Proibido tocar: qualquer arquivo em `client/` (binários).

## Testes / verificação

```bash
# os hashes do dossiê batem com o disco
cd client && sha256sum WYD.exe WYDLauncher.exe ClientPatch.dll
# links do README resolvem
for p in $(grep -o '](\([^)]*\.md\))' README.md | sed 's/](//; s/)$//'); do test -e "$p" || echo "QUEBRADO $p"; done
# gate do workspace continua verde (nenhuma crate tocada)
cargo fmt --all -- --check && cargo clippy --workspace -- -D warnings && cargo test --workspace
```

Saída esperada: hashes idênticos, `LINKS-OK`, gate com exit 0.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-client-reverso - skill criada nesta issue.
- .agents/skills/wyd-client-connect - operação e diagnóstico do cliente.
- .agents/skills/documentation-writer - estrutura do dossiê.

## Documentação de referencia

- .deprecated-go/docs/migration/client-patch-hooks.md - patches e hooks por endereço.
- .deprecated-go/docs/migration/openwyd-client-feasibility.md - paridade e divergências.
- docs/cliente/ (novo) - fonte canônica.

## Subtasks

- [x] Codificar issue (documentos)
- [x] Testar issue (hashes, links, gate)
- [ ] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] `docs/cliente/` publicado com inventário, superfície e plano de RE.
- [x] Hashes do build de referência registrados e revalidáveis por comando.
- [x] Toda afirmação de endereço tem build + hash + proveniência; o resto é UNVERIFIED.
- [x] `AGENTS.md`, README, roadmap e progresso apontam para o dossiê.

## Riscos / rollback

- Risco: tratar endereço de outro build como fato → mitigação: hash no topo de cada documento
  e reconfirmação na 08-5.
- Risco: vazar binário para o git → mitigação: `client/` continua no `.gitignore`; dossiê só
  registra hash.
- Rollback: `git checkout -- docs/cliente AGENTS.md README.md roadmap-rust/index.md`.
