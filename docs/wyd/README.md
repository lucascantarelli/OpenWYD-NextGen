# Manual de WYD (Kersef) — índice

Documentação **de domínio** do jogo: o que cada sistema é, como funciona passo a passo, o que
o jogador precisa e o que o servidor precisa implementar. Não é código: é o contrato que o
código tem de cumprir.

## Como este manual está organizado

| Documento | Responde |
|---|---|
| `lore.md` | história completa: criação, cinco deuses, guerra celestial, queda de Armia, Arch, reinos, cronologia |
| `classes.md` | 4 classes, 12 linhagens, skills 9ª a 12ª por linhagem, regras de Arch |
| `skills.md` | mecânica de pontos e maestria, custos, reset, o que o servidor valida |
| `progressao.md` | Mortal → God → Arch → Subcelestial → God2 → Nightmare; pontos por nível; **prêmios de nível reais** |
| `itens.md` | grades N/M/A/Le/Anct, **uso de cada consumível**, armas com stats, sets por classe |
| `sistemas/refino.md` | +0 a +15, Anct, celestial, God2, repleção, imbue — com taxa e falha |
| `sistemas/drops.md` | algoritmo do drop, configuração, onde farmar por nível, o que cai onde |
| `sistemas/guerras.md` | cidade, Noatun, reinos, torres, PK/chaos, máquina de estado |
| `sistemas/mundo.md` | cidades e NPCs, teleportes, **as 52 regiões**, dungeons, zonas |
| `sistemas/quests.md` | story line, Kefra, **Pista de Runas sala por sala**, Arch, destraves |
| `sistemas/eventos.md` | evento de drop, Kefra, invasões, calendário, sazonais |
| `canais-e-cidadania.md` | **servidor × canal**, cidadania (Kibita, bônus, `/getout`), canal verde/novato e guildas donas de cidade por canal |
| `fontes.md` | **de onde vem cada afirmação** e onde cada site engana |

## Confiança

| Marca | Significado |
|---|---|
| **CONFIRMADO** | duas fontes independentes, ou fonte oficial + evidência no repositório |
| **UNVERIFIED** | uma fonte só ou texto ambíguo — **nunca** vira regra de código (invariante 5 do `AGENTS.md`) |
| **E** | dado do próprio repositório (seed/migração/cliente 7662) — prevalece sobre qualquer site |

Detalhamento dos níveis A–E em `fontes.md`.

## Estado do dado no banco (2026-11-08)

Aplicado e verificado em Postgres (`0007`–`0012`):

| Conteúdo | Linhas |
|---|---:|
| `item_catalog` | 3.220 |
| `item_effect` (`EF_*`) | 55 símbolos |
| `skill_catalog` | 151 |
| `npc_template` (STRUCT_MOB 816 B) | 1.991 |
| `npc_definition` | 6.100 |
| `npc_shop_item` | 6.706 |
| `level_item` | 161 |
| `guard_spawn` / `guard_spawn_reward` | 5 / 27 |
| `game_setting` / `game_matrix` | 25 / 20 |
| `region` | 52 |
| `world_item_seed` | 49 |
| `guild_zone` | 5 |
| `content_source` | 12 |

Números que este manual usa e que **só existem** no nosso dado: prêmios de nível por classe
(`progressao.md`), as 52 regiões (`sistemas/mundo.md`), `statsapphire = 30`, `maxNightmare = 3`,
`GTorreHour = 22`, `RVRHour = 23`, `BRHour = 19`, `partydif = 200`.

## Regra de ouro

1. **Nada aqui autoriza ler arquivo em runtime.** A fonte canônica é o PostgreSQL (ADR-0009).
2. **Mecânica é estável; número não é.** Taxa de servidor privado não vira regra.
3. **Lacuna é informação.** Onde não há prova, o manual diz "UNVERIFIED" em vez de inventar —
   e o portal deve exibir "não classificado" em vez de rótulo falso.

## O que este manual **não** cobre (de propósito)

- **Cliente** (formato de `.dat`, pacotes, renderização): `docs/cliente/`.
- **Protocolo** (HEADER, checksum, catálogo de opcode): `docs/arquitetura/protocolo.md`.
- **Banco** (DDL, CHECKs, proveniência): `docs/arquitetura/modelo-banco.md`.
- **Portal** (BFF, telas): `wyd-portal/README.md`.

## Próximos aprofundamentos

| Tema | O que falta |
|---|---|
| Fórmula de dano | ordem de aplicação (defesa, absorção, crítico, perfuração) — FASE-10 |
| Curva de EXP | não publicada; exige extração do binário |
| Taxa de drop por item | idem |
| Texto de diálogo | não publicado em nenhuma fonte |
| Catálogo de 151 skills × classe | mapeamento por nome confirmado (12 feitos) |
