# Issue 51 — Launcher próprio com auto-update e login pelo portal

**Fase:** 9 · **Depende de:** 19 (sessão/play code), 32 (contas), 49 (versão do protocolo) · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Substituir a distribuição por link de fórum por um **launcher** que valida a cópia do cliente, aplica
patch por hash, gera o `serverlist.bin` do servidor e autentica pelo portal. Desenho em
[`docs/analysis/cliente-launcher.md`](../../docs/analysis/cliente-launcher.md) §4.

## Contexto

- O cliente é fechado (DX9, 32-bit); o que muda entre versões são **assets e o `serverlist.bin`**.
- Já existe `IssuePlayCode` (código de uso único) e sessão web — o launcher não precisa inventar login.
- Hoje o `serverlist.bin` é gerado à mão com `serverlist editor.exe` e existem 4 pares com IP privado
  **versionados** (a limpeza é a issue 56).

## Passos

1. **Fase 1 (MVP)**: app Go/Windows com tray — lê `manifest.json` (versão + sha256 por arquivo), baixa
   só o que mudou, aplica patch atômico (`tmp` → `rename`), gera `serverlist.bin`, botão "jogar".
2. **BFF**: `/api/launcher/manifest` e `/api/launcher/file/<sha>` (autenticados ou públicos só para
   assets), `cache-control` correto, ETag.
3. **Fase 2**: login pelo portal (OIDC/sessão), play code, "lembrar conta", status de bloqueio.
4. **Fase 3**: notícias, status do servidor, auto-reparo, telemetria opt-in.
5. **Fase 4**: manifesto assinado (ed25519), mirror, beta channel, rollback de versão.

## Arquivos

- `launcher/` (novo módulo Go + manifest schema), `portal/src/app/api/launcher/**`,
  `portal/src/app/(admin)/launcher/page.tsx` (canal/versão), `docs/analysis/cliente-launcher.md`,
  skill `wyd-launcher`.

## Verificação

```bash
# unitários do patcher (hash, diff, patch atômico, retomada)
go test ./launcher/...
# ponta a ponta local: manifesto em disco + servidor fake
go test -tags=e2e ./launcher/... -run TestPatchFlow
cd portal && npx tsc --noEmit && npx vitest run
```

## Critérios de aceite

- [ ] Launcher abre, valida hash, atualiza e inicia o jogo sem tocar em arquivo em uso
- [ ] `serverlist.bin` gerado pelo launcher aponta para o servidor configurado no portal
- [ ] Login pelo portal devolve play code e o cliente entra
- [ ] Conta bloqueada: launcher recusa antes de abrir o jogo
- [ ] Nenhum binário de terceiro versionado; manifesto e patch assinados

## Riscos

- **Antivírus**: assinar o launcher, publicar hashes, nunca reempacotar `WYD.exe`.
- **Patch corrompido**: verificação dupla (hash do arquivo e do patch) + rollback automático.
- **Legal**: distribuir só o que é nosso; o cliente vem da cópia do jogador.
