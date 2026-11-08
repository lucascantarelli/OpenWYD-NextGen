# Quests — cadeia principal, Kefra e destraves

## 1. Como as quests se organizam

WYD tem **quatro famílias** de quest, e elas não se misturam:

| Família | Escala | Exemplo |
|---|---|---|
| **Story line** | por faixa de nível | Manto do Aprendiz (101–150), Jardim de Deus (116–190) |
| **Dungeon/instância** | por sala, com limite de grupos | Pista de Runas (7 estágios) |
| **Transformação** | única, irrepetível | Arch, cristais do Arch |
| **Diária** | por conta, por dia | pool de 52 quests, teto de 3/dia |

**CONFIRMADO** para as famílias (central Kersef; WYD International "Daily Quests").

---

## 2. Story line (o caminho do Mortal)

| Nível | Quest | Objetivo | Recompensa |
|---|---|---|---|
| 1–40 | **Campo de Treino** (stages) | estágios com chave | **set A** por estágio |
| 40 | **cemitério de Armia** (X:2374 Y:2097) | primeiro desafio real | EXP/equipamento |
| 41–115 | **Spirit Guardian** | caça dirigida | **Caixa de Sabedoria** (100k EXP + 100k gold), orbes, colar |
| 101–150 | **Manto do Aprendiz** | relíquia entre as catedrais de Armia e Azran | **manto** (equipamento) |
| 116–190 | **Jardim de Deus** | oásis sagrado; herança celestial de Azran | **Lágrimas de Anjo** (200k EXP + 200k gold) |
| 116–190 | **Tumba de Kaizen** (Dark Dungeon) | **Skeleton Consery** ressuscita Kaizen | progressão + drops D |
| 201–400 | **Desert Fury** (God) | cadeia do God no Deserto | progressão de God |
| 355 | **Contrato dos Espíritos** (Arch) | 4 fases, −100kk EXP por fase | +habilidades |
| — | **Círculo Divino** | trolls e Tauron ameaçam Armia | item de reset |

**CONFIRMADO** para Manto, Jardim, Kaizen e Desert Fury (WYD International / RaidHut);
**C** para as faixas de recompensa.

> **Bug de implementação comum:** a quest de faixa entrega recompensa **uma vez por
> personagem**, não por nível. Se o jogador repetir a faixa, não pode receber duas Caixas de
> Sabedoria. Chave `(character_id, quest_id)` com `UNIQUE`.

---

## 3. Kefra — o boss e a entrada

### Pré-requisitos

1. O **Kefra precisa ter morrido naquele canal** (1× por semana, horário fixo).
2. Ir até **Lich Crunt** (2202,3866) → **entrada do Hall** (2365,3892).
3. Entregar **Pergaminho Selado** ao NPC **Sobrevivente** (2371,3887).

### Drops do kill (quem dá o último hit recebe no inventário)

| Item | Observação |
|---|---|
| **Chifre do Kefra** | entregue ao NPC **Borruk** no Hall |
| Bolsa de 10B / 2.000 GV | ouro |
| **Pedra da Fusão** | composição |
| **Ovo de Dragão Vermelho** | montaria |
| **Fragmento de Escama Bahamut** | ×5 |
| **Minério Celestial** | ×10 |
| **Essência dos Deuses** | ×2 |
| **Emblema da Valquíria** | — |
| **Cristal da Evolução** | — |
| **Lactolerium 100** | ×20 |
| **Barra de Bilhão** | raro |
| **Pedra da Benção** | **super raro** |
| **Coração do Kefra** | por canal; não é consumido, não vende a NPC |
| **Manuscrito da Concentração**, Poção Divina (7 dias), Essência da Refinação | variam por versão |

**CONFIRMADO** (Kersef "Dungeon Kefra"; WYD Brazuca; Portal UOK).

### Efeito de guilda

A guilda que mata o Kefra garante **bônus de drop e EXP no canal**. **CONFIRMADO**. É o
motivo político pelo qual guildas disputam o horário.

---

## 4. Pista de Runas — os 7 estágios

### Como começa

1. Ter **7 Poeiras de Laktolerium (PL)**.
2. Ir ao **Alquimista Odin**, Continente Desconhecido (Kefra) e compor a **Pista de Runas +0**
   — **existe chance de falha**.
3. Registrar no NPC **Uxmal** (X:3293–3301, Y:1692–1694) **somente** nos minutos:
   `XY:15–20`, `XY:35–40`, `XY:55–00`.

### Estágio por estágio

| # | Nome | Pista exigida | Grupos | Objetivo | Runas |
|---|---|---|---|---|---|
| 1 | **Lich** | +0 | 2 | matar **o lich certo** entre 8; errar ressuscita o grupo | Ansuz, Uraz, Thurisaz, Fehu, Raidho |
| 2 | **Torre** | +1 | 3 | **defender a Torre de Thor por 15 min** | Thurisaz, Fehu, Raidho, Kenaz, Naudhiz, Gebo |
| 3 | **Amon** | +2 | 3 | matar o boss, que nasce com **10 min**; só o líder entra no kill | Isa, Jara, Eihwaz, Perthro, Elhaz, Sowilo, Berkano |
| 4 | **Labirinto** | +3 | 2 | **36 salas**; matar os mobs da sala certa para invocar o boss; o líder precisa estar em cada sala (armadilha de fogo) | — |
| 5 | **Sulrang/Lobos** | +4 | — | matar **Sulrang** para o boss nascer numa das 8 salas | — |
| 6 | **Balrog / Batedor** | +5 | 2 | grupo forte; coletar sementes na sala 2 | — |
| 7 | **final** | +6 | — | conclui a cadeia | — |

**CONFIRMADO** para os estágios 1–3 (Kersef, "Clue of Runes"); **C/D** para 4–7 (WYD
International, Portal Rogue, Sky War) — a numeração das últimas salas varia por servidor.

### Recompensas

- Cada membro recebe **1 runa**; o **líder** refina a Pista (+1, +2, …).
- Drops paralelos: PL, PO, restos de Lac/Ori, **Pergaminho Selado**, **Concentração**,
  **Ressurreição**, ovo de unicórnio.
- Alguns servidores limitam **runas por IP** (3) — mecanismo anti-multibox. **C**.

### Para que servem as runas

Composição de **pedra secreta** no NPC **Odin** (2 milhões de ouro, **C**):

| Nível | Runas | Resultado |
|---|---|---|
| 40 | Fehu, Mannaz, Thurisaz, Ansuz | + 2 Selados + **Selo Mágico** |
| 90 | Sowilo, Kenaz, Uraz, Thurisaz, Isa, Tiwaz, Hagalaz | **Pedra da Fúria** + 500 de fama |

**Pedra Misteriosa:** 2 runas diferentes + Pack de Lac 10, no NPC **Ehre** (Erion). **C**.

---

## 5. Arch e Celestial

### Criar o Arch

| Requisito | Detalhe |
|---|---|
| Nível | **Mortal 400** |
| Quests | **"A procura da vida eterna"** e **"Um novo começo"** |
| Itens | **Pedra da Imortalidade**, **Sephirot**, capa **Red ou Blue** |

**CONFIRMADO**. Ao concluir, o jogo cria um **novo personagem** com o mesmo nome.

### Quest dos cristais (Arch 355+)

| Regra | Valor |
|---|---|
| Nível | Arch **355** |
| EXP mínima | **1.950.000.000** |
| Fases | **4**, ordem fixa de uso |
| Custo por fase | **−100.000.000 de EXP** |
| Repetível | **não** (1 vez, fase 1→4) |
| Comando | `/qst` para confirmar |
| Efeito | +habilidades por fase |

**CONFIRMADO** (Kersef, "(Quests Arch) Contrato dos espíritos").

### Destraves de nível

| Destrave | Requisitos |
|---|---|
| **355** | 24× Poeira de Lac + 1 Selado (NPC **Lindy**) |
| **370** | composição no **Odin** (pedras específicas) |
| **Celestial 40** | 2 Selados + 1 Selo Mágico + composição Fehu/Mannaz/Thurizas/Anzus |
| **Celestial 90** | Pedra da Fúria + 500 de fama |
| **Pedra Ideal** (Arch→Celestial) | Mortal 400 + Arch 356+ |

**C** (Portal Rogue, WYD Brazuca), consistente com a lógica oficial de Pista de Runas.

### Cythera (qualidade do Arch)

`Prateada` 356–380 · `Dourada` 381–399 · `Mística` 400. **C**.

---

## 6. Quests diárias

| Regra | Valor |
|---|---|
| NPC | Armia (X:2117 Y:2093) |
| Teto | **3 por conta por dia** de servidor |
| Pool | **52** quests configuradas |
| Tipos | coletar item (ex.: 30 Lak / 30 Ori / 100 restos) e matar N monstros |
| Recompensa | poeira, Poção Grande (PvP), Caixa de EXP |

**CONFIRMADO** (WYD International).

Modelagem: `daily_quest (account_id, quest_id, state, claimed_at)` com reset por dia **de
servidor**, não UTC do jogador. Erro aqui gera "quest não reseta" em fuso diferente.

---

## 7. Modelo de estado de quest

```
quest (id, code, name, family, min_level, max_level, repeatable, source)
quest_step (quest_id, step_index, kind, target_id, count, region_id)
quest_state (character_id, quest_id, step_index, progress, started_at, completed_at)
quest_reward (quest_id, item_index, qty, exp, gold, fame)
quest_flag (character_id, flag, value)   -- ex.: "kefra_killed_channel_2"
```

### Regras de implementação

1. **Quest é máquina de estados** (`NOT_STARTED → ACTIVE → COMPLETE → CLAIMED`). Sem isso,
   o jogador reclama recompensa duas vezes.
2. **Progresso por kill/coleta** vem do evento do tick, não de polling.
3. **Registro de Pista de Runas é por janela de tempo** — validar no servidor, não confiar no
   cliente.
4. **Limite de grupos por sala** (2 ou 3) é estado de instância, e precisa recusar entrada
   com mensagem, não teleportar e sumir.
5. **Quest irrepetível** (Arch, cristais) tem `repeatable = false` **e** checagem de estado —
   a flag sozinha não basta (relog no meio).

## 8. Lacunas

| Ponto | Status |
|---|---|
| lista completa de story line do 7662 | **UNVERIFIED** (só as faixas publicadas) |
| texto/diálogo de NPC de quest | não publicado |
| requisitos das salas 4–7 da Pista de Runas | varia por servidor |
| recompensa exata de Desert Fury | **UNVERIFIED** |
