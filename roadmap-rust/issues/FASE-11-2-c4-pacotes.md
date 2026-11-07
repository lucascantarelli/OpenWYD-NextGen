# Issue 11-2 — C4 + especificações de pacotes de rede

**Fase:** 11 · **Depende de:** 08-1 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Publicar diagramas C4 (níveis 1–3) fiéis ao código e as especificações de
pacotes vigentes em `docs/arquitetura/`.

## Contexto / evidência

`pre-release-rust.md` FASE 11: C4 + specs de pacotes. Fonte de pacotes: tabela
08-1; fonte de componentes: `Cargo.toml` do workspace e das crates.

## Passos

1. Gerar C4 nível 1 (contexto) e 2 (containers: crates, portal, banco).
2. Detalhar nível 3 dos módulos críticos (tick, parsers, persistência).
3. Transcrever as specs de pacotes confirmadas (08-1) para `docs/arquitetura/`.
4. Revisão: zero componente inventado (cada caixa cita código).

## Arquivos

- Criados: diagramas (fonte + render) e specs em `docs/arquitetura/`.
- Alterados: nenhum código.
- Proibido tocar: `crates/*/src/`, `.deprecated-go/`.

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; cada elemento do C4 com referência de código
(tabela de rastreabilidade no doc).

## Skills Relacionadas do freebuff

- .agents/skills/documentation-writer - C4 e specs.
- .agents/skills/software-architecture - fronteiras reais.

## Documentação de referencia

- pre-release-rust.md (FASE 11).
- roadmap-rust/issues/FASE-08-1-tabela-protocolo.md - pacotes fonte.
- docs/arquitetura/adr/ - decisões refletidas no C4.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] C4 1–3 publicado com rastreabilidade; specs de pacotes vigentes.

## Riscos / rollback

- Risco: diagrama divergir do código na primeira mudança → mitigação: fonte
  como código + revisão a cada revisão consolidada (02-2).
- Rollback: `git checkout -- docs/arquitetura/`.
