# Skills — aquisição, pontos, maestria e dados

Este documento trata do **sistema**, não do catálogo de nomes (isso está em `classes.md`).
Aqui está o que o servidor precisa implementar e validar.

---

## 1. Passo a passo para aprender uma skill

1. **Subir de nível caçando.** Cada level-up dá pontos de **status**, de **aprendizado** e de
   **skill**. **CONFIRMADO**.
2. **Ir a Armia** e falar com o **Skillmaster da sua classe**:

   | Classe | NPC | Observação |
   |---|---|---|
   | TransKnight | **Capitão Cavaleiros** | — |
   | Foema | **Foema Anciã** | — |
   | BeastMaster | **Mestre Archi** | no mesmo NPC há também o Ferreiro e o Premium Neil |
   | Huntress | **ForeLearner** | — |

3. **Pagar em pontos de skill.** O custo varia por skill; skills básicas custam pouco.
   **CONFIRMADO** (central Kersef, "Skills").
4. **Respeitar a linhagem.** A skill pertence a uma das três linhagens; a 9ª só pode ser
   aprendida na mesma linhagem da 8ª. **CONFIRMADO**.
5. **Respeitar o teto de maestria** (abaixo). Skill aprendida sem pontos de maestria
   disponíveis fica fraca — o custo real é a maestria, não o ponto de skill.

### 9ª, 10ª, 11ª e 12ª — onde e quando

| Skill | NPC | Nível | Fama | Pontos | Pré-requisito |
|---|---|---|---|---|---|
| 9ª | **9thMaster** (2136,2115, Armia) | 201 (Celestial/Subcelestial) | — | **255** | 8ª da mesma linhagem |
| 10ª | Skillmaster da classe | — | **4.000** | — | todas as anteriores |
| 11ª | Skillmaster / livro | **281** | **1.000** | — | 10ª da linhagem + livro |
| 12ª | **Jazeg** (Armia) | Celestial **> 320** | **12.000** | — | 11ª da mesma linhagem |

**CONFIRMADO** (central Kersef: "9th Skill", "10th Skill", "12th skills"; WYD International,
"New 9 to 12 Skills List").

> Observação operacional: a central registra que o custo da 12ª foi **reduzido para 1.200**
> durante testes de implementação. Isso é **configuração de servidor**, não regra do jogo —
> no nosso banco vira `game_setting` (`12th_skill_fame`), com default 12.000.

---

## 2. Pontos e maestria — a matemática

| Grandeza | Valor | Fonte |
|---|---|---|
| Ponto de maestria por nível | **2** | Kersef, "Skills Usage" |
| Limite de maestria por nível | **nível × 1,5** | idem |
| Teto base por linhagem | **200** | idem |
| Teto com 8ª skill / item de maestria | **255** | idem |
| Teto com 9ª skill | **320** | Kersef, "9th Skill" |
| Skills totais | **24** | Kersef, "Skills" |

Consequência de projeto: **maestria é vetor por linhagem** (`u16[3]` por classe), não um
número único. Um personagem com 320 pontos tem, na prática, 320 para distribuir entre as
três linhagens — e é isso que define se ele é "TK de Faith" ou "TK de Trans".

### O que o servidor deve rejeitar (e como)

| Tentativa | Resposta correta |
|---|---|
| aprender skill de outra classe | erro `WrongClass` |
| aprender sem pontos de skill | erro `InsufficientSkillPoints` |
| aprender 9ª sem a 8ª da linhagem | erro `Prerequisite` |
| aprender 10ª sem 4.000 de fama | erro `InsufficientFame` |
| aprender 12ª abaixo de 321 | erro `LevelTooLow` |
| passar do teto de maestria | **clamp silencioso** no teto vigente, sem erro |
| aprender a 25ª skill | erro `SkillLimit` |

Regra do repositório: entrada de rede nunca causa `panic!`; erro tipado por fronteira.

---

## 3. Fama e reset

- **Fama** é moeda de skill de topo (4.000 / 1.000 / 12.000) e também requisito de guilda
  (100 de fama + 1.000 de influência para apostar em cidade). Ver `sistemas/guerras.md`.
- **Reset de linhagem** consome **Círculo Divino**, **Benção de Deus** e a quest
  **Balanço do Poder**. **CONFIRMADO**.
- A 9ª pode ser **desaprendida** com **Círculo Divino Composto**. **CONFIRMADO**.
- O teto de maestria **cai de volta** quando a 9ª é desaprendida — é estado derivado, não
  coluna independente.

---

## 4. Custo em mana, delay e alcance

O catálogo legado (`skill_catalog`, **154 linhas** após a curadoria Kersef) traz o comportamento de cada skill:

| Coluna | Significado | Unidade |
|---|---|---|
| `skill_index` | índice no `SkillData` do cliente | — |
| `name` | nome legado (pode ter mojibake) | — |
| `skill_point` | custo de aprendizado | pontos |
| `mana_spent` | mana por uso | MP |
| `delay_ticks` | recarga | ticks do servidor |
| `range_tiles` | alcance | tiles |
| `target_type` | alvo (self, single, área, party) | enum |
| `instance_type/value` | efeito instantâneo | enum + valor |
| `tick_type/value` | efeito ao longo do tempo | enum + valor |
| `affect_type/value/time` | buff/debuff aplicado | enum + valor + duração |
| `aggressive` | conta como agressão (gera PK/aggro) | bool |
| `max_target` | alvos máximos | — |
| `party` | só funciona em grupo | bool |
| `affect_resist` | resistível | bool |
| `passive` | passiva | bool |

O que **não** está no dado e precisa de decisão humana: `class_code`, `lineage_code`,
`skill_role`. Regra vigente: preencher só por nome confirmado; o resto fica `NULL` com
`classification_source = 'unmapped'`.

---

## 5. Como o dano de skill deve ser montado (para a FASE-10)

Não inventar fórmula. O que as fontes confirmam sobre **composição** de dano:

| Fator | Confirmação |
|---|---|
| Ataque físico e mágico são separados | CONFIRMADO (classes e atributos) |
| **Aprender arma** soma dano físico | CONFIRMADO |
| Maestria da linhagem multiplica o **poder da skill** | CONFIRMADO |
| Arma equipada conta no aprendizado/dano | CONFIRMADO |
| Montaria dá aumento de dano, ataque mágico, defesa e absorção | CONFIRMADO (sistema de montaria) |
| Refino até +9 multiplica as habilidades do item em até **1,9×** | CONFIRMADO (guia clássico de refino) |
| Acima de +10 há bônus de **absorção** e **perfuração** | CONFIRMADO (patch oficial 2023) |

**UNVERIFIED:** a fórmula exata (ordem de aplicação, se defesa é subtrativa ou
multiplicativa, se absorção vem antes do crítico). Isso é FASE-10 e exige teste no cliente,
não dedução.

---

## 6. Passo a passo para implementar (visão de servidor)

1. **Tabela `skill_catalog`** já carregada — não duplicar em código.
2. **Tabela de estado** `character_skill (character_id, skill_index, learned_at, mastery)`.
3. **Tabela de maestria** `character_lineage (character_id, lineage_code, points)` — 320 máx
   com 9ª, 255 com 8ª, 200 base.
4. **Validação** no handler de aprendizado: classe → linhagem → pré-requisito → fama →
   nível → teto. Cada falha com erro tipado (tabela do §2).
5. **Pacote de skill** ao entrar no mundo: o cliente espera a lista completa; enviar
   `skill_index` + nível de maestria, não o catálogo inteiro.
6. **Cooldown** por `delay_ticks` no tick, nunca por `sleep`/timer de SO.
7. **Efeito** aplicado via `affect_type`; resistência por `affect_resist` e atributo.
8. **Log** de aprendizado com `moderator_id` quando feito por admin (regra do portal).

## 7. Armadilhas conhecidas

- **Skill passiva não gera pacote de uso**; se o cliente receber como ativa, a barra quebra.
- **9ª desaprendida** deve reduzir o teto de 320 para 255 no mesmo evento — deixar para o
  próximo login gera personagem "acima do teto" e o cliente recalcula dano errado.
- **`aggressive`** errado transforma skill de suporte em flag de PK: o jogador vira
  assassino por curar o aliado. É o erro mais caro possível nessa tabela.
- **Nomes legados com mojibake** (20 em `npc_template`, vários em skill) não devem ser
  "corrigidos" por chute: `name_recovered` existe para isso.
