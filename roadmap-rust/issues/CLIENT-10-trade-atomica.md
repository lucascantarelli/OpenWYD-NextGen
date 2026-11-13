# CLIENT-10 — Trade atômica por par (layout candidato; captura pendente)

> **ID canônico:** `CLIENT-10`
> **ID histórico:** `FASE-08-10`
> **Estado 2026-11-12:** ⛔ · layout candidato de 156 B identificado; falta o frame real do 7662.
> Ver `roadmap-rust/index.md`.

**Fase:** 8 · **Depende de:** CLIENT-06 🚧 (captura do layout) · **Bloqueia:** — · **Esforço:** M · **Estado:** ⛔ bloqueada em layout

## Objetivo

Implementar a troca jogador↔jogador sem dup, com confirmação bilateral e swap
atômico de itens + gold — SOMENTE após a captura do `MSG_Trade` real do 7662.

## Contexto / evidência

Pesquisa `docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` item 7 (26 campos).
Fluxo do legado (`flows.md` §5): `TradingItem` (oferta) → `Trade` (estado
espelhado) → ambos `MyCheck==1` → swap ATÔMICO (`Item[]`+gold) → `Trade(done)`;
`QuitTrade (0x0384)` cancela ambos (`RemoveMob`/ambos). Opcodes no catálogo
(`Trade/QuitTrade/TradingItem`, `opcode_table.rs`).

**Bloqueio:** `MSG_Trade` ainda não tem frame capturado do cliente 7662. É o
ponto desta issue em que a hipótese estática e o porte Go divergem; codar
contra qualquer um dos dois sem a captura é proibido (invariante 5 do
`AGENTS.md`; `require_ported` → `Unported`).

### Layout candidato (investigação de 2026-11-12)

O fonte legado em `Basedef.h:2435-2445` declara:

```c
struct MSG_Trade {
    _MSG          Header;       // 12 B
    STRUCT_ITEM   Item[15];     // 15 × 8 B
    char          InvenPos[15];
    int           TradeMoney;
    unsigned char MyCheck;
    unsigned short OpponentID;
};
```

`MSG_Trade` **não** está dentro de um bloco `#pragma pack`; o
`TMSrv.vcxproj` não declara `/Zp1` nem `StructMemberAlignment`, e
`protocol-spec.md` §3.5 fixa alinhamento natural do MSVC x86 como regra do
projeto. O probe C++ local (mesmas declarações, `g++`) deu:

| Offset | Campo | Tamanho |
|---:|---|---:|
| 0 | `_MSG Header` | 12 |
| 12 | `STRUCT_ITEM Item[15]` | 120 |
| 132 | `char InvenPos[15]` | 15 |
| 147 | padding | 1 |
| 148 | `int TradeMoney` | 4 |
| 152 | `unsigned char MyCheck` | 1 |
| 153 | padding | 1 |
| 154 | `unsigned short OpponentID` | 2 |
| **156** | **total do frame** | **156** |

Corpo útil = **144 B** (156 − 12). `InvenPos` é `char` (posição `-1` vira
`0xFF` no fio); `Item[i]` é o `STRUCT_ITEM` de 8 B (`sIndex` + 3 pares
efeito/valor).

O handler `_MSG_Trade.cpp` envia e compara com `sizeof(MSG_Trade)` — o fio do
servidor é o tamanho que o compilador gerou, não um buffer empacotado à mão.
Com o `.vcxproj` sem `/Zp1`, isso aponta para 156 B. A pesquisa da fase 2 fala
em “rede = `pack(1)`” como regra geral, mas o `protocol-spec` §3.5 diz “salvo
indicação, sem pack” e `MSG_Trade` não tem indicação; a contradição é
justamente o que a captura resolve.

O porte Go (`tmserver/internal/protocol/messages.go`) usa
`MsgTradeBodySize = 142` (frame 154 B, offsets `132/147/151/152`), ou seja,
assume `pack(1)`. Isso **conflita** com o C++ e não tem teste de layout; não é
prova. A captura decide qual dos dois o cliente 7662 envia.

Ferramenta de apoio já criada: `tools/client/trade-layout.mjs`.

```bash
node tools/client/trade-layout.mjs --self-test
node tools/client/trade-layout.mjs <hex-do-frame-decodificado>
```

Ela recusa `Size` diferente de 156/154, identifica o layout, lê os campos e
aponta `InvenPos` repetido, ouro fora da faixa, `MyCheck` inválido e
`OpponentID` fora de `1..999`. O frame precisa estar **desofuscado** (Type
claro), como no ponto `0x4252D6` depois do decode.

## Passos

1. Capturar o `MSG_Trade` real (CLIENT-06): dois clientes de teste, um item
   conhecido e `TradeMoney=1234`; breakpoint `0x4252D6`, `Type=0x0383`.
   Rodar `trade-layout.mjs` no frame:
   - `layout: natural` → confirmado **156 B**; offsets da tabela acima;
   - `layout: packed` → confirmado **154 B**; offsets Go;
   - qualquer outro `Size` → parar e registrar o frame bruto; não adaptar
     código por tentativa.
   Anexar o JSON da ferramenta à issue e só então promover o layout.
2. Tipar o payload `Trade` com os offsets capturados; manter
   `require_ported(Opcode::Trade)` fechado até o golden passar. Handler
   atômico por par (uma transação: dois inventários + gold), `QuitTrade`
   cancela ambos.
3. Revalidar slots/ouro na confirmação; rollback em desconexão; idempotência
   por oferta; `OpponentID` validado server-side.
4. Testes de contenção: dois swaps no mesmo slot, replay, desconexão no meio.

## Arquivos

- Criados: `tools/client/trade-layout.mjs` (verificador da captura; não é
  codec de produção) e `crates/wyd-server/tests/trade.rs` (quando o layout
  existir).
- Alterados: `crates/wyd-protocol/src/payload/commerce.rs` (só com captura),
  `crates/wyd-server/src/{cpsock,characters}.rs`, checklist 5.10.
- Proibido: tipar `Trade` sem captura; tocar a cerca de sessão única; copiar
  o codec Go de 154 B como se fosse verdade.

## Testes / verificação

```bash
node tools/client/trade-layout.mjs --self-test
cargo test -p wyd-server --test trade
```

Saída esperada: o self-test decodifica 156 e 154 sem ambiguidade; o teste de
integração prova confirmação bilateral debita/credita exatamente uma vez;
cancelamento não move nada; concorrência não duplica.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net — pacote e sessão.
- .agents/skills/wyd-rust-db — transação por par.
- .agents/skills/wyd-rust-testing — testes de contenção.
- .agents/skills/wyd-client-reverso — captura do layout.
- .agents/skills/rust-code-quality — codec e erros tipados.
- .agents/skills/rust-test-quality — testes de race e golden.

## Documentação de referencia

- `docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` (item 7).
- `docs/migration/flows.md` §5.
- `docs/arquitetura/protocolo.md` §6 (UNVERIFIED).
- `docs/cliente/depuracao.md` §7 (captura de `MSG_Trade`).

## Subtasks

- [x] Investigar layout no fonte C++ e no projeto legado (156 × 154).
- [x] Criar verificador de captura (`trade-layout.mjs --self-test`).
- [ ] Capturar `MSG_Trade` real e anexar o JSON da ferramenta.
- [ ] Promover o codec `Trade` com os offsets confirmados.
- [ ] Implementar swap atômico + `QuitTrade` + rollback.
- [ ] Testar contenção, replay e desconexão.

## Critérios de aceite

- [ ] Layout capturado e documentado (não herdado do `Basedef.h`); `Size`
      156 ou 154 decidido por frame real.
- [ ] Swap atômico auditado; nenhum dup sob contenção.
- [ ] Desconexão reverte sem perda nem duplicata.

## Riscos / rollback

- Risco: dup econômico (o pior do jogo) → mitigação: transação por par +
  idempotência + testes de race antes de qualquer merge.
- Risco: promover o layout errado (156 × 154) e corromper itens/ouro →
  mitigação: `trade-layout.mjs` recusa tamanho desconhecido e o golden da
  captura é pré-requisito do codec.
- Rollback: remover o handler; `Unported` volta a recusar.
