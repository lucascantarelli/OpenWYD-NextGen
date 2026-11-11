# Senha numérica (a "segunda senha") — pesquisa e plano

Investigação de 2026-11-09 sobre a segunda credencial que o cliente 7662 pede **depois** do login,
na tela de personagem. É o que o jogador chama de "senha de números" e o cliente chama de
*lock password*. Este documento é a base do item `0x0FDE` do mapa
[`docs/rust/08-checklist-funcionalidades.md`](../rust/08-checklist-funcionalidades.md) §1.

## 1. O que é, quando aparece e como o jogador vive com ela

| Pergunta | Resposta (com fonte) |
|---|---|
| O que é | Uma segunda credencial **por conta**, só de dígitos, separada da senha de login. |
| Quando é pedida | Depois de escolher o canal e chegar à tela de personagem — **antes** de escolher o personagem que vai jogar. |
| Primeira vez | O jogador **cadastra** a senha ali mesmo, na primeira entrada; nas seguintes só valida. |
| Troca | Pelo painel da conta no site, informando a senha numérica atual e confirmando a nova. |
| Recuperação | Pelo "esqueceu login ou senha?" do site: o e-mail vinculado recebe a senha do jogo **e** a senha numérica. |

Fontes:
- Guia para novatos do WYD Global (RaidHut): "Após escolher o servidor você será levado a tela de
  personagem onde precisará **cadastrar uma senha numérica** antes de escolher seu personagem.
  Lembre bem da senha adicionada, pois você precisará dela sempre que for entrar no jogo."
  <https://feedback.kersef.com/en/knowledge-bases/21/articles/58185-guia-para-novatos>
- FAQ de recuperação de dados do WYD Global: como alterar a senha numérica (informando a atual) e
  como recuperá-la por e-mail junto com a senha do jogo.
  <https://feedback.kersef.com/en/knowledge-bases/21/articles/64301-recuperacao-de-dados>
- `Wyd2Client` (cliente C# alternativo que loga no servidor oficial): a tela de login recebe
  "nome de usuário, senha e **senha numérica**" — a segunda credencial viaja no login.
  <https://github.com/kevinkouketsu/Wyd2Client>
- `W2.Server.Node` (emulador privado em Node/TS): lista entre as entregas a live
  "senha numérica feita" — confirma que a implementação é do **servidor**, não do cliente.
  <https://github.com/Rechdan/W2.Server.Node>

## 2. No fio (cliente 7662)

O catálogo já portado tem o pacote e o tamanho; o que falta é o handler.

| Opcode | Nome | Direções | Papel |
|---|---|---|---|
| `0x0FDE` | `_MSG_AccountSecure` | C→J tentativa · J→C aceite | leva a senha numérica |
| `0x0FDF` | `_MSG_AccountSecureFail` | J→C | recusa da senha numérica |

Corpo de 32 B, medido no legado (`docs/arquitetura/protocolo.md` §5, `Basedef.h:1586`):
`numeric_token[6]` no `+12`, `unknown[10]` no `+18`, `change_numeric: i32` no `+28`.
Já existe tipo Rust com teste de tamanho: `crates/wyd-protocol/src/payload/auth.rs`
(`AccountSecure::WIRE_LEN == 32`, opcode 4062/`0x0FDE` em `crates/wyd-protocol/src/lib.rs:90`).

Confirmação independente dos dois opcodes, com pacotes de exemplo (tentativa, recusa e aceite):
`wyd2encdec` — "Lock Password packets: try a password (Client → Server) `0xFDE`; wrong password
(Server → Client) `0xFDF`; correct password `0xFDE`".
<https://github.com/raphaelts3/wyd2encdec>

**UNVERIFIED (não inventar):** quantos dígitos o campo aceita (6 bytes dão de 1 a 6 dígitos, com
NUL de padding); o significado de `unknown[10]`; e se `change_numeric` significa "estou trocando a
senha" ou "primeiro cadastro". Nada disso vira regra antes da captura da FASE-08-6.

## 3. Estado no nosso servidor (2026-11-09)

| Peça | Estado | Onde |
|---|---|---|
| Coluna de hash | existe | `crates/wyd-db/migrations/0001_contas.sql` → `accounts.pin_hash TEXT` |
| Tipo de fio (32 B) | existe | `crates/wyd-protocol/src/payload/auth.rs` |
| Repositório de conta lê/escreve PIN | **não** | `crates/wyd-db/src/repo/accounts.rs` só expõe `id`/`login`/`role` |
| Handler `0x0FDE` no canal | **não** | `crates/wyd-server/src/cpsock.rs` atende `0x020D`, `CreateCharacter`, `DeleteCharacter`, `CharacterLogin` |
| Fluxo do cliente | **quebra aqui** | após `CNFAccountLogin` o cliente pede a senha numérica; sem resposta ele não chega à seleção |

Consequência prática: hoje o caminho de entrada termina no login (e na cerca de sessão). A tela de
seleção de personagem do cliente oficial só é alcançável porque o cliente 7662 tolera não receber
o pedido de PIN em alguns fluxos — o que **não** é o comportamento do servidor oficial e precisa ser
fechado com a implementação abaixo.

## 4. Plano de implementação (com as regras que não podem ser quebradas)

1. **Ordem**: `_MSG_AccountLogin` (`0x020D`) → `_MSG_CNFAccountLogin` → **exigir**
   `_MSG_AccountSecure` (`0x0FDE`) e só então aceitar `CreateCharacter`/`DeleteCharacter`/
   `CharacterLogin`. Timeout curto entre o `CNF` e o PIN, rate-limit por conta **e** por IP
   (`docs/rust/06-observabilidade-runbooks.md`), recusa sempre com `0x0FDF`.
2. **Primeira vez** (`pin_hash IS NULL`): o mesmo pacote cadastra. Grava **Argon2id**
   (`AGENTS.md` §3 invariante 6), audita em `admin_audit`, e a sessão só abre depois.
3. **Verificação** com `argon2id verify` (tempo constante); PIN **nunca** entra em log, métrica,
   resposta de erro ou `tracing` span.
4. **Tentativas**: 5 falhas em 10 min por conta → bloqueio temporário com auditoria e métrica;
   a resposta ao cliente não distingue "conta sem PIN" de "PIN errado".
5. **Cerca de sessão**: hoje `claim_session` abre no login (`cpsock::handshake` →
   `session.rs`). Mover a abertura para **depois** do PIN fecha o bypass de "entrar sem a segunda
   senha" e continua garantindo uma sessão viva por conta entre canais (`0030_sessao_unica.sql`).
6. **Portal** (`wyd-portal`): troca da senha numérica pelo painel da conta, informando a atual,
   passando por `adminCall` (sessão de staff/`moderator_id`), gravando só hash e auditando —
   o browser nunca vê PIN nem DSN.
7. **Testes** (na issue que implementar): golden de 32 B; PIN certo, errado e ausente; cadastro na
   primeira vez; bloqueio após N falhas; `change_numeric`; e um teste que falha se o PIN aparecer
   em qualquer linha de log ou resposta de erro.

## 5. Por que isso é prioridade do caminho de entrada

Sem PIN o fluxo do cliente oficial para na tela de personagem, e a seleção de personagem que já
temos (resultado no banco, corpo UNVERIFIED) nunca é alcançada em condições reais. É o primeiro item
⬜ do mapa de funcionalidades e depende só de nós — não de captura nova, porque o layout de 32 B já
está medido no legado.

## Persistência do bloqueio (2026-11-09)

`pin_failures` e `pin_locked_until` agora vivem em `accounts` (0001 consolidada): reiniciar o
canal não zera tentativas. 5 falhas → 10 minutos (`auth.rs`), zerado por PIN correto.
Teste: `crates/wyd-server/tests/sistema/pin_persistencia.rs` (novo `Authenticator` continua bloqueado).

Desbloqueio manual (staff, auditado pela operação):

```bash
docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "${POSTGRES_DB:-wyd}" \
  -c "UPDATE accounts SET pin_failures = 0, pin_locked_until = NULL WHERE login = '"'"'LOGIN'"'"';"'
```

## Troca de PIN em duas etapas (decisão nossa, 2026-11-09)

O `0x0FDE` traz `change_numeric: i32` em +28. Sem fonte sobre a semântica do 7662, o fluxo é:

| Passo | `change_numeric` | `numeric_token` | Servidor faz |
|---|---|---|---|
| 1 | `1` | PIN **atual** | confere via `secure_pin` (conta falhas), responde `0x0FDE` e arma a troca; **não abre sessão** |
| 2 | `2` | PIN **novo** | grava o hash novo (`change_pin`), zera falhas, responde `0x0FDE` e abre a sessão |

`change_numeric = 2` sem o passo 1, ou passo 1 com PIN errado, recebem `0x0FDF`.
Prova: `crates/wyd-server/tests/sistema/pin_troca.rs::troca_de_pin_exige_o_atual_e_invalida_o_antigo`
(inclui o PIN antigo recusado no banco e no fio depois da troca).
Se o cliente 7662 real usar outro caminho (a fonte diz que o WYD Global troca pelo site),
este fluxo é o nosso, não uma imposição do protocolo.

## Mesmo fluxo pelo portal (2026-11-09)

O portal não fala com o banco nem reimplementa o protocolo: a tela de conta chama
`POST /api/panel/pin` (BFF), que chama `account_web/change_pin` no gateway; o gateway abre
uma conexão CPSock **real** e executa `0x0FDB` (login com a senha da conta) seguido das
duas etapas do `0x0FDE` descritas acima (`crates/wyd-admin/src/service/pin.rs`).

- O `login` vem da conta da sessão do portal (`getPanelAccount`), nunca do corpo do browser.
- A senha da conta é exigida porque o `0x0FDE` só existe depois do `AccountLogin`; ela não é
  guardada em cookie nem em log.
- `0x0FDF` no passo 1 vira "PIN atual incorreto ou conta bloqueada por tentativas" — o
  servidor usa o mesmo opcode para os dois casos, então a tela não promete distingui-los.
- Prova: `crates/wyd-server/tests/sistema/portal_pin.rs::troca_de_pin_pelo_portal_funciona_e_invalida_o_antigo`
  (troca pelo caminho do portal, PIN antigo recusado e novo aceito no fio, hash conferindo só
  o novo) e `wyd-portal/src/app/api/panel/pin/route.test.ts` (sessão, validação e recusa).
