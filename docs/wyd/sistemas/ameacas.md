# Catálogo de ameaças — B1–B17 e §6 do `wyd2.md`

> **Escopo:** histórico cliente 7662 ↔ servidor Go (`Jean1dev/w2pp-OpenWYD@main`,
> `docs/migration/ingame-bugs.md`) e a lista de abusos do §6 do `wyd2.md`, confrontados
> com o Rust deste repositório.
>
> **Regra de leitura:** `PASSA` exige proteção **e** teste/verificação reproduzível no Rust.
> `PARCIAL` tem proteção em uma parte do fluxo e lacuna declarada. `ABERTO` não tem
> implementação ou não tem prova; o motivo e a issue dona ficam na linha. `OBSERVAR` e
> `ESPERADO` são decisões, não cobertura.
>
> **Fora do escopo desta prova:** cliente 7662/VM (`CLIENT-06`). Nenhum item visual é
> declarado concluído. O catálogo não executa exploit contra terceiros; a reprodução é
> defensiva, em ambiente próprio.
>
> Última revisão: **2026-11-12**.

## Legenda

| Estado | Significado |
|---|---|
| **PASSA** | O Rust recusa/neutraliza a ameaça no caminho exercitado e existe teste ou comando reproduzível. |
| **PARCIAL** | Uma parte do fluxo está protegida; a lacuna está nomeada na coluna “Aberto”. |
| **ABERTO** | Sem implementação ou sem prova; não conta como cobertura. |
| **OBSERVAR** | Sintoma histórico não reproduzido; não se mexe sem caso real. |
| **ESPERADO** | Comportamento de jogo, não vulnerabilidade. |

## 1. Matriz B1–B17 (cliente real ↔ Go → Rust)

| # | Ameaça / sintoma histórico | Estado no Rust | Evidência no repositório | O que continua aberto |
|---|---|---|---|---|
| **B1** | Jogador duplicado a cada passo (visão de outra conta) | **OBSERVAR** | `world::{scene_snapshot, plan_scene}` e `cpsock::refresh_scene` enviam `CreateMob`/`RemoveMob` por delta; `crates/wyd-server/tests/sistema/cena_mobs.rs` prova a cena de entrada e o diálogo; `world::tests` cobre entrada/saída. O Go real testou duas contas sem reproduzir. | Não há caso reproduzível; a observação visual com duas contas no 7662 continua pendente (`CLIENT-06`). |
| **B2** | Ouro inicial negativo (`0xCCCCCC00`) | **PASSA** | `character/snapshot.rs` monta `STRUCT_MOB` do banco/`BaseMob`; `tests/sistema/gold_inicial.rs::personagem_novo_entra_no_mundo_com_o_gold_do_basemob` prova `Coin@28` e `characters.gold`. | Nenhum para o valor inicial; outros offsets do dump continuam cobertos por testes de snapshot. |
| **B3** | Sem NPCs/monstros no mapa | **PASSA (existência)** | `npc_specs`/`Engine::sync_npcs` materializam `npc_definition`; `tests/sistema/cena_mobs.rs` e `cena_armia.rs` provam NPCs na cena; `crates/wyd-core/tests/integracao/npc.rs` cobre IA. | Respawn por gerador, grupos e pathfinding de jogador não são o mesmo escopo; acompanhar `GAME-06`/`GAME-18`. |
| **B4** | Preview de classe sempre TK | **PASSA** | `character/snapshot.rs` envia `Equip[slot]` do template/banco; `tests/sistema/selchar.rs` afirma rosto/arma da classe (`Equip[0]`, `Equip[1]`) e equipamento persistido. | Render visual no 7662 não foi visto; a prova é de bytes. |
| **B5** | Level +1 na seleção | **PASSA** | `tests/sistema/selchar.rs:292` compara `Score.Level` com `banco.level - 1`; o nível autoritativo permanece 1-based no banco. | Nenhum conhecido. |
| **B6** | Posição exata não persiste (spawn pela última cidade) | **DIVERGENTE / ESPERADO** | O Rust persiste a posição exata: `GAME-08` (`entrada_mundo.rs`) grava/relê 640,480; o histórico Go diz que o comportamento esperado é nascer no centro/spawn da última cidade. | **Decisão de produto:** manter posição exata (atual) ou reproduzir o spawn por cidade. Não alterar código sem decisão; candidato a issue própria. |
| **B7** | NPCs sem loja/banco | **PARCIAL** | `crates/wyd-server/tests/sistema/loja_viva.rs` e `npc_shop_item` provam compra/venda no caminho vivo; banco/baú tem `vault_items` no modelo. | NPCs de quest/combine/refino e o fluxo completo de baú no fio; ver `GAME-18` e `GAME-05`. |
| **B8** | Barra de XP não anda | **PASSA** | `world`/`cpsock` emitem `UpdateEtc`/`UpdateScore` e cena com `ESCENE_FIELD` no abate; `tests/integracao/config_gameplay.rs` cobre `_MSG_AttackOne` e EXP multiplicada; `cena_mobs` valida o header de cena. | Render da barra no 7662 (`CLIENT-06`). |
| **B9** | Personagem morto continua atacando/usando item | **PASSA (núcleo)** | `systems/combat.rs` recusa alvo morto (`ataque_em_entidade_morta_e_recusado`) e não altera HP/fila; `world` remove a criatura no golpe fatal e emite `RemoveMob`. | O cliente só entra em morte com a cena correta; prova visual pendente (`GAME-17`). |
| **B10** | Level-up não concede pontos | **PARCIAL** | `grant_exp` + `send_progress` enviam `UpdateEtc`/`UpdateScore`; a distribuição C→J (`_MSG_ApplyBonus`) ainda não está implementada (checklist 2.2). | Pontos são calculados/persistidos, mas o jogador não os distribui no fio; `GAME-05`/checklist 2.2. |
| **B11** | Compra falha em silêncio com slot ocupado | **PARCIAL** | O caminho de loja viva existe (`loja_viva.rs`); a resposta de ressincronia de slot ocupado não tem teste dedicado no Rust. | Reproduzir slot ocupado no fio e provar `SendItem`/recusa visível; `GAME-11` (inventário) ou issue de loja. |
| **B12** | Player não tira dano do mob | **PASSA** | Dano é server-authoritative: `systems/combat.rs` recebe intenção, usa `DamageModel` e recusa sem mutação (`dano_aplicado_vem_do_modelo_do_servidor`); o cliente não escolhe dano. | Cadeia L3 completa continua em `GAME-19`; a autoridade em si está provada. |
| **B13** | Proteção Elemental não protege Fogo (índices 1/2/3) | **ABERTO** | `wyd-sim` tem `formula/resist.rs` e `ResistProfile`, mas o affect 25/skill do cliente não está implementado no servidor. | Implementar affects/skills com os índices {0,1,3} e clamp 0..100; `GAME-19`/`GAME-05`, sem prova visual. |
| **B14** | Duração de habilidades alta demais | **ABERTO** | `formula/effect.rs` modela duração/maestria, mas a política de `AffectDuration` do Go (÷4, piso/teto) não foi portada. | Portar a política e testar a curva monotônica no catálogo real; `GAME-19`. |
| **B15** | Escudo Dourado curto demais | **ABERTO** | Mesmo caminho de B14: `TimedEffect` existe, mas não há aplicação de affect no fio. | Testar skill 85 e cauda curta após B14; `GAME-19`. |
| **B16** | Amunra +9 dá 100 em vez de 200 | **ABERTO** | O refino/piedade está no fio (`refino_piedade.rs`), mas o multiplicador `(sanc+10)/10` e a promoção 9→10 de acessório não estão implementados no cálculo de equipamento Rust. | Implementar `itemAbilityRefined` + isenções + `EF_RUNSPEED`; `GAME-19`/`GAME-05`. |
| **B17** | Samaritano não aumenta HP/CON | **ABERTO** | Não há affect 24 no servidor Rust; `formula/effect.rs` é genérico e não substitui a regra `v = Level*3/4 + Value`, `MaxHP += 2v`, `AffCon += v`. | Implementar affect 24 com gate `IsPlayer` e remoção no primeiro golpe; `GAME-19`. |

### Resumo da matriz

- **PASSA:** B2, B3 (existência), B4, B5, B8, B9 (núcleo), B12.
- **PARCIAL:** B7, B10, B11.
- **ABERTO:** B13, B14, B15, B16, B17.
- **OBSERVAR:** B1.
- **DIVERGENTE / decisão:** B6.
- **Prova visual pendente em todos os itens de cliente:** B1, B4, B8, B9 e a entrada do `GAME-17`; nada disso é declarado concluído.

## 2. Catálogo §6 do `wyd2.md`

Cada bloco é **uma ameaça** com os dez campos exigidos pelo §6: comportamento, versão,
causa, evidência, impacto, sistema, condições, prevenção, validação e proteção atual.
“Aberto” significa que o Rust ainda não cobre o caminho, mesmo quando existe uma peça
reutilizável.

### T01 — Duplicação de itens

1. **Comportamento:** obter a mesma unidade duas vezes por troca, drop, baú, loja ou reconexão.
2. **Versão:** WYD 7662 / Go; Rust alvo.
3. **Causa:** duas operações concorrentes leem o mesmo slot antes de gravar, ou o cliente reenvia a confirmação.
4. **Evidência:** `systems/inventory.rs` recusa item não possuído/slot errado (`inventario.rs`); `grant_loot` grava com `replace_items`; não há troca jogador↔jogador.
5. **Impacto:** economia inflada e itens fora do catálogo.
6. **Sistema:** inventário, drop, troca, baú.
7. **Condições:** duas confirmações para o mesmo slot ou replay de coleta.
8. **Prevenção:** transação por par, `UNIQUE`/idempotência por operação e revalidação do slot na confirmação.
9. **Validação:** teste de contenção no mesmo slot e replay da mesma oferta; hoje só há testes de autoridade do inventário.
10. **Proteção atual:** **PARCIAL** — inventário autoritativo testado; troca/baú no fio não existem. Dono: `CLIENT-10`, `GAME-11`.

### T02 — Duplicação de moedas

1. **Comportamento:** creditar ouro duas vezes por entrega, compra ou replay.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** repetição de `claim` sem chave idempotente ou gravação fora da transação.
4. **Evidência:** `delivery_queue.idempotency_key` UNIQUE (`0006`), `claim_gold` e `claim_item` transacionais; `tests/contrato/operacao.rs` e E2E de idempotência.
5. **Impacto:** inflação e vantagem econômica.
6. **Sistema:** grants, loja, economia.
7. **Condições:** reenvio da mesma entrega ou duas threads drenando a fila.
8. **Prevenção:** chave única, `FOR UPDATE`/status `pending` e uma transação por linha.
9. **Validação:** replay da mesma chave devolve a mesma linha; E2E `idempotencia.spec.ts`.
10. **Proteção atual:** **PASSA** para grants; **ABERTO** para compra/troca. Dono: `WEB-09`, `CLIENT-10`.

### T03 — Exploits de troca

1. **Comportamento:** trocar item já removido, confirmar duas vezes ou forjar o par.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** estado de oferta no cliente e swap não atômico.
4. **Evidência:** opcodes no catálogo, `MSG_Trade` ainda UNVERIFIED; `CLIENT-10` está ⛔ por falta de layout; não há handler.
5. **Impacto:** duplicação de itens/ouro.
6. **Sistema:** trade.
7. **Condições:** confirmação bilateral com item alterado no meio.
8. **Prevenção:** transação por par, revalidação de slots e idempotência por oferta.
9. **Validação:** teste de troca concorrente e desconexão no meio; inexistente hoje.
10. **Proteção atual:** **ABERTO** — bloqueado em captura de layout (`CLIENT-06`/`CLIENT-10`).

### T04 — Exploits de inventário

1. **Comportamento:** equipar item não possuído, mover para slot ocupado ou forjar quantidade.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** confiar no slot/quantidade enviados pelo cliente.
4. **Evidência:** `systems/inventory.rs` deriva o slot do catálogo e recusa sem mutação; `tests/integracao/inventario.rs` (9 casos) e `autoridade.rs::inventario_forjado_e_recusado_sem_alterar_item`.
5. **Impacto:** item fantasma, perda de item ou duplicação.
6. **Sistema:** inventário/equipamento.
7. **Condições:** pacote de mover/equipar forjado.
8. **Prevenção:** API recebe intenção, não resultado; limite de pilha e slot validados no servidor.
9. **Validação:** suíte de inventário e autoridade; o fio de mover/equipar ainda não existe.
10. **Proteção atual:** **PARCIAL** — núcleo PASSA; falta pacote de fio. Dono: `GAME-05`.

### T05 — Exploits de refinação

1. **Comportamento:** forçar sucesso, repetir consumo de pó ou refinar item selado.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** roll no cliente ou atualização fora da transação.
4. **Evidência:** `Gameplay::refine`/`use_item` leem SancRate viva, consomem pó e gravam item+piedade na mesma transação; `refino_piedade.rs` (5) e `config_gameplay.rs`; recusas `MissingRate`, teto +6/+9, selado e `EF_NOSANC` sem mutação.
5. **Impacto:** item +15 sem custo e duplicação de pó.
6. **Sistema:** refino.
7. **Condições:** pacote `UseItem` repetido em corrida.
8. **Prevenção:** roll no servidor, taxa por tentativa, transação e rate-limit de rede.
9. **Validação:** taxa antes/depois da recarga, piedade e `SendItem` de resposta.
10. **Proteção atual:** **PASSA** no caminho de refino com pó; **ABERTO** para concorrência explícita e B16. Dono: `QA-09`, `GAME-19`.

### T06 — Exploits de composição

1. **Comportamento:** combinar itens sem material, repetir o resultado ou forçar Anct.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** receita/roll no cliente e falha ao consumir materiais.
4. **Evidência:** `systems/compose.rs` + `tests/integracao/compose.rs` (11); sem handler de `_MSG_CombineItem`.
5. **Impacto:** equipamento de alto grau sem custo.
6. **Sistema:** compose/Anct.
7. **Condições:** pacote de combine forjado.
8. **Prevenção:** receita e roll no servidor; consumir materiais na mesma transação.
9. **Validação:** suíte de compose; falta teste de fio e concorrência.
10. **Proteção atual:** **PARCIAL** — resolvedor pronto, fluxo não ligado. Dono: `GAME-05`.

### T07 — Exploits de quests

1. **Comportamento:** repetir recompensa, pular etapa ou entregar prova de outro jogador.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** estado de quest no cliente ou recompensa sem chave única.
4. **Evidência:** `character_quest` com `UNIQUE (character_id, quest_id)`, `reward_mask`; `campo_treinamento.rs` (11) prova ordem, nível 36, prova errada e reentrega; `campo_vivo.rs` prova a etapa 1 no servidor vivo.
5. **Impacto:** recompensa dupla e progressão acelerada.
6. **Sistema:** quests.
7. **Condições:** replay do `_MSG_Quest` ou item de prova duplicado.
8. **Prevenção:** estágio/recompensa no banco, consumo da prova e recusa `AlreadyRewarded`.
9. **Validação:** fluxo 0→4 e replay; falta o fluxo completo dos guardiões/boss.
10. **Proteção atual:** **PARCIAL** — mecânica PASSA; conteúdo completo aberto (`GAME-18`, prova visual).

### T08 — Exploits de XP

1. **Comportamento:** ditar EXP no ataque, farmar flags ou pular nível.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** confiar em `current_exp` do cliente.
4. **Evidência:** `cpsock` ignora `current_exp`, `gain_exp` usa o `STRUCT_MOB` do template e `Gameplay::exp_reward`; `config_gameplay.rs` cobre `_MSG_AttackOne` real e multiplicador; `progression.rs` tem teste de level-up.
5. **Impacto:** progressão acelerada.
6. **Sistema:** combate/EXP.
7. **Condições:** pacote de ataque com EXP forjada.
8. **Prevenção:** EXP creditada pelo abate e curva do servidor.
9. **Validação:** teste de fio com EXP multiplicada; árvore completa de party/flags ainda aberta (`GAME-22`).
10. **Proteção atual:** **PASSA** no caminho base; **ABERTO** para bônus de party/eventos.

### T09 — Exploits de drop

1. **Comportamento:** escolher o drop, repetir o roll ou coletar caixa à distância.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** roll no cliente ou pickup sem distância/posse.
4. **Evidência:** `loot_at_death` roda no abate com seed e `grant_loot` entrega no inventário; `config_gameplay.rs` prova seed→rolls; `campo_vivo` prova 451.
5. **Impacto:** item raro sem esforço.
6. **Sistema:** drop/loot.
7. **Condições:** abate repetido, caixa forjada ou pickup remoto.
8. **Prevenção:** roll no servidor, posse por instância e validação `dist≤3` no pickup.
9. **Validação:** roll e entrega provados; caixa/pickup sem fio (`GAME-11`).
10. **Proteção atual:** **PARCIAL** — sorteio/entrega PASSAM; caixa no chão aberta.

### T10 — Exploits de PVP

1. **Comportamento:** atacar fora de zona, forjar alvo/duelo ou explorar penalidade.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** regra de PVP no cliente ou ausência de validação de mapa.
4. **Evidência:** `channels::ChannelRules::pvp_allowed` testado em `canal_verde.rs`; não há combate PVP no fio.
5. **Impacto:** PK sem penalidade, farm de pontos.
6. **Sistema:** PVP/PK.
7. **Condições:** ataque entre jogadores em mapa verde ou fora de zona.
8. **Prevenção:** política por mapa no servidor e validação de alvo.
9. **Validação:** teste de recusa em mapa verde existe para política; falta o fluxo de combate.
10. **Proteção atual:** **PARCIAL** — política de canal testada, combate PVP aberto (`GAME-05`).

### T11 — Exploits de skills

1. **Comportamento:** usar skill não aprendida, sem MP, fora de alcance ou repetir cast.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** confiar no `SkillIndex`/cooldown do cliente.
4. **Evidência:** `systems/skill.rs` valida skill/MP/cooldown/alvo e recusa sem cobrar (`autoridade.rs`: 3 casos); sem payload de uso no fio.
5. **Impacto:** dano infinito e vantagem em PVP.
6. **Sistema:** skills/combate.
7. **Condições:** cast forjado ou repetido.
8. **Prevenção:** livro de skills do servidor e cooldown autoritativo.
9. **Validação:** suíte de autoridade; falta o handler e os affects.
10. **Proteção atual:** **PARCIAL** — núcleo PASSA, fio aberto (`GAME-05`, `GAME-19`).

### T12 — Exploits de movimentação

1. **Comportamento:** teleportar, andar acima do orçamento ou atravessar parede.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** posição final aceita do cliente.
4. **Evidência:** `systems/movement.rs` valida orçamento/limites; `autoridade.rs::teleporte_forjado_nao_move_nem_publica` e `passo_dentro_do_orcamento_e_gravado_pelo_servidor`; sem handler de `_MSG_Move`.
5. **Impacto:** fuga de combate, acesso a mapa e speed hack.
6. **Sistema:** movimento/mapas.
7. **Condições:** pacote de movimento forjado.
8. **Prevenção:** posição autoritativa e orçamento por tick.
9. **Validação:** testes de núcleo; falta fio e colisão de jogador.
10. **Proteção atual:** **PARCIAL** — núcleo PASSA, fio aberto (`GAME-05`).

### T13 — Exploits de teletransporte

1. **Comportamento:** usar rota sem requisito, para mapa bloqueado ou em combate.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** destino/rota escolhidos pelo cliente.
4. **Evidência:** rotas catalogadas; checklist 3.6 ⬜; sem handler.
5. **Impacto:** acesso indevido e fuga.
6. **Sistema:** teleporte/mapas.
7. **Condições:** pacote de teleporte forjado.
8. **Prevenção:** validar rota, nível, item, mapa e estado no servidor.
9. **Validação:** inexistente.
10. **Proteção atual:** **ABERTO** (`GAME-05`/checklist 3.6).

### T14 — Exploits de canais

1. **Comportamento:** duplicar sessão entre canais, trocar de canal em combate ou forjar reino.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** sessão/estado não transferido atomicamente.
4. **Evidência:** `character_session` única (`0030`), `claim_session`/`release` transacionais e `sessao_unica.rs`; a troca de canal propriamente dita não existe.
5. **Impacto:** personagem em dois canais e duplicação.
6. **Sistema:** sessão/canais.
7. **Condições:** duas conexões da mesma conta ou transferência concorrente.
8. **Prevenção:** sessão única no banco e transferência com lease.
9. **Validação:** teste de mesma conta em dois canais (`sessao_unica.rs`); falta o fluxo de troca.
10. **Proteção atual:** **PARCIAL** — sessão única PASSA; transferência aberta (checklist 4.26).

### T15 — Abuso de sessões

1. **Comportamento:** manter sessão após logout, reutilizar sessão antiga ou impedir relogin.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** liberar sessão só por timeout.
4. **Evidência:** logout explícito responde `CNFCharacterLogout`, `leave_world` e `Drop`; `selchar.rs::logout_explicito_fecha_a_conexao_e_libera_a_sessao`; queda também libera.
5. **Impacto:** conta presa e dupla presença.
6. **Sistema:** sessão/login.
7. **Condições:** fechar cliente, relogar imediatamente.
8. **Prevenção:** lease com `Drop` e liberação no banco.
9. **Validação:** teste de logout explícito e sessão única.
10. **Proteção atual:** **PASSA** (golden do frame de saída ainda depende de `CLIENT-06`).

### T16 — Manipulação de pacotes

1. **Comportamento:** frame truncado, tamanho gigante, opcode desconhecido ou checksum inválido.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** parser sem limite ou confiança no tamanho declarado.
4. **Evidência:** `wyd-protocol` valida `12..=8192` antes do payload; `wyd-net` usa `Strict`; `flood.rs` cobre flood, truncamento, opcode desconhecido e fuzz mínimo; `roundtrip.rs`/golden.
5. **Impacto:** DoS, panic e leitura fora do buffer.
6. **Sistema:** protocolo/rede.
7. **Condições:** bytes arbitrários na porta.
8. **Prevenção:** validação de tamanho antes do parse, erros tipados e rate-limit.
9. **Validação:** `cargo test -p wyd-protocol -p wyd-net` e `flood.rs`.
10. **Proteção atual:** **PASSA** para framing/parse; não cobre semântica de todo opcode.

### T17 — Modificação de cliente

1. **Comportamento:** cliente alterado para mudar dano, velocidade ou inventário.
2. **Versão:** 7662; Rust alvo.
3. **Causa:** confiar em valores calculados no cliente.
4. **Evidência:** dano/EXP/inventário são autoritativos no núcleo (`autoridade.rs`, `inventario.rs`); checksum/version gate no handshake.
5. **Impacto:** trapaça e economia paralela.
6. **Sistema:** cliente/protocolo.
7. **Condições:** binário modificado falando com o servidor.
8. **Prevenção:** não confiar no cliente e validar cada intenção.
9. **Validação:** testes de pacote forjado; não há anti-cheat de integridade do executável.
10. **Proteção atual:** **PARCIAL** — autoridade PASSA onde há handler; anti-tamper não existe.

### T18 — Speed hack

1. **Comportamento:** andar mais rápido que o orçamento.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** aceitar a posição do cliente.
4. **Evidência:** `movement::step` valida orçamento e limites; `autoridade.rs::passo_dentro_do_orcamento...`; o handler `_MSG_Move` ainda não existe.
5. **Impacto:** vantagem em PVP/fuga.
6. **Sistema:** movimento.
7. **Condições:** pacotes de movimento em alta frequência.
8. **Prevenção:** orçamento por tick e rate-limit.
9. **Validação:** teste de núcleo; fio aberto.
10. **Proteção atual:** **PARCIAL** (`GAME-05`/`GAME-20`).

### T19 — Attack speed hack

1. **Comportamento:** atacar acima do cooldown ou duplicar hits.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** cooldown calculado no cliente.
4. **Evidência:** `systems/combat.rs` grava `next_attack_ms` e recusa em cooldown (`ataque_em_cooldown_nao_altera_estado`); o handler usa o relógio do processo.
5. **Impacto:** DPS acima do permitido.
6. **Sistema:** combate.
7. **Condições:** rajada de `_MSG_Attack`.
8. **Prevenção:** cooldown autoritativo e rate-limit.
9. **Validação:** teste de núcleo; falta teste de rajada no fio e a anomalia ~200% (`GAME-20`).
10. **Proteção atual:** **PARCIAL**.

### T20 — Bots

1. **Comportamento:** clientes automatizados caçando, coletando ou floodando.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** ausência de detecção comportamental e limites frouxos.
4. **Evidência:** rate-limit de handshake/pacotes (`wyd-net`), limite por IP (`limite_ip.rs`); `QA-01`/`QA-11` são as issues de bots de teste, não de detecção.
5. **Impacto:** economia e carga.
6. **Sistema:** rede/antiautomação.
7. **Condições:** muitas contas ou padrões repetitivos.
8. **Prevenção:** limite por IP, telemetria e detecção de padrão.
9. **Validação:** `flood.rs` e `limite_ip.rs`; detecção de bots não existe.
10. **Proteção atual:** **PARCIAL** — contenção de carga PASSa; antiautomação **ABERTA**.

### T21 — Automação indevida

1. **Comportamento:** macros de coleta/combate sem presença humana.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** ações repetitivas aceitas sem challenge.
4. **Evidência:** rate-limit genérico; nenhum detector de macro.
5. **Impacto:** farm e concorrência desleal.
6. **Sistema:** gameplay/rede.
7. **Condições:** sequência de ações idêntica por longo período.
8. **Prevenção:** limite de taxa e análise de telemetria.
9. **Validação:** inexistente; candidato a issue de observabilidade/antiautomação.
10. **Proteção atual:** **ABERTO**.

### T22 — Manipulação de valores

1. **Comportamento:** enviar dano, EXP, ouro, quantidade ou refino arbitrários.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** valor do cliente usado como verdade.
4. **Evidência:** dano/EXP/refino/inventário usam o snapshot do servidor (`autoridade.rs`, `config_gameplay.rs`, `refino_piedade.rs`); `current_exp` é ignorado.
5. **Impacto:** progressão e economia quebradas.
6. **Sistema:** combate/inventário/economia.
7. **Condições:** payload com campo numérico forjado.
8. **Prevenção:** derivar o valor no servidor e ignorar claims.
9. **Validação:** testes de pacote forjado; falta cobrir todos os opcodes.
10. **Proteção atual:** **PARCIAL** — caminhos implementados PASSAM; opcodes sem handler caem em `outro` tolerado.

### T23 — Replay de requisições

1. **Comportamento:** reenviar compra, entrega, quest ou logout.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** operação sem chave/idempotência.
4. **Evidência:** `delivery_queue.idempotency_key`, `reward_mask`, `AlreadyRewarded`, sessão idempotente; E2E de idempotência.
5. **Impacto:** recompensa/compra dupla.
6. **Sistema:** grants/quest/sessão.
7. **Condições:** retry de rede ou cliente malicioso.
8. **Prevenção:** chave única por operação e estado `claimed`.
9. **Validação:** replay de grant e quest cobertos; trade/compra não.
10. **Proteção atual:** **PARCIAL** (`WEB-09`, `CLIENT-10`).

### T24 — Race conditions

1. **Comportamento:** duas operações no mesmo slot/sessão/linha.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** check-then-act sem lock/transação.
4. **Evidência:** `claim_session`, `claim_gold`, `claim_item` com transação/`FOR UPDATE`; `sessao_unica.rs`; `delivery_queue` concorrente.
5. **Impacto:** duplicação, perda de item e sessão dupla.
6. **Sistema:** banco/sessão/economia.
7. **Condições:** duas threads/processos no mesmo registro.
8. **Prevenção:** transação curta, constraint e `FOR UPDATE`.
9. **Validação:** sessão/grants têm prova; inventário/troca concorrente não.
10. **Proteção atual:** **PARCIAL** (`QA-09`).

### T25 — Abuso de temporizadores

1. **Comportamento:** manter buff/portão além do tempo ou reaplicar sem cooldown.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** relógio do cliente ou timer sem autoridade.
4. **Evidência:** portões do Campo fecham por tempo no instante (`CAMP_GATES`); `TimedEffect` existe no sim; affects no fio não existem.
5. **Impacto:** buff permanente e portão aberto.
6. **Sistema:** buffs/quest/mundo.
7. **Condições:** reaplicação ou packet de action repetido.
8. **Prevenção:** timestamp do servidor e cooldown.
9. **Validação:** teste de portão (`campo_treinamento.rs`); buff sem teste de fio.
10. **Proteção atual:** **PARCIAL** (`GAME-18`, `GAME-19`).

### T26 — Falhas de persistência

1. **Comportamento:** perder item/ouro/posição após queda ou restart.
2. **Versão:** 7662 / Go; Rust alvo.
3. **Causa:** save fora de transação ou fila sem flush.
4. **Evidência:** `SaveQueue`/`SaveSink`, `replace_items` transacional, `migrate.rs` com ledger; `entrada_mundo.rs` grava/relê vitais; `pg.rs` cobre portas.
5. **Impacto:** rollback de progresso e duplicação por reentrega.
6. **Sistema:** persistência.
7. **Condições:** queda durante escrita ou dreno atrasado.
8. **Prevenção:** transação, fila com flush no shutdown e idempotência.
9. **Validação:** testes de banco e entrada; falta crash test/restart com carga (`QA-10`).
10. **Proteção atual:** **PARCIAL**.

### T27 — Falhas de autenticação

1. **Comportamento:** senha/PIN fraco, brute force ou vazamento em log.
2. **Versão:** 7662; Rust alvo.
3. **Causa:** hash fraco, PIN sem lockout ou log de segredo.
4. **Evidência:** Argon2id (`auth/password.rs`), PIN 4–6 com 5 falhas/10 min persistidos, `login.rs` e `pin_persistencia.rs`; teste de segredo ausente no log.
5. **Impacto:** takeover de conta.
6. **Sistema:** auth.
7. **Condições:** tentativas repetidas ou vazamento de log.
8. **Prevenção:** hash forte, lockout persistido e sanitização.
9. **Validação:** suíte de login/PIN; rate-limit de handshake.
10. **Proteção atual:** **PASSA** para senha/PIN; MFA não existe.

### T28 — Falhas de autorização

1. **Comportamento:** jogador acessa painel/admin ou comando GM sem papel.
2. **Versão:** 7662 / portal; Rust alvo.
3. **Causa:** rota/opcode sem checagem de papel.
4. **Evidência:** `wyd-admin` exige sessão/papel; `tests/contrato/autorizacao.rs` cobre 401/403; o portal usa BFF com sessão.
5. **Impacto:** entrega de item, ban e leitura de contas.
6. **Sistema:** admin/portal/GM.
7. **Condições:** chamada direta sem cookie/papel.
8. **Prevenção:** autorização no servidor e escopo por rota.
9. **Validação:** contrato de autorização; comandos GM no jogo ainda não existem.
10. **Proteção atual:** **PASSA** no admin; **ABERTO** para GM in-game (`WEB-08`).

### T29 — Problemas de concorrência

1. **Comportamento:** corrupção por duas tarefas no mesmo estado.
2. **Versão:** Rust alvo.
3. **Causa:** compartilhamento sem fila/lock ou teste concorrente ausente.
4. **Evidência:** engine em thread única por canal, filas `EngineHandle`, `SaveQueue`; testes de autoridade e sessão. A suíte dedicada de concorrência não existe.
5. **Impacto:** estado inconsistente e duplicação.
6. **Sistema:** engine/persistência/rede.
7. **Condições:** carga concorrente e reconexão.
8. **Prevenção:** ownership, canais e transações; medir com carga.
9. **Validação:** `QA-09` (miri/tarpaulin/concorrência) e `QA-10` (carga) estão abertas.
10. **Proteção atual:** **PARCIAL** — desenho seguro, prova de concorrência aberta.

## 3. Cobertura declarada e limites

- **B1–B17:** 7 PASSAM, 3 são parciais, 5 abertos, 1 observado e 1 divergente por decisão.
- **§6:** nenhuma linha fica vazia; as 29 ameaças têm os dez campos e um estado explícito.
- **Não é cobertura:** código sem teste, teste de núcleo sem fio, ou “o cliente não reproduziu” não viram PASSA.
- **Dependência visual:** B1/B4/B8/B9 e a entrada do `GAME-17` continuam dependentes de `CLIENT-06`; não há prova no 7662 nesta rodada.
- **Decisão pendente:** B6 (posição exata × spawn por cidade). O Rust atual persiste a posição exata; o histórico Go considera o spawn por cidade o comportamento esperado. Não alterar sem decisão do dono.
- **Próximas issues ligadas:** `GAME-05` (fio de inventário/skill/movimento), `GAME-11` (caixa/pickup), `GAME-19` (cadeia e affects B13–B17), `CLIENT-10` (trade), `QA-09`/`QA-10` (concorrência/carga) e `WEB-08` (GM/autorização).
