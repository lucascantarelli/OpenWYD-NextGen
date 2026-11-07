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
