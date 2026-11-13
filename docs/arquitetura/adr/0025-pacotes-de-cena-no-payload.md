# ADR-0025 — Pacotes de cena em `payload/scene.rs`

- **Estado:** aceita
- **Data:** 2026-11-10
- **Decisores:** dono do projeto + agente
- **Governa:** `crates/wyd-protocol` (árvore §3 de `estrutura-e-padroes.md`), FASE-03/CLIENT-06

## Contexto

A árvore alvo de `payload/` (ADR-0002, `estrutura-e-padroes.md` §3) fechou em sete domínios
(`auth`, `character`, `movement`, `combat`, `inventory`, `commerce`, `ui`). Ficou sem lugar o
pacote que faz o cliente **ver** uma entidade: `_MSG_CreateMob` (0x0364) e `_MSG_RemoveMob`
(0x0165). Encaixá-los em `movement.rs` (movimento/ação do jogador) ou em `ui.rs` (painel/clima)
misturaria domínios e quebraria a regra "um arquivo, um domínio".

O layout está provado nas duas fontes que o repositório já usa para os demais S→C
(`protocolo.md` §5.1):

- `Basedef.h`, `struct MSG_CreateMob` — `_MSG` 12 B + `PosX/PosY`, `MobID`, `MobName[16]`,
  `Equip[16]`, `Affect[32]`, `Guild`, `GuildMemberType`, `Unknow[3]`, `STRUCT_SCORE`,
  `CreateType`, `AnctCode[16]`, `Tab[26]`, `Hold`; `MSG_RemoveMob` = `_MSG` + `int RemoveType`;
- porte Go do mesmo cliente 7662 (`tmserver/internal/protocol/createmob.go`,
  `createMobSize = 232` / `removeMobSize = 16`), que fixa os offsets em teste
  (`createmob_test.go`).

Isto **não** é captura ao vivo (CLIENT-06): o documento `docs/cliente/pacotes.md` mantém a
distinção entre "corpo portado" e "captura pendente".

## Decisão

Criar `crates/wyd-protocol/src/payload/scene.rs` com `CreateMob` (232 B) e `RemoveMob` (16 B),
com offsets explícitos no `wire_payload!`, entrando em `declared_len` (`require_ported`).

Regras que o arquivo carrega:

- `CreateMob.HEADER.ID` = `ESCENE_FIELD` (30000), id da criatura em `MobID` (+16) — invariante 4
  do `AGENTS.md`;
- `RemoveMob.HEADER.ID` = id da criatura, corpo `RemoveType` (0 fora de vista, 1 morte, 2
  logout);
- nenhum campo inventado: o que não tem fonte (`Affect` de NPC, `AnctCode`, `Tab`, `Hold`) vai
  zerado, e o teste fixa os offsets em vez de confiar em `repr(Rust)`.

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| `movement.rs` | O domínio é entidade na vista, não movimento do jogador; a regra "um domínio por arquivo" cairia |
| `ui.rs` | Cena não é interface; `MessagePanel`/`Weather` não compartilham nada com mob |
| `wire/scene.rs` | `wire/` é enquadramento/opcode, não payload; quebraria a fronteira do crate |
| Deixar fora até a captura | `MSG_CreateMob` tem layout provado por `Basedef.h` + porte Go, ao contrário de `MSG_Trade`/`MSG_SetHpMp`, que seguem UNVERIFIED e fora de `payload/` |

## Consequências

- **Positivas:** o servidor pode emitir a cena sem tipo improvisado; `declared_len` cobre o
  opcode e o pipeline de decodificação reconhece o pacote; a árvore ganha um domínio real, com
  ADR, em vez de um arquivo "misc".
- **Negativas:** oitavo arquivo em `payload/` (custo de manutenção trivial).
- **Pendente:** captura ao vivo do 7662 recebendo `0x0364`/`0x0165` (CLIENT-06) — se ela
  divergir, o porte é que está errado, nunca o cliente.
