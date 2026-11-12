# Contrato de porte — transporte CPSock (`wyd-net`)

> **Data:** 2026-11-08 · **Fonte de verdade:** código legado em `origin/main`
> (`Source/Code/CPSock.{h,cpp}`, `Source/Code/Basedef.h`, `Source/Code/TMSrv/Server.cpp`) e a
> reimplementação anterior em `tmserver/internal/protocol/{header,framing,keytable,checksum,codec}.go`.
> **Autoridade:** o **C++** — é o que o cliente 7662 fala. O Go confirma o layout e é usado como
> segunda evidência; onde divergem, o documento diz qual vale e por quê (§9).
> **Alvo:** `client/WYD.exe` 2.347.008 B, SHA-256 `04c9d2a1…27ddf8` (PE32/i386).
> **Natureza:** contrato documentado (documento do tipo **Reference** — informação, não tutorial:
> `.agents/skills/documentation-writer/SKILL.md`). Nenhuma linha de Rust foi escrita nesta passada.
> Toda afirmação tem `arquivo:linha`; o que não foi provado está em **UNVERIFIED** com a pergunta
> que fecha a lacuna (§11).
>
> ## Normas do projeto que regem este contrato
>
> | Norma | O que ela já fixa (citado, não redefinido aqui) |
> |---|---|
> | `AGENTS.md` §3 invariante 2 | HEADER 12 B LE, `Size` em [12, 8192], `INITCODE = 0x1F11F311`, transformação a partir do offset 4, `repr(C)` provado por teste; **entrada de rede em política `Strict`**, `Tolerant` só em captura |
> | `AGENTS.md` §3 invariante 4 | pacote autoritativo de cena vai com `HEADER.ID = ESCENE_FIELD` (30000) |
> | `AGENTS.md` §3 invariante 1 | nada de I/O no tick; persistência por portas e canais async |
> | `AGENTS.md` §7 | estrutura antes de código; arquivo novo fora da árvore exige ADR (rito PROC-21 em `docs/arquitetura/adr/`) |
> | `.agents/skills/wyd-rust-net/SKILL.md` | regras 1–4 (validar antes de parsear, sem `panic` em input de rede, `MAX_PACKET_BYTES` como teto explícito, `ESCENE_FIELD`, rate-limit desde o handshake) e a árvore obrigatória `session/`, `transport/`, `dispatch/`, `limits/`, `telemetry/` |
> | `docs/arquitetura/estrutura-e-padroes.md` §11 (ADR-0002) | árvore arquivo-a-arquivo do `wyd-net` (`session/{state,handshake}`, `transport/{tcp,framing}`, `dispatch/router`, `error.rs`) |
> | `docs/arquitetura/protocolo.md` §1–§3 | contrato de fio vigente do HEADER, do enquadramento e da ofuscação |
>
> **O que este documento acrescenta (medição nova, não coberta pelas normas acima):** o algoritmo
> de leitura do legado passo a passo (`CPSock.cpp:360-467`), o caminho de envio (`:513-591`), o
> aceite por socket→slot (`Server.cpp:4374-4420`) e a ausência de timeout de inatividade. Onde a
> norma e a medição divergem, prevalece a norma e a divergência está nomeada em §9/§10.

## 1. Constantes de transporte (medidas no legado)

| Constante | Valor | Fonte |
|---|---:|---|
| `GAME_PORT` | **8281** | `Source/Code/Basedef.h:104` |
| `DB_PORT` | 7514 | `Source/Code/Basedef.h:105` (usado em `TMSrv/Server.cpp:4012`, `:4017`) |
| `ADMIN_PORT` | 8895 | `Source/Code/Basedef.h:106` |
| `MAX_MESSAGE_SIZE` | **8192** | `Source/Code/CPSock.h:38` |
| `RECV_BUFFER_SIZE` / `SEND_BUFFER_SIZE` | 128 KiB cada | `Source/Code/CPSock.h:35`, `:36` |
| `MAX_PENDING_CONNECTS` | 8 (backlog do `listen`) | `Source/Code/CPSock.h:31`, usado em `CPSock.cpp:155` |
| `INITCODE` | **0x1F11F311** (wire LE: `11 F3 11 1F`) | `Source/Code/CPSock.h:40`; gate em `CPSock.cpp:373` |
| `APP_VERSION` | **7640** (versão que o **servidor C++ do legado** exigia no login; no nosso servidor o gate usa `PROTOCOL_VERSION` = 12 000 — ADR-0020) | `Source/Code/Basedef.h:102` |
| `SKIPCHECKTICK` | 235543242 (tick reservado; pacote com ele é descartado) | `Source/Code/Basedef.h:232` |
| `MAX_USER` | **1000** (slots de `pUser[]`; NPCs/mobs começam acima disso) | `Source/Code/Basedef.h:116` |
| `ADMIN_RESERV` | 10 (slots reservados a admin) | `Source/Code/Basedef.h:117` |
| início da transformação | offset **4** (bytes 0..3 em claro) | `CPSock.cpp:430` (`for i = 4`), `:558` |
| tamanho do HEADER | **12 bytes** | `CPSock.h:42-50`; espelho `Basedef.h:1205-1210` |

Espelho Go que confirma cada valor: `tmserver/internal/protocol/header.go:19` (`HeaderSize`=12),
`:23` (`InitCode`), `:26` (`MaxMessageSize`=8192), `:31` (`AppVersion`=7640), `:35`
(`SkipCheckTick`), `:39` (`MaxUser`=1000), `:43` (`transformStart`=4).

## 2. HEADER de 12 bytes (little-endian, sem padding)

Definição C++ `typedef struct _HEADER` em `Source/Code/CPSock.h:42-50`; a macro `_MSG`
(`Basedef.h:1205-1210`) repete exatamente o mesmo prefixo em **todas** as structs de mensagem:

| Offset | Campo | Tipo C++ | Bytes | Observação |
|---:|---|---|---:|---|
| 0 | `Size` | `short` | 2 | tamanho **total** do pacote, incluindo o header |
| 2 | `KeyWord` | `char` | 1 | índice 0..255 usado para semear a transformação |
| 3 | `CheckSum` | `char` | 1 | `Sum2 − Sum1` (uint8, truncado) |
| 4 | `Type` | `short` | 2 | opcode com as flags de direção (`FLAG_*`) |
| 6 | `ID` | `short` | 2 | slot de conexão/jogador em `pUser[]` |
| 8 | `ClientTick` | `unsigned int` | 4 | tick do cliente |

Regras que decorrem do layout:

1. **Todos os campos são little-endian** (x86). O Go lê/escreve por offset explícito
   (`header.go:67-90`, `binary.LittleEndian`) exatamente para não depender do layout de struct.
2. **`char` do MSVC é assinado**, mas os bytes de `KeyWord`/`CheckSum` são tratados como
   `unsigned char` na aritmética (`CPSock.cpp:391-393`, `:536-537`) — no Rust são `u8`.
3. **Não há campo de versão nem de tamanho de payload**: `payload = wire[12..Size]`.
4. O `Type` inclui os bits de direção: `FLAG_GAME2CLIENT 0x0100`, `FLAG_CLIENT2GAME 0x0200`,
   `FLAG_DB2GAME 0x0400`, `FLAG_GAME2DB 0x0800`, `FLAG_DB2NP 0x1000`, `FLAG_NP2DB 0x2000`,
   `FLAG_NEW 0x4000` (`Basedef.h:1212-1221`).

## 3. Handshake: 4 bytes de INITCODE antes de qualquer frame

- O cliente abre o TCP e envia **primeiro** 4 bytes `INITCODE`; o servidor só começa a fazer
  framing depois de consumi-los (`CPSock.cpp:366-383`: se `Init == 0`, exige ≥4 bytes, compara
  com `INITCODE`, e em divergência devolve `ErrorCode = 2` com `ErrorType = <os 4 bytes lidos>`).
- O servidor **também envia** `INITCODE` — mas apenas nas conexões que **ele** abre para o
  DBSrv/BISrv (`CPSock.cpp:249-250`, `ConnectServer`/`ConnectBillServer`). No caminho de aceite
  o sentido é o inverso.
- Go: mesma regra, com diagnóstico dos primeiros 64 bytes em erro
  (`framing.go:12` `ErrBadInitCode`, `:64-79`).

Estado medido da flag: `Init = 0` só aparece no **construtor** (`CPSock.cpp:65`) e `Init = 1` é
posto depois do INITCODE válido (`:381`) ou no lado que conecta (`:252`). `StartListen` **não**
toca `Init` (`CPSock.cpp:128-173`).

**UNVERIFIED (U1):** *o caminho de aceite constrói um `CPSock` novo por conexão (rodando o
construtor em `:65`) ou reaproveita o objeto do slot?* Se reaproveitar, um slot reciclado fica
com `Init = 1` e o próximo cliente **pula o INITCODE**. Fecha com: o trecho de `WSA_ACCEPT` em
`TMSrv/Server.cpp` (que cria/atribui `pUser[User].cSock`) + captura com duas conexões no mesmo
slot.

## 4. Framing: como um fluxo de bytes vira frames

Algoritmo do legado, passo a passo (`CPSock.cpp:360-467`):

1. Acumula bytes no buffer de recepção (`Receive`, flush em `RefreshRecvBuffer`,
   `CPSock.cpp:593-600`).
2. Se `Init == 0`: exige ≥ 4 bytes e valida `INITCODE` (`:366-383`).
3. Se restam menos de `sizeof(HEADER)` = 12 bytes, **retorna e espera** (`:386-387`).
4. Lê `Size` (u16 @0), `KeyWord` (u8 @2), `CheckSum` (u8 @3) (`:390-393`).
5. **Valida `Size` antes de qualquer outra coisa**: `Size > 8192 || Size < 12` → zera
   `nRecvPosition`/`nProcPosition`, `ErrorCode = 2`, `ErrorType = Size` (`:397-406`).
6. Se o frame completo ainda não chegou, espera (`:408-411`).
7. Avança `nProcPosition += Size` e, se o buffer acabou, zera os dois cursores (`:413-422`).
8. Decodifica (transform + checksum, §5) e **devolve o pacote mesmo com checksum divergente**
   (`:457-464`).

Equivalente Go: `Framer.ReadFrame` (`framing.go:64-88`) com as mesmas duas checagens
(`ErrBadSize` = `:12`, faixa em `:86-88`). O Go **não** zera o buffer em `Size` inválido — ele
devolve erro e o chamador fecha a conexão; o efeito no fio é o mesmo (frame descartado).

## 5. Transformação de keyword e checksum (contrato byte-exato)

Tabela estática `pKeyWord[512]` em `Source/Code/CPSock.cpp:29-46` (comentário "7.xx keys");
bytes de índice **par** são "sementes KeyWord", de índice **ímpar** são "Trans"
(`keytable.go:3-13`; a cópia Go em `keytable.go:14-31` é byte-idêntica).

**Decodificação (cliente → servidor)** — `CPSock.cpp:424-455`:

```
pos = pKeyWord[iKeyWord * 2]          // iKeyWord = byte @2 do frame
Sum1 = Sum2 = 0
para i de 4 até Size-1:
    rst = pos % 256
    Sum2 += wire[i]                                  // soma do cifrado
    Trans = pKeyWord[rst * 2 + 1]
    switch i & 3:
        0: plain = wire[i] - (Trans << 1)
        1: plain = wire[i] + (Trans >> 3)
        2: plain = wire[i] - (Trans << 2)
        3: plain = wire[i] + (Trans >> 5)
    Sum1 += plain                                    // soma do plano
    pos += 1
CheckSum_calculado = Sum2 - Sum1                     // uint8, wrap
```

**Codificação (servidor → cliente)** — `CPSock.cpp:535-584`, exatamente o inverso
(`+ (Trans<<1)`, `- (Trans>>3)`, `+ (Trans<<2)`, `- (Trans>>5)`), com `iKeyWord = rand() % 256`
(`:535`) gravado no header (`:539`).

Consequências obrigatórias:

1. **A transformação cobre o header a partir do offset 4** — logo `Type`, `ID` e `ClientTick`
   também vão cifrados; um pacote só de header passa pelo transform.
2. **`KeyWord` no header é o índice `iKeyWord`**, e a semente real é **`pKeyWord[iKeyWord*2]`**
   (`:392`); trocar um pelo outro quebra a paridade em 100% dos pacotes.
3. **`CheckSum` é `Sum2 − Sum1` truncado em u8** (`:455`, `:583-584`).
4. **Checksum divergente: o legado tolerava; a stack nova não.** Medição: o C++ entrega o pacote
   com `ErrorCode = 1` (`CPSock.cpp:457-464`) e o Go preserva o mesmo comportamento
   (`checksum.go:11-17`, `codec.go:41-60`). **Norma do projeto:** entrada de rede usa
   [`Strict`] — divergência é erro tipado (`AGENTS.md` §3 invariante 2, `protocolo.md` §3,
   `crates/wyd-protocol/src/codec/checksum.rs:13-19`, `Strict` como `Default`); `Tolerant` fica
   restrito a captura/replay/diagnóstico. Divergência consciente em relação ao legado: **D-T3**
   (§10). Limite conhecido e já normatizado: o checksum é função da chave, não do conteúdo
   (`protocolo.md` §3), logo ele não é gate anti-adulteração — é detector de corrupção/emissor
divergente.
5. `Size` válido é **[12, 8192]** — e não o 4096 do comentário velho do CPSock: ver **D-T1 (resolvido)** em §10.

## 6. Aceite e identificação da sessão

| Passo | C++ | Evidência |
|---|---|---|
| Socket de escuta | `socket`/`bind`/`listen(8)` + `WSAAsyncSelect(FD_ACCEPT)` | `CPSock.cpp:128-173` |
| Porta de jogo | `ListenSocket.StartListen(hWndMain, *pip, GAME_PORT, WSA_ACCEPT)` | `TMSrv/Server.cpp:4089` |
| Aceite → slot | `int User = GetUserFromSocket(wParam)`; `User == 0` ⇒ `closesocket` | `TMSrv/Server.cpp:4376-4382`; impl. `GetUserFromSocket` em `:7512` |
| Evento que não é leitura | `WSAGETSELECTEVENT(lParam) != FD_READ` ⇒ `CloseUser(User)` | `TMSrv/Server.cpp:4384-4391` |
| Leitura | `pUser[User].cSock.Receive()`; falha ⇒ **uma** retentativa; falhou de novo ⇒ `CloseUser` | `TMSrv/Server.cpp:4393-4411` |
| Erro de socket | `-1` ⇒ `recv` de descarte + `CloseUser` | `TMSrv/Server.cpp:4412-4420` |
| Slot do jogador | `CUser pUser[MAX_USER]` com `MAX_USER = 1000` | `TMSrv/Server.h:70`; `Basedef.h:116` |

Regras:

1. **A identidade da conexão é o socket → slot** (`GetUserFromSocket`), não o `HEADER.ID`.
   `ID` é o slot declarado pelo cliente; o Go documenta que ele deve cair em `[0, MaxUser)`
   (`header.go:37-39`), mas **o C++ não valida `ID` contra o socket** nesta varredura
   (**UNVERIFIED U2**: *o dispatcher confia em `HEADER.ID` ou usa `std` do socket?* — em
   `ProcessClientMessage.cpp:66` o roteamento é por `std->Type`; o `std` vem da sessão).
2. **Nenhum timeout de inatividade existe no legado**: `LastSendTime` é declarado
   (`Server.cpp:589`, `Server.h:267`) e **não tem nenhum leitor** (varredura de `TMSrv/*.cpp`
   e `TMSrv/*.h`); o Go não chama `SetDeadline` em `world/server.go` nem em `cmd/tmserver/`.
   O único "keepalive" é o opcode `MsgPing 0x03A0`, explicitamente **sem efeito ao receber**
   (`protocol/types.go:129`).
3. **Escrita não é imediata**: pacotes entram no buffer (`AddMessage`, `CPSock.cpp:513-591`) e o
   flush acontece no laço de tick para quem tem bytes pendentes
   (`ProcessSecMinTimer.cpp:201-207`), na mesma passagem que salva jogadores (`:171-178`).

## 7. Caminho de envio (servidor → cliente)

`CPSock::AddMessage` (`CPSock.cpp:513-591`):

1. Recusa se o buffer de envio estourar (`:518-524`) ou se o socket for inválido (`:527-533`).
2. Sorteia `iKeyWord = rand() % 256` e grava `Size`, `KeyWord`, `CheckSum = 0`,
   `ClientTick = CurrentTime` (`:535-542`).
3. Cifra `[4, Size)` no buffer de envio, somando plano (`Sum1`) e cifrado (`Sum2`).
4. `CheckSum = Sum2 - Sum1` e copia os 4 primeiros bytes **em claro** (`:583-586`).
5. O envio real é `SendMessageA()` (`:617`), chamado no tick.

Contrato de porte: o `ClientTick` do servidor é o relógio do servidor (`CurrentTime`), não o do
cliente — não confundir com o `ClientTick` recebido (que é preservado apenas como dado).

## 8. Paridade mínima exigida do `wyd-net`

| Requisito | Verificação |
|---|---|
| Gate de 4 bytes `INITCODE` por conexão | teste com os 4 bytes errados ⇒ conexão encerrada sem frame |
| Header 12 B LE por offset explícito | round-trip de header com `Size`/`KeyWord`/`CheckSum`/`Type`/`ID`/`Tick` |
| `Size ∈ [12, 8192]` antes de alocar | teste de `Size = 11`, `= 13`, `= 8192`, `= 8193` |
| Transformação cobre `[4, Size)` | 256 `KeyWord` × payload de 1..64 B ⇒ byte-idêntico ao C++ |
| `KeyWord` do header = `iKeyWord`; semente = `pKeyWord[iKeyWord*2]` | golden com `iKeyWord` fixo |
| `CheckSum = Sum2 − Sum1` (u8) | idem |
| Divergência de checksum **não** rejeita | pacote com checksum corrompido deve chegar ao dispatcher + métrica |
| Slot = socket, `ID` não é identidade | teste de cliente que declara `ID` de outro slot |
| Flush no tick | pacote pendente não sai antes do tick seguinte |

## 9. Divergências C++ × Go (e a autoridade)

| # | Tema | C++ | Go | Autoridade e por quê |
|---:|---|---|---|---|
| 1 | Checagem de `Size` inválido | zera os cursores e segue (`CPSock.cpp:397-406`) | devolve `ErrBadSize` (`framing.go:86-88`) | **C++**: define o comportamento observável do servidor de referência (descarta e continua a sessão) |
| 2 | Checksum divergente | entrega o pacote com `ErrorCode=1` (`:457-464`) | devolve `mismatch` e quem chama decide (`checksum.go:11-17`, `codec.go:59-60`) | **norma do projeto** (`Strict` na entrada de rede: `AGENTS.md` §3 invariante 2, `protocolo.md` §3, `codec/checksum.rs:13-19`); o tolerante do C++ é medição e só sobrevive como `Tolerant` em captura — D-T3 |
| 3 | Leitura de `Type`/`ID` | lê **4 bytes** em @4 e **4 bytes** em @6 (`:394-395`) | `u16` em @4 e `u16` em @6 (`header.go:56-63`) | **Go/struct**: os `unsigned int` do C++ são leitura sobreposta que só se usa como diagnóstico; o roteamento usa `std->Type` (u16) |
| 4 | Semente da transformação | `rand()%256` no envio (`:535`) | `iKeyWord` injetado (`codec.go:14-15`) | **C++** para produção; o parâmetro do Go é só testabilidade |
| 5 | Excesso de bytes antes do INITCODE | buffer reiniciado só em `Size` inválido | erro de leitura | **C++**: descarte sem derrubar a conexão |
| 6 | Constantes (12/8192/INITCODE/1000) | origem | iguais, com citação ao C++ | **ambos** — sem divergência |
| 6b | Versão de cliente (7640 C++ × 12 000 medido) | ADR-0020 | 7640 é histórico; o gate compara igualdade com `PROTOCOL_VERSION` | **decidido** — divergência consciente, não defeito |

## 10. Destino Rust e defeitos já visíveis

Árvore normativa do crate: `crates/wyd-net/` = `session/`, `transport/`, `dispatch/`, `limits/`,
`telemetry/`, `error.rs` (`docs/arquitetura/visao-pastas.md` §2, ADR-0002). Hoje existem apenas
`lib.rs` (34 linhas), `limits/{mod,rate}.rs` (226) e `metrics.rs` — **não há framing nem sessão**.

| Peça do contrato | Módulo destino |
|---|---|
| gate de 4 bytes INITCODE + rate-limit do handshake | `session/handshake.rs` (árvore §11: "INITCODE + rate-limit") |
| corte por `Size` ∈ [12, 8192] e espera do frame completo | `transport/framing.rs` |
| máquina de estado `Connected → Handshaked → Authenticated` | `session/state.rs` (typestate, árvore §11) |
| transformação `[4, Size)` + checksum | **já existe** em `wyd-protocol::codec::{keyword,checksum}` — o `wyd-net` não duplica codec (skill `wyd-rust-net`, seção Estrutura) |
| vetor de slots `MAX_USER = 1000` e regra `ID` × socket | a árvore §11 não tem arquivo para isso: cabe em `session/state.rs`; **arquivo novo exigiria ADR** (`AGENTS.md` §7) |
| baldes e porte já existentes | `limits/rate.rs` (`TokenBucket`, `ConnectionGate`) |
| erros tipados da fronteira (tamanho → opcode → payload) | `error.rs` (`NetError`, árvore §11) |
| telemetria sem `println!` | `telemetry/metrics.rs` (árvore §11; hoje o crate tem `src/metrics.rs` — mover é tarefa da FASE-06, não deste contrato) |
| roteamento `Type → intenção` | `dispatch/router.rs` (`match` exaustivo — `contrato-dispatch-opcode.md`) |

**D-T1 — resolvido em 2026-11-08.** A fonte única do teto é
`crates/wyd-protocol/src/wire/header.rs:22` (`MAX_MESSAGE_SIZE = 8192`, de `CPSock.h:38`); o
`wyd-net` não repete o número: `MAX_PACKET_BYTES = wyd_protocol::MAX_MESSAGE_SIZE` e
`MIN_PACKET_BYTES = wyd_protocol::MIN_MESSAGE_SIZE` (12) em `crates/wyd-net/src/lib.rs:18-21`, com
uma única checagem de faixa (`accepts_len`, `lib.rs:28`). Antes o crate duplicava **4096** e
recusava frames legítimos entre 4097 e 8192 que o `wyd-protocol` já aceitava (`wire/frame.rs:60`);
a origem do valor errado era o comentário velho `// Maximum size a single message can have, 4K`
ao lado do **8192** em `CPSock.h:38`.

Prova por execução (`cargo test -p wyd-net packet_range_matches_the_cpsock_contract`): `8192`
aceito, `8193` recusado, `12` aceito, `11` recusado, e as constantes conferidas (`8192`/`12`) —
mexer no limite quebra o teste. O `wyd-protocol` cobre o mesmo contrato em `wire/frame.rs`
(`validate_size(8192)` ok, `8193` erro) e em `payload/mod.rs` (lista de tamanhos inclui 8192).
Nenhum buffer do crate é dimensionado por literal: não há alocação por tamanho de pacote em
`wyd-net` (o recv/send de 128 KiB do legado é buffer de socket, folgado acima de 8192).

**D-T3 (divergência de política, decidida):** checksum divergente é erro tipado na entrada de
rede (`Strict`), não métrica silenciosa — o `wyd-protocol` já implementa a decisão
(`codec/checksum.rs:13-19`, `Default = Strict`); o legado tolerante só sobrevive em `Tolerant`.

**Status de D-T1/D-T3:** D-T1 **fechado** (código + teste de fronteira acima); D-T3 **decidido e
implementado** (`Strict` é o default do `wyd-protocol`). Nenhum dos dois é UNVERIFIED.

**D-T2 — resolvido em 2026-11-08 (ADR-0020).** `AppVersion` do legado é **7640**
(`Basedef.h:102`), mas o nosso cliente oficial (`client/`) foi medido enviando **12000** no login
(`docs/cliente/superficie-conhecida.md:20`). A decisão está escrita: 7640 é **valor histórico** do
build de referência, `APP_VERSION` não é comparado com nada, e o gate (`VersionGate`, igualdade
com `PROTOCOL_VERSION` = 12 000) vive em `crates/wyd-net/src/session/handshake.rs`. Ver
`docs/arquitetura/adr/0020-versao-de-cliente-no-login.md`. O antigo "U3" fica fechado como decisão;
segue pendente só a captura de confirmação em runtime (CLIENT-06).

## 11. Lacunas UNVERIFIED e a pergunta que fecha cada uma

| # | Lacuna | Pergunta exata | Como fecha |
|---:|---|---|---|
| U1 | reset de `Init` no aceite | *um slot reciclado pode pular o INITCODE?* | ler `CUser`/`CloseUser` + captura com 2 conexões no mesmo slot |
| U2 | `ID` × socket | *o servidor usa `HEADER.ID` para algo além de diagnóstico?* | xrefs de `std->ID` em `ProcessClientMessage.cpp` |
| U3 | versão do cliente | **resolvida** — o cliente oficial mede 12000 (`superficie-conhecida.md:20`); ADR-0020 decide `APP_VERSION` (7640) como histórico e `PROTOCOL_VERSION = 12_000` (`wyd-protocol/src/lib.rs:40`) como valor de gate | `docs/arquitetura/adr/0020-versao-de-cliente-no-login.md`; falta só a captura de confirmação em CLIENT-06 |
| U4 | teto real do cliente | *o cliente já enviou frame > 8192 B?* (o teto do legado é 8192; acima disso o próprio CPSock cortaria) | captura de `CreateMob`/`ESCENE_FIELD` + contagem de `Size` em CLIENT-06 |
| U5 | timeout de inatividade | *qual timeout o servidor de referência praticava?* | evidência em `WYD.log`/produção; hoje não há código |

## 12. Reprodução da evidência

```bash
rm -rf /tmp/legacy && mkdir -p /tmp/legacy
git archive origin/main Source tmserver | tar -x -C /tmp/legacy
grep -n "INITCODE\|MAX_MESSAGE_SIZE\|typedef struct _HEADER" /tmp/legacy/Source/Code/CPSock.h
awk 'NR>=360 && NR<=467' /tmp/legacy/Source/Code/CPSock.cpp        # leitura + transform + checksum
awk 'NR>=513 && NR<=591' /tmp/legacy/Source/Code/CPSock.cpp        # envio
grep -n "StartListen\|WSA_READ\|GetUserFromSocket" /tmp/legacy/Source/Code/TMSrv/Server.cpp
grep -rn "LastSendTime" /tmp/legacy/Source/Code/TMSrv/            # 1 declaração, 0 leitores
cat /tmp/legacy/tmserver/internal/protocol/{header,framing,checksum}.go
```
