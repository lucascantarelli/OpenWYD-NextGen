# QA in-game: o que só fecha com o cliente real

Onze issues estão em 🟡 e duas (05/06, conexão e protocolo) continuam ⏳ porque a última milha é
**evidência no cliente Windows**: 13 issues no total. O servidor Go, o portal e os testes
automatizados já passam; o que falta é abrir o jogo e provar o comportamento. Esta é a fila de
trabalho prioritária (decisão registrada em `roadmap/PLANO-2026-11.md`).

## 1. Fila de verificação (ordem por risco)

O passo a passo de cada linha (SQL de preparação, comandos, o que fazer no cliente e o que capturar)
está em [`roteiros-qa.md`](roteiros-qa.md).

| Ordem | Issue | O que provar no cliente | Pré-requisito | Como capturar |
|:---:|---|---|---|---|
| 1 | 05/06 ✅ 04/11 | **feito:** status + CPSock + `0x020d`, `account login: OK … chars=2`; o cliente envia **ClientVersion 12000** | — | log do tmserver (no `issue-05.md`/`issue-06.md`) |
| 2 | 07 ✅ 04/11 | **feito:** entrou no mundo, andou, abriu loja e comprou (`0x0213` → `0x0114` → `0x036C` → `0x027B` → `0x0379`) | 05/06 ✅ | log em `issue-07.md` + sessão do usuário |
| 3 | 32 ✅ 04/11 | **feito:** recusa exibida no cliente (“Conta bloqueada Grade X…”); `0x0101` ID 0, 140 B, conexão fechada | login OK ✅ | log em `issue-32.md` + print do usuário |
| 4 | 23 | loja de NPC: abrir, comprar, vender, item no inventário | login OK | vídeo + diff de inventário |
| 5 | 24 | preço global refletido na loja após salvar no portal | 23 | compra com preço alterado (antes/depois) |
| 6 | 25 | stats/`Equip[]` de mob aplicados (defesa/ataque visíveis) | login OK | matar o mesmo mob antes/depois |
| 7 | 34 | drop por slot + `Carry[]` (frequência e item) | 25 | 200 kills com contagem por item |
| 8 | 36 | rates de refino/combine valendo de fato | 25 | 100 refinos, distribuição vs tabela |
| 9 | 44 | adds/efeitos de item e refino/combine | 36 | item criado no portal e usado |
| 10 | 45 | guilda/castelo: dono, taxa, janela de horário | 45 escrita ✅ | print do NPC de taxa + log |
| 11 | 46 | skills: nomes no cliente e efeito | 44 | cast + print |
| 12 | 48 | mapa de navegação condiz com o mundo | — | comparar coordenada do portal com o jogo |

## 2. Protocolo de evidência (o que conta como "fechado")

Para cada issue, anexar no arquivo da issue:

1. **Comando/SQL** usado para preparar o cenário.
2. **Log** do serviço (linha relevante, com timestamp).
3. **Captura** (print ou vídeo curto) do cliente.
4. **Antes/depois** quando a mudança é de conteúdo (preço, drop, stats).
5. Se falhar: `docs/migration/ingame-bugs.md` ganha linha nova com prioridade, causa provável e o que
   falta — **não** se marca a issue como fechada.

Regra do `AGENTS.md` que vale repetir: **não reiniciar serviços no meio do teste humano** (o restart
dispara save-on-shutdown e reconexão gRPC, gerando falha fantasma).

## 3. Automação antes do humano (barato → caro)

Antes de abrir o cliente, rode o que não precisa dele:

```bash
# 1. login headless (prova protocolo, versão e credencial)
MSYS_NO_PATHCONV=1 docker run --rm --network openwyd_default -v "D:/dolorestec/wyd2/OpenWYD:/src" \
  -e W2PP_E2E_ADDR=openwyd-nextgen1-tmsrv:8281 -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 \
  -e W2PP_E2E_VERSION=12000 golang:1.26-alpine \
  sh -c "cd /src && go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v"

# 2. fluxo de mundo (eventos) já existe
scripts/e2e-worldevents.sh

# 3. portal: tsc + vitest + build
make portal-test portal-build
```

Só depois disso o teste gráfico tem valor: se o headless passa e o gráfico falha, o problema é
apresentação (pacote S→C), não regra.

## 4. Bugs relatados pelos jogadores — sistema de feedback

Base para a issue 53. Desenho mínimo, sem dependência nova:

- **Tabela** `player_report(id, account_id, character_name, kind ('bug'|'sugestao'|'denuncia'),
  title, body, client_version, screenshot_url, status ('novo'|'triagem'|'confirmado'|'resolvido'|'recusado'),
  priority, created_at, updated_at, handled_by)`.
- **Portal** `/feedback`: fila com filtro por tipo/status/prioridade, detalhe, mudança de status com
  motivo (auditada como as outras), botão "virar issue" que gera arquivo em `roadmap/issues/` com o
  texto do jogador e o link do print.
- **Entrada**: página pública autenticada (`/suporte`) — o cliente não precisa mudar. Se um dia o
  launcher entrar, ele pode abrir a mesma página com `client_version` preenchida.
- **Privacidade**: nunca guardar senha/PIN, nem IP; print passa por moderação antes de virar público.
- **Métrica**: tempo médio de triagem e taxa de confirmados por versão — é o que diz se o QA está
  funcionando.

## 5. Diagnóstico dos sintomas que você relatou

| Sintoma | O que já se sabe (evidência no repo) | Próximo passo |
|---------|--------------------------------------|---------------|
| Dano "não funciona muito bem" | fórmulas portadas e testadas (`BASE_GetDamage`/`SkillDamage`, PvP `AC×3`, pisos) e 4 regressões já corrigidas (B12, B16, #230, #213). Não há golden case de **todas** as armas×classes | capturar 20 golpes por classe/arma e comparar com a fórmula |
| Velocidade de ataque | anti-speedhack é por tick (~800 ms); `SkillDelay` do cliente é `/4`; cooldown de refino está comentado no legado | medir intervalo real no cliente e comparar com o gate do servidor |
| Skills bugadas | 151 skills com affects; duração e case 25/24 já divergiram de propósito (B13–B15, B17) | rodar `skills/audit-affects.md` contra o CSV e testar as 10 skills mais usadas |
| Quests bugadas | caso documentado do novato (#395) mostrou roteamento por `Merchant` genérico e flag de progresso ausente do save | revisar cada NPC de quest pelo mesmo método (byte de template + flag persistida) |
| Drops errados | `ItemDropList.txt` é relatório, não runtime; raridade é o **slot** do `Carry[]`; RNG é LCG determinístico | conferir 200 kills por faixa e comparar com `g_pDropRate` |
| Eventos/guildas/guerras | estado persiste (`guild*`, `world_event*`, `kefra_state`), mas pacotes S→C de guerra/castelo são UNVERIFIED | capturar os pacotes no cliente durante um evento e comparar com o handler |

Conclusão: os sintomas são **plausíveis e tratáveis**, mas cada um exige captura do cliente. O caminho
é o mesmo: instrumentar, capturar, virar golden case, corrigir.
