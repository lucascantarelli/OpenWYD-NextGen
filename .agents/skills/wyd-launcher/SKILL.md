---
name: wyd-launcher
description: Build, update and UI of the game launcher (Rust core crates/wyd-launcher + planned Tauri v2 GUI): integrity manifest, atomic patch, signed distribution, serverlist delivery, portal integration (news/status), NSIS installer. Use when touching launcher code, UI, update flow, installer metadata or the launcher roadmap issue.
---

# Launcher do jogo (`wyd-launcher` + GUI)

Núcleo pronto: `crates/wyd-launcher` (CLI `check`/`update`/`launch`;
`manifest`, `patch` atômico, `bootstrap`, `distribuicao`, `sha256`).
Conceito e referências: `docs/cliente/launcher.md`. Estratégia de patch:
`docs/arquitetura/adr/0003-estrategia-patch-cliente.md`. Contrato de entrega:
`docs/cliente/customizacao.md` §3–§4.

## 1. Regras invioláveis

1. A GUI **chama** o núcleo (`manifest::verify`, `distribuicao::update`,
   `bootstrap::spawn`) — nunca reimplementa escrita no disco do jogo.
2. Fail-closed: verificação vermelha = jogo não abre, mensagem clara.
3. Pacote sem assinatura válida é recusado **antes** de tocar o disco.
4. Nunca escrever em `client/` sem backup; nunca versionar `serverlist.bin`
   com IP privado, `WYD.log`, dumps ou binários.
5. O launcher **não guarda senha**: conta/login no launcher é opcional e só
   via portal (HTTPS); o jogo autentica no CPSock.
6. Nome/marca **WYD** em janela, instalador, metadados e `LAUNCHER_NAME`
   (já `wyd-launcher`; sem prefixo morto).

## 2. Fluxo de trabalho

1. Mudança de lógica de update/patch → teste em pasta temporária, nunca em
   `client/` (padrão CLIENT-03: adulterado recusado exit 1, resume, backup).
2. Mudança de GUI → rodar o fluxo instalar → atualizar → jogar contra stack
   local antes de qualquer instalador distribuível.
3. Integração portal (notícias/status/manifesto): só endpoints GET públicos;
   sem segredo no binário; URL base configurável por canal.
4. Instalador NSIS: nome, ícone e metadados do projeto novo; preferir pasta
   do usuário (evita exigir administrador como o launcher oficial).

## 3. Verificação mínima

```bash
cargo test -p wyd-launcher
cargo fmt --all -- --check && cargo clippy --workspace --all-targets -- -D warnings && cargo test --workspace
```
