# CLIENT-12 — Comparação visual do 7662: modo de vídeo, render target e gerações

> **ID canônico:** `CLIENT-12`
> **ID histórico:** `CLIENT-12`
> **Estado 2026-11-13:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Teste manual / investigação visual · **Esforço:** S · **Depende de:** cliente 7662 e GPU/Windows local · **Bloqueia:** decisão “houve downgrade?” e o runbook de setup gráfico

## Objetivo

Responder, com print e log, se a percepção de “cliente atual mais feio que o antigo” vem de
**configuração/ambiente** (modo 800×600, falha de render target) ou de **conteúdo** (assets).
A auditoria (`docs/audit/19-client-visual-generations.md`) levantou 5 hipóteses; esta issue
falsifica as duas primeiras.

## Contexto / evidência (medido 2026-11-13)

- `client/Config.bin` (30 B) está em **modo 7 = 800×600×32**.
- `client/WYD.log`: `VertexShader HW Accel Enabled` + **`Error in Init Render Target Texture`**.
- `client/` **não tem** `Graphics.ini` nem `config_settings.json` (o cliente 7.69 do Global tem).
- Assets: `Mesh/` 5.361/189 MB e `Effect/` 468/4,6 MB contra 7.947/558 MB e 582/18,3 MB no 7.69.
- Cliente é D3D9 com skin/bones/776 animações — **não** é um cliente 2D mais simples.

## Pré-condições

1. Backup completo de `client/` (`cp -r client client.bak-validacao`).
2. Stack no ar para chegar ao server select (não precisa entrar no mundo).
3. Um segundo ambiente (outro PC/VM ou driver diferente) para o teste de render target.

## Passos

| # | Variável | Ação | O que registrar |
|---:|---|---|---|
| 1 | Baseline | abrir no modo atual e capturar tela | print + 5 linhas do `WYD.log` |
| 2 | Resolução | trocar `Config.bin` byte 2 de `07` para `08` (1024×768) | print no mesmo ponto + log |
| 3 | Driver/GPU | repetir em outro ambiente | o erro de render target desaparece? |
| 4 | Efeitos | comparar uma skill/efeito no mesmo mapa nos dois modos | print do efeito |
| 5 | Bytes 12–20 | variar **um** byte por vez (com backup) | o texto/efeito muda? log |
| 6 | Referência externa | comparar com print do Global (mesma cidade) | diferença visível |

## Arquivos

- Atualizar: `docs/audit/19-client-visual-generations.md` (§5 e §6 com resultado),
  `docs/cliente/inventario.md` (modo de vídeo documentado), `UNK-34`/`UNK-35` em
  `docs/audit/18-unknown-features.md`.
- Proibido: versionar `client/`, redistribuir asset, editar binário sem backup.

## Critérios de aceite

- [ ] Tabela com 3 configurações (atual, 1024×768, outro ambiente) e print de cada uma.
- [ ] Veredito explícito: H1 (resolução) e H2 (render target) confirmadas ou refutadas.
- [ ] `Config.bin` final documentado (modo escolhido) e backup preservado.
- [ ] Se H1/H2 não explicarem, registrar que a diferença é de conteúdo (H3) e abrir item de assets.

## Execução 2026-11-13 (tentativa sem GUI)

- `client/WYD.exe` iniciado com o `Config.bin` atual (modo 7) e encerrado após 25 s: processo vivo, sem crash.
- `client/WYD.log` novo, idêntico ao anterior: `20:08:42 VertexShader HW Accel Enabled` → `20:08:43 Error in Init Render Target Texture`.
- **Nenhuma conexão CPSock** chegou ao canal nesse intervalo (`docker compose logs server-canal-1`
  só mostra o serviço de status), ou seja, o cliente não chegou ao server select.
- O processo é protegido (`JProtectDll`): `Stop-Process`/`taskkill` sem elevação dão acesso negado.
- **Conclusão parcial:** H2 (render target) continua viva e reprodutível; falta a tela para dizer se o
  cliente para no logo ou segue para a lista. Sem input sintético, os passos 1–6 seguem humanos.

## Se falhar

Se o cliente não abrir ou não desenhar, registre o erro exato do `WYD.log` e o ambiente
(Windows, GPU, driver). Isso vira `UNK-34`, não bug de servidor.
