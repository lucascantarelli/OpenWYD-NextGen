/**
 * Metadados curados do catálogo do banco: para cada tabela, de onde vem a decisão e o quanto
 * dela está provado. Separado do parser porque isto é curadoria (humano), não derivação (SQL).
 *
 * `estado`:
 *   - `confirmado`  → existe fonte na árvore e o dado/regra está fechado.
 *   - `provisorio`  → a tabela é necessária, mas algum campo/semântica ainda não tem prova
 *                     (`nota` diz qual). Nunca usar como gate de comportamento.
 */

export const FONTES = {
  accounts: {
    dominio: 'acesso',
    fonte: 'docs/arquitetura/modelo-banco.md §2',
    estado: 'confirmado',
    nota: 'Hash de senha/PIN opaco (argon2id); nada em texto plano.',
  },
  character_session: {
    dominio: 'acesso',
    fonte: 'docs/wyd/canais-e-cidadania.md §5 + migration 0030',
    estado: 'provisorio',
    nota: 'Cerca de sessão por `account_login` (a autenticação da FASE-08-2 troca por account_id/character_id); `renewed_at` é o sinal de vida que retoma sessão órfã.',
  },
  characters: {
    dominio: 'personagem',
    fonte: 'modelo-banco.md §2 + docs/wyd/progressao.md §1 (camadas e Arch)',
    estado: 'confirmado',
    nota: '`tier`/`arch_of_id` vêm das camadas CONFIRMADAS de progressão (D-B02).',
  },
  character_scores: {
    dominio: 'personagem',
    fonte:
      'Basedef.cpp `BaseSIDCHM`/`BASE_GetHpMp`/`BASE_GetBonusScorePoint` + tmserver/internal/level/level.go',
    estado: 'confirmado',
    nota:
      'Atributos-base e pontos livres no nível 1; HP/MP máximos derivam de Con/Int e do incremento da classe.',
  },
  character_items: {
    dominio: 'inventário',
    fonte: 'modelo-banco.md §2 + docs/wyd/sistemas/refino.md §2–§3',
    estado: 'confirmado',
    nota: 'Refino/Anct/gema na criação da tabela: CHECK de gema espelha `Item` do core (D-B03).',
  },
  vault_items: {
    dominio: 'inventário',
    fonte: 'modelo-banco.md §2 + docs/wyd/sistemas/refino.md §2–§3',
    estado: 'confirmado',
    nota: 'Mesmas colunas e CHECKs de `character_items`, por conta.',
  },
  character_short_skill: {
    dominio: 'inventário',
    fonte:
      'CFileDB.cpp `memset(ShortSkill, -1, 16)` + protocolo.md §5.1 (`ShortSkill[16]@1034`)',
    estado: 'confirmado',
    nota:
      '`ShortSkill[16]` persistido do personagem; linha ausente (personagem anterior à 0033) é materializada com `0xFF` na primeira leitura.',
  },
  character_learned_skill: {
    dominio: 'personagem',
    fonte:
      '_MSG_ApplyBonus.cpp (`LearnedSkill` = máscara de 24 bits) + protocolo.md §5.1 (`Learn` no `MSG_UpdateEtc`)',
    estado: 'confirmado',
    nota:
      'Skills aprendidas por personagem (`g_pSpell[24*classe+pos]`, 24 bits). Linha ausente = nenhuma skill aprendida; nunca bit inventado.',
  },
  character_special: {
    dominio: 'personagem',
    fonte:
      '_MSG_ApplyBonus.cpp (`Special[4]` + `SpecialBonus`) + protocolo.md §5.2 (tipo 1)',
    estado: 'confirmado',
    nota:
      'Maestria por atributo e pontos de maestria; `BaseMob` traz tudo zero. Teto 200 (255 com a skill de linha 7/15/23 aprendida) e teto dinâmico `3*(nível+1)>>1` são regra de servidor.',
  },
  account_wallet: {
    dominio: 'economia',
    fonte: 'ProcessDBMessage.cpp `MSG_DBCNFAccountLogin.Coin` + protocolo.md §5.1',
    estado: 'confirmado',
    nota:
      '`Coin` da conta (distinto de `characters.gold`); teto `i32::MAX` porque o campo do fio é `int`. Conta sem linha responde zero.',
  },
  guilds: {
    dominio: 'social',
    fonte: 'modelo-banco.md §2',
    estado: 'confirmado',
    nota: '`master_character_id` DEFERRABLE: cria clã e líder na mesma transação.',
  },
  guild_members: {
    dominio: 'social',
    fonte: 'modelo-banco.md §2 + docs/wyd/sistemas/guerras.md §1',
    estado: 'confirmado',
    nota: 'Um clã por personagem (índice único) e rank 0–9.',
  },
  audit_log: {
    dominio: 'auditoria',
    fonte: 'modelo-banco.md §3.4',
    estado: 'confirmado',
    nota: 'Sem FK de propósito: auditoria não falha por integridade alheia.',
  },
  account_moderation: {
    dominio: 'moderação',
    fonte: 'pre-release-rust.md FASE 7 §7.2',
    estado: 'confirmado',
    nota: 'Bloqueio com motivo + autor; recusado no login do portal e do jogo.',
  },
  delivery_queue: {
    dominio: 'ponte web→jogo',
    fonte: 'modelo-banco.md §3.3',
    estado: 'confirmado',
    nota: 'Única escrita do Cold Path no estado de jogo; o servidor drena.',
  },
  server_channel: {
    dominio: 'canal de servidor',
    fonte: 'ADR-0022 §7 + docs/wyd/canais-e-cidadania.md §5 + tools/client/serverlist-servers.txt',
    estado: 'confirmado',
    nota:
      'Canal 0 = URL de status (≤ 32 chars); 1..10 = host que o cliente disca no 8281 fixo. ' +
      'Coloracao verde/normal e papel do processo, nao da linha: o mesmo host pode ser verde em outro grupo.',
  },
  content_version: {
    dominio: 'catálogo',
    fonte: 'ADR-0009',
    estado: 'confirmado',
    nota: 'Uma linha (`id = 1`); cada seed de conteúdo avança a versão.',
  },
  game_system: {
    dominio: 'catálogo',
    fonte: 'ADR-0009 + docs/wyd/sistemas.md',
    estado: 'confirmado',
    nota: 'Taxonomia de sistemas do jogo (20 linhas no seed).',
  },
  game_class: {
    dominio: 'catálogo',
    fonte: 'ADR-0009 + docs/wyd/classes.md',
    estado: 'confirmado',
    nota: 'Seis classes jogáveis.',
  },
  skill_lineage: {
    dominio: 'catálogo',
    fonte: 'ADR-0009 + docs/wyd/classes.md §2',
    estado: 'confirmado',
    nota: 'Linhagem por classe (13 no seed).',
  },
  item_kind: {
    dominio: 'catálogo',
    fonte: 'docs/wyd/itens.md',
    estado: 'confirmado',
    nota: 'Categoria descritiva do item; 1.532 itens ficam sem categoria por falta de prova.',
  },
  item_effect: {
    dominio: 'catálogo',
    fonte: 'docs/wyd/itens.md + ADR-0009',
    estado: 'provisorio',
    nota: 'O símbolo `EF_*` é a chave; o código numérico do `ItemEffect.h` não foi provado (`confirmed`).',
  },
  item_catalog: {
    dominio: 'catálogo',
    fonte: 'ADR-0009 + modelo-banco.md §6.5',
    estado: 'provisorio',
    nota: '3.220 itens do legado; nome não é chave (488 grupos repetem) e `class_mask` é UNVERIFIED.',
  },
  skill_catalog: {
    dominio: 'catálogo',
    fonte: 'ADR-0009 + docs/wyd/skills.md',
    estado: 'provisorio',
    nota: '154 linhas; 3 têm `skill_index` provisório e isso fica declarado em `notes`.',
  },
  content_source: {
    dominio: 'mundo',
    fonte: 'ADR-0009 + modelo-banco.md §6.7',
    estado: 'confirmado',
    nota: 'Proveniência por insumo (path/sha256/linhas); apagar o arquivo não muda o banco.',
  },
  npc_template: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.7 (STRUCT_MOB 816 B)',
    estado: 'provisorio',
    nota: '20 nomes vieram de CP949 lido como latin1; a chave não é renomeada (ver §6.6).',
  },
  npc_definition: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.7 (NPCGener)',
    estado: 'provisorio',
    nota: '`merchant` acima de 0 e `generator_data` são UNVERIFIED e não viram gate.',
  },
  npc_shop_item: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.7',
    estado: 'provisorio',
    nota: 'Sem FK em `item_index`: 22 itens vendidos não existem no catálogo do legado.',
  },
  level_item: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`LevelItem.txt`)',
    estado: 'confirmado',
    nota: '161 linhas de conteúdo após deduplicar o bloco repetido; `type_id` documentado no insumo.',
  },
  guard_spawn: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`Guard.txt`)',
    estado: 'provisorio',
    nota: 'O bloco de 8 números é preservado cru (o cabeçalho descreve 13 campos).',
  },
  guard_spawn_reward: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`Guard.txt`)',
    estado: 'provisorio',
    nota: '`values` posicional, sem semântica inventada.',
  },
  mob_merc: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`MobMerc.txt`)',
    estado: 'confirmado',
    nota: 'Uma linha no insumo atual.',
  },
  game_setting: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`gameconfig.txt`)',
    estado: 'confirmado',
    nota: '`evstart` tem 4 números: fica em `value_text` e `value_int` só existe com inteiro único.',
  },
  game_matrix: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`gameconfig.txt`)',
    estado: 'provisorio',
    nota: 'Matrizes (`Treasure`, `Item Drop Bonus`) preservadas na ordem do arquivo.',
  },
  region: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.6 (`Regions.txt`)',
    estado: 'confirmado',
    nota: 'Nome repetido é legítimo (Erion/Armia têm retângulos diferentes).',
  },
  world_item_seed: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.1 (`seed_world_item_seed`)',
    estado: 'confirmado',
    nota: 'Itens no chão por mapa (49 linhas).',
  },
  guild_zone: {
    dominio: 'mundo',
    fonte: 'modelo-banco.md §6.1 (`seed_guild_zone`) + guerras.md §2',
    estado: 'confirmado',
    nota: 'Zonas de clã com imposto e desafio (5 linhas).',
  },
  refine_rate: {
    dominio: 'refino',
    fonte: 'docs/wyd/sistemas/refino.md §2',
    estado: 'provisorio',
    nota: '+11 não tem taxa publicada (furo declarado, não chutado).',
  },
  refine_gem_effect: {
    dominio: 'refino',
    fonte: 'docs/wyd/sistemas/refino.md §3',
    estado: 'confirmado',
    nota: 'Quatro gemas (3386–3389), escala 40 × (nível − 9), drop 8% e EXP 2% fixos.',
  },
  sanc_rate: {
    dominio: 'refino',
    fonte: 'origin/main:tmserver/content/Common/Settings/SancRate.txt + Basedef.cpp g_pSancRate',
    estado: 'confirmado',
    nota: '36 células efetivas: as 28 do arquivo (PO 0–5, PL 0–9, Âmago 0–11) e as 8 que o loader mantém do default compilado (PO 6–11, PL 10–11 = 0). O arquivo sobrescreve o default; ausência de linha não é ausência de taxa.',
  },
  combine_value: {
    dominio: 'refino',
    fonte:
      'origin/main:tmserver/content/Common/Settings/CompRate.txt + internal/combine/*.go (g_pOdinRate)',
    estado: 'provisorio',
    nota: 'Taxa/custo por (família, chave) lidos pelos handlers de composição. Onde código e arquivo divergem (Celestial 35 × 5), `notes` diz qual vale — não há número inventado.',
  },
  refine_rate_override: {
    dominio: 'refino',
    fonte: '0037_config_editavel.sql + docs/wyd/sistemas/refino.md §2',
    estado: 'confirmado',
    nota: 'Camada de edição do staff sobre `refine_rate`; o loader lê `COALESCE(override, base)` e reset apaga a linha, sem reescrever o seed.',
  },
  sanc_rate_override: {
    dominio: 'refino',
    fonte: '0037_config_editavel.sql + docs/wyd/sistemas/refino.md §2',
    estado: 'confirmado',
    nota: 'Edição da célula (pó × refino atual) em percentual 0..100; o servidor recalcula em bps ao montar o snapshot.',
  },
  combine_value_override: {
    dominio: 'refino',
    fonte: '0037_config_editavel.sql + CompRate.txt',
    estado: 'provisorio',
    nota: 'Edição de taxa/custo por (família, chave); a semântica de composição ainda é provisória, então a tabela não vira gate.',
  },
  accessory_refine_rule: {
    dominio: 'refino',
    fonte: '0037_config_editavel.sql + 0038_seed_acessorio_refino.sql + guia Bahamut/Kersef',
    estado: 'provisorio',
    nota: 'Sistema posterior ao 7662: passo, taxa e modo de falha por receita; amuleto/ankh/pedra aceitam linha, mas o seed não inventa regra publicada.',
  },
  config_revision: {
    dominio: 'configuração',
    fonte: '0037_config_editavel.sql',
    estado: 'confirmado',
    nota: 'Linha única (`id = 1`); toda escrita de config incrementa `version` na mesma transação e é o sinal de recarga do servidor.',
  },
  config_change: {
    dominio: 'configuração',
    fonte: '0037_config_editavel.sql',
    estado: 'confirmado',
    nota: 'Trilha por mudança: domínio, alvo, valor anterior/novo, autor e motivo; não substitui o audit_log genérico.',
  },
  city: {
    dominio: 'economia',
    fonte: 'docs/wyd/economia.md + decisoes-servidor.md §3',
    estado: 'provisorio',
    nota: '`map_id` nulo: o build 7662 não publica o mapa de cada cidade.',
  },
  item_price: {
    dominio: 'economia',
    fonte: 'docs/wyd/economia.md',
    estado: 'confirmado',
    nota: 'Sobrepõe o preço do catálogo por NPC sem reescrever `item_catalog`.',
  },
  personal_shop: {
    dominio: 'economia',
    fonte: 'docs/wyd/economia.md + decisoes-servidor.md §3',
    estado: 'confirmado',
    nota: 'Imposto de 10% da loja pessoal; `tax_paid` acumula o já recolhido.',
  },
  city_tax_ledger: {
    dominio: 'economia',
    fonte: 'docs/wyd/economia.md',
    estado: 'confirmado',
    nota: 'Livro-caixa por cidade; 40% do acumulado vai para Noatun.',
  },
  donate_shop_item: {
    dominio: 'economia',
    fonte: 'docs/wyd/economia.md',
    estado: 'confirmado',
    nota: 'Só crédito já conciliado; dinheiro real nunca entra aqui.',
  },
  daily_reward_item: {
    dominio: 'economia',
    fonte: 'docs/wyd/economia.md',
    estado: 'confirmado',
    nota: '`day` 1..7 ou NULL para todo dia; item e/ou ouro.',
  },
  character_affects: {
    dominio: 'progressão',
    fonte: 'modelo-banco.md §8 + legado `affect` (savefmt.go offset 4572)',
    estado: 'provisorio',
    nota: 'Semântica de `type`/`time` vem do struct do cliente; a faixa não foi provada (UNVERIFIED).',
  },
  kingdom_cape_price: {
    dominio: 'economia',
    fonte: 'modelo-banco.md §8 + legado `sapphire_balance` (kingdom_cape.go)',
    estado: 'confirmado',
    nota: 'Linha única: preço por reino (CHECK 4..32) e `version` para concorrência otimista.',
  },
  daily_reward_claim: {
    dominio: 'economia',
    fonte: 'modelo-banco.md §8 + legado `daily_reward_claim` (dailyreward.go)',
    estado: 'confirmado',
    nota: 'Um resgate por conta e dia UTC (UNIQUE); o item sai por delivery_queue.',
  },
  guild_relations: {
    dominio: 'social',
    fonte: 'modelo-banco.md §8 + legado `guild_relation` (guild.go)',
    estado: 'confirmado',
    nota: '`kind` 1/2 (aliado/hostil), sem auto-relação.',
  },
  guild_tower: {
    dominio: 'social',
    fonte: 'modelo-banco.md §8 + legado `guild_tower_state` (guild.go)',
    estado: 'confirmado',
    nota: 'Linha única; `owner_guild` 0 = sem dono (por isso sem FK).',
  },
  map_definition: {
    dominio: 'mundo',
    fonte: 'docs/wyd/sistemas/mundo.md',
    estado: 'provisorio',
    nota: 'Lista de mapas do 7662 é UNVERIFIED; a tabela recebe o dado, não o inventa.',
  },
  teleport_route: {
    dominio: 'mundo',
    fonte: 'docs/wyd/sistemas/mundo.md',
    estado: 'provisorio',
    nota: 'Rota sem `content_source` não entra: procedência é obrigatória.',
  },
  mob_template_stat: {
    dominio: 'combate',
    fonte: 'modelo-banco.md §6.1 + docs/wyd/sistemas/drops.md',
    estado: 'provisorio',
    nota: 'Vazia: stats vêm do `STRUCT_MOB` e a extração do struct ainda não foi feita.',
  },
  mob_template_equip: {
    dominio: 'combate',
    fonte: 'modelo-banco.md §6.1',
    estado: 'provisorio',
    nota: 'Vazia pelo mesmo motivo de `mob_template_stat`.',
  },
  drop_rate_override: {
    dominio: 'combate',
    fonte: 'docs/wyd/sistemas/drops.md + decisoes-servidor.md §1',
    estado: 'confirmado',
    nota: 'Drop é PESO comparado ao roll 1..1.000.000, não probabilidade absoluta.',
  },
  content_rate_override: {
    dominio: 'combate',
    fonte: 'docs/wyd/sistemas/drops.md',
    estado: 'confirmado',
    nota: 'Multiplicadores em bps (10000 = 100%).',
  },
  quest: {
    dominio: 'quests',
    fonte: 'docs/wyd/sistemas/quests.md',
    estado: 'provisorio',
    nota: 'Tabela vazia: a cadeia do 7662 não está fechada; seed seria invenção.',
  },
  quest_step: {
    dominio: 'quests',
    fonte: 'docs/wyd/sistemas/quests.md',
    estado: 'provisorio',
    nota: 'Objetivo tipado; posição obrigatória só em `reach`.',
  },
  quest_reward: {
    dominio: 'quests',
    fonte: 'docs/wyd/sistemas/quests.md',
    estado: 'provisorio',
    nota: 'Recompensa por tipo; item exige `item_index`.',
  },
  character_quest: {
    dominio: 'quests',
    fonte: 'docs/wyd/sistemas/quests.md',
    estado: 'confirmado',
    nota: 'Escrita só do servidor; a web nunca toca progresso de personagem.',
  },
  world_event: {
    dominio: 'eventos',
    fonte: 'docs/wyd/sistemas/eventos.md + decisoes-servidor.md §4',
    estado: 'confirmado',
    nota: 'Kefra, Noatun, guerra de cidades e RvR.',
  },
  event_schedule: {
    dominio: 'eventos',
    fonte: 'docs/wyd/decisoes-servidor.md §4',
    estado: 'confirmado',
    nota: '`timezone` explícito para o evento não andar quando o host muda de fuso.',
  },
  kefra_state: {
    dominio: 'eventos',
    fonte: 'docs/wyd/sistemas/eventos.md',
    estado: 'confirmado',
    nota: 'Uma linha (`id = 1`); Kefra semanal.',
  },
  castle_state: {
    dominio: 'eventos',
    fonte: 'docs/wyd/sistemas/guerras.md §3',
    estado: 'confirmado',
    nota: 'Dono do castelo, cidade e cofre; cascata cidade→castelo roda no `SETTLED`.',
  },
  event_participation: {
    dominio: 'eventos',
    fonte: 'docs/wyd/sistemas/eventos.md',
    estado: 'confirmado',
    nota: 'Histórico por evento/personagem/clã.',
  },
  arena: {
    dominio: 'arena',
    fonte: 'docs/wyd/sistemas/battle-royale.md',
    estado: 'confirmado',
    nota: 'Arena Real e Battle Royale (2 equipes por round).',
  },
  arena_match: {
    dominio: 'arena',
    fonte: 'docs/wyd/sistemas/battle-royale.md',
    estado: 'confirmado',
    nota: 'Partida é histórico, não estado vivo.',
  },
  arena_round: {
    dominio: 'arena',
    fonte: 'docs/wyd/sistemas/battle-royale.md',
    estado: 'confirmado',
    nota: 'Round com desempate próprio.',
  },
  arena_score: {
    dominio: 'arena',
    fonte: 'docs/wyd/sistemas/battle-royale.md',
    estado: 'confirmado',
    nota: 'Placar por jogador/round; kills/dano nascem do combate validado no servidor.',
  },
  character_status: {
    dominio: 'progressão',
    fonte: 'docs/wyd/progressao.md §2–§3 (pontos, atributos, fama)',
    estado: 'provisorio',
    nota: 'Nomes de coluna são nossos (a fonte descreve o dado, não o schema); regra 6/2 por nível até 200.',
  },
  character_lineage: {
    dominio: 'progressão',
    fonte: 'docs/wyd/progressao.md §3 (maestria 200/255/320)',
    estado: 'provisorio',
    nota: 'CHECK cobre o maior teto conhecido; o teto por classe é regra do core (D-B05).',
  },
  character_skills: {
    dominio: 'progressão',
    fonte: 'docs/wyd/skills.md §1 + docs/wyd/classes.md §2',
    estado: 'provisorio',
    nota: '`skill_index` sem FK (catálogo parcial) e limite de 25 skills é regra, não CHECK (D-B01).',
  },
};

/** Decisões provisórias — mesmo formato do registro `D-xx` de `matematica-sim.md`. */
export const DECISOES = [
  {
    id: 'D-B01',
    decisao: '`skill_index`/`item_index` referencia o catálogo sem FK?',
    escolha: 'sem FK em `character_skills`, `npc_shop_item`, `level_item`',
    porque:
      'o catálogo cobre 154 dos 248 índices de skill e 22 itens vendidos não existem nele; FK barraria dado legítimo do cliente',
    trace: 'fechar a cobertura do catálogo pelo `SkillData.bin`/`ItemList.bin` do 7662',
  },
  {
    id: 'D-B02',
    decisao: 'Onde mora a camada de progressão (Mortal→Arch→God2)?',
    escolha: '`characters.tier` (CHECK de 6 valores) + `characters.arch_of_id` DEFERRABLE',
    porque:
      'as camadas são CONFIRMADAS em progressao.md §1 e o vínculo Mortal↔Arch é 1:1 opcional; DEFERRABLE permite criar os dois na mesma transação',
    trace: 'trace de transformação Arch no cliente para provar o momento da troca de `tier`',
  },
  {
    id: 'D-B03',
    decisao: 'Refino/Anct/gema como colunas do item',
    escolha: 'colunas na criação de `character_items`/`vault_items` com CHECK de par efeito/valor e de +10-exceto-Anct',
    porque:
      'é estado da linha, não do catálogo (refino.md §2–§3); o CHECK espelha `Item` do core e impede linha inconsistente',
    trace: 'inventário do cliente após refinar/aplicar gema (D-12/D-13 de matematica-sim.md)',
  },
  {
    id: 'D-B04',
    decisao: 'Tabelas `npc_config_meta`, `world_event_config`, `world_event_meta` do legado',
    escolha: 'consolidadas em `content_source` + colunas tipadas de `world_event`/`event_schedule`',
    porque: 'eram metadado de import; `content_source` já guarda path/sha256/linhas e o evento virou tabela com semântica',
    trace: '—',
  },
  {
    id: 'D-B05',
    decisao: 'Tabelas `item_effect_override`/`item_effect_audit` do legado',
    escolha: 'pendentes (não modeladas)',
    porque: '`item_effect.confirmed` cobre o símbolo, não o override por servidor; sem captura do efeito numérico não há coluna honesta',
    trace: 'derivar código EF_* do cliente e então modelar override com auditoria',
  },
  {
    id: 'D-B06',
    decisao: 'Tabela `castle_quest_state` do legado',
    escolha: 'pendente; `castle_state` + `quest` cobrem dono/cofre e quest genérica, não progresso de quest de castelo',
    porque: 'a cadeia de quest do castelo não está fechada em docs/wyd/sistemas/guerras.md §3',
    trace: 'fechar a cadeia e então modelar estado por castelo',
  },
  {
    id: 'D-B07',
    decisao: 'Tabela `mob_carry_override` do legado',
    escolha: 'pendente; `drop_rate_override` cobre peso de drop, não item carregado',
    porque: 'sem o layout de `STRUCT_MOB` extraído, nomear colunas seria invenção',
    trace: 'extrair o struct na FASE-10 e unificar com `mob_template_equip`',
  },
  {
    id: 'D-B08',
    decisao: 'Estado de personagem do legado ainda não modelado (chaos/PK, correio, macro/hotbar)',
    escolha: 'fora do schema até haver fonte',
    porque: 'nenhum documento da árvore atual descreve as regras desses sistemas',
    trace: 'capturar pacote do cliente e documentar antes de criar tabela',
  },
];

/** Mapa tabela legada → destino (fonte: modelo-banco.md §6.1). */
export const LEGADO = [
  ['item', 'absorvido por `item_catalog` (catálogo único por `item_index`)'],
  ['item_catalog', '`item_catalog`'],
  ['item_price', '`item_price`'],
  ['item_effect_override', 'pendente — D-B05'],
  ['item_effect_audit', 'pendente — D-B05 (evento de auditoria vai em `audit_log`)'],
  ['skill_catalog', '`skill_catalog`'],
  ['npc_template', '`npc_template`'],
  ['npc_definition', '`npc_definition`'],
  ['npc_shop_item', '`npc_shop_item`'],
  ['npc_config_meta', 'consolidado em `content_source` — D-B04'],
  ['mob_template_stat', '`mob_template_stat` (vazia até extrair o `STRUCT_MOB`)'],
  ['mob_template_equip', '`mob_template_equip`'],
  ['mob_carry_override', 'pendente — D-B07'],
  ['drop_rate_override', '`drop_rate_override`'],
  ['content_rate_override', '`content_rate_override`'],
  ['teleport_route', '`teleport_route`'],
  ['daily_reward_item', '`daily_reward_item`'],
  ['donate_shop_item', '`donate_shop_item`'],
  ['world_item_seed', '`world_item_seed`'],
  ['guild_zone', '`guild_zone`'],
  ['castle_quest_state', 'pendente — D-B06'],
  ['kefra_state', '`kefra_state`'],
  ['world_event_config', 'consolidado em `world_event` — D-B04'],
  ['world_event_meta', 'consolidado em `world_event` + `event_schedule` — D-B04'],
];
