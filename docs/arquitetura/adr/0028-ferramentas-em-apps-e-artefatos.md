# ADR-0028 — Ferramentas em `apps/tools` e política de artefatos

**Status:** aceita (2026-11-12)
**Contexto:** `tools/` acumulou scripts de análise, geração e operação sem uma
porta de entrada única; `tmp/` chegou a 147 MB, `target/` a 23,7 GB e o
`client.zip` (186 MB) vivia na raiz sem regra de versionamento/publicação.

## Decisão

1. **`apps/tools/` é o app de análise** (Node, sem dependências): CLI
   `wyd-tools` com `inventario`, `docs` e `zip-check`, biblioteca em
   `apps/tools/lib/` e testes em `apps/tools/test/` (`node --test`). `tools/`
   continua como biblioteca de RE/captura/geração — o app **reusa**
   `tools/client/pe.mjs` em vez de duplicar formato.
2. **`client.zip` não entra no git.** É artefato de distribuição, derivado de
   `client/` + `manifest.txt`; publica-se em GitHub Releases (recomendado) ou
   storage com link direto, e o launcher baixa por `WYD_LAUNCHER_CLIENT_ZIP`
   (ver `docs/cliente/launcher.md` §7.1).
3. **`target/` é cache de build** (ignorado no git). Não é fonte; limpeza é
   `make clean-target` quando o disco incomodar — nunca “apagar por apagar” no
   meio de uma sessão de testes.
4. **`tmp/` é descartável** (ignorado): política em
   `docs/arquitetura/visao-pastas.md` §limpeza, com `make clean-tmp` removendo
   só artefatos conhecidos (logs, drivers, pastas de E2E), preservando capturas
   nomeadas pelo operador.
5. **Observabilidade mora em `docker/monitoring/`** (Prometheus, Grafana, Loki,
   Alloy), junto dos demais artefatos de container; compose e docs apontam para
   o caminho novo.
6. **Fixtures de teste ficam junto do teste**: `crates/wyd-protocol/tests/fixtures/cliente/`
   (goldens de pacote), sem pasta `fixtures/` na raiz.

## Consequências

- Uma porta de entrada (`wyd-tools`) para inventário/documentação/verificação de
  pacote; scripts antigos continuam válidos e documentados em `tools/README.md`.
- Nenhuma mudança de comportamento em runtime; só caminhos e geração de docs.
- `client.zip` passa a ter verificação reproduzível (`zip-check`) antes de
  qualquer publicação.
