# Issue 49 — QA in-game em lote (fechar as issues 🟡)

**Fase:** 7 · **Depende de:** 04 · **Bloqueia:** 39 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Fechar (ou reclassificar com justificativa) as 13 issues que dependem de evidência no cliente Windows
— as 11 em 🟡 (**07, 23, 24, 25, 32, 34, 36, 44, 45, 46, 48**) mais **05** e **06** (conexão e
versão de protocolo), que são o pré-requisito de todas. É a maior concentração de trabalho "quase
pronto" do roadmap.

## Contexto

- O servidor, o portal e os testes automatizados já passam; a última milha é humana.
- Existe conflito documental sobre a versão do protocolo: `docs/migration/` diz **7640**
  (`Basedef.h:102`), `AGENTS.md` e o e2e usam **12000**. Sem resolver isso, qualquer teste gráfico é
  inconclusivo.
- Fila, pré-requisitos e protocolo de evidência estão em
  [`docs/analysis/qa-ingame.md`](../../docs/analysis/qa-ingame.md); o passo a passo executável está em
  [`docs/analysis/roteiros-qa.md`](../../docs/analysis/roteiros-qa.md).

## Passos

1. **Medir a versão**: capturar `_MSG_AccountLogin.ClientVersion` do cliente (proxy/log do tmserver) e
   registrar no doc; corrigir `AGENTS.md`/`CLAUDE.md`/README para o valor real.
2. Rodar o que é automático antes do cliente: `TestE2ESmokeLogin`, `scripts/e2e-worldevents.sh`,
   `make portal-test portal-build`.
3. Executar a fila na ordem de [`docs/analysis/roteiros-qa.md`](../../docs/analysis/roteiros-qa.md)
   (SQL de preparação + passos no cliente + captura), uma issue por vez, anexando evidência no arquivo
   da issue. Para não repetir login e restart por issue, o caminho curto é
   [`docs/analysis/roteiro-sessao-unica.md`](../../docs/analysis/roteiro-sessao-unica.md): as seis flags
   de overlay sobem juntas num restart só e a sessão cobre 24, 25, 34, 36, 44, 45, 48, 28 e 30, além
   de 08/09/10/11/12/13.
4. Cada falha vira linha em `docs/migration/ingame-bugs.md` (prioridade, causa provável, o que falta) e
   **não** fecha a issue.
5. Ao final, atualizar `roadmap/index.md` (linhas 05–48 afetadas) e o changelog.

## Arquivos

- `docs/migration/ingame-bugs.md` (achados), `docs/analysis/qa-ingame.md` (fila/estado),
  `roadmap/issues/issue-05…48.md` (evidência), `roadmap/index.md`, `AGENTS.md` (versão do protocolo).

## Verificação

```bash
# headless primeiro (não precisa do cliente)
docker run --rm --network openwyd_default -v "$PWD:/src" -w /src \
  -e W2PP_E2E_ADDR=tmserver:8281 -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 \
  -e W2PP_E2E_VERSION=12000 golang:1.26-alpine \
  go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
```

Critério de evidência: log + captura + antes/depois descritos em `qa-ingame.md §2`.

## Critérios de aceite

- [ ] Versão do protocolo medida e documentada em um único lugar
- [ ] Pelo menos 8 das 13 issues fechadas com evidência anexada
- [ ] As restantes reclassificadas com motivo (não podem continuar 🟡 sem justificativa)
- [ ] `ingame-bugs.md` atualizado com todo achado novo

## Riscos

- **Teste humano no meio de restart** → falha fantasma: não reiniciar serviço durante o teste.
- **Cliente de origem duvidosa** (launcher/IP alterados): validar hash antes de culpar o servidor.
