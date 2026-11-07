# Plano de engenharia reversa do cliente

> Este documento é **plano**, não resultado. Ele define alvos, ferramentas, marcos,
> entregáveis e critérios de aceite do desassemble profundo do `WYD.exe` 7662.
> Execução: `roadmap-rust/issues/FASE-08-5` (estático), `08-6` (dinâmico) e `08-7`
> (customização). Nada aqui autoriza redistribuir binário ou asset.
> Mapa consolidado (VAs confirmados, relação com o launcher): `docs/cliente/mapa.md`.

## 1. Objetivo

Ter o **máximo de controle e conhecimento possível** sobre o cliente fechado, de forma
reproduzível:

- saber **onde** cada mecânica vive (endereço, assinatura, structs, xrefs) e **como** o
  binário reage ao que o servidor emite;
- permitir customização **sustentável** (patch por versão, não por memória) e evolução da UI;
- alimentar o `wyd-launcher` com manifesto de integridade e versão confiáveis;
- transformar cada descoberta em documento com **build + hash + endereço**, dentro de
  `docs/cliente/`.

## 2. Princípios

1. **Estático antes de dinâmico.** Não se coloca breakpoint no que não foi lido no desassemble.
2. **Uma descoberta, uma evidência.** Endereço + hash do build + trecho de instrução. Sem isso é
   UNVERIFIED.
3. **Nada de "quase igual".** Offset de outro build não é offset deste build.
4. **Não invadir o servidor.** O cliente é hostil por definição; conhecimento de cliente nunca
   reduz validação no servidor.
5. **Ordem que falha barato.** Cada marco tem um entregável que pode invalidar o seguinte.

## 3. Alvos priorizados

| # | Alvo | Por que primeiro | Saída esperada |
|---|---|---|---|
| T1 | Pipeline de rede (`ReadMessage`, `SendPack`, `SendPacket`) | é o contrato com o servidor Rust; já há endereços conhecidos | mapa de funções + sequência de parse por opcode |
| T2 | Structs de jogo (`STRUCT_MOB`, `STRUCT_ITEM`, `STRUCT_SELCHAR`, cena) | divergência de layout é a causa nº 1 de misparse | offsets confirmados no binário, com tamanho e alinhamento |
| T3 | Loop de cena/render (`ESCENE_FIELD`, tick, interpolação) | explica "cliente congela"/"não atualiza barra" | ordem de atualização por frame e pontos de bloqueio |
| T4 | Entrada e UI (`.wyt`) | customização visual e comandos | estrutura de `.wyt` + mapa de handlers de input |
| T5 | Assets (`.msa`, `object.bin`, `ItemList.bin`) | conteúdo e ferramentas de edição | header/formato mínimo viável por arquivo |
| T6 | Login/seleção de personagem | fecha o ciclo launcher → login → cena | fluxo exato de `MSG_AccountLogin`/`CNFAccountLogin` |
| T7 | XTrap / JProtect | risco de crash e de banimento em teste | estado real (ativo/inativo) e efeitos observáveis |
| T8 | Updater (`WYDLauncher.exe` + `settings.txt`) | o `wyd-launcher` precisa substituí-lo sem regressão | protocolo de update, manifesto e fallback |

T7 é **diagnóstico**, não objetivo: se estiver ativo, o plano é contornar por configuração
documentada, não derrotar o anticheat.

## 4. Ferramentas

| Ferramenta | Uso | Observação |
|---|---|---|
| **Ghidra** (headless + GUI) | desassemble/decompilação, xrefs, RTTI, scripts de análise em lote | sem PDB; nomes vêm de assinatura e string |
| **x64dbg** (+ ScyllaHide) | breakpoints, trace de execução, dump de memória | build é **x86**, usar a versão 32-bit |
| **Detect It Easy (DIE)** | compiler, packer, entropia, seções | confirmar "sem packer" do inventário |
| **Wireshark / proxy TCP** | captura do CPSock entre cliente e servidor | combinar com `golden` do `wyd-protocol` |
| **Process Monitor / API Monitor** | I/O de arquivo, registro, winsock | prova de qual arquivo é lido em cada tela |
| **Node (`tools/client/`)** | parsing de PE, indexação de strings, diff de seções, geração de relatório | mesma linguagem do gerador de protocolo; sem Python garantido no host |
| **`cargo` + `wyd-protocol`** | reimplementar o que foi descoberto e comparar byte a byte | o teste é a prova de que o entendimento está correto |

Regra de ferramenta: nada proprietário obrigatório. O fluxo tem de rodar com Ghidra + x64dbg +
Node; IDA é conforto, não dependência.

## 5. Marcos

| Marco | Entrega | Aceite |
|---|---|---|
| **M0 — baseline** | hash, seções, imports, exports, resources, strings indexadas | relatório em `docs/cliente/` reproduzível por comando (feito na 08-4 para hash) |
| **M1 — mapa bruto** | funções nomeadas por assinatura (checksum, tabela de keyword, `ReadMessage`, `SendPack`, parse de `serverlist.bin`) | ≥ os 7 endereços já conhecidos **reproduzidos** por análise independente, não copiados |
| **M2 — structs** | `STRUCT_MOB`/`ITEM`/`SELCHAR` confirmadas por acesso a campo, mais structs de cena | offsets e tamanhos documentados; bater com `data-formats.md` e com o servidor — **não fechado na 08-5** (sem decompilador no ambiente); transferido para a 08-6 |
| **M3 — cena/tick** | ordem por frame, fontes de atualização, pontos de congelamento | explicação causal do freeze registrado em `investigacao-freeze-cliente.md` |
| **M4 — dinâmico** | trace de pacotes reais, dumps de memória, breakpoints nos hooks | captura de login→seleção→campo com golden por opcode |
| **M5 — correlação** | diff cliente × servidor Rust por pacote | divergência zero nos pacotes cobertos; as demais marcadas UNVERIFIED |
| **M6 — customização** | camada de patch sustentável + manifesto do launcher | patch aplicado por versão/hash, reversível, com rollback — ✅ implementado e testado (`crates/wyd-launcher`, 19/19; spawn real pendente) |

Cada marco só fecha com o documento correspondente atualizado **na mesma entrega**.

## 6. Artefatos (onde cada coisa mora)

| Artefato | Caminho | Estado |
|---|---|---|
| Narrativa e regras do dossiê | `docs/cliente/` | ✅ |
| Inventário e hashes | `docs/cliente/inventario.md` | ✅ |
| Superfície conhecida | `docs/cliente/superficie-conhecida.md` | ✅ |
| Mapa de funções (gerado, revisado) | `docs/cliente/mapa-funcoes.md` | ⬜ 08-5 |
| Structs confirmadas | `docs/cliente/structs.md` | ⬜ 08-5 |
| Catálogo de pacotes e goldens | `docs/cliente/pacotes.md` + `fixtures/cliente/` | 🚧 08-6 (goldens C→S gerados; captura S→C pendente) |
| Scripts de análise | `tools/client/` (PE, strings, diff, relatório) | ⬜ 08-5 |
| Camada de patch e manifesto | `docs/cliente/customizacao.md` + `crates/wyd-launcher` | ⬜ 08-7 |

Scripts em `tools/client/` seguem a mesma disciplina do gerador de protocolo: saída versionada,
com cabeçalho "GERADO/ANÁLISE" e comando de reprodução no topo.

## 7. Riscos

| Risco | Impacto | Mitigação |
|---|---|---|
| Offset de outro build | crash silencioso, falso aprendizado | hash obrigatório no topo de todo documento; 08-5 reconfirma os endereços |
| XTrap ativo | instabilidade em máquina de teste | diagnóstico antes (T7); testar em VM descartável, nunca na máquina de trabalho |
| Licença/GPL do ClientPatch | obrigação de redistribuição | patch próprio documentado ou uso local do GPL-3.0 sem vendorar |
| Tempo de desassemble | programa longo sem entrega | marcos pequenos e entregáveis por marco |
| Distribuição de binário/assets | violação de direito autoral | dossiê registra hash/endereço; nunca conteúdo |

## 8. Definição de pronto do programa

1. Todo alvo T1–T8 tem documento com build + hash + evidência.
2. O servidor Rust reproduz os pacotes cobertos byte a byte (golden em `wyd-protocol`).
3. O `wyd-launcher` valida o cliente por manifesto e aplica patch reversível.
4. Nenhum achado UNVERIFIED virou gate de servidor.
5. Nada de binário do cliente entrou no git.
