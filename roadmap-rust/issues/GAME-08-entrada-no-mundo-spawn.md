# GAME-08 — Entrada no mundo: personagem sem atributos, CP errado e nome vermelho

> **ID canônico:** `GAME-08`
> **ID histórico:** `GAME-08`
> **Estado 2026-11-13:** ⬜ · **reaberta** · ver `roadmap-rust/index.md`.

**Tipo:** Bug · **Esforço:** M · **Depende de:** `CLIENT-06` (captura do `STRUCT_MOB`/`UpdateScore` no 7662) · **Bloqueia:** onda 1 (jogável)

## Objetivo

O personagem entra no mundo com **os atributos, o CP e a aparência que estão no banco**, e o
cliente desenha exatamente isso. Hoje o relato do dono é o oposto: personagem sem atributos,
CP errado e **nome vermelho** já no primeiro login.

## Contexto / evidência

- **Relato do dono (teste manual no 7662, 2026-11-13):** “personagem sem atributos no cliente,
  com cp errado, nome vermelho”; “não cria o char direito”.
- Código: `Mob::from_persisted` (`crates/wyd-server/src/character/snapshot.rs`) carrega
  `character_items` em `equip`/`carry`, mas **não recalcula** `current_score` a partir do
  equipamento; `apply_progression` só sobrepõe nível/atributos persistidos.
- Cor do nome: `chaos_rate_de(cp)` = `CP + 75`; CP 0 = 75 (amarelo). Nome vermelho implica
  `cp <= -75` ou byte trocado no fio.
- Medição de 2026-11-13 no canal vivo: personagem nível 1 com equipamento inicial completo
  (slots 0–6, arma `1139`) causa **1 de dano** por golpe em `GremlinDecoy` (600 HP) —
  consistente com bônus de item não aplicado (`GAME-19`).
- A issue foi fechada em 2026-11-12 com testes de persistência (`entrada_mundo`, `pg`), que
  **não** olham o que o cliente desenha.

## Passos

1. Capturar no 7662 o `_MSG_CNFCharacterLogin` e o `_MSG_UpdateScore` do primeiro login e
   comparar byte a byte com o `Mob` montado no servidor (`snapshot.rs`).
2. Conferir no banco: `level`, `hp_max`, `cp`/`chaos`, `character_items` e o `Score` esperado da
   classe (Str/Dex/Int/Con) — o cliente lê `CurrentScore` em offsets fixos.
3. Fechar a conta do CP: persistido, codificado (`CP + 75`) e lido pelo cliente; provar 75 para
   personagem novo.
4. Definir onde o bônus de equipamento entra no `Score` (cálculo no login/snapshot) e cobrir com
   teste de ouro do `STRUCT_MOB`.

## Critérios de aceite

- [ ] Personagem novo mostra no cliente os mesmos atributos/HP/MP do banco e da classe.
- [ ] CP de personagem novo = neutro (nome amarelo), provado por print.
- [ ] `STRUCT_MOB`/`UpdateScore` com golden de bytes e captura do 7662.
- [ ] Nenhuma correção por “ajustar até parecer certo” sem captura.
