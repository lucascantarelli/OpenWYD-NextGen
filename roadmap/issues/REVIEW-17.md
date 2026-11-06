# REVIEW-17 — Revisão permanente: cliente WYD e engenharia reversa

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle do cliente Windows (7662/12000), do `ClientPatch_v7662` e do launcher: o que foi
medido, o que continua hipótese e o que pode ser alterado sem quebrar a conexão.

## Escopo

- Build do cliente em uso, versão de protocolo efetivamente enviada e offsets conhecidos.
- `ClientPatch_v7662`: patches aplicados, o que cada um resolve, dependência (ou não) do servidor.
- Launcher: manifesto, verificação de integridade, `serverlist.bin`, atualização.
- Assets: ícones, modelos, HUD, fontes, resolução — o que é seguro trocar.
- Comunicação: contratos observados no cliente, não os documentados no servidor.
- Toda afirmação carrega marca **CONFIRMADO** (medido) ou **UNVERIFIED** (hipótese).

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Versão enviada | `MSG_AccountLogin.ClientVersion` no log do aceite |
| Offsets usados | tabela em `docs/client.md` com build e data da medição |
| Patch do instalador | hash do arquivo + inventário de patches em `Source/Code/ClientPatch_v7662` |
| Assets trocados | lista com origem e licença |
| Achados novos | uma linha por achado, com "como foi medido" |

## Critérios de aceite (por rodada)

- [ ] Nenhuma afirmação sobre o cliente sem evidência de medição ou marca UNVERIFIED
- [ ] `client/` continua fora do git e do contexto de build Docker
- [ ] Toda alteração de cliente tem plano de reversão (backup do binário original)
- [ ] Diferença entre protocolo do cliente e do servidor detectada antes de release

## Histórico

| Data | Achados | Destino |
|---|---|---|
| 06/11/2026 | `ClientPatch_v7662` sem dependência de build/runtime no servidor (só comentários e solution) | `docs/analysis/legado-clientpatch.md`, REVIEW-07 (fechada) |
| 06/11/2026 | ESC → "Servidor" não pede troca ao servidor: nenhum frame chega, exige hook no patch | CANAL-03 (aberta) |
| 06/11/2026 | mapa de offsets do build 12000 não existe formalizado | issue 52 (aberta) |
