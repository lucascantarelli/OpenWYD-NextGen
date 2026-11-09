# Sistemas de jogo — índice

Este arquivo era o documento único de sistemas. Ele foi **dividido por domínio**, porque cada
sistema agora tem passo a passo, taxa e lacuna própria. O conteúdo completo está em:

| Sistema | Documento |
|---|---|
| Refino, Anct, +12..+15, celestial/God2, repleção, imbue | [`sistemas/refino.md`](sistemas/refino.md) |
| Drop, farm por nível, o que cai onde | [`sistemas/drops.md`](sistemas/drops.md) |
| Cidade, Noatun, reinos, torres, PK/chaos | [`sistemas/guerras.md`](sistemas/guerras.md) |
| Cidades, NPCs, teleportes, 52 regiões, dungeons | [`sistemas/mundo.md`](sistemas/mundo.md) |
| Story line, Kefra, Pista de Runas, Arch, destraves | [`sistemas/quests.md`](sistemas/quests.md) |
| Evento de drop, Kefra, invasões, calendário, sazonais | [`sistemas/eventos.md`](sistemas/eventos.md) |
| Moedas, fontes/drenos de ouro, imposto, mercado | [`economia.md`](economia.md) |
| Progressão e camadas | [`progressao.md`](progressao.md) |
| Classes, linhagens e skills | [`classes.md`](classes.md) |
| Senha numérica (segunda senha) do login | [`senha-numerica.md`](senha-numerica.md) |

## O que continua valendo daqui

**Todo sistema tem código em `game_system`** (tabela de referência) e é referenciado por
`item_catalog.primary_system` e `skill_catalog.primary_system`. Os 20 sistemas do catálogo:
`refino`, `replecao`, `mix`, `craft`, `drop`, `loja`, `donate`, `teleporte`, `armazem`,
`combate`, `guilda`, `guerra_guilda`, `castelo`, `reino`, `quest`, `evento`, `mundo`,
`progressao`, `skill`, `montaria`.

O estado de implementação de cada um desses sistemas (✅/🚧/⬜/⛔, com evidência) fica em
[`docs/rust/08-checklist-funcionalidades.md`](../rust/08-checklist-funcionalidades.md).

Regra que não mudou: **`NULL` significa "não sabemos"** — classificação parcial é intencional,
e o portal deve exibir "não classificado" em vez de rótulo inventado.
