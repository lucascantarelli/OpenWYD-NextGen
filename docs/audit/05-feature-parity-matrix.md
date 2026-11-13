# 05 — Matriz principal de paridade (legado × `main` × Rust)

> **Base:** `docs/arquitetura/contrato-dispatch-opcode.md` (63 handlers C++ × 61 rotas Go × 198
> opcodes Rust), `docs/rust/08-checklist-funcionalidades.md` (estado com evidência),
> `roadmap-rust/issues/*` e as matrizes `04-client-analysis.md`.
> **Regra:** “existe código” ≠ “portado”. `P4/P5` exigem teste **e** fio; `main` refere-se à
> branch legada preservada em `origin/main`.

## 1. Legenda

- **Legado / main:** `✅` implementado · `🚧` parcial · `❌` ausente · `—` não se aplica.
- **Rust:** `✅` com teste e fio · `🟡` com teste, sem fio · `🚧` parcial · `⬜` ausente · `⛔` bloqueado por layout/decisão.
- **Status:** vocabulário do `README.md` §4, abreviado.
- **Paridade:** P0–P5 (`README.md` §4).

## 2. Tabela A — identidade, presença e status

| ID | Categoria | Funcionalidade | Legado | main | Rust | Cliente | Protocolo | Status | Par |
|---|---|---|---|---|---|---|---|---|---|
| PAR-AUT-01 | Autenticação | Login de conta (senha) | ✅ | ✅ | ✅ | CLI-02 | `0x020D`/`0x010A` | Implementado e testado | **P4** |
| PAR-AUT-02 | Autenticação | Senha numérica (PIN) + bloqueio | ✅ | ✅ | ✅ | CLI-03 | `0x0FDE`/`0x0FDF` | Implementado e testado | **P4** |
| PAR-AUT-03 | Autenticação | Troca de PIN (2 etapas) | ✅ | ✅ | ✅ | CLI-03 | `0x0FDE change_numeric` | Divergência identificada (semântica nossa) | **P4** |
| PAR-AUT-04 | Autenticação | Sessão única entre canais | ✅ | ✅ | ✅ | — | DB `0030` | Implementado e testado | **P4** |
| PAR-AUT-05 | Autenticação | Limite de conexões por IP | ✅ | 🚧 | ✅ | — | TCP antes do login | Implementado e testado | **P4** |
| PAR-AUT-06 | Autenticação | Canal lotado / manutenção | ✅ | ✅ | ✅ | CLI-01 | `serv00.htm` + recusa | Paridade funcional confirmada | **P5** |
| PAR-AUT-07 | Autenticação | Logout explícito + ociosidade | ✅ | ✅ | ✅ | — | `0x0215` | Implementado e testado | **P4** |
| PAR-AUT-08 | Autenticação | Erro de senha visível no cliente | ✅ | ✅ | 🚧 | CLI-16 | `0x0117` **ausente** no 7662 | Divergência identificada | **P2** |
| PAR-CHR-01 | Personagem | Criação (`CreateCharacter`) | ✅ | ✅ | ✅ | CLI-04 | `0x020F`/`0x0110` | Implementado e testado | **P4** |
| PAR-CHR-02 | Personagem | Deleção (`DeleteCharacter`) | ✅ | ✅ | ✅ | CLI-05 | `0x0211`/`0x0112` | Implementado e testado | **P4** |
| PAR-CHR-03 | Personagem | Seleção e entrada no mundo | ✅ | ✅ | ✅ | CLI-06 | `0x0213`/`0x0114` | Integrado, não validado (visual) | **P4** |
| PAR-CHR-04 | Personagem | Tela de personagens (`STRUCT_SELCHAR`) | ✅ | ✅ | 🚧 | CLI-04 | corpo 844 B | Implementado parcialmente (layout `rust-v1`) | **P1** |
| PAR-CHR-05 | Personagem | Distribuição de pontos (`ApplyBonus`) | ✅ | ✅ | ⬜ | — | `0x0277` | Não iniciado em Rust | **P0** |
| PAR-CHR-06 | Personagem | Aparência / customização | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-CHR-07 | Personagem | Rename / recuperação de conta | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-MOV-01 | Mundo | Movimento no fio (ação/motion) | ✅ | ✅ | 🚧 | CLI-08 | `0x036C`/`0x036A` | Implementado sem integração | **P2** |
| PAR-MOV-02 | Mundo | Colisão de jogador | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust (só rota de NPC) | **P1** |
| PAR-MOV-03 | Mundo | Teleporte / portais | ✅ | ✅ | ⬜ | — | `0x0290`/`0x036C` | Não iniciado em Rust | **P0** |
| PAR-MOV-04 | Mundo | Mapas, regiões e coordenadas | ✅ | ✅ | ✅ | — | seed `0020/0023/0011` | Implementado e testado (dado) | **P4** |
| PAR-MOV-05 | Mundo | Spawn / respawn automático | ✅ | ✅ | 🚧 | CLI-07 | `0x0364`/`0x0165` | Implementado parcialmente | **P2** |
| PAR-MOV-06 | Mundo | Visibilidade / interest management | ✅ | ✅ | ✅ | CLI-07 | `ESCENE_FIELD` | Implementado e testado | **P4** |
| PAR-MOV-07 | Mundo | Clima / ciclo dia-noite | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-NPC-01 | NPC | Spawn/despawn por mapa | ✅ | ✅ | ✅ | CLI-07 | `CreateMob`/`RemoveMob` | Implementado e testado | **P4** |
| PAR-NPC-02 | NPC | IA (FSM + utilidade, 5 Hz) | ✅ | ✅ | ✅ | — | interno | Paridade funcional confirmada | **P5** |
| PAR-NPC-03 | NPC | Diálogo / script | ✅ | ✅ | 🚧 | CLI-15 | `0x028B` | Implementado parcialmente (merchant) | **P2** |
| PAR-NPC-04 | NPC | Loja (compra/venda) | ✅ | ✅ | 🚧 | CLI-14 | `0x027B`/`0x0379`/`0x037A` | Divergência identificada (checklist × código) | **P3** |
| PAR-NPC-05 | NPC | Teleporte por NPC / serviço | ✅ | ✅ | ⬜ | — | `0x0290` | Não iniciado em Rust | **P0** |
| PAR-CBT-01 | Combate | Ataque físico (intenção→resolução) | ✅ | ✅ | ✅ | CLI-10 | `0x0367`/`0x039D`/`0x039E` | Implementado e testado | **P3** |
| PAR-CBT-02 | Combate | Fórmulas de dano/defesa/crítico | ✅ | ✅ | ⛔ | — | — | Bloqueado por decisão (`FASE-10`) | **P1** |
| PAR-CBT-03 | Combate | Skills em combate (custo/cast/alvo) | ✅ | ✅ | 🟡 | — | sem payload | Implementado sem integração | **P2** |
| PAR-CBT-04 | Combate | Morte (remoção da cena) | ✅ | ✅ | ✅ | CLI-07 | `RemoveMob` `RemoveType=1` | Implementado e testado | **P4** |
| PAR-CBT-05 | Combate | Regen de HP/MP | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P1** |
| PAR-CBT-06 | Combate | PvP, PK, chaos e penalidades | ✅ | ✅ | ⬜ | — | `0x0399` | Não iniciado em Rust | **P1** |
| PAR-CBT-07 | Combate | Buffs/debuffs/affects | ✅ | ✅ | ⬜ | — | layout UNVERIFIED | Não iniciado em Rust | **P0** |
| PAR-CBT-08 | Combate | Boss e invasões | ✅ | ✅ | 🚧 | — | — | Implementado parcialmente (Campo) | **P1** |
| PAR-INV-01 | Inventário | Inventário e equipamento (core) | ✅ | ✅ | ✅ | CLI-06 | `STRUCT_MOB` leitura | Implementado e testado | **P4** |
| PAR-INV-02 | Inventário | Mover/equipar/vender no fio | ✅ | ✅ | 🚧 | — | `0x0374`/`0x0373` | Implementado parcialmente | **P2** |
| PAR-INV-03 | Inventário | Drop e entrega no inventário | ✅ | ✅ | ✅ | — | `_MSG_SendItem` | Implementado e testado | **P4** |
| PAR-INV-04 | Inventário | Caixa no chão + pickup | ✅ | ✅ | 🚧 | CLI-12 | `0x026E`/`0x0270`/`0x0171` | Bloqueado por layout (`CLIENT-06`) | **P1** |
| PAR-INV-05 | Inventário | Baú / armazém | ✅ | ✅ | ⬜ | — | `0x0387`/`0x0388` | Não iniciado em Rust | **P0** |
| PAR-INV-06 | Inventário | Durabilidade / reparo / quebra | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-INV-07 | Comércio | Trade jogador↔jogador | ✅ | ✅ | ⛔ | CLI-13 | `0x0383`/`0x0376` | Bloqueado por layout (156×154 B) | **P1** |
| PAR-INV-08 | Inventário | Gema no inventário (SancRate) | ✅ | ✅ | ✅ | — | ADR-0012 | Implementado e testado | **P4** |
| PAR-ITM-01 | Itens | Catálogo de itens (3.220) | ✅ | ✅ | ✅ | CLI-18 | seed `0018/0017` | Implementado e testado (dado) | **P4** |
| PAR-ITM-02 | Itens | Refino +0…+9 (taxa + piedade) | ✅ | ✅ | ✅ | CLI-11 | `0x0373`→`0x0182` | Implementado e testado | **P4** |
| PAR-ITM-03 | Itens | Anct / celestial / +12…+15 | ✅ | ✅ | 🚧 | — | — | Implementado parcialmente | **P2** |
| PAR-ITM-04 | Itens | Mix / compose | ✅ | ✅ | 🟡 | — | sem payload | Implementado sem integração | **P2** |
| PAR-ITM-05 | Itens | Craft | ✅ | 🚧 | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-ITM-06 | Itens | Imbuir / repleção | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-ITM-07 | Itens | Sets, adicional e sanc | ✅ | ✅ | 🚧 | — | — | Implementado parcialmente | **P2** |
| PAR-ECO-01 | Economia | Ouro persistido | ✅ | ✅ | ✅ | CLI-04 | `STRUCT_SELCHAR` | Implementado e testado (sem dreno) | **P4** |
| PAR-ECO-02 | Economia | Loja de NPC (preço/estoque) | ✅ | ✅ | 🚧 | CLI-14 | `ReqBuy`/`Buy`/`Sell` | Divergência identificada (doc×código) | **P3** |
| PAR-ECO-03 | Economia | Mercado / leilão | ✅ | ❌ | ⬜ | — | — | Não aplicável ao 7662 (sem cliente) | **P0** |
| PAR-ECO-04 | Economia | Imposto, taxas e drenos | ✅ | ✅ | 🚧 | — | seed `0010` | Implementado parcialmente | **P1** |
| PAR-ECO-05 | Economia | Donate / moeda premium | ✅ | ✅ | 🚧 | — | `0042`/`0045` | Implementado parcialmente | **P2** |
| PAR-SOC-01 | Social | Chat e whisper | ✅ | ✅ | ⬜ | CLI-09 | `0x0333`/`0x0334` | Não iniciado em Rust | **P0** |
| PAR-SOC-02 | Social | Party / grupo | ✅ | ✅ | ⬜ | — | `0x037F`/`0x03AB`/`0x037E` | Não iniciado em Rust | **P0** |
| PAR-SOC-03 | Social | Amigos / ignore | ✅ | ✅ | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-SOC-04 | Social | Guildas (criação, cargos, membros) | ✅ | ✅ | 🚧 | — | `0x03D5`/`0x0E12` | Implementado parcialmente (schema+roster) | **P1** |
| PAR-SOC-05 | Social | Guerra de guilda / castelo / reinos | ✅ | ✅ | ⬜ | — | `0x0E0E` | Não iniciado em Rust | **P0** |
| PAR-QST-01 | Quests | Campo de Treinamento (quest+portões) | ✅ | ✅ | 🚧 | CLI-15 | `0x028B`/`0x036C` | Implementado parcialmente | **P3** |
| PAR-QST-02 | Quests | Story line, Kefra, runas, Arch | ✅ | ✅ | ⬜ | — | `0x028B` | Não iniciado em Rust | **P0** |
| PAR-QST-03 | Quests | Diárias / repetição / limites | ✅ | 🚧 | ⬜ | — | — | Não iniciado em Rust | **P0** |
| PAR-EVT-01 | Eventos | Eventos automáticos e calendário | ✅ | ✅ | ⬜ | — | seed `0014` | Não iniciado em Rust | **P1** |
| PAR-EVT-02 | Eventos | Arena / Battle Royale | ✅ | 🚧 | ⬜ | — | seed `0015` | Não iniciado em Rust | **P0** |
| PAR-CID-01 | Cidadania | Cidadania derivada do canal | ✅ | ✅ | ✅ | — | `0041` `channel_role` | Implementado e testado | **P4** |
| PAR-CID-02 | Cidadania | Compra/atribuição de reino | ✅ | ✅ | ⬜ | — | diálogo de NPC | Não iniciado em Rust | **P0** |
| PAR-ADM-01 | Administração | BFF do portal (`adminCall`) | ✅ | ✅ | ✅ | — | HTTP/JSON | Implementado e testado | **P5** |
| PAR-ADM-02 | Administração | Grants via `delivery_queue` | ✅ | ✅ | ✅ | — | `0006` | Implementado e testado | **P4** |
| PAR-ADM-03 | Administração | Moderação de staff (papel, bloqueio) | ✅ | ✅ | ✅ | — | `0023` | Implementado e testado | **P4** |
| PAR-ADM-04 | Administração | Auditoria administrativa | ✅ | ✅ | ✅ | — | `0005` | Implementado e testado | **P4** |
| PAR-ADM-05 | Administração | Configuração 100% via portal | ✅ | 🚧 | 🚧 | — | `config_revision` | Implementado parcialmente | **P3** |
| PAR-LCH-01 | Launcher | Manifesto de integridade | ✅ | ✅ | ✅ | CLI-19 | SHA-256 por arquivo | Paridade funcional confirmada | **P5** |
| PAR-LCH-02 | Launcher | Patch atômico + rollback | ✅ | ✅ | ✅ | — | Ed25519 | Implementado e testado | **P4** |
| PAR-LCH-03 | Launcher | `serverlist.bin` + `sn.bin` | ✅ | ✅ | ✅ | CLI-01 | bytes idênticos ao Node | Paridade funcional confirmada | **P5** |
| PAR-LCH-04 | Launcher | GUI (Tauri v2) | ✅ | — | 🚧 | — | — | Implementado parcialmente (HTML) | **P1** |
| PAR-LCH-05 | Launcher | Autoupdate via portal | ✅ | ❌ | 🚧 | — | — | Implementado parcialmente | **P2** |
| PAR-CLI-01 | Cliente | Layout de `STRUCT_SELCHAR` | — | — | 🚧 | CLI-04 | 844 B | Bloqueado por captura | **P1** |
| PAR-CLI-02 | Cliente | Layouts S→C (`STRUCT_MOB`, cena) | — | — | 🚧 | CLI-06/07 | 1820 B | Bloqueado por captura | **P2** |
| PAR-CLI-03 | Cliente | Captura ao vivo / RE dinâmica | — | — | ⬜ | — | — | Bloqueado por ambiente | **P0** |
| PAR-CLI-04 | Cliente | Customização visual (`.wyt`, branding) | — | — | 🚧 | CLI-17 | — | Implementado parcialmente | **P3** |
| PAR-PLT-01 | Plataforma | Tick determinístico no processo | ✅ | ✅ | ✅ | — | — | Implementado e testado | **P4** |
| PAR-PLT-02 | Plataforma | Telemetria e métricas | ✅ | 🚧 | ✅ | — | `0x01/0x00` | Implementado e testado | **P4** |
| PAR-PLT-03 | Plataforma | Migrations no boot | ✅ | 🚧 | ✅ | — | 51 arquivos | Implementado e testado | **P4** |
| PAR-PLT-04 | Plataforma | Bots de QA/carga | ✅ | ✅ | 🚧 | — | crate próprio | Implementado parcialmente | **P2** |
| PAR-PLT-05 | Plataforma | CI/CD e release | ✅ | 🚧 | ✅ | — | `v0.1.0` | Implementado e testado | **P4** |

## 3. Tabela B — descrição, diferenças, problema, dependência, evidência e ação

| ID | Descrição | Diferenças | Problemas | Dependências | Evidências | Observações | Pri | Cx | Risco | Próximas ações |
|---|---|---|---|---|---|---|---|---|---|---|
| PAR-AUT-01 | Senha conferida antes do `CNF`, Argon2id | CNF legado 2008 B × nosso 1996 B de corpo | `0x010A` sem captura | — | `cpsock::handshake`, `tests/sistema/login.rs` | — | Alta | Baixa | Baixo | Confirmar corpo na captura |
| PAR-AUT-02 | PIN 4–6 dígitos, 5 falhas → 10 min persistidos | Schema novo (`pin_hash`, `pin_failures`) | — | — | `cpsock::authorize_pin`, `pin_persistencia.rs` | — | Alta | Baixa | Baixo | — |
| PAR-AUT-03 | Passo 1 arma, passo 2 grava | Semântica é decisão nossa | Não confirmado no cliente | captura | `pin_troca.rs`, `docs/wyd/senha-numerica.md` | Documentar como decisão | Média | Média | Médio | Capturar `change_numeric` |
| PAR-AUT-04 | `SessionGate` por conta | Legado usava sessão em memória | — | — | `0030`, `sessao_unica.rs` | — | Alta | Baixa | Baixo | — |
| PAR-AUT-05 | Conta TCP por IP antes do login | Padrão 6, `0` desliga | Conexão abandonada ocupa vaga até 10 s | — | `limite_ip.rs` | — | Média | Baixa | Baixo | — |
| PAR-AUT-06 | `Capacity`/`Maintenance` antes da sessão | — | — | — | `lotacao.rs` | — | Alta | Baixa | Baixo | — |
| PAR-AUT-07 | `CNFCharacterLogout` + `leave_world` + `Drop` | — | Golden de frame pendente | captura | `selchar.rs::logout_explicito…` | — | Alta | Baixa | Baixo | — |
| PAR-AUT-08 | Erro de senha deveria aparecer | `0x0117` não existe no 7662 | Jogador não vê motivo | `0x0101` provado | `docs/cliente/pacotes.md` | Causa raiz de `BUG-01` | Alta | Média | Médio | Emitir `0x0101` com layout provado |
| PAR-CHR-01 | Ficha inicial por classe + 5 pontos | `BaseSIDCHM`/`BASE_GetHpMp` | Sem validação de aparência | — | `personagens_crud.rs` | — | Alta | Média | Médio | — |
| PAR-CHR-02 | Soft delete com confirmação de nome | — | — | — | idem | — | Alta | Baixa | Baixo | — |
| PAR-CHR-03 | Personagem vira entidade e entra na cena | `CreateType=2` só para o dono | Render não provado | `CLIENT-06` | `cena_mobs.rs`, `world/{scene_snapshot,plan_scene}` | — | Alta | Alta | Alto | Prova visual |
| PAR-CHR-04 | 14 campos de roster | Layout `rust-v1` | Offset não confirmado | captura | `selchar.rs`, `captura-selchar.md` | Promove 4 linhas | Alta | Média | Alto | Capturar |
| PAR-CHR-05 | `_MSG_ApplyBonus` | Ausente | Jogador não distribui pontos | — | checklist 2.2 | Bloqueia build de personagem | Média | Média | Médio | Implementar handler |
| PAR-CHR-06 | Aparência | Ausente | — | — | checklist 2.4 | Cosmético | Baixa | Alta | Baixo | Adiar |
| PAR-CHR-07 | Rename/recuperação | Ausente | — | portal | checklist 2.5 | Operação, não jogo | Baixa | Média | Baixo | Portal |
| PAR-MOV-01 | Intenção de movimento no tick | Só core, sem pacote | Jogador não anda no fio | layout de ação | `systems/movement.rs` | Afeta “200%” | Alta | Alta | Alto | Definir payload provado |
| PAR-MOV-02 | Colisão de jogador | Ausente | Atravessa parede | mapas | `spatial/grid.rs` | — | Média | Alta | Alto | Derivar de `AttributeMap` |
| PAR-MOV-03 | Teleporte/portal | Ausente | Sem viagem de mapa | `0x0290` | checklist 3.6 | — | Alta | Média | Médio | Handler + rota |
| PAR-MOV-04 | 52 regiões semeadas | — | Sem regra de transição | — | `0020/0023/0011` | — | Alta | Baixa | Baixo | — |
| PAR-MOV-05 | Spawn com o mapa; morte remove | Sem respawn próprio | Mundo esvazia | recarga de definição | `world/npc.rs` | — | Média | Média | Médio | Timer de respawn |
| PAR-MOV-06 | Grade espacial + delta por sessão | Raio sem fonte provada | Visibilidade pode divergir | captura | `spatial/`, `world::tests` | — | Alta | Média | Médio | Medir raio |
| PAR-MOV-07 | Clima/ciclo | Ausente | — | — | checklist 3.9 | Só visual hoje | Baixa | Alta | Baixo | Adiar |
| PAR-NPC-01 | NPC nasce com o mapa | — | — | — | GAME-07/08/09 | — | Alta | Baixa | Baixo | — |
| PAR-NPC-02 | FSM + utilidade 5 Hz com orçamento | — | — | — | `core/npc/*`, `integracao/npc.rs` | Modelo próprio, não legado | Média | Média | Médio | Validar contra legado |
| PAR-NPC-03 | Merchant do Campo responde `0x028B` | Só um NPC | Diálogo geral ausente | scripts | `handle_quest` | — | Média | Alta | Médio | Modelo de diálogo |
| PAR-NPC-04 | Loja viva existe em teste de sistema | Checklist 7.2 diz ⬜ | Doc e código divergem | — | `tests/sistema/loja_viva.rs`, `noticias/2026-11-11-lojas-de-npc.md` | `DIV-09` | Alta | Média | Médio | Reconciliar e cobrir preço |
| PAR-NPC-05 | Teleporte por NPC | Ausente | — | MOV-03 | checklist 3.5 | — | Média | Média | Médio | — |
| PAR-CBT-01 | Ataque → dano → morte → drop → EXP | Cadência 800 ms não provada | “Hits duplicados” (`GAME-20`) | captura | `attack_at`, `config_gameplay.rs` | — | Alta | Média | Alto | Trace de cadência |
| PAR-CBT-02 | Fórmulas do legado | `wyd-sim` não é a do C++ | Dano pode divergir | FASE-10 | `formula/damage.rs` | Não inventar | Alta | Alta | Alto | Portar `Basedef.cpp:1265` |
| PAR-CBT-03 | Custo, cooldown, alcance, alvo | Sem pacote | Skill não usável | layout | `integracao/autoridade.rs` | — | Alta | Média | Médio | Payload de skill |
| PAR-CBT-04 | Golpe fatal remove da cena | — | Sem respawn | MOV-05 | `RemoveMob`, cena | — | Alta | Baixa | Baixo | — |
| PAR-CBT-05 | Regen periódica | Ausente | HP não recupera | tick | `formula/regen.rs` | — | Média | Baixa | Baixo | Ligar no tick |
| PAR-CBT-06 | PvP/PK/chaos | Só colunas (`0048`) | Sem penalidade | — | `docs/wyd/sistemas/guerras.md` | Números divergem entre fontes | Média | Alta | Alto | Definir regra + teste |
| PAR-CBT-07 | Affects (buff/debuff) | Ausente | Sem skill de suporte | captura | checklist 6.4 | Layout UNVERIFIED | Média | Alta | Alto | Capturar |
| PAR-CBT-08 | Guardiões/boss do Campo | Só pipeline, sem buff | `Cabuncle` sem efeito | GAME-18 | `integracao_campo_drops` | — | Média | Média | Médio | Buff do boss |
| PAR-INV-01 | 16–80 cargo, 120 por pilha | `MAX_CARGO` 128 no OSS vs 120 aqui | Dialeto de build | — | ADR-0012/0015, `integracao/inventario.rs` | — | Alta | Baixa | Baixo | — |
| PAR-INV-02 | Move/equip persistido | Sem pacote | Não equipa em jogo | layout | `systems/inventory.rs` | — | Alta | Média | Alto | `0x0374` |
| PAR-INV-03 | Roll na morte + `SendItem` por slot | Modelo por item, legado por slot | Peso muda distribuição | — | `config_gameplay.rs` | Ver `DIV-05` | Alta | Média | Médio | Comparar curvas |
| PAR-INV-04 | Caixa com posse (30 s/3 min) | Só a regra pura | Sem item no chão | `CLIENT-06` | GAME-11 | Direção do `0x026E` provada | Média | Alta | Alto | Handler após captura |
| PAR-INV-05 | Baú/armazém | Ausente | — | layout | `0x0387`/`0x0388` | `vault_items` citado | Média | Alta | Médio | Schema + handler |
| PAR-INV-06 | Durabilidade | Ausente | — | — | checklist 5.9 | — | Baixa | Média | Médio | — |
| PAR-INV-07 | Trade atômica por par | 156 B × 154 B | Não pode arriscar duplicação | captura | CLIENT-10, `trade-layout.mjs` | Manter `require_ported` | Alta | Alta | Alto | Capturar frame |
| PAR-INV-08 | Gema consome SancRate | — | — | — | ADR-0012 | — | Média | Baixa | Baixo | — |
| PAR-ITM-01 | 3.220 itens, 20 sistemas | Índices do cliente ≠ banco | Nome igual, função diferente | — | seeds `0017/0018` | Cruzar por índice | Alta | Baixa | Médio | — |
| PAR-ITM-02 | Taxa por tentativa + piedade | Piedade é nossa | Pode divergir do legado | — | `0043`, `refino_piedade.rs` | — | Alta | Baixa | Baixo | Comparar `g_pSuccessRate` |
| PAR-ITM-03 | Anct/celestial/+12…+15 | Cadeia não portada | Refino para no +9 | FASE-10 | checklist 5.4 | +25AC/+40dmg em aberto | Média | Alta | Médio | Portar tabela B16 |
| PAR-ITM-04 | Mix no core | Sem opcode | Mix não existe no jogo | layout | `integracao/compose.rs` | — | Média | Média | Médio | `0x03A6` |
| PAR-ITM-05 | Craft | Ausente | — | — | `game_system.craft` | — | Baixa | Alta | Baixo | Adiar |
| PAR-ITM-06 | Imbue/repleção | Ausente | — | — | `docs/wyd/sistemas/refino.md` | — | Baixa | Alta | Médio | — |
| PAR-ITM-07 | Sets/sanc/adicional | Dados sim, regra parcial | Bônus de conjunto ausente | — | `ItemSet.bin` (Global) | 7662 embute? `UNK-14` | Média | Alta | Médio | Investigar cliente |
| PAR-ECO-01 | `characters.gold` | Sem ganho/gasto pleno | Economia não fecha | loja | `selchar.rs` | — | Alta | Média | Médio | Ligar compra/venda |
| PAR-ECO-02 | Preço e estoque | Doc diz ⬜; teste existe | Divergência doc×código | — | `loja_viva.rs` | `DIV-09` | Alta | Média | Médio | Reconciliar |
| PAR-ECO-03 | Mercado | Legado tem, 7662 não expõe | Não aplicável | — | issue WEB-10 | Não implementar | Baixa | — | — | Fechar como N/A |
| PAR-ECO-04 | Imposto/drenos | Só seed | Sem dreno real | — | `0010` | Inflação | Média | Média | Médio | Regra + simulação |
| PAR-ECO-05 | Cash/donate | Parcial | Sem crédito no jogo | portal | `0042/0045` | — | Média | Média | Médio | Fechar loop |
| PAR-SOC-01 | Chat/whisper | Ausente | Jogo não tem comunicação | — | `0x0333` provado | Alto valor social | Alta | Média | Médio | Handler + filtro |
| PAR-SOC-02 | Party | Ausente | Sem grupo/EXP compartilhada | chat | `0x037F` | — | Média | Alta | Médio | — |
| PAR-SOC-03 | Amigos/ignore | Ausente | — | — | checklist 10.3 | — | Baixa | Média | Baixo | — |
| PAR-SOC-04 | Guildas | Schema + leitura | Sem handler | — | `0004` | — | Média | Alta | Médio | Criar/convidar |
| PAR-SOC-05 | Guerra/castelo | Ausente | Maior bloco não portado | guildas | `CCastleZakum.cpp` | Só após paridade base | Baixa | Muito alta | Alto | Não iniciar agora |
| PAR-QST-01 | Quest + portões + chaves | Só Campo | Story ausente | — | `integracao_campo_treinamento` 7/7 | — | Média | Média | Médio | Prova visual |
| PAR-QST-02 | Kefra/runas/Arch | Ausente | Progressão incompleta | — | `_MSG_Quest.cpp` | Conhecimento em `docs/wyd/sistemas/quests.md` | Média | Muito alta | Alto | Modelar quest genérica |
| PAR-QST-03 | Diárias | Ausente | — | QST-02 | — | — | Baixa | Alta | Baixo | — |
| PAR-EVT-01 | Eventos | Só seed | Sem calendário | tick | `0014` | — | Média | Alta | Médio | Agendador |
| PAR-EVT-02 | Arena/BR | Só seed | — | MOV/combate | `0015`, `battle-royale.md` | — | Baixa | Muito alta | Alto | Adiar |
| PAR-CID-01 | Cidadania = papel do canal | Compra não existe | Cidadania não escolhida | CID-02 | `0041`, `canal_verde.rs` | — | Alta | Baixa | Baixo | — |
| PAR-CID-02 | Compra/atribuição | Ausente | Jogador não troca de reino | diálogo | `canais-e-cidadania.md` | — | Média | Média | Médio | Diálogo NPC |
| PAR-ADM-01 | BFF puro sobre `wyd-admin` | — | — | — | `wyd-portal/src/server/admin.ts` | — | Alta | Baixa | Baixo | — |
| PAR-ADM-02 | Fila com idempotência | Item/nível pendentes | Entrega parcial | Hot Path | `0006`, E2E WEB-04 | — | Alta | Média | Médio | Drenar nível |
| PAR-ADM-03 | Papel + bloqueio com motivo | — | — | — | `0023`, E2E 6/6 | — | Alta | Baixa | Baixo | — |
| PAR-ADM-04 | `audit_log` | — | — | — | `0005` | — | Alta | Baixa | Baixo | — |
| PAR-ADM-05 | Config via portal | Rate/refino já; resto não | Ainda há arquivo manual | PROC-24 | `config_revision` | — | Média | Média | Médio | Inventariar chaves |
| PAR-LCH-01 | Manifesto por linha | — | — | — | `manifest.rs`, 19 testes | — | Alta | Baixa | Baixo | — |
| PAR-LCH-02 | tmp+rename+rollback | Spawn real não testado | — | ambiente | `patch.rs` | — | Alta | Média | Baixo | Testar no Windows |
| PAR-LCH-03 | Bytes idênticos ao Node | — | — | — | `serverlist_paridade.rs` | — | Alta | Baixa | Baixo | — |
| PAR-LCH-04 | GUI Tauri | Só HTML | Sem janela nativa | CLIENT-08 | `gui.rs` | — | Média | Alta | Médio | Shell Tauri |
| PAR-LCH-05 | Autoupdate | Ausente no portal | Atualização manual | WEB-04 | CLIENT-11 | — | Média | Alta | Médio | Endpoint assinado |
| PAR-CLI-01 | Roster 14 campos | `rust-v1` provisório | Não valida no cliente | captura | `captura-selchar.md` | Bloqueio externo | Alta | Média | Alto | CLIENT-06 |
| PAR-CLI-02 | `STRUCT_MOB` S→C | Portado de `Basedef.h` | Pode não renderizar | captura | `pacotes.md` | — | Alta | Média | Alto | CLIENT-06 |
| PAR-CLI-03 | Captura/RE dinâmica | — | Destrava 4 frentes | VM/GUI | `depuracao.md` | Único bloqueio externo | Alta | Alta | Alto | Executar runbook |
| PAR-CLI-04 | `.wyt`/branding/.msa | Sem editor de layout | UI fixa no `.exe` | ferramentas | `wyt.mjs`, `referencia-wyd-global.md` | — | Baixa | Média | Médio | Pipeline de asset |
| PAR-PLT-01 | Tick no processo | — | — | — | `tick_processo.rs` | — | Alta | Baixa | Baixo | — |
| PAR-PLT-02 | Métricas Prometheus | — | — | — | `metrics/*` | — | Média | Baixa | Baixo | — |
| PAR-PLT-03 | Migrations no boot | — | Acopla subida a schema | ADR-0024 | `migrate.rs` | — | Alta | Baixa | Médio | — |
| PAR-PLT-04 | 4 raças + campanha | Sem carga real | `QA-11` | — | `bot/campanha.rs` | — | Média | Média | Médio | 200 bots |
| PAR-PLT-05 | Pipeline + release | — | — | — | `v0.1.0`, `10-devops-ci.md` | — | Média | Baixa | Baixo | — |

## 4. Resumo por categoria

| Categoria | Linhas | P4–P5 | P3 | P2 | P0–P1 |
|---|---:|---:|---:|---:|---:|
| Autenticação | 8 | 6 | 1 | 1 | 0 |
| Personagem | 7 | 3 | 0 | 0 | 4 |
| Mundo | 7 | 2 | 0 | 2 | 3 |
| NPC | 5 | 2 | 1 | 2 | 1 |
| Combate | 8 | 2 | 1 | 1 | 4 |
| Inventário/comércio | 8 | 4 | 0 | 1 | 3 |
| Itens | 7 | 2 | 0 | 3 | 2 |
| Economia | 5 | 1 | 1 | 1 | 2 |
| Social | 5 | 0 | 0 | 0 | 5 |
| Quests | 3 | 0 | 1 | 0 | 2 |
| Eventos | 2 | 0 | 0 | 0 | 2 |
| Cidadania | 2 | 1 | 0 | 0 | 1 |
| Administração | 5 | 4 | 1 | 0 | 0 |
| Launcher | 5 | 3 | 0 | 1 | 1 |
| Cliente | 4 | 0 | 1 | 1 | 2 |
| Plataforma | 5 | 3 | 0 | 1 | 0 |
| **Total** | **86** | **33** | **8** | **14** | **31** |

**Leitura:** a força está onde o cliente e o banco foram medidos (autenticação, launcher,
administração, plataforma). A fraqueza está onde o **fio** não foi provado (social, movimento,
itens secundários) e onde o **layout** não existe (cliente, trade, caixa no chão).

## 5. Como manter

1. Só mude uma linha quando houver evidência nova (teste, captura, medição).
2. `P5` exige teste **e** fio; se o teste existe mas o pacote não sai, é `P3` no máximo.
3. Toda linha com `⛔` deve virar `UNK-NN` em `18-unknown-features.md` quando o bloqueio for de
   conhecimento, ou `DIV-NN` em `15-bugs-and-divergences.md` quando for de código.
