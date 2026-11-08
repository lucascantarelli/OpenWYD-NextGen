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

## 6. Integração com o portal (proposta)

O launcher GUI busca no portal (HTTPS, somente leitura): notícias/avisos,
status dos canais (para montar `serverlist.bin`), versão/manifesto de patch
e links (cadastro, suporte). Conta/login no launcher v1 = **opcional**
(o jogo autentica no CPSock; o launcher não guarda senha).

## 7. Plano (issue GUI criada: FASE-08-8, decisão Tauri + nome WYD em 2026-11-07)

1. ADR GUI (Tauri v2 + fronteiras com `wyd-launcher`).
2. `crates/wyd-launcher-ui` (ou `apps/`): janela, botão Jogar, barra de
   progresso de patch, painel de notícias via portal, `serverlist.bin`.
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
| O launcher **gera** o `.bin` pelo mesmo algoritmo (`10 × 11 × 64`, soma de `obfuscationKey[63-i]`), a partir do registro de canais versionado | implementação única em `crates/wyd-launcher/src/serverlist.rs`; o Node (`tools/client/serverlist.mjs`) fica como referência de operação e a paridade de **bytes** é teste (`tests/serverlist_paridade.rs`) |
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
arquivo anterior para `serverlist.bin.bak` quando difere e só então `rename` por cima; conteúdo
idêntico não é reescrito (evita reescrever por reescrita e evita backup inútil).
