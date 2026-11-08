# Issue 01-2 — Varredura de dívidas: TODO/FIXME/HACK e código morto

**Fase:** 1 · **Depende de:** 01-1 · **Bloqueia:** FASE-02 · **Esforço:** M

## Objetivo

Mapear todo débito técnico do legado (somente leitura) em lista priorizada por
domínio, sem tocar no arquivo.

## Contexto / evidência

`pre-release-rust.md` FASE 1 §1.2: varreduras por `TODO`, `FIXME`, `HACK` e
código morto, agrupadas por domínio, eliminando duplicidades (§1.2).

## Passos

1. Varrer o código Rust atual: `grep -rn --include='*.rs' --include='*.ts' --include='*.mjs' -E 'TODO|FIXME|XXX|HACK|DEPRECATED' crates tools docs wyd-portal/src | head -n 100`.
2. Repetir para docs/planejamento: mesmos marcadores em `docs/` + `roadmap-rust/` (dívida documentada, não código).
3. Agrupar por domínio (rede, gameplay, persistência, portal), eliminar
   duplicidades com issues existentes e priorizar.
4. Registrar lacunas (o que o legado nunca implementou) como novas entradas.

## Arquivos

- Criados: lista priorizada (nesta issue ou `roadmap-rust/`).
- Alterados: `roadmap-rust/index.md` (se novas issues forem abertas).
- Proibido tocar: `client/`, código Rust.

## Testes / verificação

```bash
grep -rn --include='*.rs' --include='*.ts' --include='*.mjs' -cE 'TODO|FIXME|XXX|HACK' crates tools wyd-portal/src | awk -F: '{s+=$2} END {print s " ocorrencias mapeadas"}'
make check
```

Saída esperada: contagem publicada e zerada de duplicidades com issues abertas;
`make check` verde.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - agrupamento e priorização.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (FASE 1, §§1.2, 1.1).
- roadmap-rust/issues/FASE-01-1-auditoria-estado.md - base auditada.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] 100% das ocorrências mapeadas, agrupadas e sem duplicidade.
- [ ] Lacunas registradas como issues; evidência anexada.

## Riscos / rollback

- Risco: débito documentado não se aplicar ao Rust atual → mitigação: marcar cada item como
  "corrigir", "descartar" ou "investigar" (nunca mudar sem issue dona).
- Rollback: `git checkout -- roadmap-rust/`.
