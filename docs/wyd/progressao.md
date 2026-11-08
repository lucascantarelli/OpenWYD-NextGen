# Progressão — do Mortal ao God2

A progressão de WYD é a espinha dorsal do jogo: **nível → transformação → nova camada de
poder**, repetida três vezes. Este documento descreve cada camada, o que ela destrava e o
que o servidor precisa guardar.

---

## 1. As camadas

| Camada | Como se entra | O que muda | Fonte |
|---|---|---|---|
| **Mortal** | criação do personagem | nível 1–400; pontos de status/aprendizado/skill | CONFIRMADO |
| **God (Celestial)** | nível **400** com o Mortal | nova classe escolhida, novo corpo, novo teto | CONFIRMADO |
| **Arch** | 2 quests + nível 400 + Pedra da Imortalidade, Sephirot, capa | personagem paralelo com +8 atributo/nível | CONFIRMADO |
| **Subcelestial** | evolução do Arch com Pedra Ideal | Arch nível 356+ e Mortal 400 | CONFIRMADO |
| **God2** | nível **400** com o Arch | segunda transformação | CONFIRMADO |
| **Nightmare** | patamar acima do God2, controlado por `maxNightmare` | progressão extra | CONFIRMADO no dado (`maxNightmare = 3`) |

**CONFIRMADO** para a existência das camadas e para o nível 400 como gatilho duplo
(wiki oficial: "you will become a character GOD… this transformation occurs twice, once
at 400 with MORTAL and once at 400 with your ARCH").

> Nomes que confundem: **God**, **Celestial** e **Superior** aparecem como sinônimos em
> fontes diferentes. **Subcelestial** é a camada intermediária do Arch. **God2** é a segunda
> transformação. No banco usar código (`mortal`, `god`, `arch`, `subcelestial`, `god2`,
> `nightmare`) e exibir o rótulo — nunca gravar o texto.

---

## 2. Pontos por nível

| Nível | Pontos de status | Pontos de aprendizado | Fonte |
|---|---|---|---|
| 1–200 | **6** por nível | **2** por nível | central Kersef, tabela de Arch |
| 201–355 | **6** por nível | **3** por nível | idem |
| 356–400 | **6** por nível | **1** por nível | idem |
| Arch | +**8 atributos** por nível do personagem normal | — | Kersef, "Arch" |
| Arch no 370 | — | **112** aprendizado, **288** atributos, **168** skill | idem |

O detalhe contraintuitivo: **o ganho de aprendizado cai de 3 para 1 no nível 356**. Não é
bug de fonte — é o desenho que empurra o jogador para o Arch, onde o bônus volta.

---

## 3. Prêmios de nível (equipamento) — dado real do servidor

O insumo `LevelItem.txt` (161 linhas após deduplicação) define **qual item o personagem
recebe ao atingir certo nível, por classe**. É a tabela de "kit grátis" do servidor clássico,
e ela existe no banco como `level_item`.

Colunas: `level`, `class_id`, `type_id`, `item_index`, `extras`.

- `class_id`: **0 = TransKnight, 1 = Foema, 2 = BeastMaster, 3 = Huntress**
  (documentado no cabeçalho do próprio insumo).
- `type_id` aqui **não é tipo de item**: no arquivo é o **atributo que a classe recebe**
  junto do item — `-1` independente de classe, `0` independente de pontos, `1` STR,
  `2` INT, `3` DEX.

### Conjuntos por faixa de nível

| Faixa | TransKnight | Foema | BeastMaster | Huntress |
|---|---|---|---|---|
| 29–49 | Malha (M) | Sombria (M) | Águia (M) | Metal (M) |
| 84–104 | Dourada (M) | Concha (M) | Osso (M) | Combate (M) |
| 119–139 | Anã (M) | Conjuradora (M) | Aeon (M) | Natureza (M) |
| 159–199 | Embutida (M) | Mytril (M) | Elemental (M) | Teia (M) |
| 149 | **Cavalo Leve N** (montaria) | — | — | — |
| 154 | Gram / Gungnir | Arco Élfico / Olho do Carbunkle | Lança Relâmpago / Gungnir | Arco Élfico / Presas de Behemoth |
| 254 | Foice Platinada / Asa Draconiana | Arco Guardião / Cajado Caótico | Karikas / Foice Platinada | Arco Guardião / Dianus |

Cada faixa entrega **peça por peça**, cinco níveis de intervalo:

| Peça | Nível relativo |
|---|---|
| mãos/luvas | faixa + 0 |
| botas | faixa + 5 |
| cabeça | faixa + 10 |
| pernas | faixa + 15 |
| torso | faixa + 20 |

Exemplo real (TransKnight): `29 Manoplas_de_Malha(M)`, `34 Botas_de_Malha(M)`,
`39 Elmo_de_Malha(M)`, `44 Calça_de_Malha(M)`, `49 Armadura_de_Malha(M)`.

> Por que isso importa: é a **única** fonte de progressão de equipamento "de graça" do
> servidor clássico. Sem `level_item` carregado, o jogador sobe 200 níveis pelado e a
> economia inicial não fecha.

### `type_id` na prática

O `LevelItem.txt` repete o mesmo conjunto para `type 1` e `type 2` na maioria das faixas —
ou seja, **o item não muda com o atributo**; muda o **bônus de atributo** que acompanha. Nos
casos `type 0` (Huntress) e `type 3` (DEX/claw), o mapeamento é diferente e está preservado
no dado, não inferido.

---

## 4. Fama, honra e safiras

| Recurso | Para que serve | Números confirmados |
|---|---|---|
| **Fama** | skill de topo; aposta de cidade; fama de guilda | 4.000 (10ª), 1.000 (11ª), 12.000 (12ª), 100 (aposta) |
| **Honra** | recompensa de evento/boss (Kefra dá 500 para nível 340+) | 500 |
| **Safira** | entrar no reino (nível 220); **reconfigurar pontos de status** | **30 safiras** (NPC "Mestre de Habilidade", Armia) |
| **Ouro** | refino +10 (50 milhões), aposta de guerra (1 milhão), impostos | ver `economia.md` |

O valor **30** não vem de fonte web: vem do próprio `gameconfig.txt` do servidor
(`statsapphire = 30`) e está em `game_setting`. É a prova de que este documento está
ancorado no dado e não só em wiki.

---

## 5. Quests de progressão (resumo; passo a passo em `sistemas/quests.md`)

| Nível | Quest | Efeito |
|---|---|---|
| 40 | cemitério de Armia (X:2374 Y:2097) | primeiro grande salto de caça |
| 101–150 | **Manto do Aprendiz** | equipamento |
| 116–190 | **Jardim de Deus** | up/equipamento |
| 201+ | **Desert Fury** (God 201–400) | progressão de God |
| 355 | **Contrato dos Espíritos** (Arch, 4 fases) | −100kk de EXP por fase, +habilidades |
| 355 | destrave de nível do Arch (24 Pó de Lac + 1 Selado) | nível liberado |
| 370 | destrave (composição no Odin) | nível liberado |

---

## 6. O que o servidor precisa guardar

| Estado | Onde | Por quê |
|---|---|---|
| `level`, `exp` | `character` | base de tudo |
| `layer` (`mortal`…`nightmare`) | `character` | decide teto, pacotes e visual |
| `arch_of_id` | `character` | liga Mortal ↔ Arch |
| pontos de status | `character_status` | 6/nível, respec por safira |
| pontos de aprendizado | `character_status` | 2/3/1 conforme faixa |
| maestria por linhagem | `character_lineage` | teto 200/255/320 |
| fama, honra | `character_score` | skill de topo, guilda |
| prêmios entregues | `level_item_claim` | **idempotência**: um prêmio por nível por personagem |

### Regras que evitam bug clássico

1. **Prêmio de nível é idempotente.** Relogar no nível 154 não pode duplicar a arma. Chave
   `(character_id, level_item_id)` com `UNIQUE`.
2. **Transformação consome o nível, não o personagem.** Ao virar God, o Mortal continua
   existindo (para o Arch) — não deletar.
3. **EXP acima do teto não estoura.** No nível 400 o excedente fica retido (o jogo usa isso
   para as quests de −100kk).
4. **Respec de 30 safiras** deve ser transação única: debitar safira + reembolsar pontos no
   mesmo commit.
5. **`maxNightmare = 3`** é configuração; o código não pode ter `3` hardcoded.

## 7. Curva de EXP — o que não sabemos

Nenhuma fonte pública publica a tabela de EXP por nível do build 7662. O insumo
`LevelItem.txt` dá **equipamento**, não EXP. Portanto:

- **UNVERIFIED:** a curva exata. Não inventar progressão "parecida".
- Caminho correto: extrair do binário do servidor/cliente de referência, ou medir no cliente
  oficial com personagem novo, e só então gravar `exp_table` como dado versionado.

Enquanto isso, a coluna `character.exp` existe e a fórmula fica em FASE-10 — nunca no core
como constante mágica.
