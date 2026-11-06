---
name: wyd-testing-bots
description: Testa o servidor WYD2 sem depender do cliente gráfico — prova de login e2e, harness de protocolo CPSock, fixtures e goldens, cenários de combate/comércio/quest, testes de carga do tick de mobs e bots que jogam sozinhos. Use quando precisar validar uma mudança, reproduzir um bug de protocolo, montar um bot de QA ou gerar carga no servidor local.
---

# Testes, harness e bots

O cliente gráfico é Windows, fechado e instável de automatizar. Por isso a regra do projeto:
**prove no headless primeiro, e só depois abra o cliente**. Se o headless loga e o cliente não, o
problema é cliente/rede; se o headless falha, é servidor.

## 1. Níveis de teste (use o mais barato que responde à pergunta)

| Nível | O que cobre | Como |
|-------|-------------|------|
| Unitário | codec, fórmula, handler isolado | `go test ./tmserver/internal/...` |
| Golden de protocolo | bytes exatos de S→C | `protocol/*_test.go` |
| E2E headless | login real, TCP, dbServer | `-tags=e2e` (abaixo) |
| Carga/AI | tick com milhares de mobs | `go test -bench` (`BenchmarkTickIdle10k`) |
| Bot de jogo | andar, matar, comprar, quest | harness próprio (CPSock) |
| Cliente real | aceitação humana | skill `wyd-client-connect` |

## 2. Prova de login (o teste que mais salva tempo)

```bash
# (comandos rodam na raiz do repositório)
W2PP_E2E_ADDR=127.0.0.1:8281 \
W2PP_E2E_ACCOUNT=test \
W2PP_E2E_PASSWORD=test123 \
W2PP_E2E_VERSION=12000 \
go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
```

Sem Go no host (Windows sem toolchain é o caso comum):

```bash
MSYS_NO_PATHCONV=1 docker run --rm --network openwyd_default -v "D:/dolorestec/wyd2/OpenWYD:/src" -e W2PP_E2E_ADDR=openwyd-nextgen1-tmsrv:8281 -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 -e W2PP_E2E_VERSION=12000 golang:1.26-alpine sh -c "cd /src && go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v"
```

Receita verificada no Windows/Docker Desktop: `MSYS_NO_PATHCONV=1` impede o Git Bash
de reescrever `/src`, e a rede do compose resolver `openwyd-nextgen1-tmsrv:8281` sem `--network host`.
Esperado: `LOGIN OK — CNFAccountLogin, conn=30002, body=1996 bytes` + `PASS`.


Detalhes que fazem a diferença:

- `W2PP_E2E_VERSION` é **obrigatório** e tem de casar com `-client-version` do servidor. Errado →
  resposta `0x102` (version mismatch) e o teste "falha" por um motivo que não é bug de jogo.
- `--network host` só existe no Linux. No Docker Desktop use `--network openwyd_default` e
  `W2PP_E2E_ADDR=openwyd-nextgen1-tmsrv:8281` (verificado); `host.docker.internal` é alternativa.
- A conta precisa existir (`dbserver seed-account`) e começar **sem personagens**.

## 3. Harness de protocolo

Para reproduzir um pacote específico sem cliente:

1. Monte o frame com o codec de `tmserver/internal/protocol` (header 12 bytes, `pack(1)`,
   obfuscação por tabela estática). **Não** reimplemente o codec no teste: usar o mesmo codec é o
   que dá valor ao teste.
2. Suba um servidor de teste no padrão de `handler/handler_test.go`
   (`startServer...`), com o `Config` completo — inclusive `Spells`, `BaseMobs`, `ItemEffects`,
   `ItemPrices` quando o caminho depende deles.
3. Faça o login de personagem antes do pacote alvo: quase todo handler exige `UserPlay`.
4. Leia os frames S→C recebidos e asserte **o campo do pacote**, não só o estado: a maior classe de
   bug deste projeto é "estado certo, pacote errado".

Fixtures ficam em `tmserver/test/fixtures/`; templates reais podem ser lidos de
`tmserver/content/TMsrv/run/npc/<Nome>` (816 B), como fazem os testes de quest/kefra/subcelestial.

## 4. Bots de QA

**A. Bot interno (é o padrão do projeto desde 06/11/2026).** O laboratório mora em
`tmserver/internal/bot` (biblioteca) e `tmserver/cmd/botlab` (CLI), e fala CPSock com o
`internal/protocol` do próprio servidor — o bot quebra do mesmo jeito que o cliente quebraria.

```bash
# (comandos rodam na raiz do repositório)
make bot-test                       # núcleo inteiro, offline, com -race
go run ./tmserver/cmd/botlab validate -scenario tmserver/internal/bot/scenarios/smoke.json
go run ./tmserver/cmd/botlab run -scenario tmserver/internal/bot/scenarios/smoke.json
go run ./tmserver/cmd/botlab metrics # métricas que um cenário pode usar em Success/Failure
```

O que existe hoje: conexão + INITCODE + login + seleção de personagem, FSM com guardas e histórico,
9 perfis determinísticos, cenário JSON versionado (semente, duração, população, critérios), métricas
`openwyd_bot_*` com p95/p99 e recusa de alvo não loopback (`-allow-remote` libera).

O que **não** existe ainda: ação de ataque, movimento, loot e comércio por pacote real — elas hoje só
mutam o modelo local e contam `sim_*`. Não trate `sim_*` como comportamento do servidor; o escopo para
trocar por tráfego real está em `roadmap/issues/BOT-03.md`.

**B. Bot externo.** `kevinkouketsu/WYD2Bot` (C/C++, headless) e `kevinkouketsu/Wyd2Client` (C#)
estão catalogados em `docs/agents/CLIENT-RESEARCH-2026-06-27.md`. Servem para comparação, mas exigem
offsets próprios e divergem em silêncio do binário 7662; **não** são o caminho do projeto. Risco 🟡:
nunca rode binário pré-compilado de origem duvidosa, e nunca aponte para servidor de terceiros.

Regras para qualquer bot:

- Conta dedicada (`seed-account -name bot1 -pass …`), nunca a conta de um humano testando.
- Rate limit do servidor é 200 msg/s com burst 400 (`-max-msg-per-sec`, `-msg-burst`): bot rápido
  demais é desconectado e parece bug.
- Bot não substitui teste determinístico: use para descobrir bug, depois escreva o teste que o
  reproduz sem rede.

## 5. Carga e performance

```bash
# (comandos rodam na raiz do repositório)
go test -run XXX -bench BenchmarkTickIdle10k -benchmem ./tmserver/internal/handler/
```

- O tick de IA é 1 s (`world.DefaultMobTick`) e roda **dentro** do loop: latência de tick é latência
  de jogo. Meça antes de otimizar e cite o número.
- Gate de dormência (`wakeRadius`/`roamRadius`) mantém ~10k mobs ociosos baratos; se um benchmark
  piorar, suspeite de scan de aggro sem gate ou alocação por tick (o projeto reusa scratch).
- Carga de rede: N conexões headless concorrentes; observe `recv packet` nos logs e o custo de
  `-log-sends` (não deixe ligado em teste de carga — ele distorce).

## 6. Cenários de aceitação por funcionalidade

| Funcionalidade | Cenário mínimo | Evidência esperada |
|----------------|----------------|--------------------|
| Login | conta seed → `0x020D` → `0x0213` | log `account login` ok, char no mundo |
| Criação de char | criar nas 4 classes | BaseMob aplicado (equip visual), 1.000.000 gold inicial |
| Movimento | andar 10 tiles | eco `MSG_Action`, posição no grid coerente |
| Combate PvE | matar 20 mobs do Campo de Treino | EXP/gold sobem, `MSG_Attack` com `ID=ESCENE_FIELD` |
| Level-up | chegar ao nível 35 na curva | `MSG_UpdateScore` + `MSG_Motion(14,3)` |
| Loja NPC | comprar e vender 1 item | gold e inventário batem nos dois sentidos |
| Cargo | depositar/sacar gold | persiste após relogin |
| Trade | 2 contas trocando item | item sai de A e entra em B, sem duplicar |
| Quest | Perzen (Esfera 4130 → montaria 3987) | troca consome e entrega, expira em 30d |
| Refino | +9 com poeira | taxa de `SancRate`, score recalcula |
| Persistência | deslogar e relogar | inventário, gold, cidade e affects voltam |

`scripts/e2e-worldevents.sh` cobre o caminho de eventos globais; leia antes de duplicar.

## 7. Higiene de teste

- Sempre `-race` em teste de handler/world.
- Teste que falha é informação: **não** relaxe asserção nem adicione skip para passar. Se o
  comportamento novo exige mudar o golden, mude o golden **e explique** no commit.
- Não reinicie containers no meio do teste de outra pessoa.
- Ao terminar, derrube só o que você subiu e deixe o banco intacto (sem `down -v`).
