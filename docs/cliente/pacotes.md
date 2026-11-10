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
| `0x0112` | CNFDeleteCharacter | `STRUCT_SELCHAR` (844) | **CORPO PORTADO** (mesmo layout do `CNFNewCharacter`) |
| `0x0116` | CNFCharacterLogout | só header | **PORTADO** (resposta ao `0x0215`) |
| `0x0114` | CNFCharacterLogin | STRUCT_MOB + cauda | **PENDENTE** |
| `0x0364` | CreateMob | variável (entidades visíveis) | **PENDENTE** |
| `0x7530` | ESCENE_FIELD | cena autoritativa | **PENDENTE** |

O canal responde hoje com corpo em `0x010A` (`MSG_DBCNFAccountLogin`, 1996 B), `0x0110`
(`STRUCT_SELCHAR`, 844 B), `0x0112` (o mesmo `STRUCT_SELCHAR` depois da remoção) e `0x0114`
(`MSG_CNFClientCharacterLogin`, 1820 B); os `*Fail` do domínio continuam só de cabeçalho. O
resultado de cada operação é decidido no Postgres e o corpo é o mesmo estado que o banco
serializa — a diferença para a captura ao vivo é de campo, não de fluxo.

## Fluxo de seleção no canal (`crates/wyd-server`)

| Passo | Pacote | O que o servidor faz |
|---|---|---|
| 1 | `_MSG_AccountLogin` | INITCODE → frame → checksum `Strict` → gate de versão → cerca de sessão → `CNFAccountLogin` (só cabeçalho) |
| 2 | `_MSG_CreateCharacter` | valida slot/classe/nome, exige conta em `accounts`, grava em `characters` → `CNFNewCharacter` ou `NewCharacterFail` |
| 3 | `_MSG_CharacterLogin` | resolve conta+slot, loga `stage=in_game` → `CNFCharacterLogin` ou `CharacterLoginFail` |
| 2b | `_MSG_CreateCharacter` com o slot ocupado | aloca o **primeiro slot livre** de 0–3 (nunca sobrescreve); conta cheia → `NewCharacterFail` |
| 4 | `_MSG_DeleteCharacter` | exige nome igual ao do slot, soft-delete → `CNFDeleteCharacter` com o `STRUCT_SELCHAR` do roster restante |
| 4b | `_MSG_CharacterLogout` (`0x0215`) | responde `0x0116`, sai do mundo, salva e libera a sessão **na hora** (fechar a janela não espera timeout) |
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
