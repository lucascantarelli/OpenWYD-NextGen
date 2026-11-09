# Inventário do cliente (`client/`)

> Levantado em **2026-11-07** com `ls -l`, `sha256sum` e leitura do cabeçalho PE
> (`od` sobre `e_lfanew`). Tudo aqui é verificável pelos comandos do §6.
> O que não foi medido está marcado **UNVERIFIED** — nunca deduzido.

## 1. Identidade do build

| Item | Valor |
|---|---|
| Build | **7662** (Cavaleiros de Kersef) — inferido do patch `ClientPatch_v7662` (DLL em `client/`) e do par `client-version 12000` já medido no handshake |
| Arquitetura | **PE32 / i386** (`machine=0x014c`, `magic=0x010b` nos três binários principais) |
| `WYD.exe` SHA-256 | `04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8` |
| `WYD.exe` tamanho | 2.347.008 bytes |
| Tamanho total da pasta | ~344 MB (189 MB são `Mesh/`) |
| Versionamento | **fora do git** (`.gitignore`); nenhum binário é distribuído pelo repositório |

Os patches do `ClientPatch_v7662` escrevem em `0x401000`–`0x5F3000`, o que confirma
**image base 0x400000** e seção `.text` de ~2 MB — coerente com o `SizeOfImage` de um
executável MSVC x86 desse porte.

**Confirmado pela análise estática de 2026-11-07** (`analise-estatica.md`, artefatos em
`gerado/`): entry point `0x5f3c66`, `TimeDateStamp` **2015-07-02T05:09:43Z**, `SizeOfImage`
37.257.216 B, seções `.text` 2.043.904 / `.rdata` 110.592 / `.data` 163.840 / `.rsrc` 24.576 B,
**18 DLLs e 268 imports** (d3d9, dinput8, dsound, ws2_32, wsock32, wininet…), export `WYDBR.exe!fcEXP`.
As três checagens de checksum (`0x53AC6A`, `0x53AD52`, `0x53AE7E`) já estão em `EB` **no disco**:
o build distribuído não depende de injeção em runtime para desativá-las.

## 2. Executáveis e bibliotecas

| Arquivo | Bytes | SHA-256 (prefixo) | Papel |
|---|---:|---|---|
| `WYD.exe` | 2.347.008 | `04c9d2a1235c` | Jogo (render, cena, rede CPSock) |
| `WYDLauncher.exe` | 2.674.752 | `e10c82df050a` | Launcher/updater oficial do build; usa `MSVCR` (não é .NET) |
| `ClientPatch.dll` | 41.984 | `96bbd83ec43f` | Patch em runtime (GPL-3.0, © Victor Klafke/Charles TheHouse) — reescreve `.text` em memória |
| `JProtectDll.dll` | 45.056 | `6668cc00eaa3` | Camada de proteção JProtect |
| `JPGLib.dll` | 57.344 | `665440b190aa` | Biblioteca gráfica própria (JPEG) |
| `XPva03.dll` | 280.064 | `fb674ce106e3` | Biblioteca do XTrap carregada pelo jogo |
| `msvcp140.dll` | 590.112 | — | Runtime VC++ 2015+ |
| `msvcp140d.dll` | 938.480 | — | Runtime VC++ **debug** (resíduo de build; não é necessário em produção) |
| `jlock101.sys` / `.VXD` | 12.365 / 17.686 | — | Driver/proteção "jlock" (modo kernel; **UNVERIFIED** se é carregado neste build) |
| `jlock102.sys` / `.VXD` | 12.490 / 17.167 | — | Idem, segunda geração |

### 2.1 Anticheat XTrap (`client/XTrap/`)

| Arquivo | Bytes | Observação |
|---|---:|---|
| `XTrap.xt` | 1.856.536 | Pacote do anticheat |
| `XTrapVa.dll` | 2.765.800 | Módulo de verificação (o `.exe` referencia `XTrapVa.dll`) |
| `XTrapExt.dll` | 45.056 | Extensão |
| `psapi.dll` | 23.040 | Cópia local (proxy/hook de enumeração de processos) |
| `XData0422.Log` | 142.384 | Log de execução do XTrap — **existe e foi escrito**, sinal de que o módulo rodou (**UNVERIFIED** se ainda está ativo) |
| `XDataFI0.Xtp` | 308 | Configuração do XTrap |

`client/settings.txt` (780 B, PT-BR) contém as strings do launcher/updater e o endpoint
`http://54.207.102.145/update` — é o alvo de atualização legado, não o servidor de jogo.
`client/WYD.log` registra `Error in Init Render Target Texture` e
`Reading Error : mesh\mega001.msa` — evidência de que erro de render/asset é **local**,
não de rede. A mensagem de render target é **não fatal**: com ela no log o cliente segue até a
tela de login, desenha o *server select* e conclui o handshake (prova em
`docs/rust/03-progresso.md`, 2026-11-09).

## 3. Dados de configuração e catálogos (raiz da pasta)

| Arquivo | Bytes | Papel aparente | Status |
|---|---:|---|---|
| `serverlist.bin` | 7.040 | Servidores/canais: grade 10×11×64 ofuscada com tabela própria (não é a do CPSock) | **CONHECIDO** (`formatos.md` §4) |
| `serverlist.txt` | 70 | Formato texto legível (`http://IP/serv00.htm`, `IP:8281`) | CONHECIDO |
| `sn.bin` | 143 | Nomes de canal/servidor (ofuscado) | **UNVERIFIED** (layout) |
| `Config.bin` | 30 | Configuração curta | UNVERIFIED |
| `settings.txt` | 780 | Strings do launcher/updater | CONHECIDO (texto) |
| `object.bin` | 524.292 | Objetos/cenário do mundo | UNVERIFIED |
| `ItemList.bin` | 910.004 | Catálogo de itens do cliente | CONHECIDO parcial (espelha `item_catalog` no servidor) |
| `Itemname.bin` | 215.084 | Nomes de item | CONHECIDO parcial |
| `InitItem.bin` | 768 | Itens iniciais por classe | CONHECIDO (espelha `BaseMob`) |
| `extraitem.bin` | 106 | Itens extras | UNVERIFIED |
| `SkillData.bin` | 23.812 | Tabela de skills do cliente | CONHECIDO parcial |
| `itemicon.bin` | 26.000 | Índices de ícone | UNVERIFIED |
| `itemhelp.dat` | 99.134 | Textos de ajuda de item | UNVERIFIED |
| `Mixhelp.dat` | 11.050 | Ajuda de mix/combinação | UNVERIFIED |
| `Mixlist.bin` | 272.000 | Receitas de mix | CONHECIDO parcial |
| `curse.bin` | 8.192 | Maldições/efeitos | UNVERIFIED |
| `cdata.bin` | 8.192 | Dados de personagem/classe | UNVERIFIED |
| `minimap.dat` | 4.058 | Minimapa | UNVERIFIED |
| `shader1..8.bin` | 792–1.340 | Parâmetros de shader | UNVERIFIED |
| `AniSound4.txt` | 45.164 | Mapa de som por animação | CONHECIDO (texto) |
| `notice.txt` | 1.124 | Aviso da tela inicial | CONHECIDO (texto) |
| `guilds.txt`, `Music.txt`, `sound.txt` | 0 | Vazios neste build | — |

## 4. Assets

| Pasta | Arquivos | Tamanho | Conteúdo |
|---|---:|---:|---|
| `Mesh/` | 5.361 | 189 MB | Malhas `.msa` (o log cita `mesh\mega001.msa`) |
| `UI/` | 342 | 73 MB | Telas `.wyt` (formato proprietário) |
| `Env/` | 522 | 15 MB | Cenário/ambiente |
| `Effect/` | 468 | 4.6 MB | Efeitos |
| `Sound/` | 12 | 38 MB | Áudio |
| `music/` | 13 | 26 MB | Música |
| `XTrap/` | 6 | 4.7 MB | Anticheat |
| `NUI/` | 8 | 3.3 MB | Interface nova (`main.wyt`, `mainchat.wyt`, `ccmode.wyt`…) |
| `shader/` | 18 | 40 KB | Shaders compilados |
| `ScreenShot/` | 0 | — | Saída do cliente |

## 5. O que **não** está no inventário

- Conteúdo de `.wyt`, `.msa`, `.xt` — só a estrutura inicial foi documentada (`formatos.md`);
  `.xt` (XTrap) segue sem análise.
- Mapa completo de funções do `WYD.exe` — a 08-5 confirmou **6 âncoras** e os **3 sites de
  checksum**, não o grafo de chamadas (sem decompilador neste ambiente).
- Structs de jogo (`STRUCT_MOB`/`ITEM`/`SELCHAR`) — 08-6.
- Assinatura Authenticode — não verificada; não há sinal de packer (seções legíveis, imports
  completos, strings em claro).

## 6. Como revalidar (sem ferramenta nova)

```bash
cd client
sha256sum WYD.exe WYDLauncher.exe ClientPatch.dll JProtectDll.dll JPGLib.dll XPva03.dll
ls -l WYD.exe WYDLauncher.exe *.dll *.sys *.VXD
ls -l *.bin *.dat *.txt
for d in Mesh UI Env Effect Sound music NUI shader XTrap; do printf "%-10s %s\n" "$d" "$(ls "$d" | wc -l)"; done

# arquitetura PE (sem dependência externa)
for f in WYD.exe WYDLauncher.exe ClientPatch.dll; do
  off=$(od -An -tu4 -j60 -N4 "$f" | tr -d ' ')
  printf "%s machine=0x%s magic=0x%s\n" "$f" \
    "$(od -An -tx2 -j$((off+4)) -N2 "$f" | tr -d ' ')" \
    "$(od -An -tx2 -j$((off+24)) -N2 "$f" | tr -d ' ')"
done
```

Se algum hash divergir do §1/§2, o build **não é** o 7662 de referência: todo offset do
`superficie-conhecida.md` deixa de valer e o dossiê precisa ser reaberto (FASE-08-5).
