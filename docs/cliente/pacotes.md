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
| `0x0112` | CNFDeleteCharacter | (a capturar) | **PENDENTE** |
| `0x0114` | CNFCharacterLogin | STRUCT_MOB + cauda | **PENDENTE** |
| `0x0364` | CreateMob | variável (entidades visíveis) | **PENDENTE** |
| `0x7530` | ESCENE_FIELD | cena autoritativa | **PENDENTE** |

Hoje o canal responde `0x010A`, `0x0110`, `0x0112` e `0x0114` **só com o cabeçalho**
(`Size = 12`), com o opcode de sucesso ou o `*Fail` do domínio: o resultado da operação
(criar/remover/selecionar) é decidido no Postgres, o corpo é que continua UNVERIFIED. Quem
desenha a lista de personagens é o cliente, a partir do corpo do `CNFAccountLogin` — por isso
a tela ainda mostra os dados limitados do 7662 sem o blob de `STRUCT_SELCHAR`.

## Fluxo de seleção no canal (`crates/wyd-server`)

| Passo | Pacote | O que o servidor faz |
|---|---|---|
| 1 | `_MSG_AccountLogin` | INITCODE → frame → checksum `Strict` → gate de versão → cerca de sessão → `CNFAccountLogin` (só cabeçalho) |
| 2 | `_MSG_CreateCharacter` | valida slot/classe/nome, exige conta em `accounts`, grava em `characters` → `CNFNewCharacter` ou `NewCharacterFail` |
| 3 | `_MSG_CharacterLogin` | resolve conta+slot, loga `stage=in_game` → `CNFCharacterLogin` ou `CharacterLoginFail` |
| 4 | `_MSG_DeleteCharacter` | exige nome igual ao do slot, soft-delete → `CNFDeleteCharacter` ou `DeleteCharacterFail` |
| 5 | qualquer outro | **tolerado** (`result=tolerated`), sessão continua — antes o primeiro pacote encerrava a conexão |

Só `Size` inválido ou falha de I/O encerram a conexão. A senha de 12 bytes do
`_MSG_DeleteCharacter` não é conferida nesta entrega (pertence à autenticação de conta, que
ainda não existe); a remoção é do dono da sessão e o log grava `password_checked=false`.

## Regras

1. Golden gerado nunca vira prova de comportamento do cliente — só de codec.
2. Divergência entre golden gerado e captura real = o servidor está errado, nunca o contrário.
3. Layout com `UNVERIFIED` não entra em fixture tipada: entra como esquema em texto, sem bytes.
4. Tamanho de `0x010A`/`0x0114`/`0x0364` só é fato depois da captura (o previsto acima é a
   expectativa a confirmar, não o contrato).
