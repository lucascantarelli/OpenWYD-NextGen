# Auditoria de skills — segunda passada (2026-11-12)

> **Escopo:** `.agents/skills/` e `AGENTS.md`. **Não** altera código de jogo, issues concluídas
> ou o histórico da primeira auditoria (`AUDIT-IA-2026-11-12.md`).
> **Critério:** spec Agent Skills (`https://agentskills.io/specification`) e orientação oficial
> do Freebuff para skills de projeto: frontmatter com `name`/`description`, corpo principal
> abaixo de 500 linhas, progressive disclosure e recursos carregados sob demanda.

## 1. Medição antes

| Métrica | Antes |
|---|---:|
| Skills (`SKILL.md`) | 23 |
| Linhas totais dos `SKILL.md` (frontmatter incluso) | 1.301 |
| Bytes dos `SKILL.md` | 82.932 |
| Arquivos de apoio (`references/`, `scripts/`, `assets/`, `agents/`) | 21 |
| Arquivos totais em `.agents/skills/` | 44 |
| Bytes totais em `.agents/skills/` | 297.500 |
| `AGENTS.md` | 117 linhas / 6.903 bytes |

Comandos: `find .agents/skills -name SKILL.md -print0 | xargs -0 wc -lc`,
`find .agents/skills -type f | wc -l` e `du -sb .agents/skills`.

**Diagnóstico:** o número não era o problema principal. As 23 skills tinham frontmatter
válido, `name` igual ao diretório e nenhuma passava de 500 linhas; a maior era
uma skill genérica de manutenção (360 linhas / 24.342 bytes). A sobreposição medida por 5-gramas era baixa
(maior par: `wyd-rust-core` × `wyd-rust-sim`, 4,84%), então cortar em massa seria pior para
o agente. Os problemas reais eram dois de skill — uma genérica sem uso real e uma de
desenho que duplicava a de arquitetura — mais um arquivo de apoio órfão.

## 2. Decisões

| Skill | Decisão | Justificativa |
|---|---|---|
| `github-actions-docs` | **removida** | 98 linhas / 5.032 bytes + `references/topic-map.md` (8.180 bytes); só a FASE-05 a citava. É documentação genérica de GitHub Actions, não regra do WYD; CI, lockfile e pinning ficam em `wyd-rust-toolchain` e `.github/workflows/rust.yml`. |
| `writing-plans/plan-document-reviewer-prompt.md` | **removido** | 1.713 bytes; nenhuma skill, issue ou doc o referenciava. O formato de plano já está no `writing-plans/SKILL.md`; era arquivo órfão, não progressive disclosure usada. |
| `solid-design-patterns` | **fundida** em `software-architecture` | 41 linhas / 2.203 bytes; a skill de arquitetura já tratava fronteiras, coesão, DOD/ECS e ADR. Manter duas fontes para “como desenhar” aumentava a chance de regra divergente. SOLID/patterns agora estão no corpo de `software-architecture`. |
| `software-architecture` | **atualizada** | Passa a ser a única fonte para Hot/Cold, DOD/ECS, fronteiras, ADR, SOLID, patterns idiomáticos e checklist de desenho. |
| skill genérica de manutenção | **mantida na 2ª passada; removida na 3ª (§7)** | Era 360 linhas / 24.342 bytes; a versão de conta continua disponível. Manutenção do projeto passou a ser `docs/manual-técnico/skills.md` + `tools/dev/check-skills.mjs`. |
| skills genéricas de documentação e UI, `writing-plans` | **mantidas na 2ª passada; as duas genéricas removidas na 3ª (§7)** | `writing-plans` continua; as genéricas foram substituídas por `docs/index.md` e `wyd-web-admin-ui`. |
| Demais 15 skills de domínio | **mantidas** | Codificam comandos, invariantes e fronteiras específicas do WYD; são as mais citadas nas issues. |
| Nova skill | **nenhuma** | Não há lacuna demonstrada; o problema era referência morta/duplicada, não falta de cobertura. A lacuna de verificação foi resolvida com um gate, não com mais contexto. |
| `tools/dev/check-skills.mjs` | **criado** | Valida frontmatter, `name`/diretório, `description` (1–1024), corpo ≤500 linhas, recursos citados, caminhos `.agents/skills/<slug>` em docs/issues e aliases removidos em issues. Incluído em `make check` como `skills-check`. |

## 3. Medição depois

| Métrica | Depois | Delta |
|---|---:|---:|
| Skills (`SKILL.md`) | 21 | −2 (−8,7%) |
| Linhas totais dos `SKILL.md` | 1.190 | −111 (−8,5%) |
| Bytes dos `SKILL.md` | 77.252 | −5.680 (−6,9%) |
| Arquivos de apoio | 19 | −2 |
| Arquivos totais | 40 | −4 |
| Bytes totais | 281.832 | −15.668 (−5,3%) |
| `AGENTS.md` | 117 linhas / 6.919 bytes | +16 bytes (comentário do gate) |

O corpo medido pelo gate é 1.126 linhas (exclui frontmatter); a tabela usa linhas totais
para comparar com a medição inicial. A redução de bytes totais é maior que a dos `SKILL.md`
porque `github-actions-docs` também levava um `references/topic-map.md` e havia o arquivo
órfão de `writing-plans`.

## 4. Referências corrigidas

- **Issues:** FASE-03, FASE-10 e GAME-19 passaram de `solid-design-patterns` para
  `software-architecture`; FASE-05 concentrou CI/pinning em `wyd-rust-toolchain`; FASE-09 e
  PROC-01 foram atualizadas para 21 skills.
- **Roteador:** `.agents/skills/README.md` tem uma linha para arquitetura/SOLID e não lista
  mais as duas skills removidas.
- **Documentação viva:** `docs/manual-técnico/skills.md` (inventário 21), `docs/rust/00-visao-estrutura.md`,
  `docs/rust/03-progresso.md`, `docs/rust/04-revisao-roadmap.md` e
  `docs/arquitetura/estrutura-e-padroes.md`.
- **Histórico:** `docs/manual-técnico/skills.md` mantém a tabela de sucessão com os nomes
  removidos, sempre marcados como “removida/fundida”; isso é registro, não referência ativa.

## 5. Prova

```bash
node tools/dev/check-skills.mjs
# skills-check ... 21 SKILL.md, 1126 linhas, 77252 bytes, 0 problemas

make check
# fmt + clippy + build + test + migrations-check + issue-status-check + docs-check +
# skills-check + tools-test
```

O gate falha se uma issue citar `github-actions-docs`/`solid-design-patterns`, se um
`SKILL.md` tiver frontmatter inválido, se o corpo passar de 500 linhas, se um recurso citado
não existir ou se um caminho `.agents/skills/<slug>` apontar para diretório ausente.

## 6. Revisão de conteúdo: dedup com AGENTS/docs (2026-11-12)

Depois da redução de quantidade, as 21 skills foram confrontadas com `AGENTS.md` e com os
docs normativos. **18 skills de domínio/processo** foram reescritas para o formato
“Norma → decisão específica → prova”; nesta passada as três genéricas ficaram intactas por
não duplicarem instruções do projeto — a remoção delas é a §7.

| Duplicação removida | Fonte normativa que passou a ser linkada |
|---|---|
| Hot Path, DOD/ECS, tick e `ESCENE_FIELD` | `AGENTS.md` §2.1/2.3 + `estrutura-e-padroes.md` §10 + ADR-0001/0025 |
| HEADER, opcodes, payloads e checksum | `protocolo.md` + `estrutura-e-padroes.md` §3/§11 + ADR-0005 |
| ER, migrations, seeds e `delivery_queue` | `modelo-banco.md` + `banco-catalogo.md` + `estrutura-e-padroes.md` §12 |
| Taxonomia e higiene de testes | `testes.md` §2–§4/§8 |
| Ordem de fórmulas e unidades | `matematica-sim.md` + `estrutura-e-padroes.md` §14 |
| Serviços, portas e troubleshooting da stack | `05-stack-docker.md` §2/§5–§7 + `06-observabilidade-runbooks.md` |
| Manifesto, patch e fluxo do launcher | `docs/cliente/launcher.md` + `customizacao.md` + ADR-0003/0007 |
| Inventário, hooks e endereços do cliente | `docs/cliente/` |
| DAG, patterns e SOLID | `estrutura-e-padroes.md` §1/§5/§6 |
| Ciclo, gates e fechamento do plano | `docs/rust/02-execucao.md` |

**Métricas desta passada:** 21 skills (sem alteração), 1.190 → **977 linhas totais**,
77.252 → **62.985 bytes** nos `SKILL.md`; 267.565 bytes no diretório. O corpo medido pelo gate
é 913 linhas. Arquivos totais continuam 40 (21 `SKILL.md` + 19 de apoio).

O `skills-check` foi ampliado para validar também caminhos `docs/...` e `wyd-portal/...`
citados nas skills; assim, uma skill não pode apontar para doc inexistente.

## 7. Terceira passada: remoção das skills genéricas (2026-11-12)

As três skills restantes sem conteúdo específico do WYD foram removidas do projeto. Todas
continuam disponíveis como **skills de conta/globais** no Freebuff; a cópia local só
adicionava contexto e manutenção.

| Skill | Tamanho removido | Substitutos no projeto |
|---|---:|---|
| skill genérica de UI | 71 linhas / 9.390 bytes + `LICENSE.txt` 10.174 bytes | `wyd-web-admin-ui` + `wyd-portal/README.md` §Identidade e desenho |
| skill genérica de documentação | 45 linhas / 2.748 bytes | `docs/index.md` (convenções) + `docs/rust/02-execucao.md` §5 |
| skill genérica de manutenção | 360 linhas / 24.342 bytes + 17 arquivos de apoio (191.824 bytes) | `docs/manual-técnico/skills.md` + `tools/dev/check-skills.mjs`; versão de conta disponível |

**Métricas:** 21 → **18 skills**; 977 → **501 linhas** e 62.985 → **26.505 bytes** nos
`SKILL.md`; 40 → **19 arquivos** (18 `SKILL.md` + `README.md`); diretório 267.565 →
**29.246 bytes**. As issues que citavam as skills genéricas foram corrigidas para os
docs/skills substitutos; o `skills-check` continua sem caminho morto.

## 8. Limites e próximo critério

- As três genéricas foram removidas do projeto e continuam disponíveis como skills de
  conta/globais; não há perda de capacidade. A maior skill local agora é
  `wyd-client-reverso` (28 linhas / 1.831 bytes).
- A contagem de “uso” veio das citações nas issues; não há telemetria de carregamento do
  agente. Skills mantidas por domínio continuam justificadas pelos comandos/invariantes que
  carregam, mesmo quando aparecem em poucas issues.
- Nenhum commit ou push foi feito nesta auditoria.
