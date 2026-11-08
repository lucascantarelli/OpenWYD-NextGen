# ADRs — índice das decisões de arquitetura

Uma ADR por decisão estrutural (rito em `docs/rust/02-execucao.md`, FASE-13-1). O arquivo
`0000-template.md` é o molde; toda ADR nova entra **nesta tabela** na mesma entrega. O estado
declarado aqui é o do cabeçalho da própria ADR; quando divergirem, vale a ADR.

- **Aceita** — decidida e vigente; o código deve segui-la.
- **Aceita (implementada)** — decidida e já refletida no código, com prova registrada.
- **Proposta** — decidida em planejamento, ainda sem implementação/aceite formal.
- **Substituída** — vale a ADR indicada.

| # | Decisão | Estado |
|---|---|---|
| [0001](0001-hot-path-monolitico.md) | Hot Path monolítico em Rust (processo único por nó) | aceita |
| [0002](0002-estrutura-modulos.md) | Estrutura de módulos, nomenclatura e padrões por crate | aceita |
| [0003](0003-estrategia-patch-cliente.md) | Estratégia de patch do cliente e manifesto do launcher | aceita |
| [0004](0004-ecs-manual-diferido.md) | ECS manual (SoA) agora; crate ECS diferida | aceita |
| [0005](0005-serializacao-manual-diferida.md) | Serialização manual (LE explícito + `repr(C)`) agora; `zerocopy`/`nom` diferidos | aceita |
| [0006](0006-async-diferido.md) | Runtime async (`tokio`/`sqlx`) diferido; std agora | aceita — **parcialmente substituída pela 0021** (persistência) |
| [0007](0007-gui-launcher-tauri.md) | Interface gráfica do launcher em Tauri v2 | aceita |
| [0008](0008-transporte-cold-path-http-json.md) | Transporte do Cold Path: HTTP/1.1 + JSON em loopback (sem gRPC/proto) | aceita |
| [0009](0009-conteudo-estatico.md) | Conteúdo estático no PostgreSQL, validado no boot; nenhum CSV/TXT em runtime | aceita |
| [0010](0010-ia-npc-e-bots.md) | IA de NPC determinística; bots em crate separada; LLM só no Cold Path | proposta |
| [0011](0011-estrutura-matematica-sim.md) | `wyd-sim` por sistema de jogo, com unidades e ordem explícitas | proposta (árvore materializada; aceite formal pendente) |
| [0012](0012-gema-no-inventario.md) | Aplicação de gema é transação do inventário, com gema consumida e evento único | aceita |
| [0013](0013-down-seletivo-de-seed.md) | `down` de seed é seletivo e restaura o estado anterior da fonte | aceita |
| [0014](0014-autoridade-server-side.md) | Autoridade do servidor mora em `systems/`, com dano injetado por trait | aceita |
| [0015](0015-inventario-autoritativo.md) | Inventário: intenção mínima, slot derivado do catálogo | aceita |
| [0016](0016-ia-npc-deterministica.md) | IA de NPC determinística: intenção, não efeito | aceita |
| [0017](0017-release-e-imagem-minima.md) | Release otimizado e imagem mínima (LTO thin + distroless) | aceita |
| [0018](0018-telemetria-tracing.md) | Telemetria: `tracing` + exporter Prometheus próprio | aceita |
| [0019](0019-sancrate-e-painel-do-jogador.md) | SancRate do legado e fronteira de confiança do painel do jogador | aceita |
| [0020](0020-versao-de-cliente-no-login.md) | **Versão de cliente: `APP_VERSION` (7640) é histórico; o gate compara igualdade com a versão medida do cliente oficial (`client/` = 12 000)** | aceita (implementada) |
| [0021](0021-async-sqlx-adotados.md) | `tokio` + `sqlx` adotados: pool real e save fora do tick | aceita |

## Decisões que o código já aplica

As ADRs **0001, 0002, 0004, 0005, 0008, 0009, 0012, 0013, 0014, 0015, 0016, 0017, 0018, 0019,
0020 e 0021** têm implementação com prova em `docs/rust/03-progresso.md`. As **0010 e 0011**
seguem proposta: a árvore de `wyd-sim` existe, mas o aceite e a fiação de bots/LLM não.

## ADR-0020 em uma linha (o gate de versão)

`APP_VERSION = 7640` é o valor **histórico** do cliente C++ de referência (`Basedef.h:102`), não
um conjunto aceito de builds. O servidor compara **igualdade** com `PROTOCOL_VERSION` (12 000,
medido no nosso cliente em `client/`, offset 92 do `MSG_AccountLogin`), recusa divergência como
erro tipado (`NetError::ClientVersionUnsupported`) e conta a versão recebida por métrica. O
cliente real do WYD Global (launcher em `C:\Users\Canta\AppData\Local\wyd_launcher`) é
**referência de comportamento de terceiro**, nunca fonte de requisito: o valor dele não define o
gate. Aceitar lista/faixa só em captura/replay (`Tolerant`), nunca em produção.
