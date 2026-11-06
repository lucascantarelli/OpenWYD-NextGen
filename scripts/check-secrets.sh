#!/usr/bin/env bash
# Gate de segredos (OBS-04).
#
#   bash scripts/check-secrets.sh
#
# Reprova:
#   - .env real versionado (só .env.example/.sample/.template são permitidos);
#   - SESSION_SECRET ausente, curto ou igual ao default fora de desenvolvimento;
#   - W2PP_PLAY_CODE_SECRET curto;
#   - certificado ou chave privada versionada (*.pem/key/p12/pfx/jks/crt);
#   - allowlist do gitleaks cobrindo arquivo VERSIONADO (o scan ficaria cego ali);
#   - segredo de alta confiança em arquivo versionado (gitleaks, com fallback
#     determinístico quando o binário não está instalado).
#
# O fallback cobre chave privada, AWS, GitHub (token e PAT), GitLab, Slack,
# OpenAI, Stripe live, npm e Google. Ele existe para o gate local não ficar mudo
# sem o binário; no CI quem roda é o gitleaks pinado, com as regras padrão.
#
# O gitleaks roda em modo diretório para pegar também o que ainda não foi
# commitado. O scan do CI usa o checkout limpo; localmente, as allowlists de
# .gitleaks.toml excluem .env, .next, client, .tools e tmp.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail=0
env_name="${W2PP_ENV:-${APP_ENV:-${NODE_ENV:-development}}}"
session_secret="${W2PP_PORTAL_SESSION_SECRET:-${SESSION_SECRET:-}}"
dev_default="dev-only-portal-session-secret-32-chars"

say() { printf '%s\n' "$*"; }

# 1. Nenhum .env real pode estar versionado. O exemplo é o catálogo público.
tracked_env=$(git ls-files 2>/dev/null | grep -E '(^|/)\.env($|\.)' | grep -vE '\.(example|sample|template)$' || true)
if [ -n "$tracked_env" ]; then
  say "FAIL: arquivo .env versionado (remova do índice e rotacione os valores):"
  printf '%s\n' "$tracked_env" | sed 's/^/  /'
  fail=1
else
  say "OK: nenhum .env real versionado"
fi

# 2. Fora de desenvolvimento o cookie do portal não pode usar o default nem
#    segredo curto. O compose só pode cair no default em dev local.
case "$env_name" in
  production | prod | staging | stage | homolog | homologation)
    if [ -z "$session_secret" ]; then
      say "FAIL: SESSION_SECRET ausente em $env_name"
      fail=1
    elif [ "$session_secret" = "$dev_default" ]; then
      say "FAIL: SESSION_SECRET ainda é o default de desenvolvimento em $env_name"
      fail=1
    elif [ "${#session_secret}" -lt 32 ]; then
      say "FAIL: SESSION_SECRET tem menos de 32 caracteres em $env_name"
      fail=1
    else
      say "OK: SESSION_SECRET não é o default e tem >= 32 caracteres ($env_name)"
    fi
    ;;
  *)
    if [ "$session_secret" = "$dev_default" ]; then
      say "OK: SESSION_SECRET de desenvolvimento (troque fora de dev)"
    elif [ -n "$session_secret" ] && [ "${#session_secret}" -lt 32 ]; then
      say "FAIL: SESSION_SECRET tem menos de 32 caracteres"
      fail=1
    else
      say "OK: SESSION_SECRET aceitável para $env_name"
    fi
    ;;
esac

if [ -n "${W2PP_PLAY_CODE_SECRET:-}" ] && [ "${#W2PP_PLAY_CODE_SECRET}" -lt 32 ]; then
  say "FAIL: W2PP_PLAY_CODE_SECRET tem menos de 32 caracteres"
  fail=1
fi

# 2b. Material de chave não entra no índice: o mTLS de desenvolvimento é gerado
#     por scripts/gen-certs.sh e fica fora do git.
tracked_keys=$(git ls-files 2>/dev/null | grep -Ei '\.(pem|key|p12|pfx|jks|crt)$' || true)
if [ -n "$tracked_keys" ]; then
  say "FAIL: certificado ou chave privada versionado:"
  printf '%s\n' "$tracked_keys" | sed 's/^/  /'
  fail=1
else
  say "OK: nenhum certificado ou chave privada versionado"
fi

# 2c. A allowlist do gitleaks só pode citar caminho NÃO versionado. Se um padrão
#     dela casar com arquivo do índice, o scan fica cego justamente ali. O único
#     caso aceito é o catálogo público .env.example/.sample/.template.
if [ -f "$ROOT/.gitleaks.toml" ]; then
  blind=""
  while IFS= read -r pat; do
    [ -z "$pat" ] && continue
    hits=$(git ls-files 2>/dev/null | grep -E "$pat" | grep -vE '\.(example|sample|template)$' || true)
    if [ -n "$hits" ]; then
      blind="$blind$(printf '%s\n' "$hits" | sed 's/^/  /')"$'\n'
    fi
  done < <(sed -n '/^paths = \[/,/^\]/p' "$ROOT/.gitleaks.toml" | grep -oE "'''[^']+'''" | sed "s/'''//g")
  if [ -n "$blind" ]; then
    say "FAIL: allowlist do gitleaks cobre arquivo versionado (scan cego):"
    printf '%s' "$blind"
    fail=1
  else
    say "OK: allowlist do gitleaks só cobre material não versionado"
  fi
else
  say "FAIL: .gitleaks.toml ausente (o CI instala o gitleaks e precisa da config)"
  fail=1
fi

# 3. Gitleaks. O binário pode estar no PATH ou em .tools/bin (instalado por
#    `GOBIN=$PWD/.tools/bin go install ...`). Sem ele, o fallback cobre os
#    formatos de maior confiança para o gate local não ficar cego.
gitleaks_bin=""
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks_bin=$(command -v gitleaks)
elif [ -x "$ROOT/.tools/bin/gitleaks" ]; then
  gitleaks_bin="$ROOT/.tools/bin/gitleaks"
elif [ -x "$ROOT/.tools/bin/gitleaks.exe" ]; then
  gitleaks_bin="$ROOT/.tools/bin/gitleaks.exe"
fi

if [ -n "$gitleaks_bin" ]; then
  if "$gitleaks_bin" dir "$ROOT" --config "$ROOT/.gitleaks.toml" --redact --no-banner --max-target-megabytes 5; then
    say "OK: gitleaks sem achados"
  else
    say "FAIL: gitleaks encontrou segredo (valores redigidos acima)"
    fail=1
  fi
elif [ "${W2PP_REQUIRE_GITLEAKS:-0}" = "1" ]; then
  say "FAIL: gitleaks obrigatório e ausente do PATH"
  fail=1
else
  say "SKIP: gitleaks ausente; rodando varredura determinística de fallback"
  # Padrões montados em pedaços para o script não reprovar a si mesmo.
  private_key="-----BEGIN (RSA |EC |OPENSSH )?PRIVATE"" KEY-----"
  aws_key="AKIA""[0-9A-Z]{16}"
  github_token="gh[pousr]""_[A-Za-z0-9]{36,}"
  slack_token="xox[baprs]""-[A-Za-z0-9-]{10,}"
  openai_key="sk-""[A-Za-z0-9]{32,}"
  google_key="AIza""[0-9A-Za-z_-]{35}"
  gitlab_token="glpat""-[A-Za-z0-9_-]{20,}"
  stripe_live="sk_live""_[A-Za-z0-9]{20,}"
  npm_token="npm""_[A-Za-z0-9]{36}"
  github_pat="github_pat""_[A-Za-z0-9_]{30,}"
  pattern="$private_key|$aws_key|$github_token|$github_pat|$gitlab_token|$slack_token|$openai_key|$stripe_live|$npm_token|$google_key"

  hits=$(git grep -nI -E "$pattern" -- . ':(exclude)scripts/check-secrets.sh' 2>&1)
  rc=$?
  if [ "$rc" -eq 0 ]; then
    say "FAIL: possível segredo em arquivo versionado:"
    printf '%s\n' "$hits" | head -20
    fail=1
  elif [ "$rc" -ne 1 ]; then
    say "FAIL: git grep não conseguiu varrer a árvore (rc=$rc)"
    fail=1
  else
    say "OK: fallback sem segredos de alta confiança"
  fi
fi

if [ "$fail" -ne 0 ]; then
  say "SECRETS: FALHOU"
  exit 1
fi
say "SECRETS: OK"
