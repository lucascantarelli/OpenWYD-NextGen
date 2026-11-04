# Issue 28 — Eventos globais do mundo

**Fase:** 6 · **Depende de:** 21 · **Bloqueia:** — · **Esforço:** S · **Estado:** ⏳ aberta

## Objetivo

Ligar/desligar e configurar o evento global (item do evento, taxa, faixa de índice, notice, double EXP
e evento newbie) por uma tela com prévia do efeito no jogo.

## Contexto / evidência

- `WorldEventAdminService.GetWorldEventConfig/SetWorldEventConfig` (issue #116) grava **config fria**;
  o tmserver lê via dbServer e aplica no loop (`web.proto:876-910`).
- `WorldEventConfig`: `enabled`, `item_index`, `rate`, `start_index`, `current_index`, `end_index`,
  `indexed`, `notice_enabled`, `double_exp_enabled`, `newbie_event_enabled` + `version`.
- No tmserver, `-double-exp`/`-newbie-event` são flags de boot (`main.go:115-116`) — o evento global é
  o caminho **sem restart** para ligar double EXP/newbie.
- O `gameconfig.txt` legado (`evon`, `evrate`, `double 200`, `kefrastatus`) **não** é lido pelo Go;
  a fonte real é este config + flags. Não editar aquele arquivo esperando efeito.

## Passos

1. `/eventos`: form com os 10 campos, agrupados:
   - **Evento de drop**: `enabled`, `item_index` (Combobox do catálogo), `rate` (1..10000, hint:
     quanto maior, mais raro), faixa `start_index`/`end_index` e `indexed` (usar `current_index`).
   - **Anúncio**: `notice_enabled` (mostra o texto/índice no jogo).
   - **Multiplicadores**: `double_exp_enabled`, `newbie_event_enabled`.
2. Exibir `version` recebida; ao salvar, enviar o config completo e recarregar (`GetWorldEventConfig`)
   para confirmar que o servidor aceitou — não confiar no estado local.
3. Confirmação explícita para `double_exp_enabled` (afeta toda a economia de EXP) com texto do efeito
   e link para a issue 35 (curva de EXP).
4. Linha do tempo simples: "última alteração" (do log de auditoria da issue 21) com conta e hora.
5. Estado somente leitura para `player`; `admin` e `moderator` podem escrever (o backend decide).
6. Atalho "desligar tudo" (todos os booleanos false, uma chamada) para emergência.

## Arquivos

- Criados: `portal/src/app/(admin)/eventos/page.tsx`,
  `portal/src/server/actions/events.ts`, `portal/src/lib/worldevent.ts`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: editar `Release/TMsrv/run/gameconfig.txt`; assumir que a mudança é instantânea sem
  confirmar pelo `Get` após salvar.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: ligar `double_exp_enabled`, matar o mesmo mob antes/depois no jogo e comparar a EXP
recebida (dobro); desligar e conferir o retorno. `notice_enabled=true` aparece como aviso no cliente.

## Critérios de aceite

- [ ] Ler, salvar e reler confirma o estado no servidor
- [ ] `item_index` validado contra o catálogo (não aceita índice inexistente)
- [ ] Confirmação extra para double EXP; botão "desligar tudo"
- [ ] `player` recebe 403 e auditoria registra a tentativa
- [ ] Evidência de EXP dobrada no jogo e de notice visível

## Riscos / rollback

- **Risco:** evento de drop com `rate` absurdo inundar o servidor de itens. **Mitigação:** faixa
  validada + aviso; "desligar tudo" sempre à mão.
- **Risco:** double EXP esquecido ligado. **Mitigação:** badge persistente no shell enquanto
  `double_exp_enabled` ou `newbie_event_enabled` estiverem ativos.
- **Rollback:** `SetWorldEventConfig` com o config anterior (guardar o último no log de auditoria).
