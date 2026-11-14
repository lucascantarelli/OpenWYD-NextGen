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

## Medição no cliente (2026-11-13, print do dono)

Personagem `Cantarelli` (classe TransKnight, conta `admin`). Banco: nível 1, HP 83/83,
MP 46/46, FOR/INT/DES/CON = 8/4/7/6 (`character_scores`), CP 0 (`character_chaos`).

| Campo no painel | Cliente | Banco / esperado | Veredito |
|---|---|---|---|
| Nível | **2** | 1 (fio 0) | divergente |
| HP | 83/83 | 83/83 | ✅ |
| MP | **0/0** | 46/46 | divergente |
| FOR / INT / DES / CON | **0 / 0 / 0 / 0** | 8 / 4 / 7 / 6 | divergente |
| Ataque / Defesa | 5 / 4 | template | a conferir |
| Resist (4) | **245** em todos | 0 esperado | lixo de template no fio |
| Crítico | **102,00%** | 0 esperado | lixo de template no fio |
| Nome | **vermelho** | CP 0 → `ChaosRate` 75 (amarelo) | divergente |

O HP correto mostra que o `STRUCT_MOB`/`Score` chega alinhado até `Hp` (offset 116 do mob);
MP (120) e atributos (124+) vêm errados. `Mob::from_persisted` preserva `Critical`,
`Resist` e `SkillBar` do template (comentário no código) — é a origem provável de 245/102%.
O caminho do `select` também precisa ser conferido contra `list_by_account`, que carrega
`character_scores`/`character_chaos` à parte.

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
