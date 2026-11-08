# ADR-0020 — Versão de cliente: `APP_VERSION` é histórico, o gate usa a versão medida do cliente oficial

- **Estado:** aceita (2026-11-08) · **implementada** em `crates/wyd-net/src/session/handshake.rs`
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (contratos de porte, U3/U-A)
- **Governa:** `crates/wyd-protocol/src/wire/header.rs`, `crates/wyd-net/src/session/handshake.rs`
  (gate do login, FASE-08), `crates/wyd-protocol/src/payload/auth.rs`
- **Depende de:** ADR-0002 (estrutura), `docs/arquitetura/protocolo.md` §1,
  `docs/cliente/superficie-conhecida.md`

## Contexto

O legado exige igualdade exata com `APP_VERSION = 7640` (`Source/Code/Basedef.h:102`) no login. O
cliente que este servidor vai atender é o **`client/` (build 7662)**, nosso artefato oficial, e ele
**envia 12000** no campo `ClientVersion` do `MSG_AccountLogin` — offset 92 do payload de 116 B
(`crates/wyd-protocol/src/payload/auth.rs:14`), medido em runtime e registrado em
`docs/cliente/superficie-conhecida.md:20`.

O workspace tem hoje as duas constantes, sem que nenhuma decida nada:

| Constante | Valor | Origem | Consumidores |
|---|---:|---|---|
| `APP_VERSION` | 7640 | `Basedef.h:102` (`crates/wyd-protocol/src/wire/header.rs:31`, `i32`) | **nenhum** (só reexport em `lib.rs:35`, `wire/mod.rs:12`) |
| `PROTOCOL_VERSION` | 12000 | medido no cliente `client/` (`wyd-protocol/src/lib.rs:40`) | `session/handshake.rs` (`VersionGate::from_protocol_version`) e `payload/auth.rs:52` (fixture) |

Até esta decisão não existia gate nenhum: `PROTOCOL_VERSION` só aparecia em fixture. Se o gate
nascesse copiando `APP_VERSION`, **o nosso próprio cliente seria recusado no login**. O **WYD Global
(RaidHut)** — `C:\Users\Canta\AppData\Local\wyd_launcher`, launcher Electron 2.7.3 — é referência
de um **cliente real em funcionamento**, mas é terceiro: o valor dele **não** define o gate.

## Decisão

**`APP_VERSION = 7640` é valor histórico de um cliente específico — não é conjunto aceito de
versões.** Conjunto aceito existiria se o servidor mantivesse uma lista de builds válidos; ele não
mantém: existe **uma** versão esperada por vez.

Adotamos `APP_VERSION = 7640` como **constante histórica** do build de referência (documentação e
rastreabilidade com `Basedef.h`) e `PROTOCOL_VERSION` — a versão medida no cliente oficial
`client/` — como o **valor esperado no login**, porque o servidor atende o nosso cliente, não o
build de 2011 da referência.

Regras que decorrem dela:

1. O gate compara `AccountLogin::client_version` com `PROTOCOL_VERSION`; `APP_VERSION` não é
   comparado com nada, e o comentário da constante diz isso na própria definição.
2. Divergência é **erro tipado** (variante própria em `NetError`/`ProtocolError`), nunca `panic`,
   e conta uma métrica por valor recebido — para saber qual build tentou entrar
   (`AGENTS.md` §3 invariante 2, skill `wyd-rust-net` regra 1).
3. O valor esperado é **configurável no boot**, com o default vindo de `PROTOCOL_VERSION`; aceitar
   lista (`Tolerant`/captura) é permitido em diagnóstico e replay, nunca no caminho de produção.
4. Trocar de cliente (novo build oficial em `client/`) exige **nova medição** do handshake e bump de
   `PROTOCOL_VERSION` — não se copia número de `Basedef.h` nem do cliente de terceiro.

### Onde 7640 aparece hoje (cada ocorrência, com papel)

| Arquivo:linha | O que é | Papel na decisão |
|---|---|---|
| `crates/wyd-protocol/src/wire/header.rs:31` | `pub const APP_VERSION: i32 = 7640;` | **única definição**; documentada como histórico, sem comparação |
| `crates/wyd-protocol/src/lib.rs:35` e `wire/mod.rs:12` | reexports de `APP_VERSION` | superfície pública do protocolo; não decide nada |
| `crates/wyd-net/src/session/handshake.rs:3` | doc do módulo citando o ADR | explicação do gate; não compara |
| `crates/wyd-net/src/session/handshake.rs:101-113` | teste `recusa_a_constante_historica_7640` | prova que 7640 **não** entra |
| `crates/wyd-net/src/metrics.rs:270-277` | teste da métrica por versão (`version="7640"`) | prova que 7640 é **contado** quando tentam entrar |
| `crates/wyd-net/src/error.rs:38-42` | teste de `Display` de `ClientVersionUnsupported` com 7640 | prova que a recusa é erro tipado, sem `panic` |

Fora do workspace, 7640 aparece apenas como **referência histórica do legado**:
`Source/Code/Basedef.h:102` (citado em `docs/arquitetura/contrato-transporte-cpsock.md` §2). O
launcher/packaging e o portal **não** usam 7640: `grep` em `crates/wyd-launcher`, `tools/`,
`wyd-portal/src` e nos manifests de deploy retorna zero ocorrências.

### Como o servidor compara (evidência de código)

| Pergunta | Resposta medida |
|---|---|
| Algoritmo da comparação | **igualdade** — `handshake.rs:72`: `received == i32::from(self.expected)` |
| Faixa, mínimo ou lista? | **não existem**; a única comparação de versão do workspace é essa igualdade (nenhum `contains`/`matches!`/intervalo sobre versão) |
| Versão esperada | `VersionGate::from_protocol_version()` → `PROTOCOL_VERSION` (12 000, `wyd-protocol/src/lib.rs:40`); `VersionGate::new(x)` permite trocar no boot |
| Política | `Strict` (default, produção) recusa; `Tolerant` aceita **qualquer** valor e só registra — só para captura/replay |
| Onde a checagem vive | `crates/wyd-net/src/session/handshake.rs:68-80` (`check`), chamado pela sessão antes do login autenticar |
| O que o `client/` declara | offset 92 do `MSG_AccountLogin` (116 B) = **12000**, medido em runtime (`docs/cliente/superficie-conhecida.md:20`, `payload/auth.rs:14`) |
| Divergência entre pontos do código | **nenhuma**: legado compara igualdade em 7640, nós comparamos igualdade em 12000 — mesmo algoritmo, valor esperado diferente. Não há ponto que aceite faixa nem lista em produção |

## Consequências

- **Positivas:** o cliente oficial entra sem ajuste; o 7640 deixa de ser uma armadilha silenciosa;
  a divergência vira dado observável (métrica de versão por tentativa de login) em vez de "não
  conecta"; a versão esperada passa a ter fonte única (`PROTOCOL_VERSION`).
- **Negativas:** aceitar só 12000 recusa qualquer outro build — inclusive o do WYD Global
  (`blogal`, referência de cliente real em funcionamento, nunca fonte de requisito) — por desenho;
  se um dia houver dois clientes suportados, a decisão vira "conjunto aceito" (ADR nova, com
  medição por build).
- **Invariantes afetados:** nenhum gate fixo em constante histórica; `APP_VERSION` nunca é critério
  de aceitação; captura/replay pode aceitar outras versões, produção não.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Gate em 7640, como o legado | Recusaria o cliente oficial (`client/`, 12000) — o oposto do objetivo do porte |
| Aceitar qualquer versão | Perde o diagnóstico de build incompatível e deixa um cliente de protocolo diferente falar com o servidor |
| Gate por hash do executável | Frágil a patch de launcher e já coberto pelo manifesto do `wyd-launcher` (ADR-0003); a versão no handshake é o que o cliente manda de graça |
| Gate na lista `[7640, 12000]` | Aceitaria um build que não é o nosso e cujo protocolo não foi verificado; 7640 é referência histórica, não cliente atendido |

## Implementação (2026-11-08)

A decisão está no código, não só no texto:

| Peça | Onde |
|---|---|
| gate | `crates/wyd-net/src/session/handshake.rs` — `VersionGate::{new, from_protocol_version, tolerant}` e `check(received, &NetMetrics)` |
| erro tipado | `crates/wyd-net/src/error.rs` — `NetError::ClientVersionUnsupported { received, expected }` |
| métrica por versão | `crates/wyd-net/src/metrics.rs` — `wyd_client_version_total{version="…"}`, `…_accepted_total`, `…_rejected_total` |
| constante histórica | `crates/wyd-protocol/src/wire/header.rs` — `APP_VERSION` documentada como não-gate |

Prova executada em 2026-11-08 (`cargo test -p wyd-net session::`): **4/4 ok** —
`aceita_a_versao_do_cliente_oficial` (12000 → `Ok`), `recusa_a_constante_historica_7640`
(7640 → `ClientVersionUnsupported { received: 7640, expected: 12_000 }`),
`tolerante_aceita_mas_nao_muda_o_esperado` e `gate_e_configuravel_no_boot` (13 001 → `Ok`, 12 000 →
`Err`). Com a feature `metrics`: `cargo test -p wyd-net --features metrics` conta por valor recebido.

## Rastreabilidade

- Issues consumidoras: FASE-08 (transporte/sessão), FASE-08-6 (captura de login)
- Docs atualizadas nesta ADR: `docs/arquitetura/contrato-dispatch-opcode.md` §4 (D-D1/U-A),
  `docs/arquitetura/contrato-transporte-cpsock.md` §11 (U3), `docs/rust/03-progresso.md`
- Evidência: `docs/cliente/superficie-conhecida.md:20` (12000 confirmado em runtime);
  `crates/wyd-protocol/src/lib.rs:40` (`PROTOCOL_VERSION`), `wire/header.rs:26` (`APP_VERSION`)

## Revisão

- Revisar quando: novo build oficial entrar em `client/`, ou quando um segundo cliente passar a ser
  suportado (aí a decisão vira "conjunto aceito", com ADR nova e medição por build).
