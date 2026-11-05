# Issue 54 — Hub de documentação, ADRs e runbooks

**Fase:** 7 · **Depende de:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Transformar 104 arquivos de documentação em um **conjunto navegável**: uma porta de entrada única,
índice de sistemas de jogo, decisões registradas (ADRs) e runbooks de operação. Nada de reescrever a
engenharia reversa, que já é boa.

## Contexto

O diagnóstico está em [`docs/analysis/estado-atual.md`](../../docs/analysis/estado-atual.md) §6:
existe muito material de baixo nível (58/58 handlers, protocolo, formatos, regras) e pouco de
conjunto. `docs/agents/` tem relatórios de junho/2026 que ninguém sabe se valem; `docs/migration/` tem
planos e implementações misturados; não há ADR nem runbook.

## Passos

1. **`docs/README.md`** como hub: "quero entender X → leia Y", mapa de pastas, status de cada
   documento (atual / histórico / plano).
2. **`docs/game/systems/`** — um arquivo por sistema (evolução, classes, skills, combate, drop, refino,
   quests, guildas, guerras, castelo/Kefra, eventos, economia, comandos), cada um com: regra,
   estado (implementado/parcial/ausente), arquivos de código e de documento, issues relacionadas.
3. **`docs/adr/`** — ADRs curtos (contexto, decisão, consequências, alternativas) para: Go, estado
   single-owner, overlay+auditoria, `delivery_queue`, `logpipe`, conteúdo no banco, launcher,
   cliente/engine. Começar pelos 5 já decididos.
4. **`docs/runbooks/`** — subir/parar stack, restore de banco, rotação de `SESSION_SECRET`, migração
   com rollback, incidente de fila (`delivery_queue`), cliente que não conecta.
5. **Arquivar** `docs/agents/` em `docs/archive/agents-2026-06/` com nota de contexto; mover planos já
   executados para `docs/archive/plans/`.
6. **Lint de links** no CI (`lychee` ou script Go) para doc não apodrecer.

## Arquivos

- `docs/README.md`, `docs/game/systems/*.md`, `docs/adr/*.md`, `docs/runbooks/*.md`,
  `docs/archive/**`, `AGENTS.md` (§6 aponta o hub), `.github/workflows/ci.yml` (job de links).

## Verificação

```bash
# todo link relativo dos docs resolve
python - <<'PY'
import re, pathlib, sys
bad = []
for p in pathlib.Path("docs").rglob("*.md"):
    for target in re.findall(r"\]\((?!https?:|#)([^)]+)\)", p.read_text(encoding="utf-8", errors="ignore")):
        q = (p.parent / target.split("#")[0]).resolve()
        if target.endswith("/") or q.suffix:
            if not q.exists():
                bad.append(f"{p}: {target}")
print("\n".join(bad) if bad else "links ok")
sys.exit(1 if bad else 0)
PY
```

## Critérios de aceite

- [ ] `docs/README.md` responde "onde está X" para código, jogo, operação e análise
- [ ] Um arquivo por sistema de jogo com estado e evidência
- [ ] Pelo menos 5 ADRs registrados (decisões já tomadas)
- [ ] 6 runbooks executáveis por alguém que não participou do projeto
- [ ] Links verificados no CI; `docs/agents/` arquivado com contexto

## Riscos

- **Duplicação**: sistema descrito em `docs/game/systems` e em `docs/migration` → o hub deve apontar,
  não copiar; `docs/migration` continua sendo a fonte da engenharia reversa.
- **Doc morta**: cada arquivo novo nasce com dono e data; revisão trimestral no roadmap.
