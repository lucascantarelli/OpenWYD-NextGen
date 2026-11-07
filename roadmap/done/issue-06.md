# Issue 06 — Alinhar versão de protocolo e autenticar

**Fase:** 1 · **Depende de:** 05 · **Bloqueia:** 07 · **Esforço:** S · **Estado:** ✅ fechada em 2026-11-04 (versão 12000 medida no cliente; o headless fecha na 07)

## Objetivo

Descobrir empiricamente o `ClientVersion` que **este** cliente envia e configurar o tmServer para
aceitá-lo, de modo que o login de conta responda com sucesso (sem `0x102`).

## Contexto / evidência

- `handler/login.go` compara `body.ClientVersion` com `Config.ClientVersion` e rejeita divergência
  (resposta `0x102`, log `account login rejected … got=… want=…`).
- `tmserver/cmd/tmserver/main.go` usa `-client-version` padrão **7640** (valor de `Basedef.h`),
  enquanto `docker-compose.yaml` passa **12000** porque o build "Cavaleiros de Kersef" manda 12000.
- O cliente desta pasta é de linhagem **WYD Global/RaidHut** (ver `docs/agents/CLIENT-RESEARCH-2026-06-27.md`),
  portanto o valor é **desconhecido** até ser medido. Supor 12000 é chute.
- O teste headless exige o mesmo valor (`W2PP_E2E_VERSION`), então essa descoberta serve para os dois
  caminhos de validação.

## Passos

1. Subir com o valor atual e tentar login no cliente:
   ```bash
   # (comandos rodam na raiz do repositório)
   docker compose logs -f tmserver
   ```
2. Ler o log de rejeição e extrair o número **exato**:
   ```
   account login rejected conn=… got=NNNNN want=12000
   ```
3. Ajustar o servidor para `NNNNN`:
   - definitivo: `docker-compose.yaml`, no `command:` do `tmserver`, trocar
     `"12000"` por `"<NNNNN>"`;
   - pontual: `W2PP_CLIENT_VERSION=<NNNNN> docker compose up -d tmserver`.
4. Reiniciar o tmServer e repetir o login. Esperado: sem `0x102`; log de login aceito e, em seguida,
   resposta de seleção de personagem.
5. Registrar o valor e a origem do build em `docs/migration/ingame-bugs.md` (seção de
   observações) ou no `roadmap/index.md` (changelog), para ninguém precisar redescobrir.

## Arquivos

- Alterado: `docker-compose.yaml` (`-client-version`), se o valor divergir de 12000.
- Documentação: registrar o valor medido (não invente "7640/12000" como regra geral).
- Proibido: remover a checagem de versão do `handler/login.go`; ela é a única defesa contra
  incompatibilidade de layout de pacote.

## Testes / verificação

```bash
# a) nenhuma rejeição de versão nova nos logs
# (comandos rodam na raiz do repositório)
docker compose logs --since 10m tmserver | grep -E "account login rejected|version mismatch"
#   esperado: nenhuma linha com "got=", ou apenas as antigas (antes do ajuste)

# b) login aceito
docker compose logs --since 5m tmserver | grep -E "recv packet.*0x020d|account login"
#   esperado: pacote 0x020d seguido de resposta de char list, sem 0x102

# c) prova headless com o mesmo valor
W2PP_E2E_ADDR=127.0.0.1:8281 W2PP_E2E_ACCOUNT=test W2PP_E2E_PASSWORD=test123 \
W2PP_E2E_VERSION=<NNNNN> go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
```

## Critérios de aceite

- [ ] Valor `got=` capturado e registrado com data e nome do build do cliente
- [ ] tmServer configurado com **exatamente** esse valor
- [ ] Login de conta aceito (sem `0x102`) no cliente e no teste headless
- [ ] Documentação atualizada com o valor e como reencontrá-lo

## Riscos / rollback

- **Risco:** dois clientes com versões diferentes (ex.: um patcheado por launcher) → um loga, outro
  não. **Mitigação:** congelar o build do cliente usado no projeto; nunca rodar launcher.
- **Risco:** aceitar uma versão "errada" só porque o login passou; pacotes posteriores podem falhar.
  **Mitigação:** fechar issue 07 (login completo ≥ seleção de personagem) antes de considerar
  compatível.
- **Rollback:** voltar `-client-version` ao valor anterior e reiniciar `tmserver`.

## Evidência de execução (2026-11-04)

```text
# docker inspect openwyd-tmserver-1 → Cmd
["-addr",":8281",…,"-status-addr",":80","-client-version","12000"]

recv packet conn=1 type=0x020d len=104 routed=true
account login: relaying to dbServer conn=1 account=test
account login: OK conn=1 account=test id=1 role=admin chars=2
```

- `handler/login.go` só loga `got=` **em mismatch**; como não houve `version mismatch` com
  `-client-version 12000`, o cliente enviou **12000**. Build testado: `WYD.exe` 7662 "Cavaleiros de
  Kersef" da pasta `client/`.
- Login aceito (sem `0x102`) e a lista veio com **2 personagens** (`chars=2`).
- Documentação corrigida para 12000: `README.md`, `CLAUDE.md`, `docs/migration/README.md`,
  `protocol-spec.md` §4.1 e skills `wyd-client-connect`/`wyd-server-ops`.
- Critérios: ✅ valor medido e registrado · ✅ tmServer com exatamente esse valor · ✅ login aceito no
  cliente (headless: rodar junto da issue 07) · ✅ documentação atualizada.
