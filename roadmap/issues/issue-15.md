# Issue 15 — Ferramentas de administração e economia para o agente

**Fase:** 4 · **Depende de:** 03, 04 · **Bloqueia:** 16, 17 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Dar ao agente comandos **seguros e auditáveis** para as operações que hoje exigem sessão de jogo ou
SQL manual: criar conta, dar item/gold/nível, banir, consultar estado, medir economia.

## Contexto / evidência

Hoje o caminho é: `/gm` digitado no cliente (exige cliente aberto e personagem em jogo) ou `psql`
(escrita direta em dado que o loop possui — risco de sobrescrita e sem auditoria). O servidor já tem
`dbserver seed-account` e `/gm` auditado em slog, mas não há uma interface para o agente.

Restrições de arquitetura que a ferramenta precisa respeitar:

- Estado de personagem é do **loop único**; escrita externa concorrente é perigosa.
- Concessões fora de sessão devem fluir por fila/entrega, como no plano da plataforma web
  (`docs/migration/web-platform-plan.md`, `delivery_queue`).
- Papel administrativo é `account.role` (`moderator`/`admin`) e vale no login.

## Passos

1. Inventariar as operações reais (poucas e claras):
   `account create|block|unblock|role`, `character list|show|give-item|give-gold|set-level`,
   `economy price|drop|exp`, `world status|players|mobs`, `backup`.
2. Implementar como **subcomandos Go** (não script solto), no padrão de
   `dbserver/cmd/dbserver` (`seed-account`, `convert`), com `-dry-run` obrigatório nas mutações.
3. Toda mutação: valida item/limite (ex.: `qtd` 1–120), grava linha de auditoria (quem, o quê, quando,
   antes/depois) e **não** escreve estado vivo fora do fluxo permitido — usar `delivery_queue` ou
   delegar ao tmServer via RPC quando a operação exigir personagem online.
4. Expor leitura de economia (preço, taxa de refino, curva, EXP de mob) direto do conteúdo, para o
   agente responder "por que esse item custa X" sem abrir o CSV na mão.
5. Documentar cada subcomando na skill correspondente (`wyd-game-admin`, `wyd-economy`) e no `--help`.

## Arquivos

- Criar: `cmd/wydadmin/main.go` (ou `tools/adminctl/`) + lógica em
  `internal/adminctl/` (nada de pacote `util`).
- Alterar: `Makefile` (alvo `admin`), skills `wyd-game-admin`/`wyd-economy`.
- Migração: se precisar de tabela de auditoria, `internal/migrations/00NN_admin_audit.up.sql`.
- Proibido: escrever direto em `character`/`item` sem passar pelo fluxo autorizado.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine go build ./... 
# 1. dry-run não muta
... adminctl give-item -char Teste -item 4130 -qty 1 -dry-run
docker compose exec -T db psql -U postgres -tAc "select count(*) from item where item_index=4130;"
# 2. execução real muta e audita
... adminctl give-item -char Teste -item 4130 -qty 1
docker compose exec -T db psql -U postgres -c "select * from admin_audit order by id desc limit 3;"
# 3. limite respeitado
... adminctl give-item -char Teste -item 4130 -qty 999   # deve recusar
```

Saída esperada: `--dry-run` inalterado; execução real cria exatamente 1 item; auditoria com autor,
alvo, args e resultado; `qty=999` recusado com erro claro.

## Critérios de aceite

- [ ] Subcomandos implementados com `--help` e `-dry-run`
- [ ] Toda mutação auditada (autor, alvo, antes/depois, resultado)
- [ ] Nenhuma escrita em estado vivo fora do fluxo permitido
- [ ] Leitura de economia (preço/EXP/refino) sem abrir CSV manualmente
- [ ] Skills atualizadas com exemplos reais
- [ ] Teste cobrindo dry-run, sucesso, validação e auditoria

## Riscos / rollback

- **Risco:** ferramenta virar "cheat console" sem rastro. **Mitigação:** auditoria obrigatória e
  `-dry-run` default em produção.
- **Risco:** concorrência com o save do loop. **Mitigação:** delegar ao tmServer ou usar fila;
  nunca UPDATE direto em personagem online.
- **Rollback:** `adminctl` é aditivo; desabilitar o binário não afeta o servidor.
