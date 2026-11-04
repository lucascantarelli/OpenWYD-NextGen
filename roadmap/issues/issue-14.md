# Issue 14 — Camada de agente: AGENTS.md, skills e disciplina de execução

**Fase:** 4 · **Depende de:** — (contínua) · **Bloqueia:** 15, 16 · **Esforço:** M · **Estado:** ✅ v1 concluída (2026-11-02)

## Objetivo

Ter um agente de IA que conhece o projeto, executa as rotinas certas sem redescobrir tudo, e **prova**
o que fez — com instruções versionadas no repositório, não com memória de conversa.

## Contexto / evidência

O workspace não tinha `AGENTS.md` na raiz (só `AGENTS.md`, restrito ao subdiretório) e nenhuma
skill de domínio. O `` já tinha `.agents/skills/openspec-*` e `.claude/skills/*` (comandos
`/opsx`), que foram preservados.

Formato adotado: skills no padrão aberto carregado pelo Codebuff — diretório
`.agents/skills/<nome>/SKILL.md` com frontmatter YAML (`name` igual ao diretório, `description` que diz
**o que faz e quando usar**). O Codebuff lê `.agents/skills/` (prioridade máxima) e `.claude/skills/`,
e cada skill vira `/skill:<nome>`.

## O que foi entregue (v1)

| Arquivo | Conteúdo |
|---------|----------|
| `AGENTS.md` (raiz) | contexto do workspace, caminho feliz de bring-up, 8 invariantes, árvore de conteúdo, escada de verificação, mapa de conhecimento, índice de skills, convenções |
| `.agents/skills/wyd-server-ops/SKILL.md` | subir/inspecionar/derrubar stack, portas, flags do tmServer, banco/migrações/backup, tabela de sintoma→causa |
| `.agents/skills/wyd-client-connect/SKILL.md` | handshake do cliente, geração de `serverlist.bin`, versão de protocolo, proteções/launcher, diagnóstico |
| `.agents/skills/wyd-game-admin/SKILL.md` | `/gm` (14 subcomandos), comandos de jogador, papéis no banco, ban/unban, auditoria |
| `.agents/skills/wyd-economy/SKILL.md` | catálogo/preço/requisito, refino, combines, EXP e eventos, drops, rotina de balanceamento |
| `.agents/skills/wyd-server-dev/SKILL.md` | mapa do código, invariantes (loop único, offsets, RNG, `ESCENE_FIELD`), como adicionar handler, persistência, armadilhas |
| `.agents/skills/wyd-testing-bots/SKILL.md` | níveis de teste, login headless sem Go no host, harness, bots, carga, cenários de aceitação |

## Passos para evoluir (não refazer)

1. **Validar disparo das skills.** Rodar prompts reais que deveriam acionar cada uma (ex.: "o servidor
   não aceita login", "quero dar 500k de gold pro personagem X") e confirmar que a skill certa é lida.
2. **Corrigir descrições que não disparam.** A `description` é o único gatilho; se uma skill não foi
   usada numa tarefa em que era óbvia, reescreva-a citando os sintomas do usuário, não o nome técnico.
3. **Evitar duplicação com `AGENTS.md`.** Regra: invariantes de código vivem lá; rotina
   operacional vive nas skills; este workspace só aponta.
4. **Manter factual.** Toda skill cita caminho/comando existente. Quando o roadmap muda algo
   (ex.: porta de status, flag padrão), atualize a skill na mesma tarefa.
5. **Skill nova só com dor real.** Criar quando uma rotina se repetir 3+ vezes (ex.: "análise de
   paridade de pacote", "migração de conta legada").

## Arquivos

- Criados: os oito acima.
- Proibido: alterar `.agents/skills/openspec-*` e `.claude/commands/opsx` (ferramentas do time).

## Testes / verificação

```bash
# 1. estrutura e frontmatter
for f in .agents/skills/*/SKILL.md; do
  d=$(basename "$(dirname "$f")")
  n=$(sed -n '2s/^name: //p' "$f")
  [ "$d" = "$n" ] && echo "OK $d" || echo "MISMATCH $d vs $n"
done

# 2. toda skill tem description
grep -L "^description:" .agents/skills/*/SKILL.md || echo "todas com description"

# 3. os caminhos citados existem
grep -oh "[A-Za-z0-9_./-]*" .agents/skills/*/SKILL.md AGENTS.md | sort -u | head -20
```

Saída esperada: um `OK` por skill (6), nenhum `MISMATCH`, todos os caminhos citados existentes.

## Critérios de aceite

- [x] `AGENTS.md` na raiz cobre workspace, bring-up, invariantes e índices
- [x] 6 skills com `name` = diretório e `description` orientada a gatilho
- [x] Skills citam comandos e caminhos reais (verificáveis)
- [ ] Validação de disparo feita com pelo menos 3 prompts reais (pendente)
- [ ] Nenhuma skill contradiz `AGENTS.md`

## Riscos / rollback

- **Risco:** skill desatualizada virar fonte de erro pior que nenhuma skill. **Mitigação:** cada issue
  que muda comportamento operacional deve atualizar a skill correspondente.
- **Risco:** excesso de skills diluir o gatilho. **Mitigação:** máximo ~10; fundir quando o escopo
  sobrepuser.
- **Rollback:** skills são arquivos isolados; remover um diretório desativa a skill sem afetar código.
