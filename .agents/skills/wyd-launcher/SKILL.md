---
name: wyd-launcher
description: Build, update and UI of the game launcher (Rust core crates/wyd-launcher + planned Tauri v2 GUI): integrity manifest, atomic patch, signed distribution, serverlist delivery, portal integration (news/status), NSIS installer. Use when touching launcher code, UI, update flow, installer metadata or the launcher roadmap issue.
---

# Launcher do jogo (`wyd-launcher` + GUI)

Norma: `docs/cliente/launcher.md`, `docs/cliente/customizacao.md` §§3–4, ADR-0003 e
ADR-0007. Fluxo de update, manifesto, patch e contrato do portal estão lá; não os copie.

## Decisões específicas

- A GUI chama o núcleo (`manifest::verify`, `distribuicao::update`, `bootstrap::spawn`);
  não reimplementa escrita no disco do jogo.
- Fail-closed: assinatura/verificação vermelha impede abrir o jogo e mostra mensagem
  acionável. Nada toca `client/` sem backup.
- Launcher não guarda senha; login é opcional e via portal HTTPS. O jogo autentica no CPSock.
- Integração de notícias/status/manifesto usa endpoints GET públicos; URL configurável por
  canal e nenhum segredo no binário.
- Teste de patch em pasta temporária; instalador/NSIS só depois de instalar → atualizar →
  jogar contra stack local.

## Prova

`cargo test -p wyd-launcher`; `make check`.
