# OBS-04 — `gitleaks` e checklist de segredos

**Fase:** revisão · **Depende de:** OBS-01 ✅ · **Esforço:** S · **Estado:** ⏳ proposta

## Objetivo

O CI reprova segredo versionado e o repositório tem um checklist verificável de que
`SESSION_SECRET` não é o default de dev e que `.env`/credenciais não entram no git.

## Contexto

`.gitignore` já cobre `.env`, mas não há gate. O compose usa um `SESSION_SECRET` de
desenvolvimento e a stack aceita mTLS opcional; falta provar que o ambiente de dev não vaza para
produção. Esta issue foi separada da `OBS-01` porque não depende de `/metrics`.

## Passos

1. Rodar `gitleaks detect` (ou `gitleaks dir`) no CI e no `make gate`, com baseline só para
   falsos positivos documentados.
2. Script de pré-voo que recusa `SESSION_SECRET` default/curto fora de dev e lista variáveis
   obrigatórias sem imprimir valores.
3. Documentar rotação de segredos e o procedimento de restore/backup no runbook.

## Arquivos

- `.github/workflows/`, `Makefile`, `scripts/`
- `docs/migration/config-ops.md` / runbook de operação

## Testes/verificação

```bash
gitleaks detect --no-git --redact
bash scripts/check-secrets.sh   # exit 1 com SESSION_SECRET default
```

## Critérios de aceite

- [ ] `gitleaks` no CI e no gate, com prova de vermelho em segredo plantado.
- [ ] Checklist recusa o default de dev fora de dev.
- [ ] Rotação e backup documentados.

## Riscos/rollback

Risco: falso positivo travar o CI. Mitigação: allowlist mínima e auditada, nunca desligar a regra.
