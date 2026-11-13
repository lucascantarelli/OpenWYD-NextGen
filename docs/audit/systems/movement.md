# Sistema — Movimentação e mundo

> **Paridade: P2 no fio, P4 na infraestrutura.** O motor tem tick, grade espacial e cena
> autoritativa; o jogador ainda não anda no protocolo e não há colisão server-side.

## 1. Escopo

Movimento, coordenadas, direções, velocidade, colisão, obstáculos, mapas, regiões, teleporte,
portais, troca de mapa, visibilidade, distâncias, sincronização, estados.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `ProcessSecMinTimer.cpp`, `_MSG_Action/Motion/NoViewMob.cpp`, `_MSG_ReqTeleport/ChangeCity.cpp` |
| Cliente | `0x036C` Action (52 B), `0x036A` Motion, `0x0369` NoViewMob; `Env/FieldXXXX.dat/.trn` |
| Rust | `core/{tick,spatial}`, `systems/movement.rs`, `world/{tick,scene,npc}.rs` |
| Banco | `0020/0023_seed_mundo`, `0011_rotas_mundo` |
| Testes | `integracao_determinism`, `autoridade`, `tick_processo`, `cena_mobs`, `cena_armia` |
| Doc | `docs/wyd/sistemas/mundo.md`, `docs/arquitetura/autoridade-servidor.md` |

## 3. Regras confirmadas

| Regra | Estado |
|---|---|
| Tick determinístico com seed | ✅ |
| Grade espacial para visibilidade | ✅ |
| Posição persistida no personagem | ✅ |
| Cena por delta (entrar/sair) | ✅ `ESCENE_FIELD` |
| Teto de mobs por refresh | `WYD_SCENE_MAX_PER_REFRESH` (64) |
| Faixa de id | jogador `< 1000`, criatura `>= 1000` |
| Velocidade | **não provada** (`GAME-20`, `DIV-11`) |
| Colisão de jogador | **ausente** (`DIV-10`) |
| Teleporte/portal | **ausente** |

## 4. Fluxo de entrada na cena

```
CharacterLogin → posição salva → entidade no mundo
  → UpdateScore (dono) → ESCENE_FIELD (delta) → CreateMob (232 B por criatura)
  → saída/morte → RemoveMob (16 B, RemoveType 0/1/2)
```

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Tick/grid/visibilidade | Implementado e testado | P4 |
| Cena autoritativa | Implementado e testado | P4 |
| Mapas/regiões (dado) | Implementado e testado | P4 |
| Movimento no fio | core sem pacote | **P2** |
| Colisão | ausente | **P1** |
| Teleporte/portais | ausente | P0 |
| Respawn | parcial (recarga) | P2 |
| Clima/ciclo | ausente | P0 |

## 6. Lacunas

1. Payload de movimento não provado (`UNK-09`).
2. Raio de visão do cliente desconhecido (`UNK-21`) — o delta pode “piscar”.
3. Sem colisão: risco de atravessar parede se o cliente permitir.
4. `Env/Field*.dat` não é lido pelo servidor; `AttributeMap.dat` (`UNK-18`) é o caminho para
   colisão.

## 7. Ações

1. Definir o payload de ação com captura e implementar no `cpsock`.
2. Derivar colisão de `AttributeMap.dat` (ou de grade de atributos semeada).
3. Implementar teleporte/portal com validação de destino.
4. Medir raio de visão com teste de dois clientes.

## 8. Riscos

| Risco | Consequência |
|---|---|
| Payload adivinhado | personagem “teleporta”/trava no cliente |
| Sem colisão | exploit de mapa |
| Delta grande demais | cliente ignora cena |
| Velocidade errada | combate e fuga fora do original |
