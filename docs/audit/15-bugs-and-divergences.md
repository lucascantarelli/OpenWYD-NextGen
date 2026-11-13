# 15 — Bugs, divergências e inconsistências

> Cada item tem **evidência**, **classificação** e **ação proposta**. Nada foi corrigido nesta
> auditoria (§2.5 do prompt). IDs `DIV-NN` são referenciados nos demais documentos.

## 1. Divergências de contrato e de documento

| ID | Item | O que diz uma fonte | O que diz a outra | Classificação | Impacto | Ação |
|---|---|---|---|---|---|---|
| `DIV-01` | Erro de senha invisível | C++/catálogo têm `0x0117 NewAccountFail` | o roteador do 7662 **não** compara `0x0117` (zero ocorrências) | **Confirmado** | jogador não vê motivo da recusa (`BUG-01`) | emitir `0x0101` com layout provado |
| `DIV-02` | Direção do `CreateItem` | `opcode_table.rs` marca `Flow::Client2Game` | binário: `0x495dc5 cmp edx,0x26e` → handler de **recebimento** (S→C) | **Confirmado** | contrato e dispatch errados para item no chão | corrigir tabela congelada em rito próprio |
| `DIV-03` | Tamanho do `CharacterLogin` | `Basedef.h`: 20 B | 7662 envia **36 B** | **Confirmado** | leitura de 16 B a mais | já tratado no crate; documentar |
| `DIV-04` | Layout de `MSG_Trade` | natural: 156 B (`Basedef.h:2435-2445`) | Go `pack(1)`: 154 B | **Divergente** | trade não pode ser implementado às cegas | captura (`CLIENT-10`) |
| `DIV-05` | Modelo de drop | legado: taxa/bônus **por slot do `Carry`** | Rust: **um roll por item** da tabela do snapshot | **Divergente** | distribuição de loot muda | comparar curvas no `wyd-sim` e decidir com dado |
| `DIV-06` | Piedade de refino | legado: `g_pSuccessRate` | Rust: `0043_refine_pity` com regra nossa | **Fortemente indicado** | taxa efetiva pode divergir | documentar como decisão + comparar |
| `DIV-07` | Duas verdades de catálogo | Postgres (`item_catalog`, 3.220) | cliente `ItemList.bin` (6.500 posições) | **Confirmado** | nome/índice divergentes | cruzar sempre por índice; checklist de expansão (`11`) |
| `DIV-08` | Documentação atrás do código | `analise-portabilidade-legado.md`: 7 crates, `wyd-net` 458 linhas | hoje: **9 crates**, `wyd-net` **833** linhas | **Confirmado** | leitor planeja com número errado | atualizar na PROC-01 |
| `DIV-09` | Loja de NPC | checklist 7.2 = ⬜ (“não existe”) | `tests/sistema/loja_viva.rs` existe e `noticias/2026-11-11-lojas-de-npc.md` anuncia entrega | **Divergente** | priorização errada | reconciliar checklist × código |
| `DIV-10` | Colisão de jogador | legado valida no servidor | Rust não tem colisão de jogador | **Confirmado** | atravessar parede se o cliente permitir | derivar de `AttributeMap` |
| `DIV-11` | Velocidade ~200% e hits duplicados | comportamento esperado: cadência ≥ 800 ms | não provado no Rust | **Hipótese** | combate mais rápido que o original | trace de cadência (`GAME-20`) |
| `DIV-12` | Tamanho das tabelas do cliente | `ItemList` 140 B / `SkillData` 96 B (este build) | 7.69+: 164 B / 104 B | **Fortemente indicado** | build é de geração anterior | registrar no dossiê; não copiar layout |
| `DIV-13` | Chaos/PK | doc oficial EN: “less than **−65**” | doc oficial PT: “**−75**” | **Divergente (externo)** | regra de PK ambígua | escolher e registrar fonte ao implementar |
| `DIV-14` | “Familiares” | projeto tem `GAME-13` e `docs/wyd/sistemas/familiares.md` | documentação oficial não tem sistema com esse nome (há montarias e animais do BeastMaster) | **Fortemente indicado** | pode ser nomenclatura de servidor privado | tratar como sistema nosso, não “oficial” |

## 2. Bugs conhecidos (issues abertas)

| ID | Sintoma | Causa raiz (evidência) | Sev | Issue | Correção proposta |
|---|---|---|---|---|---|
| `BUG-01` | Mensagens do cliente ausentes; personagem não entra | `0x0117` inexistente no 7662 (`DIV-01`) + falhas de fio | **Alta** | `BUG-01` | usar `0x0101`; matriz mensagem × exibe/não exibe |
| `BUG-02` | “Falha ao conectar” ao entrar no mundo | seleção decidida antes do PIN; PIN opcional | Alta | `BUG-02` | ordem e gate já corrigidos; falta prova viva |
| `GAME-11` | Item não aparece no chão | `CreateItem`/`GetItem` sem handler; direção errada no catálogo (`DIV-02`) | Média | `GAME-11` | handler após captura |
| `GAME-17` | Entrada no mundo não renderiza no cliente | faixa de `ClientID`, `UpdateScore`, `CreateType` | **Alta** | `GAME-17` | prova visual (só falta isso) |
| `GAME-18` | Campo de Treinamento incompleto | buff do `Cabuncle`, portão visível, boss no mundo | Média | `GAME-18` | templates + prova visual |
| `GAME-20` | Velocidade ~200%, hits duplicados | cadência não provada (`DIV-11`) | Média | `GAME-20` | trace em `CLIENT-06` |
| `CLIENT-10` | Trade não implementada | layout 156×154 (`DIV-04`) | Média | `CLIENT-10` | captura do frame real |
| `PAR-MOV-02` | Sem colisão de jogador | ausente (`DIV-10`) | **Alta** | — | derivar de `AttributeMap` |

## 3. Achados desta auditoria (sem issue ainda)

| ID | Achado | Evidência | Sev | Ação |
|---|---|---|---|---|
| N-1 | Cliente falha em criar *render target* → efeitos de pós-processamento degradam | `client/WYD.log`: `Error in Init Render Target Texture` | Média (visual) | testar em outra GPU/Windows; registrar no runbook |
| N-2 | `client/` é repack de comunidade, não retail | `ClientPatch_v7662` GPL, `XPva03.dll`, `jlock*`, `msvcp140d` | Informativo | declarar proveniência no dossiê |
| N-3 | `extraitem.bin` de 106 B é stub (Global: 110.390 B) | medição local × `referencia-wyd-global.md` | Baixa | investigar loader antes de usar |
| N-4 | Sem respawn próprio de NPC | só recarga de definição | Média | timer de respawn |
| N-5 | Sem regen de HP/MP | `PAR-CBT-05` | Média | ligar `formula/regen.rs` no tick |
| N-6 | Checklist declara 1.11 ✅ mas resíduo visual; 7.2 ⬜ com teste de loja | `08-checklist-funcionalidades.md` | Baixa | reconciliar (`DIV-08`, `DIV-09`) |
| N-7 | `MoveOperation`/inventário sem pacote | `PAR-INV-02` | Alta | `0x0374` |
| N-8 | `client/` sem `Graphics.ini` | `ls client` | Baixa | documentar setup gráfico |
| N-9 | 124 `unwrap()` em `src` | `grep` | Média | ver `14` §7 |
| N-10 | Sem teste de concorrência de save | `UNK-23` | Média | teste de dois saves |

## 4. Severidade e tratamento

| Severidade | Critério | Itens |
|---|---|---|
| **P0** | integridade de dados / duplicação / segurança | nenhum aberto (contido por `require_ported`) |
| **P1** | bloqueia jogar de ponta a ponta | `BUG-01`, `GAME-17`, `PAR-MOV-02`, layout S→C |
| **P2** | funcionalidade importante incompleta | `GAME-11`, `GAME-18`, `CLIENT-10`, `DIV-05`, N-4/N-5/N-7 |
| **P3** | cosmético/documental | `DIV-08`, `DIV-09`, N-1/N-2/N-8 |

## 5. Regra de correção (para a etapa seguinte, com autorização)

1. **Nenhuma correção de layout sem captura**: `DIV-02` se corrige no catálogo (dado), mas
   `DIV-04` não se “adivinha”.
2. **Toda correção nasce com teste** que falha antes (`AGENTS.md` §2.8).
3. **`DIV-05`/`DIV-06` são decisões**, não bugs: precisam de ADR/nota com fonte e simulação, não
   de “consertar para o legado” sem medir.
4. **`DIV-08`/`DIV-09` se resolvem atualizando a fonte única** (checklist/portabilidade) na mesma
   entrega que mudou o comportamento.
5. **Não tornar checksum rejeitante** e **não remover `require_ported`** — são as duas defesas
   que impedem regressão de integridade.
