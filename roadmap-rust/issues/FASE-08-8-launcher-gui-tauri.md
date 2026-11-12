# Issue 08-8 — GUI do launcher em Tauri v2

**Fase:** 8 · **Depende de:** 08-3, 08-7 · **Bloqueia:** FASE-07 (distribuição) · **Esforço:** L · **Status:** 🚧

## Objetivo

Dar rosto ao núcleo `wyd-launcher`: janela WYD com notícias do portal, barra
de progresso de patch, botão Jogar adaptativo (Instalar/Jogar/Atualizar) e
instalador NSIS por canal — sem reimplementar nenhuma escrita no disco do jogo.

## Contexto / evidência

Núcleo pronto (08-3: `check`/`update`/`launch`, 27 testes). Decisão
ADR-0007 (Tauri v2). Referência de mercado: launcher oficial Electron +
Squirrel (`docs/cliente/launcher.md` §3). Skill vigente:
`.agents/skills/wyd-launcher`.

## Passos

1. Escorar Tauri v2 no workspace sem contaminar as crates (`tauri.conf.json`,
   frontend mínimo, `wyd-launcher` como dependência — capability só o
   necessário).
2. Telas: notícias/status via portal (GET público), progresso de
   `distribuicao::update`, botão Jogar → `bootstrap::spawn` (fail-closed:
   verificação vermelha = jogo não abre).
3. `serverlist.bin` montado do status dos canais; instalador NSIS com nome,
   ícone e metadados **WYD** (pasta do usuário, sem exigir administrador).
4. Teste em máquina limpa: instala → abre → atualiza → joga; pacote
   adulterado recusado na GUI como na CLI; `make check`.

## Arquivos (lista fechada)

- Criar: frente Tauri (`src-tauri/`, `ui/` ou `apps/launcher-ui/` — definir na
  execução, sem espalhar), teste de fluxo documentado.
- Alterar: `crates/wyd-launcher` só para expor API que a GUI precise (nada de
  lógica duplicada); `docs/cliente/launcher.md` (§7) ao concluir.
- Proibido: reescrever `patch`/`distribuicao`, tocar `client/` sem backup,
  versionar `serverlist.bin` com IP privado, guardar senha no launcher,
  dependência nova sem ADR + PoC (FASE-14).

## Testes / verificação

```bash
cargo test -p wyd-launcher
make check
```

Saída esperada: gates verdes + evidência do fluxo instalar → atualizar →
jogar em pasta temporária + recusa de pacote adulterado na GUI.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-launcher - regras da GUI e do núcleo.
- .agents/skills/wyd-client-connect - serverlist e diagnóstico.
- .agents/skills/rust-code-quality - testes.

## Documentação de referencia

- docs/cliente/launcher.md - conceito, referência Electron, plano.
- docs/arquitetura/adr/0007-gui-launcher-tauri.md - decisão.
- roadmap-rust/issues/FASE-08-3-launcher.md - núcleo reutilizado.

## Subtasks

- [x] Frente Tauri mínima (2026-11-12): `apps/launcher-ui` (bin `WYDLauncher`) consome `configuracao::resolver_flags` + `lancamento::iniciar`; janela serve o HTML da identidade efetiva e o Jogar abre o jogo; `--instalar <dir>` copia o executável para a pasta do jogo (instalado em `client/`, com `.bak` do launcher original)
- [x] Arte da janela (2026-11-12): ICO fornecido pelo dono em `apps/launcher-ui/icons/icon.ico` e fundo embutido como `data:` URI (`branding::fundo_data_uri`), com teste de determinismo
- [x] E2E WebDriver real (2026-11-12): `tools/launcher/e2e-janela.mjs` (tauri-driver + msedgedriver) abre a janela do exe na pasta do jogo, confere DOM/notícia, clica Jogar (1 processo) e fail-closed sem escrita
- [x] Portal do launcher (2026-11-12): `/configuracoes/launcher` (publica `launcher.conf` assinado + histórico) e `/monitoramento/launcher` (telemetria por versão + operação)
- [x] Cliente real jogável (2026-11-12): `client/manifest.txt` derivado de 7.131 arquivos reais e `client/canais.txt` derivado de `serverlist.bin`/`sn.bin` (`dist-manifest.mjs`/`canais.mjs`, ambos com `--check`); manifesto aceita caminho com espaço; `cd client && ./WYDLauncher.exe --check` = 7.131 íntegros, exit 0
- [x] Instalação/atualização por `client.zip` (2026-11-12): leitor ZIP (stored/deflate) em `cliente_zip.rs`, origem configurável (`WYD_LAUNCHER_CLIENT_ZIP`/`WYD_LAUNCHER_CLIENT_MANIFEST`, local ou URL via `curl`), verificação por manifesto e `serverlist.bin` sempre regenerado
- [x] Reparo por hash (2026-11-12): `reparo.rs` + rotas `/api/launcher/cliente/{manifesto,arquivo/<caminho>}`; arquivo adulterado é rebaixado do portal e restaurado ao hash do manifesto (teste Rust com servidor HTTP + prova real em `client/`)
- [x] Harness na pasta real (2026-11-12): `node tools/launcher/e2e-janela.mjs --cliente client` abre a janela com manifesto/canais reais e jogo falso, prova um processo e nenhuma mensagem de manifesto
- [ ] Barra de progresso de patch + instalador NSIS
- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] GUI instala/atualiza/joga via núcleo, sem caminho próprio de escrita.
- [ ] Notícias + status vêm do portal; pacote adulterado recusado na GUI.
- [ ] Instalador NSIS WYD instala em pasta do usuário; `make check` verde.

## Riscos / rollback

- Risco: frontend pesado virando segundo produto → mitigação: mesma stack do
  portal, telas mínimas v1 (notícias + status + Jogar).
- Rollback: remover frente Tauri; CLI continua valendo (08-3 intacta).
