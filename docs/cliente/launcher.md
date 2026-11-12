# Launcher — conceito, legado, referência e plano (2026-11-07)

O launcher é o **único programa que o jogador instala e abre**: janela gráfica
com notícias/aviso, botão Jogar, verificação de versão, download de patch e
disparo do `WYD.exe` com `serverlist.bin` válido. Ele **não é o jogo** — é
instalador + atualizador + ponto de contato (notícias, status, links).

## 1. O que o launcher faz (conceito, 5 papéis)

1. **Apresentar**: notícias/eventos, banner, status dos canais, botão Jogar.
2. **Verificar**: versão local × manifesto remoto (hash por arquivo).
3. **Atualizar**: baixar só o que mudou, com resume; aplicar de forma atômica
   (nunca meio patch); recusar pacote sem assinatura válida.
4. **Entregar serverlist**: escrever `serverlist.bin` apontando para os
   canais atuais (IP:8281).
5. **Disparar o jogo**: iniciar `WYD.exe` e fechar/monitorar (fail-closed: se a
   verificação falhar, o jogo não abre).

## 2. Legado (`client/`, build 7662)

- `WYDLauncher.exe` + skins `Launcher.bmp`/`Launcher.hat`.
- Fontes de update embutidas: `http://www.wyrd.co.kr/update_regular6/` (oficial
  coreano) e `settings.txt` com URL de servidor privado
  (`http://54.207.102.145/update`) — prova de que privados reusavam o mesmo
  launcher trocando a URL.
- `serverlist.txt` local (`192.168.100.3:8281` — IP privado, nunca versionar).

## 3. Referência real: launcher oficial atual (instalado nesta máquina)

Analisado em `C:\Users\Canta\AppData\Local\wyd_launcher` (versão 2.7.3):

- **Electron + Squirrel.Windows**: `With Your Destiny.exe`, `Update.exe`,
  `app-2.7.3/` (Chromium + `resources/app`), `packages/wyd_launcher-2.7.3-full.nupkg`
  + `RELEASES` (hash SHA1 por pacote). Autoupdate = delta/full `.nupkg`.
- **Jogo em subpasta**: `WYD Global/` (AniSound4.txt, HeightMap.dat, Effect/…).
- **`launcher.json` = só `{ "Launcher": { "version": "2.7.3" } }`** — sem endpoints:
  URLs de update/notice vivem no código Electron (`resources/app`), não em config.
- Manifesto do EXE exige **elevação de administrador** (`requireAdministrator`)
  — necessário porque ele escreve em `Program Files`/raiz; nosso instalador
  deve preferir pasta do usuário para evitar esse atrito.

Lição: o padrão de mercado é **shell web (HTML) + backend nativo + update
assinado + jogo em subpasta**. É exatamente o desenho do Tauri (§5).

## 4. O que já temos (núcleo Rust, sem GUI)

`crates/wyd-launcher` — CLI `check`/`update`/`launch` + biblioteca
(`manifest`, `patch` atômico, `bootstrap`, `distribuicao`, `sha256`),
estratégia em ADR-0003, contrato em `docs/cliente/customizacao.md` §3–§4,
entregue na FASE-08-3 (27 testes, pacote adulterado recusado, resume).
É o **backend do futuro launcher GUI**: a interface chama essas funções, não
as reimplementa.

Mapa client↔launcher e plano de RE: `docs/cliente/mapa.md`;
inventário de assets: `docs/cliente/inventario.md`.

## 5. GUI: proposta Tauri v2 (a confirmar no questionário)

- **Tauri v2**: backend Rust (reusa `wyd-launcher` como crate), frontend web
  (qualquer framework; sugestão: o mesmo padrão do portal), WebView2 no
  Windows (já vem no Win10 1803+/Win11 — sem Chromium embutido, instalador
  de ~5–10 MB contra ~100 MB+ do Electron), **updater com assinatura
  obrigatória** (casa com ADR-0003), NSIS/MSI prontos.
- Precedente open-source de launcher de jogo em Tauri v2: **Sparus**
  (manifesto `state.json` remoto, instalar/atualizar/jogar, botão adaptativo).
- Alternativas descartadas por padrão (só reconsiderar com PoC):
  Electron (pesado, duplica o que o SO já tem), egui/iced (nativo, mas sem
  HTML — portal não seria reusável na tela de notícias).

## 6. Integração com o portal (contrato, LAUNCHER-01)

O launcher lê do portal (somente leitura, `crates/wyd-launcher/src/portal_client.rs`):

| Endpoint | Conteúdo | Fonte | Cache |
|---|---|---|---|
| `GET /api/launcher/versao` | `client_build` 7662, `protocol_version` 12000, `config_revision` viva (null com gateway fora), `gateway_up` | constantes documentadas + gateway via token de processo (mesmo padrão de `/api/health`) | 60 s |
| `GET /api/launcher/eventos` | grade de eventos (código/nome/estado/horários; vazia com gateway fora) | `world_event_admin/get_world_event_config` via token de processo — só sai dado público de jogo | 60 s |
| `GET /api/launcher/manifesto` | `manifest.txt` real da distribuição (`build`, `protocol`, SHA-256 + tamanho por arquivo); `404` sem publicação | pasta montada em `LAUNCHER_DIST_DIR`, gerada por `tools/launcher/dist-manifest.mjs` | 60 s |
| `GET /api/launcher/pacote` | binário do launcher com `x-checksum-sha256` no header; `404` sem pacote | mesmo diretório (`LAUNCHER_PACKAGE`, padrão `wyd-launcher.exe`) | 5 min |
| `GET /api/launcher/configuracao` | `launcher.conf` **assinado** (revisão, `portal.base`, caminhos e sobreposições de marca/canais); `404` sem publicação | mesmo diretório, assinado por `tools/launcher/sign-config.mjs` | 60 s |
| `GET /api/launcher/noticias` | vitrine (`slug`, `titulo`, `data`, `tag`, `resumo`, `url`) do JSON de `docs/noticias`; sem corpo markdown e sem sessão | `server/noticias.ts` (mesma fonte do site) | 60 s + `stale-while-revalidate=300` |
| `POST /api/launcher/telemetria` | evento anônimo e fechado (`versao`, `build_cliente`, `operacao`, `ok`, `duracao_ms`, `arquivos`, `falhas[≤10]`); `400` fora do contrato, `413` acima de 4 KiB, `415` sem JSON, `204` sem corpo | `server/launcher-telemetria.ts` (agregação em processo + log estruturado, sem PII) | `no-store` |

Sem TLS neste módulo (`http://` de loopback/rede local; `https://` entra com o canal
de distribuição assinado). **Publicação real (2026-11-11):** o portal serve o manifesto e o
pacote a partir de uma pasta de distribuição montada (`LAUNCHER_DIST_DIR`), gerada por
`node tools/launcher/dist-manifest.mjs dist/launcher 7662 12000`; sem pasta, as duas rotas
respondem `404` e a página de downloads mostra “não publicada” — nunca manifesto vazio.
**Consumo no launcher (2026-11-12):** as duas rotas acima são usadas em código real.
- **Notícias:** `gui` (e `gui --dry-run`) busca a vitrine com timeout curto de 700 ms e
desenha até 5 itens no painel `Notícias` da janela existente (`LauncherGUI.html`). Rota
fora, lenta ou com erro não impede a janela de abrir: a seção aparece com o motivo
(`notícias indisponíveis: …`) e o motivo também vai para o stderr (`notícias: …`).
- **Telemetria:** após `check`, `config`, `update` (patch) e `launch` — inclusive nas
falhas — o launcher faz `POST` do contrato fechado, com `duracao_ms`, `arquivos` e até
10 `falhas[{arquivo,motivo}]`. Sem base de portal (config remota ausente e
`WYD_LAUNCHER_PORTAL` vazio) o envio é omitido; falha de envio só aparece como
`telemetria: …` no stderr, nunca muda o exit code nem trava a janela. `--dry-run` não
envia patch.

**Ainda fora do contrato:** painel de monitoramento da telemetria no portal, `/admin/patches`
+ canal de versão publicável (exige migration + `ConfigStore` + rotas, escopo backend
próprio) e autoupdate do binário. Conta/login no
launcher v1 = **opcional** (o jogo autentica no CPSock; o launcher não guarda senha).

## 7. Plano (issue GUI criada: FASE-08-8, decisão Tauri + nome WYD em 2026-11-07)

1. ADR GUI (Tauri v2 + fronteiras com `wyd-launcher`) — **aceita** (ADR-0007).
2. **Frente mínima entregue (2026-11-12):** `apps/launcher-ui` (bin
   `WYDLauncher`) abre janela Tauri com o HTML da identidade efetiva (mesma
   `configuracao::resolver_flags` da CLI), notícias do portal com fallback e o
   botão Jogar chamando `lancamento::iniciar` — fail-closed, sem escrita própria.
   `WYDLauncher.exe --instalar <pasta-do-jogo>` copia o executável para
   `<pasta>/WYDLauncher.exe` (atômico, com backup); rodando de dentro da pasta,
   a identidade e o jogo padrão (`manifest.txt`, `canais.txt`) vêm dali.
   `--check` imprime a identidade resolvida sem abrir janela e `--dir` aponta
   outra pasta do jogo. A pasta do jogo real é `client/` (ignorada no git):
   `WYDLauncher.exe --instalar client` deixa o launcher ao lado do `WYD.exe`
   (o `WYDLauncher.exe` original fica em `.bak`) e, rodando de lá sem `--dir`,
   ele resolve a própria pasta.
   **Arte:** o ícone da janela/instalador é o ICO entregue pelo dono em
   `apps/launcher-ui/icons/icon.ico`; o fundo (`apps/launcher-ui/assets/`,
   otimizado para `crates/wyd-launcher/assets/fundo.jpg`, 135 KB) entra embutido
   no HTML como `data:` URI, com véu escuro por cima para manter contraste —
   o mesmo desenho no HTML da CLI e na janela Tauri.
   **Portal:** `/configuracoes/launcher` revisa e publica o `launcher.conf`
   assinado com histórico append-only (autor, SHA-256, bytes) e
   `/monitoramento/launcher` mostra a telemetria agregada por versão + operação.
   **E2E real da janela:** `node tools/launcher/e2e-janela.mjs` (tauri-driver +
   msedgedriver) sobe portal falso, abre a janela do executável **dentro da
   pasta do jogo**, confere o HTML injetado e a notícia, clica em Jogar e prova
   processo único + fail-closed sem reescrever `serverlist.bin`.
   Ainda falta: barra de progresso de patch e instalador NSIS.
3. Instalador NSIS por canal (WYD), ícone e metadados WYD.
4. Teste: máquina limpa instala → abre → atualiza → joga; pacote adulterado
   recusado na GUI como na CLI.

## 8. Entrega do `serverlist.bin` (contrato, 2026-11-08)

O launcher é quem escreve o `serverlist.bin` (papel 4 do §1). O formato está decodificado em
`formatos.md` §4; o que falta é a **regra de entrega**, para não voltar ao ciclo de editar o
binário à mão:

| Regra | Por quê |
|---|---|
| Fonte única é o **registro de canais do servidor** (um grupo = um servidor; canais 1..10 = hosts das instâncias), nunca um `.bin` versionado | `client/` e `serverlist.bin` ficam fora do git; o `.bin` é artefato de ambiente |
| O launcher **gera** o `.bin` pelo mesmo algoritmo (`10 × 11 × 64`, soma de `obfuscationKey[63-i]`), a partir do registro de canais versionado | implementação única em `crates/wyd-launcher/src/serverlist.rs`; o Node (`tools/client/serverlist.mjs`) fica como referência de operação e a paridade de **bytes** é teste (`tests/golden/serverlist_paridade.rs`) |
| Escrita **atômica com backup**: `serverlist.bin.tmp` → `rename`, guardando `serverlist.bin.bak` | arquivo meio escrito = cliente sem lista, e o jogador não sabe reverter |
| URL de status com **≤ 32 caracteres** e **um IP por canal** (porta de jogo é 8281 fixa) | limites medidos no cliente 7662; violá-los deixa a coluna Canal vazia |
| `sn.bin` (nomes + contadores) é **do cliente**, não do servidor: o launcher só o entrega se o build exigir, e nunca zera o contador 0 | é ele que decide quais grupos aparecem e com que rótulo |
| Falha de geração/verificação = **fail-closed**: não abre o jogo com `.bin` velho apontando para canal morto | evita “não conecta” sem diagnóstico, o pior sintoma para o jogador |
| Sem IP privado nem credencial no artefato publicado | invariante do repositório (`AGENTS.md` §3) |

Ordem de entrega no boot: (1) consultar o registro de canais; (2) verificar o manifesto de patch
(ADR-0003) e aplicar o que faltar; (3) gerar e instalar o `serverlist.bin` de forma atômica;
(4) só então disparar `WYD.exe`. O status HTTP de cada canal (`:80/serv00.htm`) é do **servidor de
jogo**, não do launcher — o launcher só precisa do endereço.

Teste mínimo dessa entrega (offline, sem servidor no ar): gerar o `.bin` para 1 servidor e 1 canal,
decodificar e comparar célula a célula com a entrada; depois gerar 1 servidor e 3 canais com IPs
distintos e provar que a grade muda só nas células 1..3.

### 8.1 Como rodar (implementado em 2026-11-08)

```bash
cargo run -p wyd-launcher -- serverlist tools/client/serverlist-servers.txt client/ [--dry-run]
cargo run -p wyd-launcher -- launch client <manifesto> tools/client/serverlist-servers.txt [--dry-run] [-- args]
```

`launch` executa a ordem do §8 e é **fail-closed**, com código de saída real: manifesto
inacessível/inválido → `2` (nada é escrito, o jogo não inicia); verificação suja → `1`; registro
inválido → `1` (erro nomeado: URL de status > 32 caracteres, porta no endereço de canal, IP
repetido, grade fora de 10×11/11 canais); sucesso → `0`. `serverlist` é o mesmo passo isolado
(gerar + instalar), útil quando só a lista mudou.

A instalação é **atômica com backup**: escreve `serverlist.bin.tmp` no mesmo diretório, copia o
arquivo anterior para `serverlist.bin.bak` quando difere e só então `rename` por cima;conteúdo idêntico não é reescrito (evita reescrever por reescrita e evita backup inútil).

## 9. Identidade única e a tela de entrada (2026-11-09)

Antes desta data o launcher só tinha `serverlist`/`launch`; a skin e os nomes de servidor eram
editados à mão. Agora existe **uma** definição de identidade e ela é consumida por três lados:

| Lado | Como consome |
|---|---|
| Portal | `wyd-portal/src/app/globals.css` declara as variáveis `--bg/--surface/--accent/--space-*` com os valores de `docs/arquitetura/identidade.tokens` |
| Launcher (Rust) | `include_str!` do mesmo arquivo em `crates/wyd-launcher/src/identidade.rs` (parser estrito, `chave = valor`); nenhum literal de cor ou nome em Rust |
| Gerador Node | `tools/client/serverlist.mjs` lê `servidor.N` do arquivo (`identidadeServers()`); não existe mais o nome fixo `NextGen` |

Guarda de divergência: `crates/wyd-launcher/tests/contrato/identidade.rs` compara os tokens com o CSS e
proíbe qualquer cor de 6 dígitos no CSS fora da identidade; `wyd-portal/src/identidade.test.ts`
faz a mesma checagem do lado do portal. Mexer num valor sem mexer no outro quebra o gate.

Dois comandos escrevem a partir da identidade:

```bash
cargo run -p wyd-launcher -- serverlist tools/client/serverlist-servers.txt client/  # + sn.bin
cargo run -p wyd-launcher -- branding client/                                      # Launcher.bmp + .hat
```

**Cadeia da tela de entrada, arquivo por arquivo** (medida nesta máquina):

1. `WYDLauncher.exe` — carrega `%s/Launcher.hat` (string do próprio binário) e não abre o jogo
   antes de `check` verde;
2. `client/Launcher.hat` — HTML de 268 B: `bgcolor="#0f1115"` (= `cor.bg`) e `<img src="Launcher.bmp">`;
3. `client/Launcher.bmp` — BMP 598×369, 24 bpp, 662.778 B, gerado por `branding::bmp()`
   (gradiente `cor.surface` → `cor.bg`, faixa `cor.accent`, wordmark WYD);
4. `client/sn.bin` — 143 B: `d6` + `eserved` no slot 0, `WYD` no slot 1 (identidade), contadores
   `[3, 2, 0…]` (cabeçalho = canais + 1, slot = grupo + 1). É o que dá nome à linha do servidor
   na seleção de canais;
5. `client/serverlist.bin` — 7.040 B, gerado do registro `tools/client/serverlist-servers.txt`
   (grupo 1: status `http://192.168.100.3/serv00.htm`, canal 1 `192.168.100.3`, canal 2 `192.168.100.4`).

A moldura que o `WYD.exe` desenha por cima (`NUI/ServerList.wyt` 321.470 B = **287×280 32 bpp**,
`NUI/main.wyt` 616×300, `mainchat/mainparts/ccmode/popup/iconon/iconoff.wyt`, `UI/*.wyt`) **é
decodificável desde 2026-11-09**: `tools/client/wyt.mjs` lê o cabeçalho de 22 B e o raster
(`22 + w*h*bpp + 8 = tamanho`, confirmado em 256/259 amostras) e exporta PNG/BMP — a prova visual
é `tmp/wyt/ServerList.png` (moldura ornamental + duas colunas de slots, sem texto: o texto é
desenhado pelo `.exe`).

O que continua faltando não é o formato, é a **carga**: `strings client/wyd.exe` mostra `sn.bin`,
`UI\minimap.wyt`, `%s%02d%02d.wyt` e `mesh\*.wyt`, mas nenhuma string literal
`NUI\ServerList.wyt` — o caminho até esse diretório passa por um `%s` montado em runtime. Antes de
substituir o arquivo no `client/` é preciso provar esse caminho (ou aceitar o teste com o cliente
aberto). A segunda aba de cidadania/atributos continua dependente da captura de
`STRUCT_SELCHAR`/`STRUCT_MOB` (FASE-08-6) — checklist de campos na issue.

## 10. Configuração remota assinada (2026-11-11)

Marca, canais e URLs deixam de exigir recompilação: o launcher baixa `launcher.conf` do portal,
confere a assinatura e sobrepõe as chaves à identidade embutida (`configuracao.rs`). O padrão
continua sendo o `include_str!` — mas a janela resolve `<dir>/launcher.conf` sozinha
(`--key-file` ou `WYD_LAUNCHER_CONFIG_KEY`) e cai para a embutida com motivo quando o arquivo
não é confiável.

Formato (UTF-8, LF, `chave = valor`, `#` comenta):

```text
# wyd-launcher-config v1
revisao = 3
portal.base = http://127.0.0.1:3000
launcher.manifesto = /api/launcher/manifesto
launcher.pacote = /api/launcher/pacote
cor.accent = #ff0000
servidor.2 = Novo
assinatura = <64 hex>
```

| Regra | Por quê |
|---|---|
| `assinatura` é **a última linha com conteúdo** e o HMAC-SHA256 cobre os bytes exatos anteriores a ela | texto depois da assinatura não seria coberto por ela; o parser recusa |
| chave **simétrica própria da config** (`--key-file`, `configuracao::carregar_chave`), comparação em tempo constante | a config pode ser re-assinada pelo operador sem tocar no par Ed25519 do pacote; nenhuma chave dentro do binário |
| chave desconhecida, `revisao` ausente, `portal.base` fora de `http://host[:porta]` e caminho sem `/` reprovam **com a linha** | config errada não vira "marca estranha" silenciosa |
| o merge valida o que os consumidores usam (cor, número, nome/slot de canal), não só a sintaxe | config inválida é recusada no `config`, não na hora de pintar a skin |
| instalação atômica com `.bak` (`branding::install`) | config meio escrita é pior que config velha |

Fluxo (publicar → consumir → renderizar sem recompilar):

```bash
# 1. publica: o assinador anexa a linha `assinatura` ao corpo
node tools/launcher/sign-config.mjs tools/launcher/launcher.conf.exemplo \
  tmp/chave.bin dist/launcher/launcher.conf

# 2. consome: baixa do portal, verifica e instala em client/
cargo run -p wyd-launcher -- config client/ http://127.0.0.1:3000 --key-file tmp/chave.bin

# 3. usa a marca/canais remotos no skin e no sn.bin
cargo run -p wyd-launcher -- branding client/ \
  --config client/launcher.conf --key-file tmp/chave.bin
```

`--config` é o caminho de **operador** (estrito: assinatura inválida é erro, não fallback) e
vale para `branding`, `gui`, `serverlist`, `launch` e `retheme`. Sem ele, esses mesmos comandos
resolvem `<dir>/launcher.conf` com a chave de `--key-file` ou de `WYD_LAUNCHER_CONFIG_KEY`.
O `launcher.conf` publicado é servido por `GET /api/launcher/configuracao` (404 quando não há
publicação) e o launcher nunca interpreta os bytes antes de a assinatura conferir.

### 10.1 Resolução da identidade efetiva (a janela que o jogador abre)

A janela não tem parser próprio: `configuracao::resolver` devolve a `Efetiva` (identidade +
URLs + origem + motivo) e `gui::html` desenha exatamente isso. Nenhuma etapa faz `panic` e
nenhum caminho fica sem marca — quando a config não é confiável, o rodapé mostra o motivo.

| Situação | O que o jogador vê |
|---|---|
| `launcher.conf` válido + chave | cor, nomes/slots de canal e links do portal do arquivo; rodapé sem motivo |
| arquivo ausente | identidade embutida + `identidade embutida: launcher.conf ausente em <dir>` |
| sem chave | identidade embutida + motivo pedindo `WYD_LAUNCHER_CONFIG_KEY`/`--key-file` |
| assinatura inválida (adulterado) | identidade embutida + motivo com a recusa |
| assinado mas fora do contrato | identidade embutida + motivo do `validar()` |

O CLI loga a mesma origem (`identidade: remota revisão N · portal ...` ou
`identidade: embutida (<motivo>)`), então o diagnóstico não depende de abrir a janela.
Testes: `configuracao` cobre remoto/ausente/sem chave/adulterado/fora do contrato/binário e a
prova de que **trocar o `launcher.conf` muda a janela sem recompilar**; `gui` cobre paleta,
escape de nome de canal e o motivo visível no fallback.

## 11. Assinatura do pacote de patch (Ed25519, 2026-11-11)

O pacote (`manifesto` + `arquivos/`) é assinado com **Ed25519** pela chave privada do
operador/CI; o launcher confere com a chave **pública embutida**
(`distribuicao::CHAVE_PUBLICA_HEX`) e o `update` não aceita `--key-file`. Isso substitui o
HMAC simétrico da FASE-08-3, em que quem tivesse o binário também tinha a chave que assina.

| Peça | Onde vive |
|---|---|
| chave pública (32 bytes em hex) | constante no binário; trocar é publicar launcher novo |
| chave privada (PKCS#8) | secret do CI; nunca no repositório |
| `assinatura` no pacote | 128 hex sobre os bytes exatos do `manifesto` |
| emissor | `node tools/launcher/sign-package.mjs sign <pacote> <chave.pem>` |

`verify_strict` recusa assinatura malleável; assinatura truncada ou de outra chave é
`DistError::BadSignature` (sem `panic`) e o pacote é recusado **antes** de tocar no
diretório do jogo. O rollback é o do `patch::apply` + `PostVerify`: falha no meio restaura
os arquivos já trocados (`falha_no_meio_do_patch_restaura_o_backup`).

Prova de paridade emissor↔verificador: `tests/golden/assinatura_ed25519.rs` verifica um
pacote assinado pelo Node com a chave embutida; trocar manifesto ou payload (mesmo com
assinatura presente) é recusado. Rotação: gerar par novo (`keygen`), embutir a pública,
distribuir o launcher e só então publicar pacotes com a privada nova.
