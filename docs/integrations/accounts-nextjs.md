# Contas, papéis e banimento no portal (`/contas`, issue 32)

Guia da borda web: o que o `AccountAdminService` garante, como o BFF chama e quais erros são de
negócio. Complementa `npc-admin-nextjs.md` (mesmo padrão de `adminCall`/`AdminResult`).

## 1. Por que existe

Antes desta issue, promover alguém ou banir uma conta era `update account set role=…` no `psql`.
Sem autor, sem motivo, sem antes/depois — e a skill `wyd-game-admin` documentava isso como se fosse
procedimento normal. O objetivo aqui é o oposto: **toda** mudança de papel/bloqueio tem motivo,
autor e histórico, e as regras que protegem o acesso administrativo não dependem de lembrar de um
comando SQL.

O que **não** entra: redefinir senha/PIN, apagar conta, escrever personagem. `account.role` e
`account.is_blocked` são estado frio — valem no próximo login.

## 2. Contrato (`api/web/v1/web.proto`)

```proto
service AccountAdminService {
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse); // staff
  rpc GetAccount(GetAccountRequest)     returns (GetAccountResponse);   // staff
  rpc SetRole(SetRoleRequest)           returns (AccountWriteResponse); // admin
  rpc SetBlocked(SetBlockedRequest)     returns (AccountWriteResponse); // staff, alvo player p/ moderator
  rpc CreateAccount(AdminCreateAccountRequest) returns (AdminCreateAccountResponse); // admin
}
```

- `AccountRow` **não tem** `pass_hash`, `pin_hash` nem `block_pass_hash`; o tipo de wire foi desenhado
  para que um vazamento exija mudar o proto.
- `ListAccounts` é keyset (`after_id`), com `total_count` do mesmo filtro. `blocked` é tri-estado:
  `-1` todas, `0` ativas, `1` bloqueadas.
- `GetAccount` devolve a conta + as 50 linhas mais novas de `account_admin_audit`.
- Escritas devolvem `AdminResult` **e** `message`: a recusa de negócio vem em texto pronto
  ("esta é a última conta admin ativa — promova outra antes"). `adminCall` prefere esse `message` ao
  texto genérico de `AdminResult` — sem isso a tela mostraria "dados inválidos" para uma regra que
  tem explicação.

## 3. Regras (onde elas moram de verdade)

`internal/store/account_admin.go`, **dentro da transação**:

| Regra | Onde | Resultado |
|-------|------|-----------|
| Ator precisa ser `admin` (re-lido do banco) | `requireAdmin` | `FORBIDDEN` |
| Ninguém altera o próprio papel/bloqueio | `SetAccountRole`/`SetAccountBlocked` | `INVALID` + mensagem |
| Última conta **admin ativa** não é rebaixada nem bloqueada | `lockActiveAdmins` (`FOR UPDATE`) | `INVALID` + mensagem |
| `moderator` só bloqueia `player` | `SetAccountBlocked` | `FORBIDDEN` |
| Toda escrita grava `account_admin_audit` | mesma transação | — |

`lockActiveAdmins` tranca as linhas de admin ativo antes de contar, então duas demissões
concorrentes serializam: a segunda enxerga o resultado da primeira. A regra é redundante com
"ninguém altera a si mesmo" no caminho normal — mas é a rede de segurança para o caso de um admin
bloqueado com sessão antiga ainda válida.

Ganchos em `account`:

- `role_changed_at`, `role_changed_by` — a última troca de papel (nulos até a primeira);
- `blocked_reason` — o motivo do bloqueio atual; desbloquear limpa.

## 4. BFF (`portal/src`)

- `server/accounts.ts` — leituras (`listAccounts`, `getAccount`) via `adminCall`.
- `server/actions/accounts.ts` — `changeAccountRole`, `changeAccountBlocked`, `createAccount`.
  Validam com `lib/account.ts` (mesmas mensagens da UI), chamam `adminCall` e fazem `revalidatePath`.
- `lib/account.ts` — rótulos, tons de badge, formatação em `America/Sao_Paulo` e as validações
  (login 4–12 `a-z0-9`, senha ≥ 4, motivo ≥ 10).
- Páginas: `(admin)/contas/page.tsx` (lista + filtros + criar) e `(admin)/contas/[id]/page.tsx`
  (detalhe + auditoria + editor).

Nada de `moderator_id` vindo do corpo: o `adminCall` injeta o da sessão.

## 5. Verificação

```bash
# unitários do serviço e do gRPC
go test ./webserver/internal/accountadmin/ ./webserver/internal/grpcsrv/

# portal
cd portal && npx tsc --noEmit && npx vitest run && npx next build

# e2e contra um web-api rodando (cria contas zq*, apaga no fim)
W2PP_E2E_ADMIN_ID=1 \
W2PP_E2E_WEB_API=127.0.0.1:7600 \
W2PP_E2E_DSN=postgres://postgres:dev@127.0.0.1:5432/postgres \
go test -tags=e2e ./webserver/internal/accountadmin -run TestE2EAccountAdmin -v
```

O e2e cobre: criação já com papel, promoção com `role_changed_by`, auditoria com motivo,
`moderator` recusado ao alterar papel, auto-bloqueio recusado, `moderator` bloqueando `moderator`
recusado, bloqueio com motivo visível e desbloqueio limpando o motivo, lista por papel, e `player`
recebendo `FORBIDDEN` na leitura.

## 6. Armadilhas

- **`protoc` não existe no host**, mas `protoc-gen-go`/`protoc-gen-go-grpc` sim. Gere o proto pelo
  container, com as versões do header do arquivo (senão o diff vira ruído):

  ```bash
  MSYS_NO_PATHCONV=1 docker run --rm -v "$PWD:/src" -w /src golang:1.26-alpine sh -c \
    "apk add --no-cache protobuf >/dev/null && \
     go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 && \
     go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2 && \
     PATH=\$PATH:/root/go/bin protoc --go_out=. --go_opt=module=openwyd-nextgen \
       --go-grpc_out=. --go-grpc_opt=module=openwyd-nextgen api/web/v1/web.proto"
  ```

- **`int64` chega como `number`** no portal (`longs: Number` no proto-loader) — `role_changed_by`
  compare como número, não string.
- **Papel no cookie é dica de UI.** O web-api relê `account.role`; um `moderator` promovido só vê a
  interface nova depois de relogar, e um `admin` rebaixado continua com a casca antiga até o cookie
  expirar — a RPC, porém, já o recusa.
- **Não prometa efeito imediato.** Um personagem online continua na sessão; o bloqueio só aparece
  quando ele reconectar.
- **`NAME_TAKEN`** chega como `ADMIN_RESULT_INVALID` + mensagem "já existe uma conta com esse nome"
  (HTTP 422). Não existe um `AdminResult` para conflito de nome; a mensagem é o contrato.
