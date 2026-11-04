---
name: wyd-game-admin
description: Executa e audita tarefas de administração in-game do servidor WYD2 — comandos /gm (kick, notice, goto, summon, spawn, item, setlevel, setgold, ban, unban, guildname, guildfame), comandos de jogador (/cp, /nick, /gritar, teleportes), concessão de papel de moderador/admin, bloqueio de conta e verificação de auditoria. Use quando a tarefa pedir dar item ou nível a um personagem, banir/desbanir conta, avisar jogadores, mover personagens, investigar abuso ou preparar uma conta de teste.
---

# Administração in-game (WYD local)

A autoridade administrativa **não** vem de "nível ≥ 1000" (fraqueza do legado): vem da coluna
`account.role` (`moderator` / `admin`), carregada no login. Comando negado é **silencioso**, e toda
execução é auditada em `slog` (conta, alvo, args). Implementação: `tmserver/internal/handler/gm.go`.

## 1. Como os comandos chegam

O cliente manda comandos de barra como um **sussurro** cujo alvo é o nome do comando
(`_MSG_MessageWhisper`, quirk do protocolo legado). `handler/chat.go` intercepta antes da entrega
normal. Consequência para o agente: você não "chama uma API"; você digita no chat do personagem, ou
usa o caminho de teste headless (`wyd-testing-bots`).

## 2. Comandos de GM

Digitados como `/gm <subcomando> <args>`:

| Comando | Efeito | Cuidados |
|---------|--------|----------|
| `/gm kick <jogador>` | desconecta o jogador | não derruba GM de nível igual/superior |
| `/gm notice <texto>` (ou `aviso`) | anúncio global, linha `[GM]` | sai como chat prefixado (pacote dedicado é UNVERIFIED) |
| `/gm goto <jogador>` (ou `ir`) | teleporta você até o jogador | requer estar em jogo |
| `/gm summon <jogador>` (ou `puxar`) | traz o jogador até você | idem |
| `/gm spawn <id>` | cria criatura de teste na sua posição | índice do roster de summons |
| `/gm item <id> [qtd]` | item no inventário; `qtd` 1–120 cria stack | `id` é índice do `ItemList`, não nome |
| `/gm setlevel <n>` | sobe o nível | **só sobe**, não rebaixa |
| `/gm setgold <n>` | define o ouro carregado | não mexe no ouro do cargo/banco |
| `/gm ban <jogador\|conta>` | bloqueia conta e derruba se online | grava `account.is_blocked` |
| `/gm unban <jogador\|conta>` | remove bloqueio | idem |
| `/gm guildname <id> <nome>` | registra nome de guild (memória) | não há fluxo de criação de guild |
| `/gm guildfame <id> <fama>` | registra fama de guild | admin-only, equivalente ao `+guildfame set` |

Antes de usar `item`/`spawn`, descubra o índice certo em `Release/Common/ItemList.csv`
(coluna 0 = índice) — chutar índice é a forma mais rápida de dar o item errado e sujar a economia.
Para itens de progressão (Pedra Ideal 1742, Cythera 3502/3507, Pedra da Fúria 3020, Pergaminho
Selado 4127, Semente de Cristal 4032), confira a regra do sistema em `docs/game.md`.

## 3. Comandos de jogador úteis para teste

| Comando | Uso no QA |
|---------|-----------|
| `/cp` | pontos de caos (`PKPoint-75`; 0 = nick branco) |
| `/nick <jogador>` | nick, guild, cidadania, fama do alvo |
| `/gritar <msg>` (`/spk`) | grito global; consome 1 Trombeta Mágica (item 3330) |
| `/armia` `/azran` `/erion` `/noatun` `/gelo` `/kefra` `/torre` `/arch` | teleportes de cidade/evento |
| `/rei` (`/king`) / `/reino` (`/kingdom`) | teleporte conforme a capa de reino equipada |
| `/buffs` | limpa todos os buffs (útil quando affect buga teste) |
| `/destravar40` `/destravar90` `/arcana` | gates do Celestial (efetivo só para Celestial) |
| `/sair` | sai da guild |

Comandos marcados ⏳ em `docs/game.md` (`/crias`, `/create`, `/guild`) **não existem** ainda:
não prometa nem documente como disponíveis.

## 4. Conceder ou remover papel administrativo

**Prefira o portal** (`/contas`, issue 32): login do alvo → detalhe → papel + motivo. Ele grava
`role_changed_at`/`role_changed_by` e a linha de `account_admin_audit` (antes/depois + motivo) na
mesma transação, exige `admin` e recusa rebaixar a si mesmo ou a última conta admin ativa.

Papel vive no banco e vale no **próximo login** (é carregado na autenticação) — sessão web aberta
continua com o papel antigo.

SQL direto só como **emergência** (portal indisponível), e sabendo que não haverá auditoria do BFF:

```bash
# (comandos rodam na raiz do repositório)
docker compose exec db psql -U postgres -d postgres -c \
  "update account set role='admin', role_changed_at=now() where name='test';"
# conferir
docker compose exec db psql -U postgres -d postgres -c \
  "select name, role, is_blocked, blocked_reason from account order by name;"
```

Para voltar a jogador comum, `role='player'` e relogar. Nunca edite `character`/`item` à mão para
"dar" algo: o loop é dono desse estado e o próximo save sobrescreve.

## 5. Banimento e abuso

```bash
# pelo portal (preferido, auditado): /contas → bloqueio com motivo
# pelo jogo:                        /gm ban fulano
# pelo banco (offline, sem auditoria):
docker compose exec db psql -U postgres -d postgres -c \
  "update account set is_blocked=true, blocked_reason='motivo' where name='fulano';"
```

O login já rejeita conta bloqueada (o `is_blocked` é lido na autenticação do jogo e no
`VerifyCredentials`). No portal, `moderator` só bloqueia conta de jogador; ninguém bloqueia a si
mesmo e a última conta admin ativa não pode ser bloqueada. A migração do ban administrativo para o
binServer (entitlement) está pendente (`docs/migration/web-platform-plan.md §binServer`) — não
invente esse fluxo.

## 6. Auditoria e verificação

- Toda execução de `/gm` é auditada via slog: procure `gm` nos logs do tmServer.
- Confirme efeito no **estado e no pacote**: nível/ouro também precisam chegar ao cliente
  (`MSG_UpdateScore`). Se o log mostra o valor certo e o cliente não, suspeite do pacote S→C, não da
  regra (lição registrada no `SESSION-PRIMER`).
- Para ações em lote (dezenas de itens, vários personagens), prefira escrever um teste/harness a
  digitar comando por comando: é reproduzível e não depende de cliente aberto.

## 7. Limites

- Não use `/gm` para "consertar" economia em produção sem registrar o que foi feito e por quê.
- `setgold`/`item` não são idempotentes: repetir duplica valor. Anote o antes/depois.
- `spawn` cria criatura permanente até morrer/despawnar: em área pública pode virar lixo de teste.
- `notice` fala com **todos** — inclusive durante um teste de outro humano na mesma stack.
