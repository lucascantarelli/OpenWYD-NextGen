# Análise estática — build 7662

> **Data:** 2026-11-07 · **Alvo:** `client/WYD.exe` (2.347.008 B)
> **SHA-256:** `04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8`
> **Comando:** `node tools/client/analyse.mjs`
> Artefatos brutos: `docs/cliente/gerado/{pe,strings,evidencia,formatos}.json` + `ancoras.txt`.
> Toda classificação abaixo é **CONFIRMADO** ou **REFUTADO** com evidência lida do binário.

## 1. Método e ferramentas efetivamente disponíveis

| Ferramenta | Situação | Uso |
|---|---|---|
| Node 26 | disponível | parser PE próprio, strings, evidência, formatos |
| `objdump` (mingw64) | disponível | desassemble de janelas por endereço |
| `xxd`/`od`/`strings` | disponível | conferência bruta |
| Python | **ausente** | — |
| Ghidra / IDA / rabin2 | **ausentes** | sem decompilação e sem xrefs completos |

Consequência declarada: o pipeline confirma **endereços e instruções**, não reconstrói funções
inteiras. Decompilação e xrefs completos ficam para 08-6, quando houver ferramenta adequada.

## 2. Cabeçalho PE (confirmado)

| Campo | Valor |
|---|---|
| Formato | **PE32 / i386**, image base `0x400000` |
| Entry point | `0x5f3c66` (RVA `0x1f3c66`) |
| `SizeOfImage` | 37.257.216 B |
| `TimeDateStamp` | **2015-07-02T05:09:43Z** |
| Seções | `.text` 2.043.904 B · `.rdata` 110.592 B · `.data` 163.840 B · `.rsrc` 24.576 B |
| Diretórios não vazios | export, import, resource, import address table |
| Imports | **18 DLLs, 268 funções** |
| Export | DLL `WYDBR.exe`, 1 nome: `fcEXP` |

DLLs importadas: `ADVAPI32`, `d3d9`, `DINPUT8`, `DSOUND`, `GDI32`, `IMM32`, `iphlpapi`,
`KERNEL32`, `MSVFW32`, `ole32`, `OLEAUT32`, `SHELL32`, `SHLWAPI`, `USER32`, `WININET`, `WINMM`,
`WS2_32`, `WSOCK32`.

Leitura: render por **Direct3D 9**, input por DirectInput 8, áudio por DirectSound, rede por
Winsock (`WS2_32`/`WSOCK32`) e HTTP próprio por `WININET`. Não há import de `.NET` nem sinal de
packer.

## 3. Âncoras herdadas — todas CONFIRMADO

Cada linha foi confirmada por duas provas: bytes no endereço e o endereço aparecer como **início
de instrução** no desassemble.

| Símbolo herdado | VA | Instrução no endereço | Entrada de função? | Status |
|---|---|---|---|---|
| `ReadMessage` | `0x4252d6` | `jge 0x4253cc` | não | **CONFIRMADO** |
| `SendChat` | `0x4676c5` | `jne 0x467740` | não | **CONFIRMADO** |
| `SendPack` | `0x40126a` | `push ecx` | não | **CONFIRMADO** |
| `SendPacket` | `0x54da23` | `push ebp` | **sim** | **CONFIRMADO** |
| `AddVolatileMessageItem` | `0x422007` | `cmp [ebp+0xc], 0x15e2` | não | **CONFIRMADO** |
| `SetVolatileMessageBoxText` | `0x41fb30` | `cmp [ebp-0x1ac], 0xd3` | não | **CONFIRMADO** |

**Achado que corrige a documentação herdada:** cinco das seis âncoras **não são entradas de
função**; são instruções internas que o `ClientPatch` reescreve (o hook mantém a condição e
desvia para o trampolim). Só `SendPacket` (`0x54da23`) é prólogo real:
`push ebp; mov ebp,esp; sub esp,0x8`. Portanto “endereço da função `ReadMessage`” era impreciso:
o correto é “ponto de hook dentro da rotina de recebimento”.

## 4. Desativação de checksum — os três sites já estão patchados no disco

| VA | Bytes | Instrução | Status |
|---|---|---|---|
| `0x53ac6a` | `eb 04 33 c0 eb 35` | `jmp 0x53ac70` | **CONFIRMADO** (já contém `EB`) |
| `0x53ad52` | `eb 07 33 c0 e9 5e 01 00` | `jmp 0x53ad5b` | **CONFIRMADO** (já contém `EB`) |
| `0x53ae7e` | `eb 04 33 c0 eb 35` | `jmp 0x53ae84` | **CONFIRMADO** (já contém `EB`) |

Isso é mais forte do que o esperado: o build distribuído **já vem com as três checagens
desativadas** (`jz` → `jmp`), sem depender de injeção em runtime. Consequência para o servidor
Rust: manter checksum **não rejeitante** é obrigatório para este cliente; rejeitar quebraria o
build de referência.

## 5. CPSock — o que ficou comprovado e o que não

| Item | Evidência no binário | Status |
|---|---|---|
| Tabela `pKeyWord[512]` | bloco de 512 bytes byte-idêntico ao gerado (`SHA-256 e47996fe5e92de5d…`) localizado em `.data` | **CONFIRMADO** |
| `INITCODE = 0x1F11F311` | constante `11 f3 11 1f` presente em `.data` | **CONFIRMADO (dado)** |
| Referência de código ao INITCODE | padrões `push imm32` / `mov reg, imm32` **não localizados** nesta passada | **NÃO LOCALIZADO** |
| Header de 12 B (`Size/KeyWord/CheckSum/Type/ID/Tick`) | tabela embutida + INITCODE presente + limite 8192 | **CONFIRMADO INDIRETO** |
| Offsets de campo do header por instrução | obtidos por instrução na 08-6: envio de `GetItem` grava `Type` em +4 (`0x4fee36`) e `ID` em +6 (`0x4fee32`), com o resto do header completado por `SendPacket` (`0x54da23`, recebe o tamanho) | **CONFIRMADO** para os opcodes de caixa/pickup (ver `pacotes.md`); demais `Type`s seguem sem leitura |

Candidatos a checagem de tamanho (`cmp reg, 0x2000`) foram registrados em
`gerado/evidencia.json` (`cpsock.sizeLimit.candidates`) para o trace dinâmico usar como breakpoint.

## 6. Strings relevantes (495 indexadas)

| String | VA | Seção | Leitura |
|---|---|---|---|
| `./serverlist.bin` | 6433080 | `.data` | confirma que o caminho do serverlist é relativo à pasta do jogo |
| `.\ItemList.bin` | 6431376 | `.data` | catálogo de itens lido do disco |
| `CheckSum Error` / `CheckSum Error : %d,%d …` | 6379716 / 6402932 | `.data` | mensagens de checksum existem, mas os sites estão desativados |
| `XTrapVa.dll` / `.\XTrap\%s` | 6492488 / 6492612 | `.data` | carregamento do anticheat pelo próprio executável |
| `sn.bin` | 6435012 | `.data` | lista de nomes de canal lida do disco |
| `#define ITEM_CHECKSUM %d` | 6431476 | `.data` | resíduo de geração do `ItemList.bin` |

## 7. Limitações desta passada

1. Sem decompilador: não há assinatura de funções além das âncoras, nem xrefs completos.
2. O header de 12 B é confirmado por **duas evidências indiretas**, não por leitura de campo;
   a semântica de `Type`/`ID`/`Tick` no código segue UNVERIFIED.
3. `objdump` só desassembleia janelas; para cada âncora o pipeline tenta vários alinhamentos
   até o VA aparecer como início de instrução (determinístico e registrado no artefato).
4. Nenhuma execução do jogo foi feita nesta etapa — nada aqui vem de runtime.

## 8. Reprodução

```bash
node tools/client/analyse.mjs      # exit 1 se o SHA-256 não for o do build 7662
sha256sum client/WYD.exe           # 04c9d2a1…27ddf8
```

Duas execuções consecutivas produzem artefatos **byte-idênticos** (verificado por `sha256sum`).
