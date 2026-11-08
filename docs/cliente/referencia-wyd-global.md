# Referência externa: cliente real do WYD Global (RaidHut)

> **Data da medição:** 2026-11-08 · **Local:** `C:\Users\Canta\AppData\Local\wyd_launcher`
> (instalação do launcher oficial `2.7.3` + jogo em `WYD Global/`).
> **Natureza:** cliente **de terceiro**, fechado, **não** é o nosso cliente e **não** é contrato.
> Serve para: conferir arquitetura de launcher/atualização, reconhecer domínios de conteúdo que
> nasceram depois do 7662 e validar decisões do `wyd-launcher`. Distinto de `client/`
> (build 7662, `docs/cliente/README.md`), que é o alvo do servidor.

## 1. Identificação medida

| Item | Valor |
|---|---|
| Raiz | `wyd_launcher` com `Update.exe`, `With Your Destiny.exe`, `app-2.7.3/`, `packages/`, `WYD Global/`, `gid.txt` |
| Versão do launcher | **2.7.3** (`app-2.7.3/launcher.json` = `{"Launcher":{"version":"2.7.3"}}`) |
| Mecanismo de update | **Electron + Squirrel.Windows**: `packages/wyd_launcher-2.7.3-full.nupkg` (117.555.531 B) + `RELEASES` com SHA1 `825E5374F4FAEB9C7057F6F4C69F9ACD677D4976` |
| Shell do launcher | `app-2.7.3/With Your Destiny.exe` 176.796.680 B (Chromium embarcado) |
| `gid.txt` | `Global=C:\Users\Canta\AppData\Local\wyd_launcher` |
| Jogo | `WYD Global/WYD.exe` **6.494.000 B**, PE32/i386 (`machine=0x014c`), `e_lfanew=304`, `TimeDateStamp=1780386287` (≈2026-06) |
| Porte do transporte | o binário contém a constante `11 f3 11 1f` (`INITCODE 0x1F11F311`) e a string `./serverlist.bin` |
| `serverlist.bin` | **ausente** na pasta do jogo (o cliente espera o arquivo; quem entrega é o launcher/updater) |
| Runtime extra | `JCore.dll` (28.976 B, assembly gerenciado), `Updater.exe` (.NET 4.6.1, 7.168 B), `Newtonsoft.Json.dll`, `System.IO.Compression*.dll`, `WydConfig.exe` + `WydConfig.swf` (configurador Flash), `Adobe AIR/`, `META-INF/` |
| Config de update | `WYD Global/config.json` → `{"path":".","infoJson":"https://d16tndoz98nwcc.cloudfront.net/info.json","gameId":"Global"}` |
| Avisos | `WNotice.ini` aponta para imagens em `https://wydglobal.raidhut.com/images/…` |
| Aparência/gráficos | `config_settings.json` (resolução, textura, efeito, cursor), `Graphics.ini`, `Lang/` com 73 arquivos |

## 2. Conteúdo: o que existe lá e não existe no 7662

Todos os tamanhos **medidos** nesta data; o *papel* de cada `.bin` é **hipótese pelo nome**, não
layout verificado (o 7662 é a única fonte com estrutura inicial documentada em `formatos.md`).

| Arquivo | Bytes | Hipótese de domínio |
|---|---:|---|
| `ItemList.bin` | 2.494.004 | catálogo de itens — **2,7×** o nosso (910.004 B) → base de itens muito maior |
| `ItemSet.bin` | 928.004 | conjuntos de item (sets) como tabela própria |
| `Missionitems.bin` | 26.880.000 | itens de missão (volume anômalo; provável tabela por missão/NPC) |
| `MountData.bin` / `MountDataV.bin` | 70.004 / 14.004 | montarias |
| `CollectionData.bin` | 23.520 | coleções/álbum |
| `ArenaVisualData.bin` | 18.164 | visuais de arena |
| `ItemPrice.bin` | 800 | preços separados do catálogo |
| `SkillData.bin` | 25.796 | skills — **1,08×** o nosso (23.812 B) → a tabela de skills quase não cresceu |
| `Mixlist.bin` | 545.600 | mix/combinação — **2,0×** o nosso (272.000 B) |
| `HeightMap.dat` | 16.777.216 | 4096×4096 bytes = 1 B por célula, coerente com `data-formats.md` |
| `ZoneXY.txt` | 791 | pares de zona (`1,1`, `1,4`, …) — domínio de zonas nomeadas |

**Leitura para o nosso modelo:** `ItemSet`, `MountData`, `CollectionData`, `Missionitems` e
`ArenaVisualData` são **domínios que o 7662 não tem como arquivo separado**. Não viram issue por
existirem no Global; viram **pergunta** ("o 7662 carrega sets/mounts dentro de outro `.bin`?") a
responder com o dossiê do 7662, nunca copiando layout de terceiro.

## 3. Assets (medido, por pasta de `WYD Global/`)

| Pasta | Arquivos | Bytes |
|---|---:|---:|
| `mesh/` | 7.947 | 558.921.773 |
| `UI/` | 396 | 153.373.289 |
| `music/` | 13 | 82.442.796 |
| `sound/` | 350 | 38.069.980 |
| `Env/` | 605 | 35.471.773 |
| `Effect/` | 582 | 18.318.960 |
| `NUI/` | 20 | 9.121.596 |
| `Adobe AIR/` | 8 | 38.247.138 |
| `Lang/` | 73 | 2.541.949 |
| `shader/` | 18 | 7.920 |

Comparação com o 7662 (`inventario.md` §4): malhas 7.947 × 5.361 arquivos e UI 396 × 342 — mesma
família de formatos (`.msa`/`.wyt`), escala maior. **Não redistribuir** nenhum dos dois.

## 4. Arquitetura de launcher/atualização — o que confirma e o que rejeitar

Confirmado no disco:

1. **Shell web + backend nativo + jogo em subpasta** (Electron aqui; Tauri no nosso desenho) —
   valida `docs/cliente/launcher.md` §3 e a decisão da FASE-08-8.
2. **Autoupdate do launcher é independente do update do jogo**: Squirrel cuida do app
   (`Update.exe` + `.nupkg` + `RELEASES`); o jogo tem **outro atualizador** dentro da pasta
   (`Updater.exe` .NET 4.6.1) e um `info.json` em CDN (`cloudfront`) como manifesto remoto.
   Ou seja: **dois canais de distribuição separados**, exatamente o que o nosso
   `wyd-launcher` une em um só manifesto (`manifest` + `distribuicao`, ADR-0003).
3. **`launcher.json` não carrega endpoint** — só versão. Endereço de update vive no código.
   O nosso modelo é melhor para operação: manifesto versionado e verificável.

Rejeitar do modelo deles:

- **SHA1 no `RELEASES`** como única integridade → nós usamos SHA-256 + HMAC (FASE-08-3).
- **Delta/full `.nupkg` sem fail-closed**: nosso patch é atômico (tmp + rename) e recusa pacote sem
  assinatura **antes** de tocar o disco.
- **Configurador Flash** (`WydConfig.swf`/`Adobe AIR`) e `.NET Framework 4.6.1` como dependência
  extra → nós ficamos em binário Rust único + WebView2 do sistema.
- **Instalação em `AppData\Local` + manifesto `requireAdministrator`** (registrado em
  `launcher.md` §3): nosso instalador prefere pasta do usuário sem elevação.

## 5. Limites de uso (não negociável)

1. É **terceiro**: nada dele entra no git, no manifesto do nosso launcher ou em fixture.
2. **Não é contrato de protocolo.** Ter `INITCODE` no binário prova só que a família de transporte
   é a mesma; layout de struct do Global é **UNVERIFIED** e provavelmente diferente do 7662.
   Nenhum offset do Global vira regra no `wyd-protocol`.
3. **Não é fonte de conteúdo.** `ItemList.bin` de 2,49 MB não vira seed: a verdade do nosso
   conteúdo é o PostgreSQL semeado do 7662 (`docs/wyd/README.md`).
4. **Não é alvo de patch.** Não se escreve no cliente do Global; ele é observação.
5. Qualquer afirmação daqui que não seja tamanho/hash/string é **hipótese** e precisa da marca.

## 6. O que aproveitar de fato

| Achado | Uso no nosso projeto |
|---|---|
| Dois canais (launcher × jogo) com manifestos distintos | manter **um** manifesto com escopo declarado (app + jogo) e documentar por que é melhor |
| `info.json` em CDN como "versão + notícias" | formato de notícias/status do portal para a GUI (FASE-08-8) sem segredo no binário |
| `serverlist.bin` entregue pelo launcher (ausente na instalação do jogo) | confirma nosso desenho: o launcher escreve `serverlist.bin` (`launcher.md` §1 papel 4) |
| `ZoneXY.txt`/`Lang/` como dados de config fora do executável | reforça ADR-0009: configuração e conteúdo são dados (banco), não código |
| Domínios `ItemSet`/`MountData`/`CollectionData`/`Missionitems` | lista de **perguntas** para a FASE-03-4/10-1 sobre o que o 7662 carrega embutido |

## 7. Como revalidar (comandos usados)

```bash
G="/c/Users/Canta/AppData/Local/wyd_launcher"
ls -la "$G"; cat "$G/app-2.7.3/launcher.json"; cat "$G/packages/RELEASES"
sha256sum "$G/With Your Destiny.exe" "$G/app-2.7.3/With Your Destiny.exe"
cat "$G/WYD Global/config.json"
off=$(od -An -tu4 -j60 -N4 "$G/WYD Global/WYD.exe" | tr -d ' ')
od -An -tx2 -j$((off+4)) -N2 "$G/WYD Global/WYD.exe"        # machine=0x014c (PE32/i386)
od -An -tu4 -j$((off+8)) -N4 "$G/WYD Global/WYD.exe"        # TimeDateStamp
LC_ALL=C grep -oba $'\x11\xf3\x11\x1f' "$G/WYD Global/WYD.exe" | head  # INITCODE
strings -a "$G/WYD Global/WYD.exe" | grep -a "serverlist"
```

Hashe desta medição: launcher raiz `ba3ef224328507b52387110755df783494126586d3d1f9714e503bd6aeb75fda`;
`app-2.7.3/With Your Destiny.exe` `735fe9776928a677b1fafcdf7a2a25deaa3d247f3aa0e381ad6d945058be8655`.
Se os hashes mudarem, a instalação é **outra versão** e toda afirmação deste documento precisa
ser reaberta.
