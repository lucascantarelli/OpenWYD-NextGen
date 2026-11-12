# WEB-08 — Contas, segurança, auditoria e área pública × administrativa

> **ID canônico:** `WEB-08`
> **ID histórico:** `PORTAL-02`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** Código (portal+gateway) · **Esforço:** L · **Depende de:** PROC-24, WEB-03 ✅ · **Bloqueia:** —

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
- wyd-web-admin-ui — BFF, sessão e `adminCall`.

## Documentação de referência

- `wyd-portal/README.md` (regra `adminCall`, PIN), `docs/wyd/senha-numerica.md`
- `docs/arquitetura/adr/0023-moderacao-auditoria-economia.md`, `docs/rust/06-observabilidade-runbooks.md`

## Subtasks

- [x] Contas paginadas + ficha com histórico (`account_admin/list_accounts`, `get_account`; `auth.login` filtrado da trilha na ficha)
- [ ] 2FA staff + recuperação por e-mail + bloqueio automático — BLOQUEADO: sem rota no gateway (sem TOTP, sem envio de e-mail, `accounts/recovery` só registra sem emitir token)
- [x] Auditoria com filtros/export/relatórios (filtro ator/ação/alvo/período, CSV, resumo por ação + contagem de recusas)
- [x] Home pública + descoberta de admin (`/` pública sem link de staff; guarda real segue no proxy/layout/gateway)
- [x] Área do jogador: sessões (`character_session`) e histórico de acesso (`auth.login`) em `/conta`, rota `panel/history` (2026-11-11)
- [x] Downloads reais: `manifest.txt` + pacote do launcher servidos pelo portal (`/api/launcher/manifesto`, `/api/launcher/pacote`), pasta `LAUNCHER_DIST_DIR`

## Critérios de aceite

- [ ] Staff sem 2FA não acessa admin (quando habilitado); recuperação não enumera contas.
- [x] Toda ação admin auditada com autor real (moderator_id da sessão).
- [ ] Área pública completa para jogador (história, drops, classes, guias, downloads) sem vazar admin.

## Riscos / rollback

- Risco: lockout de staff por 2FA → mitigação: códigos de recuperação + admin break-glass auditado.
- Rollback: desabilitar 2FA por flag sem remover trilhas.

## Verificação on-line (2026-11-11)

Stack real: `db` + `gateway` + `portal` do compose, com um gateway de QA extra
(`:7610`, `MIGRATIONS_DIR=crates/wyd-db/migrations`) e o portal em `next dev`
(`:3010`) apontando para ele, para entrar como staff sem tocar na conta `admin`
do `.env`. Conta de QA criada pelo seed (`qa-portal`, admin).

- **Contas**: `/contas` abriu com 50 contas, filtro por papel e ficha
  (`ver ficha`). Corrigido nesta rodada: cópia sem jargão ("sem segredo",
  "exige leitura de moderação") e coluna de ação alinhada à direita.
- **Auditoria**: `/auditoria` abriu com 200 ações, resumo por ação, filtros e
  CSV. Corrigido: ações traduzidas ("leitura do catálogo", "presença do
  servidor"), removidas as menções a `audit_log`/`auth.denied:*`, rodapé em
  linguagem de operação.
- **Ficha/histórico de login**: ainda depende de `auth.login` na janela da
  trilha; sem rota de histórico por conta no gateway.
- **2FA / recuperação por e-mail / bloqueio automático**: continua BLOQUEADO.
  `accounts/recovery` só registra a intenção, não emite token; não há TOTP nem
  envio de e-mail no gateway. Implementar isso exige migration nova
  (segredo cifrado, códigos de recuperação) + rota `account_admin/two_factor_*`
  + transporte de e-mail — não cabe em fatia de UI.

## Portal público (2026-11-11, segunda rodada)

O `/` deixou de ser uma tela de login com atalho de staff e passou a ser um portal
de jogador de verdade:

| Rota | Conteúdo |
|---|---|
| `/` | hero com status vivo (gateway/banco/revisão), últimas notícias, top 5 do ranking, primeiros passos, classes e atalhos |
| `/noticias`, `/noticias/[slug]` | notícias em markdown em `docs/noticias/`, geradas por `npm run noticias` |
| `/ranking` | top 50 por nível (nome, classe, nível; sem identidade de conta) |
| `/downloads` | build 7662, protocolo 12000, revisão de conteúdo e o comando do launcher |
| `/login` | entrada do jogador (mesma conta do jogo); papel decide `/conta` × `/painel` |

**Separação pública × administrativa:** nenhuma página pública linka a
administração (verificado por `grep` em `(public)`, home, header e footer);
`src/app/robots.ts` desindexa `/painel`, `/contas`, `/monitoramento`,
`/auditoria`, `/economia`, `/login` e as demais rotas de staff. O staff entra por
URL direta; a guarda continua no `proxy.ts` + `(admin)/layout` + gateway.

## Verificação visual (2026-11-11)

| Tela | Resultado |
|---|---|
| `/contas` | lista + ficha + moderação operáveis |
| `/auditoria` | filtros, resumo e CSV operáveis |
| `/login` | entrada do jogador; staff segue para `/painel` |
