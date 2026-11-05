# NORMALIZE-01 — Fim de linha: `.gitattributes` ausente e índice não determinístico

**Fase:** revisão · **Depende de:** — · **Esforço:** S · **Estado:** ✅ concluída (05/11/2026)

## Objetivo

O conteúdo do índice não depender do sistema operacional de quem edita, e um gate barato que falhe
quando a normalização se perder.

## Contexto/evidência (05/11/2026)

Medido neste checkout, logo depois do rename do módulo:

```bash
ls .gitattributes                  # não existia
git config --get core.autocrlf     # true
git ls-files --eol | awk '{print $1, $2}' | sort | uniq -c
#   718 i/lf w/crlf     <- LF no índice, CRLF no worktree
#   556 i/lf w/lf
#    28 i/-text w/-text
#     2 i/none w/none
#     1 i/lf w/mixed     <- fim de linha misturado no mesmo arquivo
```

Sintoma diário: todo `git add` emite `warning: LF will be replaced by CRLF the next time Git touches
it`. O efeito real é mais caro que o aviso: uma varredura de texto (como o rename do módulo) roda sobre
bytes diferentes conforme o checkout, e um arquivo com EOL misto quebra `sed`/`grep` de forma
silenciosa.

## O que foi feito

- `.gitattributes` com `* text=auto eol=lf` + `binary` para os formatos de imagem/áudio/binário.
- `scripts/check-eol.sh`: falha se faltar a regra, se houver CRLF no índice ou EOL misto no worktree.
- Arquivo com EOL misto normalizado (`sed -i 's/\r$//'`).

## Verificação (falha antes, passa depois)

```
ANTES_RC=1
FAIL: .gitattributes ausente (nenhuma regra de normalizacao)
FAIL: 1 arquivo(s) com fim de linha misto no worktree
DEPOIS_RC=0
OK: normalizacao de fim de linha em dia
```

```bash
bash scripts/check-eol.sh
```

## Critérios de aceite

- [x] `.gitattributes` versionado com `* text=auto eol=lf` e binários declarados.
- [x] Gate `scripts/check-eol.sh` falhando antes e passando depois.
- [x] Nenhum arquivo com EOL misto no worktree.
- [ ] Rodar o gate no CI (entra junto com o `make ci` da CI-01).

## Riscos/rollback

Risco: o primeiro checkout após entrar `.gitattributes` reescrever os arquivos de texto para LF e
aparecer um diff grande no worktree. Mitigação: `git add --renormalize .` é opcional aqui (o índice já
estava em LF); o diff de conteúdo é zero. Rollback: remover o arquivo e o gate.
