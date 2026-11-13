# 18 — Catálogo de dúvidas e funcionalidades desconhecidas

> Cada item tem **pergunta**, **por que está desconhecido**, **como resolver**, **custo** e
> **impacto**. Nenhum destes pode virar regra de código antes de fechado (`AGENTS.md` §2.4).

## 1. Protocolo e cliente

| ID | Pergunta | Por que desconhecido | Como resolver | Custo | Impacto |
|---|---|---|---|---|---|
| `UNK-01` | A `main` local é idêntica ao remoto `w2pp-OpenWYD`? | sem `git fetch`/diff nesta auditoria | `git fetch` + `git diff --stat origin/main` | S | médio (fonte do legado) |
| `UNK-02` | Offsets de campo do HEADER (12 B) | nunca capturado | captura em `recv`/`SendPacket` | M | alto |
| `UNK-03` | Layout de `STRUCT_SELCHAR` (844 B) | `rust-v1` provisório | captura de `0x0110` | M | **alto** |
| `UNK-04` | Layout de `STRUCT_MOB` S→C (1820 B) | portado de `Basedef.h` + Go | captura de `0x0114` | M | **alto** |
| `UNK-05` | Corpo de `ESCENE_FIELD` (30000) | só o header é provado | captura + teste visual | L | alto |
| `UNK-06` | O cliente **responde** à cena? | nenhuma evidência | captura bidirecional | S | médio |
| `UNK-07` | Id de item no chão (faixa/geração) | `CreateItem` sem handler | captura de `0x026E` | M | alto (`GAME-11`) |
| `UNK-08` | `MSG_Trade`: 156 × 154 B | conflito natural × `pack(1)` | captura de `0x0383` | M | alto (`CLIENT-10`) |
| `UNK-09` | Semântica de `Motion`/`NoViewMob` | opcode sem corpo | desassemblage + captura | M | médio |
| `UNK-10` | Qual `Combine*` pertence a qual NPC | 11 opcodes, 1 resolvedor | `Mixhelp.dat` + diálogo | M | médio |
| `UNK-11` | `0x0101` aceita que layout de texto? | só “é a via de texto livre” | captura | S | alto (`BUG-01`) |
| `UNK-12` | Por que o cliente ignora `0x0117` | opcode ausente do roteador | já provado por desassemblage; falta captura | — | — |
| `UNK-13` | `change_numeric` do `0x0FDE` | semântica nossa | captura do cliente | S | baixo |

## 2. Conteúdo e dados do cliente

| ID | Pergunta | Por que desconhecido | Como resolver | Custo | Impacto |
|---|---|---|---|---|---|
| `UNK-14` | O 7662 tem bônus de **set**? | não há `ItemSet.bin` | RE no loader de item | M | médio |
| `UNK-15` | Layout de `ItemList.bin` (140 B) e `SkillData.bin` (96 B) | só tamanho | Ghidra em `BASE_Read*` | M | alto (editor/expansão) |
| `UNK-16` | Existe slot de **familiar**? | `GAME-13` bloqueado | captura de UI + struct | M | médio |
| `UNK-17` | Layout de `FieldXXXX.dat/.trn` | opaco | Ghidra + comparação | L | alto (mapa) |
| `UNK-18` | Dimensão/origem de `AttributeMap.dat` | não medido | leitura binária | S | alto (colisão) |
| `UNK-19` | Layout de `.ani`/`.bon` | opaco | RE no `InitBoneAnimation` | L | médio (animação) |
| `UNK-20` | `minimap.dat` é texto ou binário? | 4.058 B ambíguo | `head -c`/`file` | S | baixo |
| `UNK-21` | Raio de visão do cliente | sem fonte | teste com NPC a N células | S | médio (cena) |
| `UNK-22` | `extraitem.bin` (106 B) é stub? | 1/1000 do Global | comparar builds | S | baixo |

## 3. Regras de jogo

| ID | Pergunta | Por que desconhecido | Como resolver | Custo | Impacto |
|---|---|---|---|---|---|
| `UNK-23` | Dois saves simultâneos da mesma conta corrompem estado? | sem teste | teste de concorrência | S | **alto** (integridade) |
| `UNK-24` | Existe erro engolido (`.ok()`, `let _ =`) em caminho de rede? | não medido | auditoria dirigida | S | médio |
| `UNK-25` | Curva de EXP por nível do 7662 | nenhuma fonte oficial | captura/`level_item` | M | alto (progressão) |
| `UNK-26` | Fórmula de dano do build | C++ citado, não validado aqui | portar + golden | M | alto (`PAR-CBT-02`) |
| `UNK-27` | Drop: por item (Rust) × por slot (legado) | modelos diferentes | simulação comparada | S | médio |
| `UNK-28` | Refino: piedade nossa × `g_pSuccessRate` | decisão nossa | comparar com legado | S | médio |
| `UNK-29` | Números de PK/chaos (−65 × −75) | fontes oficiais divergem | escolher + registrar | S | médio |
| `UNK-30` | Guerra: agenda e regras do 7662 | só doc do Global | pesquisa + decisão | L | baixo agora |
| `UNK-31` | Cidadania: como se compra? | “pertence ao diálogo de NPC” | RE de diálogo | M | médio |
| `UNK-32` | Durabilidade existe no 7662? | não modelada | catálogo + cliente | M | baixo |

## 4. Ambiente e operação

| ID | Pergunta | Por que desconhecido | Como resolver | Custo | Impacto |
|---|---|---|---|---|---|
| `UNK-33` | `XTrap`/`jlock` carregam no Windows atual? | não instrumentado | ProcExp/ProcMon | S | alto (captura) |
| `UNK-34` | O erro de *render target* é driver, D3D9 ou asset? | só o sintoma no log | testar GPU/Windows/resolução | S | médio (visual) |
| `UNK-35` | Cliente roda em 1024×768 sem distorção? | `Config.bin` em modo 7 | testar modo 8 | S | médio (visual) |
| `UNK-36` | Qual build a Ongame usava em 2016? | sem fonte | arquivo/wayback | M | baixo |
| `UNK-37` | O “7663” existe? | zero evidência | não pesquisar mais | — | — |

## 5. Prioridade de fechamento

```
Imediato (S, alto impacto): UNK-02, UNK-11, UNK-18, UNK-23, UNK-33, UNK-34
Curto (M, alto impacto):     UNK-03, UNK-04, UNK-07, UNK-08, UNK-15, UNK-25
Médio:                       UNK-05, UNK-10, UNK-14, UNK-16, UNK-21, UNK-31
Longo (L):                   UNK-17, UNK-19, UNK-30
```

## 6. Regra de registro

Ao fechar um `UNK`, a resposta entra em `docs/cliente/` ou `docs/wyd/` **com build/hash/fonte**,
e a linha da matriz (`05`) muda de `P0/P1` para o novo nível na mesma entrega. Se a resposta for
“não existe no 7662”, o item vira `Não aplicável` — não vira pendência eterna.
