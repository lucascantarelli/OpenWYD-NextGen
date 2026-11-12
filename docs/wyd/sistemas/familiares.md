# Familiares — slot, itens e o que cada um faz

> Escopo: itens que ocupam o **slot Familiar** do `STRUCT_MOB` — o mesmo slot de Nyerds, Imp,
> Griupan, Trajes e das **fadas**. Cada linha traz o nível de evidência (`A` central oficial,
> `B` guia de servidor, `C` relato/comunidade); o que não tem fonte fica **UNVERIFIED**.
> Fadas têm documento próprio: `sistemas/fadas.md`.

## 1. O slot

O cliente equipa no slot Familiar: **Nyerds**, **Imp**, **Griupan** e **Trajes** (cosméticos).
Guia do WYD Brazuca (2016): *"é preciso equipá-lo no slot Familiar (aquele onde equipamos Nyerds,
Imp, Griupan e Trajes)"*. **B**.

O mesmo slot recebe o **Círculo Divino Puro**, usado para resetar uma linha de skill — ou seja,
o slot aceita tanto item de efeito passivo quanto item de uso em NPC. **B**.

**Slot provado: `Equip[13]`** (índice **13** do `Equip[16]`). Fonte primária: servidor legado
(`Source/TMSrv/.../Functions.cpp`) — `GetTeamAlive` conta "estar de Nyerds" por
`player->Equip[13].Index == 769`, e o timer de fada entrega com `SendItem(cId, EQUIP, 13, Item)`.
O guia de servidor (**B**) confirma a mesma posição ("espalhe no slot familiar"). Logo, Nyerds,
Imp, Griupan, Trajes e fadas compartilham **um** índice — e nenhum código deve tratar o slot como
"reservado a cosmético".

## 2. Nyerds

| Ponto | Fato | Nível |
|---|---|---|
| O que é | item equipável de Familiar, **refinável** com poeira de Ori/Lac como arma e set | B |
| Onde compra | NPC **Seller** | A (patch note WYD Global) |
| Efeito ao renascer | restaura **100% do HP e do MP** ao renascer, com ela equipada | A |
| Origem em evento | entregue como recompensa `+0` em troca de Baús do Tesouro | C |

O efeito de HP é antigo; o MP foi **adicionado** no patch do WYD Global (Sandbox Episódio VIII,
09/12/2024): *"Implementamos um ajuste na Nyerds que restaura 100% do MP do jogador ao renascer ao
possuir ela equipada, assim como já ocorre com o HP."* — regra de servidor moderno, **não** do
build 7662.

Consequência para nós: Nyerds é o único familiar com efeito **provado** e implementável. O efeito
é no respawn do personagem (não no tick), e o item entra no refino que já existe
(`Gameplay::refine`), bastando o item no catálogo.

## 3. Imp

- Equipa no slot Familiar (mesma lista da Nyerds). **B**
- Dropado por **boss de dungeon** — "Imp no Boss!" nas drop lists de dungeon 7.58. **C**
- Efeito passivo exato: **UNVERIFIED**. Nenhuma fonte descreve o bônus; não implementar por
  palpite (invariante 5 do `AGENTS.md`: campo sem prova não vira regra).

## 4. Griupan (Grewpain)

- Item de Familiar citado junto de Nyerds/Imp nos guias. **B**
- O guia da Steam *"Criando Griupan! Grewpain!"* trata de **como criar** ⇒ é **composto**, não
  dropado. **C**
- Receita exata (itens e NPC): **UNVERIFIED**.

## 5. Boneco de neve

**UNVERIFIED.** Não foi encontrada fonte WYD que descreva pet/familiar "boneco de neve": as
ocorrências de *snowman / snow imp* são de RuneScape e Mabinogi. O que existe no WYD é **Traje de
Boneco de Madeira** (cosmético de 30 dias do Premium Neil) — outro item, outra função. Não criar
mecânica sem fonte.

## 6. O que o servidor precisa (e o que está bloqueado)

| Requisito | Estado |
|---|---|
| slot Familiar no `Equip[16]` | **provado: índice 13** (legado: `Equip[13].Index == 769` para Nyerds e `SendItem(..., EQUIP, 13, ...)` para fada) |
| Nyerds: HP/MP 100% no respawn | regra clara; depende do respawn de personagem (FASE-04) |
| Nyerds refinável | já cabe no refino existente; falta o item no catálogo |
| Imp/Griupan: efeito passivo | **bloqueado** — sem fonte do bônus |
| Trajes: cosmético com duração | fora do escopo do servidor clássico |
| Quests de familiar | nenhuma confirmada — **UNVERIFIED** |

Issue de acompanhamento: `roadmap-rust/issues/GAME-13-familiares-slot-e-itens.md`.
