# Sistema — Autenticação e conta

> **Paridade global: P4.** É o sistema mais bem coberto do projeto: senha, PIN, sessão, limite por
> IP e recusa por lotação têm teste e fio. O único buraco visível é a **mensagem de erro**
> (`DIV-01`).

## 1. Escopo investigado

Login, logout, segunda senha (PIN), validação, bloqueio, sessão, reconexão, expiração, limite de
tentativas, limite por IP, seleção de canal, recusa por lotação/manutenção, mensagens.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_AccountLogin.cpp`, `_MSG_AccountSecure`, `_MSG_CharacterLogout.cpp` |
| Cliente | `0x020D` (116 B, versão 12000), `0x0FDE`/`0x0FDF`, `0x0215`; `sn.bin` + `serv00.htm` |
| Rust | `cpsock/{handshake,login}.rs`, `auth/{password,pin}.rs`, `character/session.rs`, `boot/healthcheck.rs` |
| Banco | `0030_sessao_unica`, `0040_capacidade_canal`, `0041_papel_do_canal` |
| Testes | `login.rs`, `pin_persistencia.rs`, `pin_troca.rs`, `sessao_unica.rs`, `limite_ip.rs`, `lotacao.rs`, `handshake.rs` |
| Doc | `docs/wyd/senha-numerica.md`, `docs/cliente/pacotes.md` |

## 3. Regras confirmadas

| Regra | Valor | Fonte |
|---|---|---|
| Hash de senha | Argon2id, conferido **antes** do `CNF` | `auth/password.rs` |
| PIN | 4–6 dígitos, Argon2id em `accounts.pin_hash` | `auth/pin.rs` |
| Bloqueio de PIN | 5 falhas → 10 min persistidos (`pin_failures`, `pin_locked_until`) | `pin_guard` |
| Primeira entrada | cadastra o PIN | `primeira_entrada_cadastra_o_pin` |
| Troca de PIN no jogo | 2 etapas (atual → novo), `change_numeric` | `pin_troca.rs` |
| Troca de PIN no portal | exige PIN atual, `POST /api/panel/pin` | `portal_pin.rs` |
| Sessão única | 1 personagem/conta entre canais | `0030` |
| Ociosidade | 120 s | `selchar.rs` |
| Limite por IP | padrão 6, `0` desliga | `limite_ip.rs` |
| Lotação | `Capacity` antes de abrir sessão; `WYD_SERVER_MAX_PLAYERS` | `lotacao.rs` |
| Manutenção | `WYD_SERVER_MAINTENANCE=1` recusa | idem |
| Timeout de handshake | 10 s | `cpsock` |
| Versão | 12000 | handshake |

## 4. Fluxo resumido

```
TCP → INITCODE → 0x020D → senha (Argon2id) → 0x010A (roster)
    → 0x0FDE (PIN; bloqueio se 5 falhas) → 0x0213 → 0x0114 (STRUCT_MOB)
    → cena; logout 0x0215 → 0x0116 → libera sessão
```

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Login | Implementado e testado | P4 |
| PIN + bloqueio | Implementado e testado | P4 |
| Troca de PIN (jogo/portal) | Implementado, semântica própria | P4 |
| Sessão única | Implementado e testado | P4 |
| Logout/ociosidade | Implementado e testado | P4 |
| Limite por IP | Implementado e testado | P4 |
| Lotação/manutenção | Paridade confirmada | P5 |
| **Erro de senha visível** | Divergência (`0x0117` ausente) | **P2** |
| Criação de conta pelo jogador | não existe (staff/portal) | P1 |
| Reconexão automática | não existe (sessão nova) | P0 |

## 6. Lacunas e ações

1. Emitir `0x0101` com texto para senha/versão/PIN errados (`BUG-01`).
2. Confirmar `change_numeric` com captura (`UNK-13`).
3. Definir política de reconexão (hoje é sessão nova; semântica do legado desconhecida).
4. Teste de sessão órfã após crash do canal.

## 7. Riscos

| Risco | Mitigação atual |
|---|---|
| Força bruta de PIN | bloqueio persistido + limite por IP |
| Vazamento em log | testes afirmam que PIN/senha não aparecem |
| Duas sessões | sessão única no banco |
| Enumeração de conta | mensagem única (mas **invisível**, `DIV-01`) |
