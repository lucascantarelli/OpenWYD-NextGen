# Issue 09 — Persistência de personagem, cargo e contas/papéis

**Fase:** 2 · **Depende de:** 07, 08 · **Bloqueia:** 12, 15, 17 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Provar que o que o jogador faz sobrevive a logout, troca de personagem e reinício do servidor — e que
contas/papéis administrativos funcionam de forma auditável.

## Contexto / evidência

- Persistência hoje cobre: inventário/equipamento, gold, **última cidade**, exp/level/MaxHP/MaxMP,
  affects, skill bar/short skill, cargo (banco da conta, gold e itens), quests (Celestial, classes).
- **A posição exata NÃO é persistida por design**: ao relogar, o personagem nasce no spawn da última
  cidade visitada. Isso é comportamento esperado (B6), não bug.
- Saves de logout/desconexão rodam fora do loop (`World.Go`); o logout-para-seleção usa
  `SaveCharacterThen`, que só confirma ao cliente após o commit. `shutdown()` espera saves em voo.
- Papel administrativo vem de `account.role` (`moderator`/`admin`) e é carregado **no login**.

## Passos

1. Com um personagem em jogo: pegar itens, mudar gold, andar até outra cidade, aprender/usar skill,
   abrir buff, depositar gold no cargo, equipar/desequipar.
2. Deslogar (para a seleção e depois para o login), relogar e conferir cada item acima.
3. Reiniciar o `tmserver` e conferir de novo (prova de que não era só estado em memória).
4. Trocar de personagem na mesma conexão e verificar que **buffs não vazam** entre eles
   (`Entity.ResetAffects` no char-login/logout).
5. Conceder `role='admin'` a uma conta, relogar, executar um `/gm` e conferir a linha de auditoria no
   log.
6. Bloquear e desbloquear a conta (`/gm ban` / `/gm unban`), conferindo o login.

## Arquivos

- Consulta: `tmserver/internal/handler/character.go` (save/load, `SaveCharacterThen`),
  `handler/cargo.go`, `handler/skill.go`, `world/affect.go`, `dbserver/` (RPC de save),
  `internal/migrations/` (tabelas `character`, `item`, `affect`, `cargo`).
- Nenhuma alteração de código esperada; se faltar persistência, abra issue específica (não amplie esta).

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
# 1. estado no banco antes/depois
docker compose exec -T db psql -U postgres -c \
  "select name, level, exp, coin, last_city from character order by name;"
docker compose exec -T db psql -U postgres -c \
  "select count(*) from item; select count(*) from affect;"

# 2. reinício limpo
docker compose restart tmserver && docker compose logs --since 1m tmserver | tail -20

# 3. papel e bloqueio
docker compose exec -T db psql -U postgres -c \
  "select login, role, is_blocked from account order by login;"
docker compose logs --since 10m tmserver | grep -i "gm"
```

Saída esperada: valores idênticos antes e depois do relogin/restart; `role` refletido após novo
login; linha de auditoria do `/gm` presente; conta bloqueada recusando login.

## Critérios de aceite

- [ ] Inventário, equip, gold e cargo sobrevivem a logout e a restart do tmserver
- [ ] Exp/level/MaxHP/MaxMP preservados
- [ ] Última cidade preservada e spawn coerente (posição exata não é requisito)
- [ ] Affects e skill bar reidratados no login; nenhum buff vaza entre personagens da mesma conexão
- [ ] `role` e `is_blocked` funcionam, com auditoria de GM no log
- [ ] Evidência SQL + logs anexada

## Riscos / rollback

- **Risco:** `docker compose restart` no meio do teste de outra pessoa gera falha fantasma.
  **Mitigação:** avisar/esperar; reiniciar só o `tmserver`.
- **Risco:** imagem em cache sem o save novo depois de mudança em `.proto`.
  **Mitigação:** `build --no-cache dbserver tmserver`.
- **Risco:** concluir "perdeu item" quando o cliente só não re-renderizou o slot. **Mitigação:**
  comparar banco × pacote S→C antes de acusar o save.
- **Rollback:** nenhum dado é apagado nesta issue; se um personagem ficar inconsistente, use `/gm`
  no jogo em vez de UPDATE manual.
