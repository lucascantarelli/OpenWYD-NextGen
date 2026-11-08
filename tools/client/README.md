# `tools/client/` — análise estática do cliente

Pipeline reproduzível de engenharia reversa do `client/WYD.exe` (build 7662). Não é código
de jogo: entra `WYD.exe`, saem artefatos versionados de análise. Plano: `docs/cliente/engenharia-reversa.md`.

## Comando único

```bash
node tools/client/analyse.mjs
```

Pré-requisitos: Node 20+ (testado em v26). `objdump` (binutils) é **opcional**: se existir,
enriquece `gerado/ancoras.txt` com o desassemble das âncoras; se não, o relatório registra
`desassemble indisponível` e o resto continua.

O comando **falha (exit 1)** se o SHA-256 do executável não for o do build de referência —
offset de outro build não é evidência.

## Organização

| Arquivo | Responsabilidade | Contrato |
|---|---|---|
| `pe.mjs` | ler o formato PE32/PE32+ | `parsePe(buf)`, `vaToOffset`, `offsetToVa`, `readImports`, `readExports`, `readRichHeader` — sem I/O na leitura |
| `strings.mjs` | indexar strings ASCII/UTF-16 e classificar | `extractStrings`, `classify`, `indexStrings` |
| `keytable.mjs` | tabela `pKeyWord[512]` e transformação CPSock | `loadKeywordTable`, `transform` (mesmas regras de `crates/wyd-protocol`) |
| `evidence.mjs` | confirmar/refutar endereços herdados | `ANCHORS`, `PATCH_SITES`, `analyseEvidence`, `disassemble` |
| `formats.mjs` | estrutura inicial de `.wyt`, `.msa`, `sn.bin`, `serverlist.bin` | `analyseFormats` |
| `extract-tables.mjs` | tabelas de dados (`ItemList.bin`, `SkillData.bin`): XOR, registro e perfil de colunas | CLI única; grava `docs/cliente/gerado/tabelas.json` |
| `analyse.mjs` | orquestrar e gravar artefatos | CLI única; nada de lógica de análise aqui |
| `captura.mjs` | gerar (`--gen`) e validar (`--check`) goldens do caminho login→campo | CLI única; fontes só de `payload.rs` + tabela de opcodes | CLI única; nada de lógica de análise aqui |

## Padrões

- Um módulo, uma responsabilidade; `analyse.mjs` só compõe.
- Toda função de análise é **pura** em relação ao arquivo lido; I/O só na borda (CLI).
- Saída **determinística**: JSON com 2 espaços, listas ordenadas, sem data/hora e sem caminho
  absoluto. Rodar duas vezes no mesmo build produz bytes idênticos.
- Nenhum offset é aceito sem evidência: o status é `CONFIRMADO` ou `REFUTADO`/`NÃO LOCALIZADO`,
  com bytes e seção. `UNVERIFIED` continua sendo marcado como tal, nunca promovido por dedução.
- Nenhum binário do cliente é copiado para o repositório.

## Artefatos gerados (`docs/cliente/gerado/`)

| Arquivo | Conteúdo |
|---|---|
| `pe.json` | cabeçalho, seções, diretórios, imports, exports, rich header |
| `strings.json` | totais e strings relevantes (arquivos de dados, rede, proteção, protocolo, versão, UI) |
| `evidencia.json` | âncoras, patches de checksum, CPSock (tabela/INITCODE/limite), resumo |
| `formatos.json` | amostras de `.wyt`/`.msa`, `sn.bin`, `serverlist.bin` (inclui tentativa de decodificação) |
| `ancoras.txt` | desassemble das âncoras e dos patches (quando `objdump` existe) |
| `goldens.json` | resultado da validação de `fixtures/cliente/*.json` |

## Limitações registradas

- Sem Ghidra/IDA/rabin2 e sem Python neste ambiente: não há decompilação nem xrefs completos.
  Isso impede nomear funções além das âncoras e impede fechar a semântica de campo do header
  de 12 bytes por análise estática — fica para a 08-6 (trace dinâmico).
- `objdump` confirma prólogo e instruções, não assinatura de função; a classificação usa
  prólogo + fronteira de instrução e é declarada como tal.
