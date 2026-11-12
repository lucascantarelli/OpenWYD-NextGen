# `apps/tools` — ferramentas de análise do projeto

App de linha de comando para engenharia reversa e geração de documentação do
cliente. **Nunca é runtime**: nada aqui entra no servidor, no portal ou no
launcher; é leitura de bytes e geração de artefatos (docs/pacotes).

```bash
# resumo medido do cliente (arquivos, bytes, PEs, SHA-256)
node apps/tools/bin/wyd-tools.mjs inventario client
node apps/tools/bin/wyd-tools.mjs inventario client --json

# análise PE por binário: seções, entrypoint, imports por DLL, exports, PDB,
# strings com offset/VA e candidatos a função (heurística declarada)
node apps/tools/bin/wyd-tools.mjs binarios client

# tabela completa de strings classificadas de um binário
node apps/tools/bin/wyd-tools.mjs strings client --bin WYD.exe

# protocolo: catálogo de opcodes × handlers no servidor × structs × capturas
node apps/tools/bin/wyd-tools.mjs pacotes

# regenera TODOS os docs derivados em docs/cliente/ (inventário + RE)
node apps/tools/bin/wyd-tools.mjs docs client

# confere o pacote de distribuição contra o manifesto antes de publicar
node apps/tools/bin/wyd-tools.mjs zip-check client.zip --manifest client/manifest.txt

# testes do app (node:test, sem dependências)
node --test apps/tools/test/
```

## O que cobre

| Camada | Como |
| --- | --- |
| Inventário do cliente | `lib/inventario.mjs`: caminhada determinística, bytes por extensão/diretório, SHA-256 e resumo PE por `.exe`/`.dll` (parser único em `tools/client/pe.mjs`) |
| Binários PE | `lib/pe-analise.mjs`: seções, entrypoint, imports por DLL, exports, PDB (CodeView/RSDS), strings indexadas (`tools/client/strings.mjs`), candidatos a função e **variáveis globais candidatas** (referências absolutas a seções de dados) |
| Protocolo | `lib/pacotes.mjs`: `opcode_table.rs`, referências `Opcode::*` em `crates/wyd-server/src`, `struct`+`OPCODE` em `payload/*.rs` e capturas em `tmp/` |
| Documentação | `lib/re-docs.mjs` + `lib/docs.mjs`: Markdown sem timestamp, ordenado, com o comando de regeneração em cada arquivo |
| Pacote `client.zip` | `lib/zip.mjs`: lê diretório central, infla stored/deflate e confere cada entrada do manifesto (tamanho + SHA-256) |
| Geração do ZIP | `scripts/client-zip.ps1`: entradas relativas (sem prefixo `client/`), sem artefatos mutáveis |

## Heurística de funções (inferência, não nome)

`lib/pe-analise.mjs` conta, na seção de código:

1. destinos de `call rel32` (`E8`) e `jmp rel32` (`E9`) que caem dentro da seção;
2. prólogos x86 `55 8B EC`, `55 89 E5`, `8B FF 55 8B EC`;
3. exports nomeados são a única fonte de **nome** medido.

Variáveis globais candidatas usam referências absolutas (`A1/A3` e `8B/89/FF` com
`modrm=00 r/m=101`) cujo destino cai em `.data`/`.rdata`/`.bss`: o endereço é
medido, a existência de um global ali é inferência e o nome não é inventado.

Não há desmontagem completa: não há nomes de função, tipos de variável ou
assinatura de struct do executável. O que a doc marca como "inferência" vem
dessa contagem; "medido" vem do parser PE, do catálogo Rust ou da captura.

## Limitações registradas (não preenchidas com suposição)

- Capturas JSON em `tmp/` guardam frames cifrados; o `Type` só é decodificável
  com a keyword da sessão, ausente no arquivo. A doc lista apenas os tipos que
  já estão em texto (ex.: `diff.json`) e marca o resto como não decodificado.
- `WYD.exe` passa por `JProtectDll`/`XTrap`; os candidatos a função são
  subconjunto (destinos diretos + prólogos), não o mapa completo de rotinas.
- Nomes de campo de struct do cliente exigiriam RTTI/PDB; o PDB é procurado no
  diretório de debug e a ausência é registrada por binário.
