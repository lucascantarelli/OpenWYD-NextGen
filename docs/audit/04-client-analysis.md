# 04 — Auditoria do cliente oficial (`client/`)

> Build de referência: **7662**, `WYD.exe` 2.347.008 B,
> `SHA-256 04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8`,
> `TimeDateStamp` 2015-07-02, PE32/i386, entry `0x5f3c66`, 18 DLLs / 268 imports.
> Pasta **não versionada** (~344 MB). Medições de inventário desta auditoria: 2026-11-13.
> Base documental: `docs/cliente/*` (CLIENT-04/05/06).

## 1. Identidade e proveniência

| Item | Valor | Confiança |
|---|---|---|
| Versão de protocolo enviada no login | **12000** | Confirmado (medido no handshake) |
| Build/patch | **7662** (“Cavaleiros de Kersef”) | Confirmado pelo nome do `ClientPatch_v7662` |
| Patches de runtime | `ClientPatch.dll` (41.984 B, **GPL-3.0**, © Victor Klafke/Charles TheHouse) | Confirmado |
| Proteções | `JProtectDll.dll` (45.056 B), `XTrap/` (`XTrap.xt` 1.856.536 B, `XTrapVa.dll` 2.765.800 B), `jlock101/102.sys`+`.VXD` | Confirmado (presença); eficácia **Desconhecido** |
| Runtime | `msvcp140.dll` **e** `msvcp140d.dll` (debug) | Confirmado — resíduo de build |
| Checksums de integridade | 3 sites já em `EB` **no disco** (`0x53AC6A`, `0x53AD52`, `0x53AE7E`) | Confirmado |
| `Graphics.ini` / `config_settings.json` | **ausentes** (o Global os tem) | Confirmado (`ls`) |
| `Config.bin` | 30 B; modo `0x07` (800×600×32) | Confirmado o byte; semântica do resto **Hipótese** |
| `WYD.log` | `VertexShader HW Accel Enabled` + **`Error in Init Render Target Texture`** | Confirmado (log local) |

**Leitura de proveniência (Fortemente indicado):** a pasta é um **repack de comunidade**, não uma
instalação retail da Ongame — combina patch GPL de terceiro, anticheat XTrap, DLL de proteção
jlock e runtime *debug*. Isso explica por que “o cliente atual parece diferente do que eu lembro”.

## 2. Arquitetura gráfica — é 3D com skin, não 2D

| Evidência | Contagem em `client/` | Significado |
|---|---:|---|
| `Mesh/*.msa` | **1.707** | malhas (estáticas/efeito), FVF `0x112`/`0x142` |
| `Mesh/*.msh` | **903** | malhas **skinned** (header com `palette`/`influences`) |
| `Mesh/*.bon` | **81** | tabelas de ossos (registros de 8 B) |
| `Mesh/*.ani` | **776** | animações |
| `Mesh/*.wys` | **1.865** | texturas de malha (família DDS, `WS10` — **UNVERIFIED** localmente) |
| `shader/*.bin` | 18 | bytecode D3D9 SM 1.1: `pseffect1-6`, `vseffect1-4`, `skinmesh1-8` |
| `shader1..8.bin` (raiz) | 8 | shaders por efeito |
| `Env/FieldXXXX.{dat,trn}` | 522 arquivos em `Env/` | terreno **por campo**, não heightmap global |
| `Effect/` | 468 | partículas/efeitos (`.msa` + `.wys`) |
| `UI/*.wyt` / `NUI/*.wyt` | 342 / 8 | telas raster (`WT10`) |

**Conclusão (Confirmado por inventário + formato):** o cliente 7662 tem pipeline de malha com
esqueleto, animação e *pixel/vertex shaders*. A premissa “o cliente atual é uma versão 2D mais
simples” é **Divergente** com a evidência. Ver análise completa em
[`19-client-visual-generations.md`](19-client-visual-generations.md).

## 3. Tabelas de dados — aritmética dos tamanhos

> O layout não está provado para nenhuma destas linhas; o que está provado é o **tamanho** e a
> **divisibilidade**. Serve para levantar hipótese falsificável, nunca para escrever parser.

| Arquivo | Bytes | Aritmética | Hipótese | Confiança |
|---|---:|---|---|---|
| `ItemList.bin` | 910.004 | `6.500 × 140 + 4` | catálogo de itens, registro de 140 B + trailer | Fortemente indicado (bate com o `.bin` de servidor do w2pp) |
| `Itemname.bin` | 215.084 | `3.163 × 68` | nomes de item, 68 B cada | Fortemente indicado |
| `SkillData.bin` | 23.812 | `248 × 96 + 4` | 248 skills, 96 B + trailer | Fortemente indicado |
| `InitItem.bin` | 768 | `96 × 8` | itens iniciais por classe | Hipótese |
| `itemicon.bin` | 26.000 | `6.500 × 4` | índice de ícone por item | Fortemente indicado |
| `Mixlist.bin` | 272.000 | `6.800 × 40` **ou** `8.500 × 32` | receitas de mix | Desconhecido (divisor ambíguo) |
| `object.bin` | 524.292 | `524.288 + 4` | **máscara de meshes** (bitmask), **não** lista de objetos de mapa | Fortemente indicado (mesmo tamanho do v769 auditado) |
| `cdata.bin` | 8.192 | `2.048 × 4` | tabela de checksum do terreno | Fortemente indicado |
| `curse.bin` | 8.192 | `256 × 32` | tabela de maldições | Hipótese |
| `extraitem.bin` | 106 | — | no Global tem 110.390 B; aqui é **stub** | Divergente |
| `minimap.dat` | 4.058 | — | texto/coordenadas (no v769 é texto CP1252) | Hipótese |
| `serverlist.bin` | 7.040 | `10 × 11 × 64` | grade de servidores, XOR próprio | **Confirmado** (`formatos.md` §4) |
| `sn.bin` | 143 | `99 + 11×4` | 11 nomes de 9 B + 11 contadores | **Confirmado** (`formatos.md` §3) |

**Achado de geração:** o `ItemList.bin` de 140 B/registro e o `SkillData.bin` de 96 B/registro
são **menores** que os do cliente 7.69+ (164 B e 104 B). Fortemente indicado que o 7662 usa um
*schema* anterior — coerente com “o build é de 2015”.

## 4. Matriz de funcionalidades do cliente

### 4.1 Capacidade e comunicação

| ID | Sistema | Funcionalidade | Arquivo / localização | Componente | Comunicação | Status |
|---|---|---|---|---|---|---|
| CLI-01 | Entrada | Lista de servidores/canais | `serverlist.bin`, `sn.bin`, `NUI/ServerList.wyt` | rotina `0x46d6ce`–`0x46dc76`; **inversão** de nomes no último grupo | `GET http://IP:80/serv00.htm` + TCP `IP:8281` | Confirmado |
| CLI-02 | Entrada | Handshake + login de conta | `WYD.exe` | header 12 B, `INITCODE`, tabela 512 B | envia `0x020D` (116 B) com ClientVersion 12000 | Confirmado |
| CLI-03 | Entrada | Senha numérica (2ª senha) | `WYD.exe` | `AccountSecure` | `0x0FDE`; recusa `0x0FDF` | Confirmado no servidor; semântica do `change_numeric` **Hipótese** |
| CLI-04 | Personagem | Lista de personagens | `WYD.exe` | copia **840 B** de `frame+0x10` | recebe `0x0110` (corpo 844 B, sel@4) | Confirmado |
| CLI-05 | Personagem | Deleção | `WYD.exe` | mesmo `memcpy` 840 B | recebe `0x0112` em frame de **852 B** | Confirmado |
| CLI-06 | Personagem | Entrada no mundo | `WYD.exe` | `CNFCharacterLogin` | envia `0x0213` (cliente 7662: **36 B**, não 20) | Confirmado |
| CLI-07 | Mundo | Cena autoritativa | `WYD.exe` | `HEADER.ID = 30000` | recebe `ESCENE_FIELD`, `0x0364` CreateMob (232 B), `0x0165` RemoveMob (16 B) | Confirmado (servidor); render **não provado** |
| CLI-08 | Mundo | Movimento / ação | `WYD.exe` | `SendPacket 0x54DA23` | envia `0x036C` Action (52 B), `0x036A` Motion | Confirmado (tamanho); semântica de rota **Desconhecido** |
| CLI-09 | Social | Chat e whisper | `WYD.exe` | sender `0x4695b3`, receptor `0x48241f` (lê falante do `HEADER.ID`) | envia `0x0333` (140 B, `text[128]@12`), `0x0334` | Confirmado |
| CLI-10 | Combate | Ataques | `WYD.exe` | três variantes | envia `0x0367`, `0x039D`, `0x039E` | Confirmado (opcode); cadência **Desconhecido** |
| CLI-11 | Item | Uso de item / refino | `WYD.exe` | `UseItem` | envia `0x0373` | Confirmado (servidor ligado) |
| CLI-12 | Item | Item no chão | `WYD.exe` | handler `0x486c25` | **recebe** `0x026E` (30 B, packed); **envia** `0x0270` (28 B) | **Divergente** com `Basedef.h` (direção) |
| CLI-13 | Comércio | Trade | `WYD.exe` | `MSG_Trade` | envia `0x0383`, `0x0384`, `0x0376` | Layout **Desconhecido** (156 × 154 B) |
| CLI-14 | Comércio | Loja de NPC | `WYD.exe` | `REQShopList`/`Buy`/`Sell` | envia `0x027B`, `0x0379`, `0x037A`, `0x0398` | Confirmado (opcode); regra ausente no Rust |
| CLI-15 | Quest | Quest/diálogo | `WYD.exe` | `Quest` | envia `0x028B` (20 B) | Parcial (só Campo de Treinamento) |
| CLI-16 | UI | Texto livre e caixas | `WYD.exe` | `0x0101` MessagePanel é a **única** via de texto livre | recebe `0x0101`; `0x0102` não desenha sozinho | Confirmado por desassemblage |
| CLI-17 | UI | Telas raster | `UI/`, `NUI/` | `.wyt` = `WT10` + raster 24/32 bpp | — (desenho local; layout no `.exe`) | Confirmado (256/259 amostras fecham fórmula) |
| CLI-18 | Catálogo | Tabelas de item/skill | raiz | `ItemList`, `Itemname`, `InitItem`, `SkillData`, `itemicon` | carregadas localmente, não vêm do servidor | Confirmado (tamanho); layout **Hipótese** |
| CLI-19 | Distribuição | Launcher/updater oficial | `WYDLauncher.exe`, `settings.txt` | endpoint `http://54.207.102.145/update` (morto) | HTTP próprio | Confirmado |
| CLI-20 | Proteção | Anticheat/proteção | `XTrap/`, `JProtectDll.dll`, `jlock*.sys` | verificação de processo/arquivo | — | Presença confirmada; atividade **Desconhecido** |

### 4.2 Efeito no servidor e no Rust

| ID | Envia (C→S) | Recebe (S→C) | Legado | Rust hoje | Diferença / Melhoria |
|---|---|---|---|---|---|
| CLI-01 | — | `serv00.htm` (texto) | lê contadores | `status/service.rs` serve o `serv00.htm` | OK; IP/porta na célula são UNVERIFIED |
| CLI-02 | `0x020D` | `0x010A` (1996 B de corpo) | `_MSG_AccountLogin` | ✅ `cpsock::handshake` | senha errada **não** desenha nada (`DIV-01`) |
| CLI-03 | `0x0FDE` | `0x0FDE`/`0x0FDF` | `_MSG_AccountSecure` | ✅ | troca em 2 etapas é decisão nossa |
| CLI-04 | — | `0x0110` | `_MSG_CharacterLogin` roster | ✅ mas layout `rust-v1` | offsets **não** confirmados |
| CLI-05 | `0x0211` | `0x0112` | `_MSG_DeleteCharacter` | ✅ | frame 852 B correto |
| CLI-06 | `0x0213` | `0x0114` (1820 B) | `_MSG_CharacterLogin` | ✅ | `STRUCT_MOB` S→C sem captura |
| CLI-07 | — | `0x7530`, `0x0364`, `0x0165` | `SendFunc.cpp` | ✅ (`ESCENE_FIELD`, `CreateMob`, `RemoveMob`) | render no 7662 **pendente** (`GAME-17`) |
| CLI-08 | `0x036C`/`0x036A` | `ESCENE_FIELD` delta | `_MSG_Action/Motion` | 🚧 movimento no core, sem fio | falta pacote de movimento (`PAR-MOV-01`) |
| CLI-09 | `0x0333`/`0x0334` | `0x0333` | `_MSG_MessageChat` | ❌ | chat é requisito social básico |
| CLI-10 | `0x0367`/`0x039D`/`0x039E` | `RemoveMob`, `UpdateScore` | `_MSG_Attack*` | ✅ morte + drop + EXP | cadência **não** provada (`GAME-20`) |
| CLI-11 | `0x0373` | `0x0182` SendItem | `_MSG_UseItem` | ✅ refino | resto do `UseItem` (consumível) parcial |
| CLI-12 | `0x0270` (28 B) | `0x026E` (30 B), `0x0171`, `0x0175`, `0x016F` | `_MSG_GetItem/DropItem` | ❌ | corrigir direção no catálogo (`DIV-02`) |
| CLI-13 | `0x0383`… | resposta de trade | `_MSG_Trade` | ❌ (layout candidato) | 156 B × 154 B (`DIV-04`) |
| CLI-14 | `0x027B`… | lista/preço | `_MSG_REQShopList/ReqBuy` | ❌ | loja de NPC é 🟡 no roadmap |
| CLI-15 | `0x028B` | `0x028B` | `_MSG_Quest` | 🟡 Campo de Treinamento | resto das quests ❌ |
| CLI-16 | — | `0x0101` | `_MSG_MessagePanel` | parcial (`send_message`) | usar `0x0101` para erros visíveis |
| CLI-17 | — | — | — | — | `.wyt` é editável (`wyt.mjs`) |
| CLI-18 | — | — | `content/TMsrv` | catálogo no Postgres | divergência de índices é risco (`DIV-07`) |
| CLI-19 | HTTP | — | launcher legado | `wyd-launcher` substitui | endpoint morto, não suportar |
| CLI-20 | — | — | — | — | não depender de XTrap/jlock |

## 5. Superfície de protocolo que o cliente realmente trata

| Medida | Valor | Fonte |
|---|---:|---|
| Opcodes do nosso catálogo comparados pelo cliente | **91** (de 198) | desassemblage 2026-11-11 |
| `0x0117 NewAccountFail` | **ausente** (zero `cmp`) | idem |
| `0x0102 MessageBoxOk` | compara, mas cai no bloco de `0x0101/0x0105/0x0106` | idem |
| `0x0110` / `0x0112` / `0x0119` / `0x011a` / `0x011b` | presentes | idem |
| Validação de `Size` por opcode | **não existe** no decoder | idem |
| Tamanho máximo de frame | 8.192 B (candidato `cmp …,0x2000`) | candidato |

> Consequência: emitir um opcode que o cliente **não** compara é silêncio, não erro. Isso muda a
> estratégia de teste: ausência de reação no cliente não prova que o servidor não enviou.

## 6. O que este documento **não** cobre

- Captura ao vivo (VM/GUI/`Wireshark`): offsets de campo do header, `STRUCT_MOB`/`SELCHAR` S→C,
  layout de `ESCENE_FIELD` de resposta, trade.
- Grafo de chamadas do `WYD.exe` (sem decompilador neste ambiente; `objdump` só deu âncoras).
- Atividade real de XTrap/jlock neste Windows.
- `.xt`, `Effect/*`, `Sound/*` conteúdo (só contagem/tamanho).
