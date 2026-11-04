# Issue 17 — Observabilidade, backup, CI e hardening

**Fase:** 5 · **Depende de:** 04, 09 · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Tornar a operação sustentável: saber o que aconteceu, recuperar dados, impedir regressão e reduzir
superfície de risco — sem quebrar a compatibilidade com o cliente fechado.

## Contexto / evidência

- Logs são `slog` texto em stdout, com **todo pacote recebido** logado (`recv packet … type=0x…`) e
  `-log-sends` para o sentido S→C. Em produção isso é volume alto e sem retenção.
- O estado real está no PostgreSQL (volume Docker `pgdata`); não há rotina de backup no repositório.
- CI existe em `.github/workflows/` (badge no README), mas o caminho de bring-up local
  (`run-local.sh`) não é exercitado.
- Hardening já feito: argon2id, roles, auditoria de GM, mTLS opcional (`docker-compose.mtls.yaml`),
  checksum rejeitante disponível. Pendências: ban via binServer, keytable estática, exposição de portas.
- Lições operacionais que devem virar automatismo: imagem stale, IPAM esgotado, restart no meio de
  teste, conteúdo ausente.

## Passos

1. **Logs**: separar diagnóstico de produção — nível configurável, `recv packet` amostrado ou
   desligável por flag, retenção/rotação no compose; manter auditoria de GM e de economia sempre.
2. **Métricas mínimas**: jogadores online, mobs vivos, tick médio, conexões aceitas/recusadas por tipo,
   erros de gRPC, fila de respawn. Expor endpoint local (não público) consumível pelo agente.
3. **Backup**: script versionado (`scripts/backup.sh`) com `pg_dump` + timestamp + verificação de
   restauração em banco descartável; documentar RPO/RTO no `docs/`.
4. **Smoke automatizado**: `scripts/preflight.sh` (issue 02) + bot smoke (issue 16) rodando antes de
   qualquer validação humana e no CI.
5. **CI**: `go build`, `go vet`, `go test -race ./...`, `golangci-lint` e build de imagem; falha
   rápida. Sem segredos no workflow.
6. **Hardening** (ordem de risco):
   a. não publicar 7514/3000/7600 no host (já são internos, manter assim);
   b. exigir mTLS nos links internos para qualquer ambiente não-local;
   c. migrar ban administrativo para o binServer (entitlement) quando a plataforma web entrar;
   d. revisar `Release/` versionado vs. `.gitignore` (`[Rr]elease/`) para não perder conteúdo;
   e. documentar que `-reject-checksum` só liga com captura que confirme checksums.
7. **Runbook**: uma página com subir, verificar, reiniciar com segurança, backup, restaurar, rollback de
   deploy e "quem avisar antes de reiniciar".

## Arquivos

- Criar: `scripts/backup.sh`, `scripts/restore-check.sh`, `docs/operations/runbook.md`,
  workflow de CI (se não existir equivalente).
- Alterar: `docker-compose.yaml` (logging/healthcheck), `tmserver/cmd/tmserver/main.go`
  (flag de nível/amostragem de log), skills `wyd-server-ops`/`wyd-server-dev`.
- Proibido: versionar dump de banco, credencial, ou reduzir log de auditoria para "economizar espaço".

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
# 1. backup + restauração verificada em base descartável
./scripts/backup.sh && ./scripts/restore-check.sh
# 2. suíte completa e lint (se disponível)
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  sh -c "go vet ./... && go test -race ./..."
# 3. healthcheck dos serviços
docker compose ps
curl -s -o /dev/null -w "status_http=%{http_code}\n" http://127.0.0.1/
```

Saída esperada: backup criado com timestamp; restauração confere contagem de contas/personagens;
suíte verde; healthchecks `healthy`.

## Critérios de aceite

- [ ] Backup automatizado e **restauração testada** (não só gerar arquivo)
- [ ] Smoke de preflight + bot no CI
- [ ] Log de produção utilizável (sem afogar em `recv packet`), auditoria preservada
- [ ] Métricas mínimas expostas localmente
- [ ] Runbook publicado e referenciado nas skills
- [ ] Portas internas não expostas; plano de mTLS documentado para ambiente não-local

## Riscos / rollback

- **Risco:** reduzir log e perder a evidência de um incidente. **Mitigação:** amostragem configurável,
  nunca remover categorias de auditoria.
- **Risco:** backup que nunca foi restaurado não é backup. **Mitigação:** `restore-check.sh` no CI.
- **Risco:** ligar mTLS e derrubar a stack local. **Mitigação:** overlay separado (`make certs` +
  `docker-compose.mtls.yaml`), testado antes de virar padrão.
- **Rollback:** cada mudança de infraestrutura em commit isolado, revertível sem tocar dados.
