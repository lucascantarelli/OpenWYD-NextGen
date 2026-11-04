//go:build e2e

// E2E da administração de contas (issue 32) contra um web-api rodando.
//
// Excluído do build padrão (tag `e2e`), igual ao smoke de login do tmserver. Ele
// cria contas de teste com prefixo `zq` e, se `W2PP_E2E_DSN` for informado, apaga
// tudo (inclusive a auditoria) no final — sem isso, o banco de dev fica com
// contas zq órfãs.
//
//	W2PP_E2E_ADMIN_ID=1 \
//	W2PP_E2E_DSN=postgres://postgres:dev@127.0.0.1:5432/postgres \
//	go test -tags=e2e ./webserver/internal/accountadmin -run TestE2EAccountAdmin -v
package accountadmin_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	webv1 "github.com/jeanluca/w2pp-openwyd/api/web/v1"
)

func TestE2EAccountAdmin(t *testing.T) {
	adminID := int64(envInt(t, "W2PP_E2E_ADMIN_ID"))
	addr := envOr("W2PP_E2E_WEB_API", "127.0.0.1:7600")

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()
	c := webv1.NewAccountAdminServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suffix := strconv.FormatInt(time.Now().Unix()%100_000, 10)
	playerName, modName, otherName := "zqp"+suffix, "zqm"+suffix, "zqo"+suffix

	const reason = "teste e2e da issue 32 (papéis e bloqueio)"

	// 1. Criação já com papel (só admin) — auditoria na mesma transação.
	created := mustOKCreate(t, c, ctx, adminID, playerName, "player", reason)
	mod := mustOKCreate(t, c, ctx, adminID, modName, "moderator", reason)
	other := mustOKCreate(t, c, ctx, adminID, otherName, "player", reason)
	t.Logf("contas criadas: %s=%d %s=%d %s=%d", playerName, created, modName, mod, otherName, other)

	// Cleanup registrado já aqui: uma falha no meio não pode deixar lixo no banco.
	if dsn := os.Getenv("W2PP_E2E_DSN"); dsn != "" {
		t.Cleanup(func() { cleanup(t, dsn, "zq%") })
	} else {
		t.Logf("W2PP_E2E_DSN ausente: contas %s/%s/%s ficaram no banco", playerName, modName, otherName)
	}

	// 2. Promoção auditada: quem mudou e quando.
	if res, err := c.SetRole(ctx, &webv1.SetRoleRequest{
		ModeratorId: adminID, AccountId: created, Role: "moderator", Reason: reason,
	}); err != nil || res.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("promover %d: result=%v err=%v", created, res.GetResult(), err)
	}
	got, err := c.GetAccount(ctx, &webv1.GetAccountRequest{ModeratorId: adminID, AccountId: created})
	if err != nil || got.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("get %d: result=%v err=%v", created, got.GetResult(), err)
	}
	if got.GetAccount().GetRole() != "moderator" {
		t.Fatalf("role = %q, want moderator", got.GetAccount().GetRole())
	}
	if got.GetAccount().GetRoleChangedBy() != adminID || got.GetAccount().GetRoleChangedAtUnix() == 0 {
		t.Fatalf("role_changed_by/at não gravados: by=%d at=%d",
			got.GetAccount().GetRoleChangedBy(), got.GetAccount().GetRoleChangedAtUnix())
	}
	if len(got.GetAudit()) == 0 || got.GetAudit()[0].GetReason() != reason {
		t.Fatalf("auditoria ausente ou sem motivo: %+v", got.GetAudit())
	}

	// 3. Moderador não altera papel (nem o do próprio colega).
	denied, err := c.SetRole(ctx, &webv1.SetRoleRequest{
		ModeratorId: mod, AccountId: created, Role: "player", Reason: reason,
	})
	if err != nil {
		t.Fatalf("set role por moderador: %v", err)
	}
	if denied.GetResult() != webv1.AdminResult_ADMIN_RESULT_FORBIDDEN {
		t.Fatalf("moderador alterou papel: result=%v", denied.GetResult())
	}

	// 4. Ninguém bloqueia a si mesmo — e a recusa traz mensagem.
	self, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: adminID, AccountId: adminID, Blocked: true, Reason: reason,
	})
	if err != nil {
		t.Fatalf("auto-bloqueio: %v", err)
	}
	if self.GetResult() != webv1.AdminResult_ADMIN_RESULT_INVALID || self.GetMessage() == "" {
		t.Fatalf("auto-bloqueio: result=%v msg=%q", self.GetResult(), self.GetMessage())
	}

	// 5. Moderador só bloqueia jogador: alvo moderador é escopo negado.
	scope, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: mod, AccountId: created, Blocked: true, Reason: reason,
	})
	if err != nil || scope.GetResult() != webv1.AdminResult_ADMIN_RESULT_FORBIDDEN {
		t.Fatalf("moderador bloqueando moderador: result=%v err=%v", scope.GetResult(), err)
	}

	// 6. Bloqueio com motivo, visível na leitura; desbloqueio limpa o motivo.
	blocked, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: mod, AccountId: mod, Blocked: true, Reason: reason,
	})
	if err != nil {
		t.Fatalf("auto-bloqueio do moderador: %v", err)
	}
	if blocked.GetResult() != webv1.AdminResult_ADMIN_RESULT_INVALID {
		t.Fatalf("moderador bloqueou a si mesmo: result=%v", blocked.GetResult())
	}
	if res, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: adminID, AccountId: created, Blocked: true, Reason: reason,
	}); err != nil || res.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("bloquear: result=%v err=%v", res.GetResult(), err)
	}
	afterBlock, err := c.GetAccount(ctx, &webv1.GetAccountRequest{ModeratorId: adminID, AccountId: created})
	if err != nil || !afterBlock.GetAccount().GetIsBlocked() ||
		afterBlock.GetAccount().GetBlockedReason() != reason {
		t.Fatalf("bloqueio não persistiu: %+v err=%v", afterBlock.GetAccount(), err)
	}
	if res, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: adminID, AccountId: created, Blocked: false, Reason: reason,
	}); err != nil || res.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("desbloquear: result=%v err=%v", res.GetResult(), err)
	}
	afterUnblock, err := c.GetAccount(ctx, &webv1.GetAccountRequest{ModeratorId: adminID, AccountId: created})
	if err != nil || afterUnblock.GetAccount().GetIsBlocked() ||
		afterUnblock.GetAccount().GetBlockedReason() != "" {
		t.Fatalf("desbloqueio deixou motivo: %+v err=%v", afterUnblock.GetAccount(), err)
	}

	// 7. O caminho de login enxerga o bloqueio (mesma coluna que o tmserver lê).
	web := webv1.NewAccountWebServiceClient(conn)
	if res, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: adminID, AccountId: created, Blocked: true, Reason: reason,
	}); err != nil || res.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("rebloquear: result=%v err=%v", res.GetResult(), err)
	}
	login, err := web.VerifyCredentials(ctx, &webv1.VerifyCredentialsRequest{
		Name: playerName, Password: "senha-e2e-32",
	})
	if err != nil || !login.GetOk() || !login.GetBlocked() {
		t.Fatalf("login de conta bloqueada: ok=%v blocked=%v err=%v", login.GetOk(), login.GetBlocked(), err)
	}
	if res, err := c.SetBlocked(ctx, &webv1.SetBlockedRequest{
		ModeratorId: adminID, AccountId: created, Blocked: false, Reason: reason,
	}); err != nil || res.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("desbloquear de novo: result=%v err=%v", res.GetResult(), err)
	}
	login, err = web.VerifyCredentials(ctx, &webv1.VerifyCredentialsRequest{
		Name: playerName, Password: "senha-e2e-32",
	})
	if err != nil || !login.GetOk() || login.GetBlocked() {
		t.Fatalf("login após desbloqueio: ok=%v blocked=%v err=%v", login.GetOk(), login.GetBlocked(), err)
	}

	// 8. Lista: staff vê, jogador não; o filtro de papel acha a conta promovida.
	list, err := c.ListAccounts(ctx, &webv1.ListAccountsRequest{
		ModeratorId: mod, Query: "zq", Role: "moderator", Blocked: -1, Limit: 100,
	})
	if err != nil || list.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK {
		t.Fatalf("list como moderador: result=%v err=%v", list.GetResult(), err)
	}
	found := false
	for _, row := range list.GetAccounts() {
		if row.GetId() == created {
			found = true
		}
	}
	if !found {
		t.Fatalf("conta %d não apareceu no filtro role=moderator", created)
	}

	deniedList, err := c.ListAccounts(ctx, &webv1.ListAccountsRequest{
		ModeratorId: other, Query: "zq", Limit: 10, Blocked: -1,
	})
	if err != nil || deniedList.GetResult() != webv1.AdminResult_ADMIN_RESULT_FORBIDDEN {
		t.Fatalf("list como jogador: result=%v err=%v", deniedList.GetResult(), err)
	}
}

func mustOKCreate(t *testing.T, c webv1.AccountAdminServiceClient, ctx context.Context, adminID int64, name, role, reason string) int64 {
	t.Helper()
	res, err := c.CreateAccount(ctx, &webv1.AdminCreateAccountRequest{
		ModeratorId: adminID,
		Name:        name,
		Password:    "senha-e2e-32",
		Email:       name + "@example.invalid",
		Role:        role,
		Reason:      reason,
	})
	if err != nil {
		t.Fatalf("criar %s: %v", name, err)
	}
	if res.GetResult() != webv1.AdminResult_ADMIN_RESULT_OK || res.GetAccountId() == 0 {
		t.Fatalf("criar %s: result=%v msg=%q", name, res.GetResult(), res.GetMessage())
	}
	return res.GetAccountId()
}

func cleanup(t *testing.T, dsn, prefix string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Logf("cleanup: connect: %v", err)
		return
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `
		DELETE FROM account_admin_audit
		WHERE target_id IN (SELECT id FROM account WHERE name LIKE $1)`, prefix); err != nil {
		t.Logf("cleanup auditoria: %v", err)
	}
	tag, err := conn.Exec(ctx, `DELETE FROM account WHERE name LIKE $1`, prefix)
	if err != nil {
		t.Logf("cleanup contas: %v", err)
		return
	}
	t.Logf("cleanup: %d contas de teste removidas", tag.RowsAffected())
}

func envInt(t *testing.T, key string) int {
	t.Helper()
	raw := os.Getenv(key)
	if raw == "" {
		t.Skipf("%s não definido: o e2e precisa do id de uma conta admin", key)
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		t.Fatalf("%s=%q inválido", key, raw)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var _ = fmt.Sprintf
