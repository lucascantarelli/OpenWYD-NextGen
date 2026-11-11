# Issue PORTAL-02 — Contas, segurança, auditoria e área pública × administrativa

**Tipo:** Código (portal+gateway) · **Esforço:** L · **Depende de:** ROAD-01, FASE-07-3 ✅ · **Bloqueia:** —

## Objetivo

Expandir contas (histórico login, atividades, permissões, filtros/busca/ordenação), endurecer segurança (2FA, recuperação, bloqueio de suspeitas, auditoria completa) e separar de verdade área pública × administrativa (acesso liberado pós-login com botão/perfil na navbar quando staff).

## Contexto / evidência

- Hoje: `/contas` (busca, criação, moderação papel/bloqueio), `/auditoria` (admin, `audit_log` append-only), `/conta` (jogador: conta, personagens, ranking, troca de PIN), guarda em `proxy.ts`, `adminCall` injeta `moderator_id` da sessão.
- `prompt.md` pede: histórico de login/atividades/permissões; UI com filtros/busca/ordenação; 2FA, recuperação de senha, bloqueio de suspeitas; auditoria/relatórios/alertas; separação pública/admin com descoberta de acesso no perfil.

## Passos

1. Contas: `account_admin/list_accounts` paginado com filtros (nome, papel, bloqueio, ordenação) + ficha com histórico de login (`character_session`/audit), atividades recentes e permissões efetivas; tudo via `adminCall`.
2. Segurança: 2FA TOTP para staff (enroll + verify, segredo cifrado, códigos de recuperação), recuperação por e-mail (token único, expiração curta, sem enumerar contas), bloqueio automático de suspeitas (novo `account_admin/flag_suspicious` + motivo) — PIN/senha/hash nunca em log ou resposta.
3. Auditoria: `/auditoria` com filtros (ator, ação, alvo, período), exportação CSV/NDJSON, relatórios (ações por staff, concessões por período) e alertas de ação suspeita.
4. UX pública × admin: `/` pública (jogo, notícias, eventos, downloads, suporte) + descoberta de admin no perfil da navbar quando `isStaff(session.role)` (link, não botão obrigatório — encaixar no layout); guarda de rota continua no servidor (`proxy.ts` + `requireStaff`).
5. Economia do servidor (escopo desta issue só na parte de contas): trilhar concessões por conta na ficha (origem `delivery_queue`, sem escrever estado vivo).

## Arquivos

- Criados: `wyd-portal/src/app/(admin)/contas/historico-*`, `wyd-portal/src/app/(admin)/auditoria/export/route.ts`, `wyd-portal/src/server/two-factor.ts`, testes.
- Alterados: `wyd-portal/src/{proxy,server/{accounts,admin-client,session,audit}}.ts`, `wyd-portal/src/app/(public)/page.tsx` (home pública), `wyd-portal/src/components/*nav*`, `crates/wyd-admin/src/transport/*` (novas rotas), migrations (2FA, login_history — uma por domínio, reversível).
- Proibido: escrever estado de personagem vivo pelo portal (só `delivery_queue`); segredo/DSN no browser.

## Testes / verificação

```bash
cd wyd-portal && npx tsc --noEmit && npx vitest run
cargo test -p wyd-admin --test contract
```

Saída esperada: typecheck 0, vitest verde, contrato 11/11 + novos testes de 2FA/bloqueio.

## Skills relacionadas

- wyd-rust-db — modelagem, migrations reversíveis, `delivery_queue`.
- wyd-rust-obs — auditoria, alertas de segurança.
- frontend-design, ui-ux-practices — tabelas densas, filtros acessíveis.

## Documentação de referência

- `wyd-portal/README.md` (regra `adminCall`, PIN), `docs/wyd/senha-numerica.md`
- `docs/arquitetura/adr/0023-moderacao-auditoria-economia.md`, `docs/rust/06-observabilidade-runbooks.md`

## Subtasks

- [x] Contas paginadas + ficha com histórico (`account_admin/list_accounts`, `get_account`; `auth.login` filtrado da trilha na ficha)
- [ ] 2FA staff + recuperação por e-mail + bloqueio automático — BLOQUEADO: sem rota no gateway (sem TOTP, sem envio de e-mail, `accounts/recovery` só registra sem emitir token)
- [x] Auditoria com filtros/export/relatórios (filtro ator/ação/alvo/período, CSV, resumo por ação + contagem de recusas)
- [x] Home pública + descoberta de admin (`/` pública com atalho para staff; guarda real segue no proxy/layout/gateway)

## Critérios de aceite

- [ ] Staff sem 2FA não acessa admin (quando habilitado); recuperação não enumera contas.
- [ ] Toda ação admin auditada com autor real (moderator_id da sessão).
- [ ] Área pública completa para jogador (história, drops, classes, guias, downloads) sem vazar admin.

## Riscos / rollback

- Risco: lockout de staff por 2FA → mitigação: códigos de recuperação + admin break-glass auditado.
- Rollback: desabilitar 2FA por flag sem remover trilhas.
