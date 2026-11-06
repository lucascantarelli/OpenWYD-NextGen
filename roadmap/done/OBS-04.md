# OBS-04 — `gitleaks` e checklist de segredos

**Fase:** revisão · **Depende de:** OBS-01 ✅ · **Esforço:** S · **Estado:** ✅ concluída em 06/11/2026

## Objetivo

O CI reprova segredo versionado e o repositório tem um checklist verificável de que
`SESSION_SECRET` não é o default de dev e que `.env`/credenciais não entram no git.

## Resultado

- `scripts/check-secrets.sh` (novo alvo `make check-secrets`, executado também por `make gate`):
  recusa `.env` real versionado; fora de `development` recusa `SESSION_SECRET` ausente, curto
  (< 32) ou igual ao default do compose; valida `W2PP_PLAY_CODE_SECRET` quando presente; e roda
  `gitleaks`.
- `.gitleaks.toml` usa as regras padrão e uma allowlist mínima para material **não versionado**
  (`.env`, `portal/.next`, `client`, `.tools`, `tmp`, `_legacy`) e para os defaults públicos de dev.
  Nenhuma regra do gitleaks foi desligada.
- `.github/workflows/ci.yml` ganhou o job `Secret scan`, que roda em todo push/PR (inclusive
  documentação) com `gitleaks` pinado em `v8.28.0` e `gitleaks dir`.
- `docs/migration/config-ops.md` §0.1 documenta o gate, a instalação local, a rotação do
  `SESSION_SECRET`/códigos de uso único e o restore sem segredo no dump.

## Evidência

```bash
$ W2PP_ENV=production W2PP_PORTAL_SESSION_SECRET=dev-only-portal-session-secret-32-chars \
    bash scripts/check-secrets.sh
FAIL: SESSION_SECRET ainda é o default de desenvolvimento em production
SECRETS: FALHOU                  # exit 1

$ W2PP_ENV=production W2PP_PORTAL_SESSION_SECRET='<48 caracteres aleatórios>' \
    bash scripts/check-secrets.sh
OK: gitleaks sem achados
SECRETS: OK                      # exit 0
```

Prova de vermelho com segredo sintético (arquivo removido depois):

```text
gitleaks dir secret-probe --config .gitleaks.toml --redact
leaks found: 1                   # exit 1
```

Scan real da árvore: `gitleaks dir .` → `scanned ~41.84 MB`, `no leaks found`. O scan local ignora
`.env`/`.next` por allowlist; o CI parte de um checkout limpo e roda o mesmo comando.

## Critérios de aceite

- [x] `gitleaks` no CI e no `make gate`, com prova de vermelho em segredo plantado.
- [x] Checklist recusa o default de dev fora de dev.
- [x] Rotação e backup documentados.

## Riscos/rollback

Falso positivo em arquivo versionado deve ser tratado com allowlist **mínima e justificada**, nunca
desligando a regra. Rollback: remover `check-secrets` do alvo `gate` e o job `Secret scan`; os
arquivos de configuração podem ficar sem efeito.

## Fora do escopo

Não implementa `trace_id` (OBS-02) nem dashboards/alertas (OBS-03).
