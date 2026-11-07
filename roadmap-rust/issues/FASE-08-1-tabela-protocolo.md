# Issue 08-1 — Tabela do protocolo (headers, opcodes, tamanhos, payload)

**Fase:** 8 · **Depende de:** 01-2 · **Bloqueia:** 08-2 · **Esforço:** M

## Objetivo

Publicar em `docs/arquitetura/` a tabela exata do protocolo (cabeçalhos, opcodes,
tamanhos, payload) com marcação UNVERIFIED onde não confirmado.

## Contexto / evidência

`pre-release-rust.md` FASE 8 §8.1. Fontes (leitura): `protocol-spec.md`,
`data-formats.md`, `handlers/` (58/58), `flows.md` em
`.deprecated-go/docs/migration/`; `Structures.h` legado; `STRUCT_*` conhecidas
(MOB=816, ITEM=8, SELCHAR=840, ACCOUNTFILE=7952); `PROTOCOL_VERSION=12000`,
`ESCENE_FIELD=30000`.

## Passos

1. Extrair HEADER/opcodes/tamanhos de `protocol-spec.md` + `handlers/`.
2. Publicar tabela (opcode → nome → tamanho → payload → fonte) marcando
   UNVERIFIED o não confirmado.
3. Registrar divergências encontradas entre docs e código legado.
4. Revisão cruzada: nada UNVERIFIED vira requisito de 08-2.

## Arquivos

- Criados: doc do contrato em `docs/arquitetura/`.
- Alterados: nenhum código.
- Proibido tocar: `crates/*/src/`, `.deprecated-go/`, `client/`.

## Testes / verificação

```bash
make check && ls docs/arquitetura/ | grep -i -E "protocolo|contrato|wire"
```

Saída esperada: gates verdes; tabela publicada com fontes e marcações UNVERIFIED.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net - leitura do protocolo.
- .agents/skills/documentation-writer - tabela de contrato.
- .agents/skills/wyd-client-connect - versão de protocolo.

## Documentação de referencia

- .deprecated-go/docs/migration/protocol-spec.md.
- .deprecated-go/docs/migration/handlers/ - contrato por `_MSG_*`.
- .deprecated-go/docs/migration/flows.md.
- .deprecated-go/docs/migration/data-formats.md.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Tabela completa com fonte por linha; UNVERIFIED marcado e isolado.

## Riscos / rollback

- Risco: tratar hipótese como fato → mitigação: marcação obrigatória + revisão
  da FASE-01.
- Rollback: `git checkout -- docs/arquitetura/`.
