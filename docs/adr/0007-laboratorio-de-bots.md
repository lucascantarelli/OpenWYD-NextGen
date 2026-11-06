# ADR 0007 — Laboratório de bots determinístico dentro do módulo Go

- **Status:** aceito em 2026-11-06
- **Contexto do projeto:** servidor local do WYD2 (`tmserver` + `dbserver` + `binserver` + `webserver`), portal Next.js e cliente Windows fechado.
- **Decisão relacionada:** issue 16 (`roadmap/issues/issue-16.md`), `BOT-01`/`BOT-02` e o documento de missão `pre-release-bots.md`.

## Contexto

Precisamos de uma população artificial de jogadores para provar o servidor sob carga
realista, reproduzir bugs e validar economia e balanceamento. As alternativas em jogo eram:

1. **Bot interno em Go**, reusando `tmserver/internal/protocol` (o codec do próprio servidor).
2. Bot externo em C/C++ (`kevinkouketsu/WYD2Bot`) ou C# (`Wyd2Client`), já catalogados em
   `docs/agents/CLIENT-RESEARCH-2026-06-27.md`.
3. Automatizar o cliente gráfico (Windows, protegido por `JProtectDll.dll`).

A restrição que decide: o formato dos pacotes é a única coisa que não pode ser inventada. Um bot
externo exige offsets próprios e divergiria do binário 7662 sem ninguém perceber; automatizar o
cliente é caro, instável e não roda em CI. O pacote `internal/protocol` já é a fonte de verdade do
wire — inclusive para o e2e de login e de troca de canal.

## Decisão

Implementar o laboratório **dentro do módulo Go**, como `tmserver/internal/bot` (biblioteca) e
`tmserver/cmd/botlab` (CLI), importando `tmserver/internal/protocol`.

- `internal/` do `tmserver` é o único lugar de onde o codec pode ser importado sem duplicação: a
  regra de `internal` do Go proíbe um pacote irmão na raiz (`botlab/`) de importá-lo. Foi o próprio
  compilador que forçou essa localização.
- Núcleo **determinístico**: máquina de estados finita, perfis declarativos, sementes por bot
  (`seed + índice × 7919`) e cenários versionados em JSON. Nenhuma dependência de LLM.
- LLM é **opcional e externo**: se um dia entrar (diálogo social, decisão estratégica), fica atrás de
  uma interface e nunca no caminho de movimento, ataque ou pacote.
- O executor recusa alvo que não seja loopback, a menos que se passe `-allow-remote`; carga pesada
  exige ambiente autorizado, porque o laço do mundo é único.

## Consequências

**Positivas**

- O bot quebra do mesmo jeito que o cliente quebraria: código do wire é compartilhado, não copiado.
- Roda em CI sem Docker, sem Windows e sem rede (servidor fake no teste).
- Cenário, semente e veredito são versionados, então um bug reproduzido vira cenário de regressão.
- Métricas no formato Prometheus com prefixo próprio (`openwyd_bot_*`), separadas das do servidor.

**Negativas / limites aceitos**

- O binário do bot vive no mesmo módulo do servidor: aumenta o tempo de `go build ./...` (medido:
  irrelevante, o pacote tem 7 arquivos).
- O núcleo **não** envia pacotes de ataque, movimento ou comércio ainda. Essas ações hoje só mutam o
  modelo local e contam métricas `sim_*`; trocá-las por tráfego real exige provar cada formato contra
  o cliente 7662, e isso é trabalho próprio (`BOT-03`).
- Não há isolamento de rede por container ainda: a orquestração é por processo e por cenário
  (`BOT-05`).

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| Bot externo em C/C++ | offsets próprios, divergência silenciosa do wire, binário de origem duvidosa |
| Automação do cliente gráfico | Windows fechado, sem paralelismo viável, não roda em CI |
| Python para o núcleo | reimplementaria o codec; Python fica para análise de relatórios, se precisar |
| LLM no laço de decisão | latência, custo e não determinismo num caminho que precisa ser reproduzível |

## Verificação

- `go build ./...`, `go vet ./tmserver/internal/bot/...` e `go test -race ./tmserver/internal/bot/...`.
- `go run ./tmserver/cmd/botlab validate -scenario tmserver/internal/bot/scenarios/smoke.json`.
- Prova de recusa de alvo remoto e de reprodução por semente nos testes do pacote.
