# Issue 07 — Login ponta-a-ponta (headless + cliente gráfico)

**Fase:** 1 · **Depende de:** 03, 04, 05, 06 · **Bloqueia:** 08–13, 16 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Provar, com evidência dos dois lados, que **conta → seleção de personagem → entrada no mundo**
funciona: primeiro no teste headless, depois no cliente gráfico modificado.

## Contexto / evidência

- Teste pronto: `tmserver/internal/world/e2e_smoke_test.go` (`-tags=e2e`,
  `TestE2ESmokeLogin`), que exercita tmServer → dbServer → Postgres.
- A conta `test`/`test123` começa **sem personagens** (issue 03) — o login headless deve cobrir a
  etapa de conta e listar a seleção vazia; a criação de personagem é a issue 08.
- Lições registradas no `SESSION-PRIMER`: (a) servidor correto ≠ cliente correto — confirme o
  **pacote** S→C; (b) reiniciar serviços no meio do teste gera falha fantasma.

## Passos

1. **Headless primeiro** (sem cliente Windows):
   ```bash
   # (comandos rodam na raiz do repositório)
   docker run --rm --network host -v "$PWD":/src -w /src \
     -e W2PP_E2E_ADDR=127.0.0.1:8281 -e W2PP_E2E_ACCOUNT=test \
     -e W2PP_E2E_PASSWORD=test123 -e W2PP_E2E_VERSION=<valor da issue 06> \
     golang:1.26-alpine go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
   ```
   No Docker Desktop (Windows), trocar `--network host` por
   `--add-host=host.docker.internal:host-gateway` e `W2PP_E2E_ADDR=host.docker.internal:8281`.
2. **Cliente gráfico**: abrir `WYD.exe` (nunca o launcher), logar `test`/`test123`, conferir que a
   tela de seleção abre (sem personagens) e que o cliente **não** fecha nem mostra "versão
   incorreta".
3. Observar os logs simultaneamente (`docker compose logs -f tmserver`) e anotar a sequência de
   `recv packet` — esse é o mapa real do handshake deste cliente.
4. Se falhar no cliente e passar no headless, tratar como problema de cliente/rede:
   porta 8281/80, `serverlist.bin`, versão, firewall. Se falhar nos dois, tratar como servidor.

## Arquivos

- Alterados: nenhum esperado.
- Se algum passo exigir código: registrar em issue própria (não conserte no meio desta).
- Evidência a anexar: saída do teste headless + captura/trecho de log do login gráfico.

## Testes / verificação

```bash
# 1. headless verde
#    esperado: PASS e log do servidor com 0x020d → resposta de char list
# 2. servidor registrou o login gráfico
docker compose logs --since 10m tmserver | grep -E "recv packet.*0x020d|recv packet.*0x0213"
# 3. nenhum erro de gRPC no caminho de persistência
docker compose logs --since 10m dbserver | grep -iE "error|fail" | head
```

Saída esperada: `PASS` no teste; log do servidor com o pacote de login; dbserver sem erro de
conexão/migração.

## Critérios de aceite

- [ ] `TestE2ESmokeLogin` passa com a versão correta
- [ ] Cliente gráfico chega à tela de seleção de personagem
- [ ] Log do servidor mostra `0x020d` e a resposta de seleção para a conexão do cliente
- [ ] Nenhuma reinicialização de serviço durante o teste
- [ ] Sequência de pacotes do handshake registrada para referência futura

## Riscos / rollback

- **Risco:** `--network host` não funciona no Docker Desktop → confundir falha de rede do teste com
  bug do servidor. **Mitigação:** usar `host.docker.internal` e validar antes com
  `Test-NetConnection`.
- **Risco:** cliente fecha na seleção por falta de `BaseMob` (char sem template). **Mitigação:**
  conferir `base mob templates loaded classes=4` (issue 04) antes de investigar o cliente.
- **Rollback:** nenhum estado é alterado além de sessões; basta fechar o cliente.
