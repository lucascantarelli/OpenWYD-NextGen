# Sistema — Personagens e progressão básica

> **Paridade global: P3–P4 na criação/seleção; P1 no layout da tela; P0 em aparência e rename.**
> O personagem existe, entra no mundo e luta; o que falta é a **tela** e a **distribuição de
> pontos**.

## 1. Escopo

Criação, seleção, exclusão, limites por conta, classes, atributos, status iniciais, pontos,
nomes, aparência, inventário/equipamento inicial, persistência, entrada no mundo, CP/chaos.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_CreateCharacter.cpp`, `_MSG_DeleteCharacter.cpp`, `_MSG_CharacterLogin.cpp`, `BASE_GetHpMp`, `BaseSIDCHM` |
| Cliente | `0x020F` (36 B), `0x0110` (840 B de corpo), `0x0112` (frame 852), `0x0213` (**36 B** no 7662) |
| Rust | `character/{crud,equipment,session,snapshot}.rs`, `cpsock/{selection,login}.rs`, `personagens::StructMob` |
| Banco | `0002_personagens`, `0032_character_scores`, `0004_guildas`, `0048_chaos_points` |
| Testes | `personagens_crud.rs`, `selchar.rs`, `entrada_mundo.rs`, `gold_inicial.rs`, `cena_mobs.rs` |
| Doc | `docs/wyd/classes.md`, `docs/wyd/progressao.md`, `docs/cliente/captura-selchar.md` |

## 3. Regras confirmadas

| Regra | Valor |
|---|---|
| Ficha inicial | `BaseSIDCHM[classe]` + `BASE_GetHpMp` |
| Pontos livres | 5 por nível (3 de skill) em `character_scores` |
| Deleção | exige nome igual; **soft delete** |
| Limite de personagens | por conta (schema) |
| Seleção | decidida no banco; emissão de `STRUCT_MOB` |
| Entrada | personagem vira entidade na posição salva e entra na cena |
| Id de cena | jogador `< 1000` (`MAX_USER`), criatura `>= 1000` |
| `CreateType` | 2 **só** para o dono |
| Ouro inicial | entregue (`gold_inicial.rs`) |

## 4. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Criação | Implementado e testado | P4 |
| Deleção | Implementado e testado | P4 |
| Seleção/entrada | Integrado, sem prova visual | P4 |
| Tela (`STRUCT_SELCHAR`) | layout `rust-v1` | **P1** |
| Atributos/pontos (`ApplyBonus`) | ausente | **P0** |
| Aparência | ausente | P0 |
| Rename/recuperação | ausente | P0 |
| CP/chaos | coluna + migration, sem regra | P1 |
| Progressão por nível/tier | schema + doc, fórmula pendente | P1 |

## 5. Lacunas

1. `STRUCT_SELCHAR`: 14 campos com dados reais, offsets provisórios (`UNK-03`).
2. `_MSG_ApplyBonus` (`0x0277`): sem handler — jogador não distribui pontos.
3. Sem aparência (cor/rosto) e sem rename.
4. Progressão: `level_item`/`game_setting` existem; curva não portada (`UNK-25`).
5. CP/chaos: número oficial é divergente (−65 × −75, `DIV-13`).

## 6. Ações

1. Capturar `0x0110`/`0x0114` e fixar offsets (`CLIENT-06`).
2. Implementar `ApplyBonus` com validação de pontos.
3. Modelar aparência apenas depois de saber quais campos o `STRUCT_SELCHAR` carrega.
4. Portar a curva de EXP para `wyd-sim` com golden.

## 7. Riscos

| Risco | Nota |
|---|---|
| Layout adivinhado | nome/classe corretos com offset errado = tela embaralhada |
| Pontos não distribuíveis | personagem “quebrado” para o jogador |
| Id de cena fora da faixa | cliente ignora a criatura (`GAME-17`) |
