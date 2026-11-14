# 00 — Relatório executivo

> Auditoria de 2026-11-13 sobre `docs/arquiva-devops-10` @ `9cfcaa2`. Investiganção somente
> leitura: nada em `crates/`, `client/` ou `roadmap-rust/` foi alterado.

## 1. Veredito em um parágrafo

O projeto **não está começando**: existe um servidor Rust funcional no caminho crítico
(handshake → senha → PIN → seleção → cena → combate/drop → refino → logout), com 810 testes e
evidência de fio em loopback. O que falta para “jogar de ponta a ponta” **não é volume de
código, é layout de cliente**: os pontos de maior retorno (entrada no mundo no cliente real,
item no chão, trade, inventário no fio) estão travados pelo mesmo bloqueio externo — a captura
do build 7662 (`CLIENT-06`). Em paralelo, existe uma dívida silenciosa de **sistemas inteiros
sem nenhuma linha em Rust** (quests story line, eventos, guildas, guerra/castelo, PK/chaos,
party, chat, loja de NPC, baú, teleporte), que não bloqueiam o loop atual mas bloqueiam qualquer
afirmação de “paridade com o WYD2”.

A “missão extra” sobre a qualidade visual do cliente tem resposta **divergente da premissa**:
o build atual não é um cliente 2D mais antigo. É um cliente Direct3D9 com malhas com skin,
ossos e animações (1.707 `.msa`, 903 `.msh`, 81 `.bon`, 776 `.ani`, 1.865 `.wys`). A percepção
de “pior” tem três causas testáveis e uma delas está **provada no log local**: o cliente falha
em criar o *render target* (`Error in Init Render Target Texture`) e roda em 800×600.

## 2. Números medidos nesta sessão

| Métrica | Valor | Como |
|---|---:|---|
| Crates no workspace | **9** + `apps/launcher-ui` | `Cargo.toml` members |
| Arquivos `.rs` em `crates/` | **269** | `find` |
| Linhas em `crates/*/src` | **55.675** | `wc -l` |
| Linhas em `crates/` (com testes/benches) | **75.001** | `wc -l` |
| Testes (`#[test]`/`#[tokio::test]`) | **810** | `grep -c` (bate com a soma por crate) |
| Arquivos em `crates/*/tests/` | **64** | `find` |
| Migrations | **51** (`0001`…`0050` + `README`) | `ls` |
| Opcodes no catálogo Rust | **198** | `opcode_table.rs` (via contrato) |
| Handlers no C++ legado | **63** `case` / 58 arquivos `_MSG_*.cpp` | contrato de dispatch |
| Rotas no Go anterior | **61** | contrato de dispatch |
| Cliente: entradas na raiz | **61** | `ls` |
| Cliente: `Mesh/` | **5.361** arquivos / **189 MB** | `docs/cliente/inventario.md` |
| Cliente: `UI/`, `Env/`, `Effect/` | 342 / 522 / 468 | idem |
| `unwrap()` em `src` | **124** | `grep` (36 adicionais em `tests`) |
| `unsafe` real | **0** | 5 ocorrências são comentários afirmando ausência |
| `TODO/FIXME/todo!/unimplemented!` | **16**, todos em testes | `grep` |

## 3. Dez achados que mudam decisão

| # | Achado | Classificação | Evidência |
|---:|---|---|---|
| 1 | O loop de entrada está fechado no fio: paridade alta até a cena autoritativa. | Confirmado | `docs/rust/08-checklist-funcionalidades.md` §1 (1.5–1.11, 1.16, 1.17) + testes `tests/sistema/{login,selchar,pin_*,sessao_unica,lotacao}.rs` |
| 2 | `0x0117 NewAccountFail` **não existe** no roteador do 7662: senha errada não desenha nada. | Fortemente indicado | `docs/cliente/pacotes.md` §“O que o build 7662 realmente trata” (zero `cmp` para o opcode) → causa raiz de `BUG-01` |
| 3 | `CreateItem` (`0x026E`) está catalogado como C→J, mas o binário o **recebe** (S→C). | Divergente | `docs/cliente/pacotes.md` §“Caixa de item” (`0x495dc5 cmp edx,0x26e`) |
| 4 | O cliente 7662 compara **91** opcodes do nosso catálogo de 198; o resto é silêncio. | Confirmado (contagem por desassemblage) | mesmo documento |
| 5 | O build distribuído em `client/` **já vem com os 3 checksums desativados** (`EB` no disco) — rejeitar por checksum quebraria o alvo. | Confirmado | `docs/cliente/superficie-conhecida.md` §2 |
| 6 | O cliente tem pipeline 3D completo (skin + bones + anim), **não** é 2D; a “piora” visual é configuração/ambiente + volume de assets. | Divergente / Fortemente indicado | `04-client-analysis.md` §2, `19-client-visual-generations.md` |
| 7 | `client/` é um **repack de comunidade** (ClientPatch GPL-3.0, `XPva03.dll`, `JProtect`, sem `Graphics.ini`), não uma instalação retail da Ongame. | Fortemente indicado | inventário local + `docs/cliente/inventario.md` §2 |
| 8 | O legado **não está na árvore**: vive só em `origin/main` (C++ 70.987 linhas + Go 99.606). Auditar “o legado” exige `git show origin/main:…`. | Confirmado | `analise-portabilidade-legado.md` §2; `ls` da raiz |
| 9 | Metade dos opcodes do legado nunca passa pelo socket do jogador (TMSrv↔DBSrv na 7514). | Confirmado | `contrato-dispatch-opcode.md` §3 |
| 10 | A documentação está **à frente do código em alguns pontos e atrás em outros** (ex.: checklist diz 1.11 ✅; portabilidade ainda descreve `wyd-net` com 458 linhas e 7 crates). | Divergente | `15-bugs-and-divergences.md` `DIV-08` |

## 4. Riscos por severidade

**P0 — integridade/negabilidade**

| Risco | Estado | Nota |
|---|---|---|
| Duplicação de item/economia por layout adivinhado | **Contido por desenho** | `require_ported` recusa opcode UNVERIFIED (`payload/mod.rs:219`); trade segue fechado. Não afrouxar. |
| Checksum rejeitante | **Evitado** | O build alvo exige checksum não rejeitante; “endurecer” aqui quebra o cliente. |
| Segredo em log/git | **Sem achado** | `client/` fora do git; `.env` presente e ignorado; testes afirmam que PIN/senha não vazam. |

**P1 — paridade funcional**

1. Sistemas ausentes por completo: quests (story/Kefra/runas/Arch), eventos, guildas, guerra/
   castelo, PK/chaos, party, chat, loja de NPC, baú, teleporte, montaria, fadas/familiares no fio.
2. Layouts UNVERIFIED que travam quatro frentes: `STRUCT_SELCHAR`, `STRUCT_MOB` S→C,
   `MSG_Trade`, id de item no chão/`ESCENE_FIELD` de resposta.
3. Sem regen/respawn próprio; morte remove o mob e ele só volta com recarga de definição.

**P2 — engenharia**

4. 124 `unwrap()` em `src` (não são `panic!` de entrada de rede, mas são pontos de queda).
5. `systems/{skill,compose,inventory}` prontos e **sem fio** — regra que não chega ao jogador.
6. Documentação duplicada/desatualizada (PROC-01 aberta).

## 5. O que esta auditoria **não** prova

- Nada sobre runtime: **nenhum** `cargo test`, `docker compose up` ou consulta ao Postgres foi
  executado nesta sessão. “Provado” aqui significa “existe teste/evidência registrada no repo”.
- Nada sobre o cliente ao vivo: sem VM, sem GUI, sem captura de pacote (`CLIENT-06`).
- Nada sobre o legado por leitura linha a linha: as afirmações vêm dos contratos já publicados.
- Nenhuma fórmula de dano/EXP/drop foi validada como “a do WYD2”: as fontes externas são
  engenharia reversa comunitária (nível C/D em `13-external-references.md`).

## 6. Cinco próximas ações, em ordem

1. **Destravar a captura** (`CLIENT-06`): é a raiz de 4 das 6 frentes bloqueadas. Sem ela, todo
   o resto é estimativa.
2. **Corrigir o catálogo de direção de `CreateItem`** (rito próprio, tabela congelada) — é dado
   errado que já enganou o contrato.
3. **Fechar o loop de combate**: regen, respawn e HP/MP no fio (4.4 da matriz) — hoje a morte
   funciona, mas o mundo não se recupera sozinho.
4. **Ligar o que já existe**: skill e compose têm regra testada e nenhum opcode. É o maior
   ganho de paridade por esforço.
5. **Reconciliar a documentação** (PROC-01): o checklist e a análise de portabilidade divergem
   entre si sobre o mesmo código.

## 7. Leitura recomendada

- Decisão de produto/prioridade → este documento + [`17-priorities-and-roadmap.md`](17-priorities-and-roadmap.md).
- Trabalho técnico do dia → [`05-feature-parity-matrix.md`](05-feature-parity-matrix.md) + [`15-bugs-and-divergences.md`](15-bugs-and-divergences.md).
- Cliente e customização → [`04-client-analysis.md`](04-client-analysis.md) + [`09-client-customization.md`](09-client-customization.md) + [`19-client-visual-generations.md`](19-client-visual-generations.md).
- Antes de codar perto de protocolo → [`06-protocol-catalog.md`](06-protocol-catalog.md) + [`18-unknown-features.md`](18-unknown-features.md).
