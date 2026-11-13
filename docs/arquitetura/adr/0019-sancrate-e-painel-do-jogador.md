# ADR-0019 — SancRate do legado e fronteira de confiança do painel do jogador

**Status:** aceito · **Data:** 2026-11-08 · **Fases:** 04 (dado) e WEB-02 · **Depende de:** ADR-0008
(transporte Cold Path), ADR-0009 (conteúdo estático)

## Contexto

1. **Dado de refino incompleto.** O seed `0022_seed_refino.sql` deixou o passo `+10 → +11` sem
   linha e usou **3500 bps (35%)** no `+9 → +10`, um "ponto médio" da faixa 30–40% — valor
   inventado. A pesquisa mostrou o caminho real: `+10 → +11` é **refino com Poeira de
   Lactolerium (Lac)**, 3% com poeira normal e 100% com o item de cash "Lac 100%"; não é
   composição. O arquivo `tmserver/content/Common/Settings/SancRate.txt` do legado
   (`origin/main`, sha256 `80e90f4a…`) tem a tabela de sucesso por pó e índice de refino
   (PO 0–5, PL 0–9, Âmago 0–11) — 28 células que nunca entraram no banco.
2. **Painel do jogador sem contrato.** A WEB-02 precisa de conta, personagens, ranking e
   recuperação. O transporte (ADR-0008) autentica **processo→processo** com um token de staff;
   não há JWT de jogador verificável em Rust (sem dependência de cripto aprovada) nem binário do
   servidor para emitir sessão ao backend.

## Decisão

### 1. Dado de taxa: literal, sem interpolação

- Nova tabela `sanc_rate(powder, refine_index, percent, content_source, confidence, notes)` com
  PK `(powder, refine_index)` e **exatamente** as 28 linhas do `SancRate.txt`. Índice ausente não
  é preenchido: ausência é dado (o loader do legado mantém o default).
- `refine_rate` corrigido: `+9 → +10` = **3000 bps (30%)** (doc §2, não a média da faixa) e
  `+10 → +11` = **300 bps (3%)**, `kind = 'dust'`, fonte "wikis WYD: PL +10 → +11", com a
  divergência do `SancRate` Âmago (5%, configuração de outro servidor) registrada em `notes`.
- `docs/wyd/sistemas/refino.md` §2 passa a citar 3% (Lac) em vez de "30% (composição)".

### 2. Painel do jogador: BFF confiável, escopo do cookie

- Rotas `POST /admin/v1/panel/{self,characters,rankings}` e `/accounts/recovery` no mesmo
  roteador do ADR-0008.
- O **BFF** (portal, mesmo host, loopback, token processo→processo) injeta `account_id` lido do
  cookie `httpOnly` assinado; o browser nunca envia nem escolhe esse id. O backend valida
  `account_id > 0`, devolve só dados daquela conta e audita `panel.*` com o alvo interno.
- Ranking não leva `account_id` (dado público: nome, classe, nível; top ≤ 100).
- Recuperação não escreve senha: registra a solicitação auditada (`accounts.recovery`,
  `target = login:*`) e responde **o mesmo** resultado exista ou não o login. O reset segue ato
  de staff (WEB-03) — não há provedor de e-mail nem tabela de token aprovados.

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| Verificar o JWT do portal no Rust agora | Exigiria `jsonwebtoken`/`ring`/`hmac` (deps novas, ADR-0006/0017) e um segredo compartilhado; a WEB-01 já tem esse item aberto para quando houver binário |
| Endpoint público de ranking sem sessão | Expõe o serviço de administração a qualquer processo local; hoje o token é a única barreira |
| Preencher `+11` com 5% do SancRate Âmago | Valor existe, mas é de outra linha/configuração (passo especial `GEM_LEVEL`); a taxa publicada do caminho PL é 3% |
| Deixar `+11` sem linha | O dado existe (3% com Lac); furo declarado só quando não houver fonte |

## Consequências

- `SCHEMA_VERSION` = 25; 25 migrations, 60 tabelas, 463 colunas; `refine_rate` 15 linhas,
  `sanc_rate` 28; ciclo up/down/up idêntico.
- Quem editar taxa no portal no futuro lê a **mesma** tabela que o motor (SancRate literal), sem
  valores derivados.
- O painel depende do BFF ser honesto quanto ao `account_id`; é o mesmo modelo de confiança do
  `moderator_id` da sessão (ADR-0008 §2) e fica documentado para a fiação JWT da WEB-01.

## Evidência

`cargo test -p wyd-admin --test contrato_painel` (6 casos: 401, conta, personagens escopados, ranking
ordenado sem `account_id`, recuperação sem vazar existência, parâmetros inválidos) e
`cargo test -p wyd-db` (catálogo/`SCHEMA_VERSION`). Números e corpos em
`roadmap-rust/changelog.md` (WEB-02) e `docs/rust/evidencias/WEB-02-painel-vivo.md`.

## Emenda 2026-11-08 — `sanc_rate` é a tabela **efetiva**, não o arquivo

A auditoria do refino (`docs/wyd/sistemas/refino-auditoria.md`) mostrou que o item 1 desta
ADR estava incompleto em dois pontos:

1. **Ausência de linha não é ausência de taxa.** O loader do legado parte do default
   compilado (`Basedef.cpp g_pSancRate`) e o `SancRate.txt` só sobrescreve os índices que
   lista. Um índice não listado vale **0**, não "sem dado". A migração
   `0026_seed_sanc_rate_defaults.sql` acrescenta as 8 células que faltavam (PO 6–11, PL 10–11),
   levando `sanc_rate` a **36 células** e `SCHEMA_VERSION` a **26**. A regra de "não
   interpolar" continua: nenhum valor foi estimado, todos vêm do default compilado.
2. **A célula Âmago 10/11 (5%) não é a taxa do +10 → +11.** `BASE_GetSuccessRate`
   curto-circuita `level == GEM_LEVEL` para **15%** (ou 100% na linha Âmago) e não consulta a
   tabela. O 5% é inalcançável pelo caminho de pó; a taxa real do passo é 15% no legado e 3%
   na fonte pública (valor que o `refine_rate` mantém).

A leitura da tabela é em **`nível + 1`**, documentada em `refine_rate`/`sanc_rate` e provada
por `crates/wyd-sim/tests/golden/refino_seed.rs`.
