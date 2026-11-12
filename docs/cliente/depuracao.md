# Roteiro de trace dinâmico do cliente

> Playbook executável da captura ao vivo (marcos M3–M5 de `engenharia-reversa.md`).
> Status: **roteiro publicado** (CLIENT-06); a captura ao vivo ainda não foi feita — exige
> servidor executável + VM com o jogo, fora deste ambiente. Nada aqui inventa endereço:
> todos os VAs vêm de `docs/cliente/analise-estatica.md`.

## 1. Preparação do ambiente (uma vez)

1. Conferir o build: `sha256sum client/WYD.exe` = `04c9d2a1…27ddf8`. Hash diferente = parar.
2. **Backup** de `client/` inteiro para fora do repositório. Pasta não é reconstruível.
3. Máquina descartável (VM): x64dbg **32-bit** + ScyllaHide, Process Monitor, Wireshark,
   API Monitor. Nunca depurar na máquina de trabalho — XTrap/JProtect reagem a debugger.
4. Rede: servidor de teste acessível em `tcp/8281` + `http/80` (status `serv00.htm`).
5. Conta de teste dedicada **sem nenhum dado pessoal** (ver §5).

## 2. Breakpoints (endereços confirmados na CLIENT-05)

| VA | Instrução | Para quê | O que registrar |
|---|---|---|---|
| `0x4252D6` | `jge 0x4253cc` | todo pacote **recebido** passa por aqui | `Type` (u16 @pacote+4), `Size` (u16 @pacote+0), `KeyWord` (u8 @pacote+2) |
| `0x40126A` | `push ecx` | caminho de **envio** | `Type` de cada pacote C→S |
| `0x54DA23` | `push ebp` (entrada) | função de envio original | pilha de chamada do envio |
| `0x53AC6A/0x53AD52/0x53AE7E` | `jmp` (já `EB`) | somente leitura — **não mexer** | confirmar que continuam `EB` |
| candidatos `0x2000` | ver `gerado/evidencia.json` (`cpsock.sizeLimit.candidates`) | checagem de enquadramento | qual deles valida `Size` do frame |

Comandos base do x64dbg: `bp <va>` (software), `bphws <va>` (hardware, sobrevive a escrita
em `.text`), `r` (registradores), `db/d <end>` (dump), `t`/`p` (passo), `r esp`,
`db esp` (argumentos na pilha). Trace com condicional nos tipos de interesse
(`0x010A`, `0x0364`, `0x7530`) para não afogar o log.

## 3. Fases da captura (login → campo)

| # | Fase | Gatilho | O que conferir |
|---|---|---|---|
| 0 | Conexão | TCP para `:8281` | primeiros 4 bytes = `11 F3 11 1F` (INITCODE little-endian) |
| 1 | AccountLogin C→S | breakpoint em `0x40126A` | `Type=0x020D`, `Size=116`, `KeyWord` anotado |
| 2 | CNFAccountLogin S→C | breakpoint em `0x4252D6` | `Type=0x010A`, **`Size` exato do frame** (previsto 2008) |
| 3 | Tela de seleção | — | anotar horário; captura de `serverlist.bin`/`sn.bin` no Process Monitor |
| 4 | CharacterLogin C→S | `0x40126A` | `Type=0x0213`, `Size=20` |
| 5 | Entrada no mundo | `0x4252D6` | burst de `0x0364` (CreateMob) e `0x7530` (ESCENE_FIELD) |
| 6 | Ação de movimento C→S | `0x40126A` | `Type=0x036C`, `Size=52`, coordenadas na rota |

Wireshark (na máquina do jogo): filtro `tcp.port == 8281`; exportar **apenas** os bytes TCP
do stream jogo (sem credencial — ver §5).

## 4. Reprodução do freeze (investigação aberta do legado)

Sintoma herdado: jogo congela minutos após o login, TCP continua com ACK, cliente descarta
tudo (hipóteses registradas em `docs/cliente/pacotes.md` — frame S→C malformado, fila de
keyword do cliente, NPC com visual inválido).

Protocolo de reprodução:

1. Controle: ficar só em Armia por 5 min. Anotar se congela.
2. Teleporte para `/reino` {1702,1728}: andar 2–3 passos, anotar.
3. Teleporte para `/gelo` {3650,3128} e Erion (2456,2007): mesma anotação.
4. Rajada de `ApplyBonus` (~1 clique/130 ms, 30–60 s): o legado travou após 262 cliques.
5. Quit com `0x03AE` + `0x0215` (logout-cancelado): observar relogin em conexão nova.
6. Em todo freeze: anotar **hora exata do congelamento** (não do kill) e colher o tail de
   pacotes recebidos no breakpoint `0x4252D6` + bytes crus do Wireshark.

Critério de fechamento: tail com `Size` anômalo para o `Type` (encoder culpado) ou stream
íntegro com o cliente descartando (hipótese 2/3 → investigar `RecvQueue`/NPC da área).

## 5. Higiene das fixtures (obrigatório)

1. Toda captura real contém **conta e senha**: antes de versionar, substituir por
   `GOLDEN`/zeros e marcar `sanitized: true` + método no JSON.
2. `crates/wyd-protocol/tests/fixtures/cliente/<fase>-<opcode>-<n>.json` segue o esquema de `captura.mjs`
   (`payloadHex`, `wireHex`, `checksum`, `header`, `fields`).
3. Validar com `node tools/client/captura.mjs --check`; falha = fixture inválida, não gate quebrado.
4. Nunca versionar `.pcap`, dumps de memória ou logs do WYD.

## 6. O que falta para a captura ao vivo

- ~~Servidor executável falando o dialeto 7662~~ — o canal Rust já sobe (`wyd-server`, `:8281`)
  com login (senha + senha numérica) e seleção de personagem.
- VM Windows 32-bit com D3D9 funcional e o jogo instalável.
- Executor humano para jogar o roteiro do §3–§4.
- Instrumentos prontos: proxy e conversão em `tools/client/captura-live.mjs`, diff em
  `tools/client/diff-captura.mjs`. Roteiro completo, com a variação de campo a campo, em
  [`captura-selchar.md`](captura-selchar.md).
