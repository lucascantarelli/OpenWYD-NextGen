# ADR-0007 — Interface gráfica do launcher em Tauri v2

- **Status:** aceita (2026-11-07, decisão do dono).
- **Governa:** CLIENT-08 (`roadmap-rust/issues/CLIENT-08-launcher-gui-tauri.md`).

## Contexto

O núcleo do launcher existe e é testado (`crates/wyd-launcher`: `check`/
`update`/`launch`, patch atômico, assinatura — CLIENT-03, 27 testes), mas é
só CLI. O jogador precisa de janela: notícias, progresso de patch, botão
Jogar. O launcher oficial de referência (instalado nesta máquina,
`C:\Users\Canta\AppData\Local\wyd_launcher` 2.7.3) é Electron + Squirrel:
~100 MB de Chromium embutido por cópia, update por `.nupkg`, jogo em
subpasta, `launcher.json` só com versão. Análise completa em
`docs/cliente/launcher.md`.

## Decisão

GUI em **Tauri v2**: backend Rust (reusa `wyd-launcher` como crate, sem
reimplementar escrita no disco), frontend web (mesmo padrão do portal),
WebView2 no Windows (já vem no Win10 1803+/Win11), **updater com assinatura
obrigatória** (casa com ADR-0003), instaladores NSIS/MSI prontos (~5–10 MB).
Integração v1 com o portal só leitura: notícias, status dos canais (para
montar `serverlist.bin`), manifesto de patch. Sem senha no launcher.

## Consequências

- Positivas: binário pequeno, mesma linguagem do servidor, updater assinado
  por padrão, tela de notícias reaproveita padrão do portal.
- Negativas: exige WebView2 (pré-instalado no Windows atual; o instalador
  Tauri providencia se faltar) + toolchain Node para o frontend.
- Neutras: marca/janela/instalador usam o nome **WYD** (pós-rename 2026-11-07).

## Alternativas consideradas

- Electron (referência oficial): descartado — duplica o Chromium que o SO já
  tem; ~10× o tamanho para a mesma função.
- egui/iced (nativo puro): descartado por padrão — sem HTML, a tela de
  notícias não reaproveita nada do portal. Só reconsiderar com PoC se o
  WebView2 virar impedimento real.
- Manter só CLI: descartado — jogador não abre terminal;launcher invisível
  não é produto.
