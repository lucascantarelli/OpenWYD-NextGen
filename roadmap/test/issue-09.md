# Issue 09 — Persistência de personagem, cargo e contas/papéis

**Fase:** 2 · **Depende de:** 07, 08 · **Bloqueia:** 12, 15, 17 · **Esforço:** M · **Estado:** 🟡 persistência básica documentada; falta a evidência SQL + restart anexada (regularizada em 06/11/2026)

## Objetivo

Provar que o que o jogador faz sobrevive a logout, troca de personagem e reinício do servidor — e que
contas/papéis administrativos funcionam de forma auditável.

## Evidência que já existe (com a fonte)

| Fato | Fonte | Tipo |
|---|---|---|
| Inventário, gold e **última cidade** persistem e voltam no relogin | `docs/migration/SESSION-PRIMER.md` §1 (linhas 20–23), comportamento confirmado no servidor | documentado + medido |
| A **posição exata não é persistida por design**: ao relogar, nasce no spawn da última cidade (B6) | `docs/migration/SESSION-PRIMER.md` §1, nota "Regra confirmada" | design, não bug |
| Saves de logout/desconexão rodam fora do loop (`World.Go`); logout-para-seleção aguarda o commit (`SaveCharacterThen`); `shutdown()` espera saves em voo | `tmserver/internal/handler/character.go`, `tmserver/internal/world/` | código |
| Papel administrativo vem de `account.role` e é carregado no login; banimento recusado com mensagem no cliente | issue 32 (fechada em 04/11, `0x0101` ID 0 com 140 B + conexão encerrada) | medido |

## O que falta — roteiro humano, sem código novo

Uma sessão no cliente com a mesma conta, anexando aqui as saídas:

- [ ] `select name, level, exp, coin, last_city from characters order by name;` **antes** e **depois** do
      relogin (valores idênticos)
- [ ] `select count(*) from items;` / `select count(*) from affects;` antes e depois
- [ ] `docker compose restart openwyd-nextgen1-tmsrv` e reconferir (prova de que não era só memória)
- [ ] cargo (gold e item depositado) sobrevivendo ao relogin
- [ ] trocar de personagem na mesma conexão sem vazamento de buff entre eles (`Entity.ResetAffects`)
- [ ] `role='admin'` refletido no login seguinte + linha de auditoria do `/gm` no log
- [ ] conta bloqueada recusando login, e desbloqueada entrando

## Contexto / invariantes

- Persistência cobre hoje: inventário/equipamento, gold, última cidade, exp/level/MaxHP/MaxMP, affects,
  skill bar/short skill, cargo (banco da conta), quests (Celestial, classes).
- **Não** "consertar" a posição não persistida: é comportamento esperado (B6).
- Se aparecer perda real de item/dado, abrir issue específica — não ampliar esta.

## Testes / verificação

```bash
docker compose exec -T db psql -U openwyd_app -c \
  "select name, level, exp, coin, last_city from characters order by name;"
docker compose restart openwyd-nextgen1-tmsrv
docker compose logs --since 1m openwyd-nextgen1-tmsrv | tail -20
docker compose exec -T db psql -U openwyd_app -c \
  "select login, role, is_blocked from accounts order by login;"
```

## Critérios de aceite

- [ ] Inventário, equip, gold e cargo sobrevivem a logout **e** a restart do tmserver
- [ ] Exp/level/MaxHP/MaxMP preservados
- [ ] Última cidade preservada e spawn coerente (posição exata não é requisito)
- [ ] Affects e skill bar reidratados; nenhum buff vaza entre personagens da mesma conexão
- [ ] `role` e `is_blocked` funcionam com auditoria de GM no log
- [ ] Evidência SQL + logs anexada nesta issue

## Riscos / rollback

- **Risco:** `docker compose restart` no meio do teste de outra pessoa gera falha fantasma.
  **Mitigação:** avisar/esperar; reiniciar só o canal 1.
- **Risco:** concluir "perdeu item" quando o cliente só não re-renderizou o slot. **Mitigação:**
  comparar banco × pacote S→C antes de acusar o save.
- **Rollback:** nenhum dado é apagado nesta issue; personagem inconsistente se corrige com `/gm`.
