# Auditoria completa do WYD2 — índice e método

> **Executada em:** 2026-11-13 · **Árvore:** `docs/arquiva-devops-10` @ `9cfcaa2`
> (“Arquiva a DEVOPS-10 com a v0.1.0 publicada”). `develop` e `main` presentes na mesma árvore.
> **Natureza:** investigação **somente leitura**. Nenhum arquivo de `crates/`, `wyd-portal/`,
> `client/` ou `roadmap-rust/` foi alterado; a auditoria produziu apenas este diretório.
> **Escopo:** ecossistema completo — legado (`origin/main`), Rust (`crates/` + portal +
> launcher), cliente `client/` (build 7662), `docs/`, referências externas e possibilidades de
> customização.

## 1. Por que este diretório existe

O repositório já tinha conhecimento disperso: `docs/cliente/` (dossiê do 7662),
`docs/manual-técnico/analise-portabilidade-legado.md` (de-para legado→Rust),
`docs/rust/08-checklist-funcionalidades.md` (mapa funcional), `docs/wyd/` (lore e sistemas) e
`roadmap-rust/` (trabalho). O que faltava era **um lugar único que cruze as três bases**
(legado × cliente × Rust) com evidência, classificação de confiança e lacuna declarada.

Este diretório **não substitui** os documentos acima: ele os cita e reconcilia. Onde há
divergência entre documentação e código, o código vence e a divergência fica registrada em
[`15-bugs-and-divergences.md`](15-bugs-and-divergences.md).

## 2. Regras de execução aplicadas

| Regra do prompt | Como foi cumprida |
|---|---|
| Não remover/simplificar funcionalidade | Todo sistema encontrado entra na matriz, inclusive os não implementados; `docs/wyd/sistemas.md` lista 20 sistemas, a matriz cobre os 20 + os transversais. |
| Não presumir implementação correta | “Existe código” nunca foi tratado como “está portado”: a coluna *Ligado ao fio?* vem do checklist e do dispatch real (`cpsock/dispatch`). |
| Rastreabilidade | Toda linha aponta arquivo, caminho, teste ou medição; conclusão sem fonte aparece marcada como Hipótese. |
| Classificação | Tabela do §3. |
| Não modificar o projeto | Só `docs/audit/` foi criado. Nenhum offset, layout, issue ou seed foi tocado. |
| Pesquisa contínua | Cada achado que abriu domínio novo (ex.: `object.bin` não é lista de objetos) gerou entrada em [`18-unknown-features.md`](18-unknown-features.md). |

## 3. Classificação de confiança

| Marca | Significado | Exemplo neste diretório |
|---|---|---|
| **Confirmado** | Medido nesta árvore ou provado por teste reproduzível. | `WYD.exe` = 2.347.008 B, SHA-256 `04c9d2a1…27ddf8`; 810 testes em `crates/`. |
| **Fortemente indicado** | Evidência técnica consistente, falta confirmação definitiva. | `0x0117 NewAccountFail` ausente do roteador do cliente (desassemblage, sem captura ao vivo). |
| **Hipótese** | Interpretação plausível, ainda não validada. | Bytes 12–20 do `Config.bin` controlarem textura/efeito. |
| **Desconhecido** | Sem informação suficiente. | Layout de `STRUCT_MOB` no sentido servidor→cliente. |
| **Divergente** | Fontes conflitam. | `CreateItem` (`0x026E`): `Basedef.h` diz C→J, o binário 7662 prova S→C. |
| **Não aplicável** | Não se aplica à versão/arquitetura. | `dnSpy`/`ILSpy` para o `WYD.exe` (nativo PE32, sem .NET). |

## 4. Status e paridade

**Status** (mais preciso que “feito/pendente”):

`Não investigado` · `Investigação em andamento` · `Identificado no legado` · `Identificado no cliente` ·
`Não iniciado em Rust` · `Planejado em Rust` · `Em desenvolvimento` · `Implementado parcialmente` ·
`Implementado sem integração` · `Integrado, mas não validado` · `Implementado e testado` ·
`Paridade funcional confirmada` · `Paridade parcial` · `Divergência identificada` ·
`Bloqueado por dependência` · `Descontinuado na referência` · `Não aplicável`

**Paridade:**

| Nível | Critério usado aqui |
|---|---|
| P0 | Nenhuma implementação correspondente. |
| P1 | Estrutura inicial ou implementação superficial. |
| P2 | Fluxo principal parcialmente implementado. |
| P3 | Fluxo principal funcional, com divergências. |
| P4 | Equivalente, com exceções/casos extremos pendentes. |
| P5 | Equivalência comportamental comprovada por teste **e** pelo fio. |

> **P5 é raro de propósito.** Nesta árvore, só chegam perto de P5: transporte CPSock,
> senha/PIN, seleção de personagem, pacote autoritativo de cena e refino — e AINDA assim com
> ressalva de captura do cliente. `Implementado com teste` sem emissão/recepção no fio fica em P3.

## 5. Entregáveis (22 do §15.1 do prompt)

| # | Entregável | Documento |
|---:|---|---|
| 1 | Inventário do projeto | [`01-project-inventory.md`](01-project-inventory.md) |
| 2 | Inventário da implementação Rust | [`03-rust-analysis.md`](03-rust-analysis.md) |
| 3 | Inventário do legado | [`02-legacy-analysis.md`](02-legacy-analysis.md) |
| 4 | Inventário do cliente | [`04-client-analysis.md`](04-client-analysis.md) |
| 5 | Matriz geral de paridade | [`05-feature-parity-matrix.md`](05-feature-parity-matrix.md) |
| 6 | Matriz específica do cliente | [`04-client-analysis.md`](04-client-analysis.md) §4 |
| 7 | Catálogo de protocolos | [`06-protocol-catalog.md`](06-protocol-catalog.md) |
| 8 | Fluxos de comunicação | [`07-network-flows.md`](07-network-flows.md) |
| 9 | Mapeamento de eventos | [`07-network-flows.md`](07-network-flows.md) §5 · [`systems/events.md`](systems/events.md) |
| 10 | Catálogo de funcionalidades | [`05-feature-parity-matrix.md`](05-feature-parity-matrix.md) |
| 11 | Catálogo de itens | [`11-items-and-equipment.md`](11-items-and-equipment.md) |
| 12 | Recursos gráficos | [`10-assets-and-formats.md`](10-assets-and-formats.md) · [`12-maps-and-animations.md`](12-maps-and-animations.md) |
| 13 | Relatório de engenharia reversa | [`08-reverse-engineering.md`](08-reverse-engineering.md) |
| 14 | Clientes e códigos-fonte disponíveis | [`13-external-references.md`](13-external-references.md) |
| 15 | Comparação de ferramentas de RE | [`08-reverse-engineering.md`](08-reverse-engineering.md) §2 |
| 16 | Customização | [`09-client-customization.md`](09-client-customization.md) |
| 17 | Arquitetura Rust | [`14-rust-architecture-audit.md`](14-rust-architecture-audit.md) |
| 18 | Bugs e divergências | [`15-bugs-and-divergences.md`](15-bugs-and-divergences.md) |
| 19 | Estratégia de testes | [`16-test-strategy.md`](16-test-strategy.md) |
| 20 | Roadmap | [`17-priorities-and-roadmap.md`](17-priorities-and-roadmap.md) |
| 21 | Dúvidas e desconhecidos | [`18-unknown-features.md`](18-unknown-features.md) |
| 22 | Relatório executivo | [`00-executive-summary.md`](00-executive-summary.md) |

**Missão extra** (qualidade visual do cliente entre gerações):
[`19-client-visual-generations.md`](19-client-visual-generations.md).

**Recortes por sistema** (detalhe que não cabe na matriz):
`systems/authentication.md`, `characters.md`, `movement.md`, `combat.md`, `inventory.md`,
`items.md`, `npcs.md`, `quests.md`, `events.md`, `guilds.md`, `economy.md`, `citizenship.md`,
`additional-systems.md`.

## 6. Cobertura declarada (sem porcentagem inventada)

Não existe metodologia de amostragem que sustente “X% auditado”. O que existe é a lista do que
foi **lido/medido diretamente**, o que veio de **evidência já registrada no repositório** e o que
**não foi possível executar**:

| Base | Cobertura real |
|---|---|
| Estrutura do repositório, crates, LOC, contagem de testes, migrations, ferramentas | **Medido** nesta sessão (comandos no §7). |
| Cliente `client/`: inventário, hashes, contagem de assets, tamanho de tabelas, log de execução | **Medido** (leitura de diretório + `od`/`ls`); conteúdo binário **não** foi decodificado além do já publicado em `docs/cliente/`. |
| Protocolo: framing, opcodes, direções, tamanhos | **Evidência do repo** (`docs/arquitetura/contrato-dispatch-opcode.md`, `docs/cliente/pacotes.md`) gerada por desassemblage/`Basedef.h`. Não re-executei `objdump` nesta sessão. |
| Legado C++/Go | **Evidência do repo** (`analise-portabilidade-legado.md`, contrato de dispatch). O código vive em `origin/main`, não na árvore; não foi re-medido linha a linha nesta sessão. |
| Comportamento em runtime (stack Docker, banco, canal) | **Não executado** nesta auditoria (nenhum `docker compose up`, nenhum `cargo test`). |
| Cliente ao vivo (GUI, VM, captura de pacote) | **Não executado** — depende da máquina do usuário/VM (`CLIENT-06`). |
| Referências externas | **Pesquisadas** em 2026-11-13, com URL e nível de confiança em `13-external-references.md`. |

Consequência prática: **nada aqui afirma comportamento de runtime observado nesta sessão.**
Onde o documento diz “provado no fio”, a prova é o teste citado, registrado no repositório.

## 7. Como reproduzir as medições

```bash
# inventário Rust
for c in crates/*/; do printf "%-20s %s\n" "$(basename $c)" "$(find $c/src -name '*.rs' | wc -l)"; done
find crates -name '*.rs' | xargs wc -l | tail -1
grep -rn '#\[test\]\|#\[tokio::test\]' crates --include=*.rs | wc -l
ls crates/wyd-db/migrations | wc -l

# cliente (não versionado; pasta local)
cd client && sha256sum WYD.exe && ls -1 | wc -l
for d in Mesh UI Env Effect NUI shader; do printf "%-8s %s\n" "$d" "$(ls $d | wc -l)"; done
od -An -tx1 -N30 Config.bin        # modo de vídeo e flags
tail -5 WYD.log                    # erro de render target

# legado (somente leitura, sem checkout)
git show origin/main:Source/Code/Basedef.h | wc -l
git ls-tree -r --name-only origin/main | wc -l
```

## 8. Convenções de identificação

- **`PAR-<sigla>-NN`** — linha da matriz de paridade (`05-feature-parity-matrix.md`).
- **`CLI-NN`** — linha da matriz do cliente (`04-client-analysis.md` §4).
- **`PKT-0xNNNN`** — pacote no catálogo (`06-protocol-catalog.md`).
- **`UNK-NN`** — pergunta aberta (`18-unknown-features.md`).
- **`DIV-NN`** — divergência/bug (`15-bugs-and-divergences.md`).
- **`VIS-NN`** — recurso visual comparado entre gerações (`19-client-visual-generations.md`).

## 9. Manutenção

Este diretório é **fotografia datada**, não documento vivo de contrato. Quando uma issue mudar o
comportamento, atualize a linha correspondente na mesma entrega (mesmo rito de
`docs/rust/08-checklist-funcionalidades.md`). Se um item daqui virar regra de código, ele precisa
de teste — a auditoria não promove hipótese a contrato (`AGENTS.md` §2.4).
