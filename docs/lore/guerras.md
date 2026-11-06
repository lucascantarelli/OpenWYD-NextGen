# Guerras

Guerra, neste servidor, é **mecânica de castelo**: uma guilda domina, as outras disputam, e o resultado
muda preço e acesso. Esta página descreve o que roda hoje e o que ainda é só planejamento.

## Guerra de castelo (implementada)

| Peça | Onde | O que faz |
|---|---|---|
| Janela de disputa | `tmserver/content/Common/Settings/CastleQuest.txt` | define quando a disputa abre e o que ela paga |
| Estado do castelo | tabela de estado no Postgres + memória do mundo | quem domina, desde quando |
| Taxa de quest do castelo | `CastleQuest.txt` (rates) | multiplicador de recompensa no período |
| Painel | `/castelo` no portal | leitura do estado e ajuste auditado |

Regra que vale para qualquer guerra: **o servidor de jogo é a autoridade**. O portal lê e envia
intenção; ele não decreta vencedor escrevendo no banco.

## Guerra entre guildas (parcial)

Existe guilda com membros, cargos e depósito. O que **não** existe hoje:

- declaração formal de guerra entre duas guildas com prazo e placar;
- recompensa automática de guerra ao fim do período;
- histórico consultável de guerras encerradas.

Enquanto isso não existe, "guerra" é disputa pelo castelo — e é assim que deve ser dito a um jogador,
para não prometer sistema que o servidor não tem.

!!! warning "UNVERIFIED"
    A duração e a janela exatas da disputa vêm do `CastleQuest.txt` **do servidor original**. O
    arquivo deste repositório é uma cópia de conteúdo; se ele divergir do que o cliente espera, o
    comportamento muda sem nenhum código mudar. Ao mexer nisso, registre o antes/depois no
    `docs/migration/game-rules.md`.

## O que uma guerra afeta de verdade

1. **Preço e acesso** de itens vendidos no castelo.
2. **Recompensa de quest** no período de domínio.
3. **Orgulho e recrutamento** — efeito social, não mecânico. Não invente bônus que o servidor não
   aplica.
