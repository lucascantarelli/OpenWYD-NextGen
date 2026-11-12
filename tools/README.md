# `tools/` — bibliotecas de análise, captura e geração

A **porta de entrada** de análise é o app [`apps/tools`](../apps/tools/README.md)
(`wyd-tools inventario|binarios|strings|pacotes|docs|zip-check`). Esta pasta
continua sendo a **biblioteca** usada por ele (parsers de PE/strings, captura e
formatos) e pelos scripts de operação. Nada aqui é runtime, e não há segundo CLI
de documentação: `docs/cliente/re-*.md` só é gerado pelo `wyd-tools`.

| Pasta | O que tem | Quando usar |
|---|---|---|
| `apps/tools/` | CLI e lib do app | inventário, análise PE/strings, protocolo, documentação e verificação de `client.zip` |
| `tools/client/` | parser PE, `strings`, `.wyt`, `serverlist`, captura ao vivo | **biblioteca** de engenharia reversa e captura (`tmp/captura*` é dado de entrada, não código) |
| `tools/protocol/` | `gen-protocol.mjs` | regenerar tabelas de protocolo |
| `tools/content/` | seeds, catálogo do banco, enciclopédia, migrations | conteúdo e validação do banco |
| `tools/launcher/` | manifest, canais, assinatura, E2E da janela | distribuição do launcher |
| `tools/dev/` | `bench-gate`, `monitoring-smoke`, `clean-tmp` | gates de operação |

## Regras

- Ferramenta **nunca** vira dependência de runtime; não é importada por `crates/`.
- Caminho de dado de teste: `crates/wyd-protocol/tests/fixtures/cliente/`.
- Artefatos gerados (`docs/cliente/inventario-gerado.md`, `client.zip`) não são
  editados à mão; regenere pelo comando documentado.
- Script descartável nasce em `tmp/`; `tools/` só recebe o que tem propósito
  declarado aqui (política em `docs/arquitetura/visao-pastas.md` §1.1).
