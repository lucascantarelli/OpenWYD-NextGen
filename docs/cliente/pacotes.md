# Pacotes do caminho login → campo

> Tabela viva do contrato C↔S com o estado de cada pacote. `GERADO` = golden sintético
> (`fixtures/cliente/*.json`) validado por `captura.mjs`. `PENDENTE` = só fechado com a
> captura ao vivo (roteiro em `depuracao.md`). Tamanhos e fluxos vêm de
> `protocol-spec.md` §3.5 + `flows.md`; o crate usa os mesmos valores (`payload.rs`).

## C→S (gerados)

| Opcode | Nome | Tamanho | Fixture | Status |
|---|---|---:|---|---|
| `0x020D` | AccountLogin | 116 | `account_login.json` | **GERADO** |
| `0x020F` | CreateCharacter | 36 | `create_character.json` | **GERADO** |
| `0x0213` | CharacterLogin | 20 | `character_login.json` | **GERADO** |
| `0x036C` | Action | 52 | `action.json` | **GERADO** |
| `0x0333` | (framing) | 12 | `header_only.json` | **GERADO** (só header) |

## S→C (aguardando captura)

| Opcode | Nome | Tamanho esperado | Status |
|---|---|---:|---|
| `0x010A` | CNFAccountLogin | 2008 | **PENDENTE** (conferir `Size` no wire) |
| `0x0110` | CNFNewCharacter | STRUCT_SELCHAR | **PENDENTE** |
| `0x0114` | CNFCharacterLogin | STRUCT_MOB + cauda | **PENDENTE** |
| `0x0364` | CreateMob | variável (entidades visíveis) | **PENDENTE** |
| `0x7530` | ESCENE_FIELD | cena autoritativa | **PENDENTE** |

## Regras

1. Golden gerado nunca vira prova de comportamento do cliente — só de codec.
2. Divergência entre golden gerado e captura real = o servidor está errado, nunca o contrário.
3. Layout com `UNVERIFIED` não entra em fixture tipada: entra como esquema em texto, sem bytes.
4. Tamanho de `0x010A`/`0x0114`/`0x0364` só é fato depois da captura (o previsto acima é a
   expectativa a confirmar, não o contrato).
