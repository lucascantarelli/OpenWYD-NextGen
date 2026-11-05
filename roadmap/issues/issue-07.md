# Issue 07 — Login ponta-a-ponta (headless + cliente gráfico)

**Fase:** 1 · **Depende de:** 03, 04, 05, 06 · **Bloqueia:** 08–13, 16 · **Esforço:** M · **Estado:** ✅ fechada em 2026-11-04 (login, **entrada no mundo**, movimento, loja e compra no cliente; headless verde)

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

- [x] Cliente gráfico chega à tela de seleção de personagem — 2026-11-04, lista com 2 personagens
- [x] Log do servidor mostra `0x020d` e a resposta de seleção para a conexão do cliente
- [x] Nenhuma reinicialização de serviço durante o teste
- [x] Sequência de pacotes do handshake registrada (até a lista: `accept` → `0x020d` → `0x010a`)
- [x] `TestE2ESmokeLogin` passa com a versão correta (`W2PP_E2E_VERSION=12000`) — PASS em 04/11
- [x] Selecionar personagem, **entrar no mundo** e andar — 04/11, `CNFCharacterLogin body=1820` e
  rota de 6 passos

## Riscos / rollback

- **Risco:** `--network host` não funciona no Docker Desktop → confundir falha de rede do teste com
  bug do servidor. **Mitigação:** usar `host.docker.internal` e validar antes com
  `Test-NetConnection`.
- **Risco:** cliente fecha na seleção por falta de `BaseMob` (char sem template). **Mitigação:**
  conferir `base mob templates loaded classes=4` (issue 04) antes de investigar o cliente.
- **Rollback:** nenhum estado é alterado além de sessões; basta fechar o cliente.

## Evidência de execução (2026-11-04)

```text
19:14:12.798  connection accepted conn=1 ip=172.28.0.1:41680
19:14:12.809  recv packet conn=1 type=0x020d len=104 routed=true
19:14:12.810  account login: relaying to dbServer conn=1 account=test
19:14:13.074  account login: OK conn=1 account=test id=1 role=admin chars=2
19:14:20.538  connection drop reason conn=1 ip=172.28.0.1:41680 err=EOF
19:14:20.539  session send stats conn=1 frames=1 bytes=2008 out_high_water=1 by_type=0x010a:1
19:14:20.539  session last sends conn=1 tail=0x010a(30002)/1996/-7463ms
19:14:20.541  connection closed conn=1
```

Banco no mesmo momento: `Cantarelli` nível 399 e `Cantarelli` nível 39, ambos da conta `test`.

**Leitura honesta do log:**

- `accept` → `0x020d` → `relaying` → `OK chars=2`: a conta autenticou e a **lista de 2 personagens**
  foi enviada ao cliente num único frame de 1.996 B (`0x010a`). A tela de seleção foi alimentada.
- **Sete segundos depois a conexão caiu com `err=EOF`, sem nenhum pacote de seleção** (`0x0213` ou
  equivalente) no log. Ou o cliente foi fechado na tela de seleção, ou travou/fechou ao abrir a lista.
  **Não houve entrada no mundo** nesta sessão.
- Critérios: ✅ cliente chega à seleção · ✅ `0x020d` + resposta no log · ✅ nenhum restart durante o
  teste · ✅ sequência do handshake registrada (até a lista) · ⏳ `TestE2ESmokeLogin` headless ·
  ⏳ selecionar personagem, entrar no mapa e andar.

### Confirmação no cliente (2026-11-04, 20:00–20:01)

```text
20:00:10.712  account login: OK conn=1 account=test id=1 role=admin chars=2
20:00:17.501  recv packet type=0x0fde len=20      (MsgAccountSecure — token de PIN)
20:00:18.804  recv packet type=0x0213 len=24      (MSG_CharacterLogin)
20:00:18.804  character login request conn=1 slot=1 mode=11
20:00:19.054  char login: sending CNFCharacterLogin path=template slot=1 name=Cantarelli spawn=(2089,2106) body=1820
20:00:24.453  movement: first action after login type=876 target=(2095,2107) speed=6 route=[57 54 54 54 54 54 0 …]
20:00:24.687  shop opened conn=1 npc=12902 merchant=1
20:00:30.286  buy ok conn=1 npcPos=8 item=5134 myPos=5 price=0 gold=44474191
20:01:54.986  connection drop reason conn=1 err=EOF
```

- **Entrou no mundo:** `0x0213` slot 1 → `CNFCharacterLogin` (corpo de 1.820 B, spawn 2089,2106) e o
  personagem `Cantarelli` nível 39 (o que tem 44.474.191 de ouro no banco).
- **Andou:** `0x036C` (`MsgAction`) com rota de 6 passos e a posição do servidor acompanhando.
- **Comércio:** loja aberta em três mercadores (`0x027B` → `MSG_ShopList`) e **compra aceita**
  (`0x0379` → `buy ok`, item 5134 no slot 5). O item tinha **preço 0**, então não há débito de ouro
  a provar nesta compra (a prova de preço é a issue 24).
- **Fluxo S→C da sessão:** `0x0364:49` (CreateMob — mobs visíveis), `0x0165:31`, `0x0181:8`,
  `0x0336:2` (UpdateScore), `0x0114:1` (CNFCharacterLogin), `0x0379:1`, total 104 frames / 17.880 B.
- **Headless:** `TestE2ESmokeLogin` **PASS** com `W2PP_E2E_VERSION=12000`
  (`LOGIN OK — CNFAccountLogin, body=1996 bytes`).

**Achado para depois:** `0x03AE` (942) chegou às 20:01:49 com `routed=false` e a conexão caiu por
`EOF` 5 s depois — candidato a pacote de logout/quit do cliente, ainda sem handler. Não bloqueia a
issue (o save on disconnect funcionou), mas convém mapear em `docs/migration/protocol-spec.md`.
