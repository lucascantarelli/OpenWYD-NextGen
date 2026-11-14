# Sistema — Guildas, guerra e castelo

> **Paridade: P1 em guildas (schema + leitura), P0 em guerra/castelo.** É o maior bloco de
> conhecimento do legado ainda não portado.

## 1. Escopo

Criação, convite, aceite/recusa, exclusão, gerenciamento, hierarquia, cargos, permissões,
comunicação, membros, liderança, regras, conflitos, eventos, guerras, benefícios, persistência.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_InviteGuild.cpp`, `_MSG_GuildAlly.cpp`, `_MSG_War.cpp`, `CCastleZakum.cpp`, `CWarTower.cpp` |
| Cliente | `0x03D5` InviteGuild, `0x0E12` GuildAlly, `0x0E0E` War |
| Rust | ausente (só leitura de guilda no roster) |
| Banco | `0004_guildas` (`guild_members`), leitura em `selchar.rs` |
| Testes | nenhum de handler |
| Doc | `docs/wyd/sistemas/guerras.md`, `docs/arquitetura/contrato-dispatch-opcode.md` §4 |

## 3. O que existe

| Item | Estado |
|---|---|
| Tabela de guilda e membros | ✅ |
| Guilda exibida no roster (`STRUCT_SELCHAR`) | ✅ |
| Criar guilda | ❌ |
| Convidar/aceitar | ❌ |
| Cargos/permissões | ❌ |
| Chat de guilda | ❌ (chat geral também ausente) |
| Guerra/cerco/torre | ❌ |
| Aliados | ❌ |
| Persistência de guerra | ❌ |

## 4. Regras históricas (referência, nível B/C)

| Regra | Fonte | Observação |
|---|---|---|
| Criação por comando `/criar`, 100kk, 3 sub-líderes, medalhas | guia Ongame (Wayback) | número de época |
| Guerra de Noatun a cada 2 semanas, 21h–22h | WYD Global | configuração de servidor |
| Castelo em duas zonas (cidade × guerra), Torre de Thor | WYD Global | mecânica estável |
| Soberano da cidade vira Rei | WYD Global | depende de cidadania |

**Mecânica** é referência válida; **preço/horário** não é regra do nosso servidor.

## 5. Lacunas e ações

1. CRUD de guilda com permissões (líder/sub-líder/membro) e auditoria.
2. Convite/aceite com estado (pendente/aceito/recusado) e expiração.
3. Chat de guilda depende do chat geral (`PAR-SOC-01`).
4. Guerra: só depois de cidadania e colisão/movimento estáveis.
5. Aliados e benefícios: modelar como dados, não como `if`.

## 6. Riscos

| Risco | Nota |
|---|---|
| Guilda sem permissão | qualquer membro expulsa líder |
| Guerra sem agenda | ciclo impossível de testar |
| Chat acoplado | mensagem para membro offline sem fila |
| Preço copiado | economia distorcida |
