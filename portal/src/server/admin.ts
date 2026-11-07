/**
 * O **único** caminho para chamar RPC admin do web-api.
 *
 * `adminCall` faz quatro coisas de uma vez, e é por isso que ninguém deve falar
 * com o cliente gRPC direto numa page:
 *
 * 1. exige sessão de staff (`requireStaff`);
 * 2. injeta `moderator_id` **da sessão** — nunca do corpo da request;
 * 3. aplica deadline de 10s na RPC;
 * 4. mapeia `AdminResult` → HTTP e registra auditoria — com o `trace_id` da
 *    chamada, o mesmo que o web-api registra (OBS-02).
 *
 * `AdminResult` viaja no corpo, com `result: 1..5`; só falha de transporte é
 * erro gRPC (`docs/integrations/npc-admin-nextjs.md §4.1`).
 */
import { ADMIN_RESULT, httpStatusFor, isOk, messageFor } from "./admin-result";
import { logAudit } from "./audit";
import { AdminError } from "./errors";
import { promisify, type Unary } from "./promisify";
import { getSession, isStaff, type Session } from "./session";
import { newTraceId } from "./trace";

export { ADMIN_RESULT, httpStatusFor, messageFor } from "./admin-result";

export interface AdminCallAudit {
  /** Verbo + recurso, curto e estável: `npc.delete`, `item.price.set`. */
  action: string;
  /** Identificador do alvo em texto (`npc:42`). Nunca nome de personagem. */
  target?: string;
}

/** Extrai o `moderator_id` do request, para que ele não venha do browser. */
type WithoutModerator<Req> = Omit<Req, "moderator_id" | "moderatorId">;

/**
 * Remove as duas grafias do id do corpo, em runtime.
 *
 * O tipo `WithoutModerator` só protege quem compila: a Server Action recebe
 * JSON do browser, e um `moderator_id`/`moderatorId` plantado ali sobreviveria ao
 * spread e chegaria à RPC. O web-api usa `keepCase`, mas quem garante a autoria
 * é este ponto — o corpo do browser nunca escolhe o moderador.
 */
function withoutModerator(req: object): Record<string, unknown> {
  const { moderator_id: _snake, moderatorId: _camel, ...rest } = req as Record<string, unknown>;
  void _snake;
  void _camel;
  return rest;
}

/**
 * Resposta admin vista pelo `adminCall`. Alguns serviços (contas, issue 32)
 * devolvem `message` com a recusa de negócio ("última conta admin ativa");
 * quando ela existe, é mais útil que o texto genérico do `AdminResult`.
 */
type AdminResponse = { result: number; message?: string };

/**
 * Sessão de staff ou erro tipado — **auditando a tentativa negada**.
 *
 * O `requireStaff` sozinho lançaria antes de qualquer registro: uma conta
 * `player` tentando uma ação admin não deixaria rastro. Aqui a negação entra na
 * trilha com `result=FORBIDDEN`; sem sessão o `accountId` é `0` (anônimo).
 */
async function guardStaff(audit: AdminCallAudit, traceId: string): Promise<Session> {
  const session = await getSession();
  if (session && isStaff(session.role)) return session;

  const status = session ? 403 : 401;
  const message = session
    ? "sua conta não tem permissão para esta ação"
    : "sessão expirada: entre novamente";

  logAudit({
    accountId: session?.accountId ?? 0,
    role: session?.role ?? "player",
    action: audit.action,
    target: audit.target,
    result: ADMIN_RESULT.FORBIDDEN,
    status,
    traceId,
  });
  throw new AdminError(status, message);
}

export async function adminCall<Req extends object, Res extends AdminResponse>(
  fn: Unary<Req, Res>,
  req: WithoutModerator<Req>,
  audit: AdminCallAudit,
): Promise<Res> {
  // Um id por ação: a negação de permissão também é auditada com ele, então a
  // tentativa recusada e a RPC (quando acontece) dividem a mesma correlação.
  const traceId = newTraceId();
  const session = await guardStaff(audit, traceId);

  const res = await promisify(fn)(
    {
      ...withoutModerator(req),
      moderator_id: session.accountId,
    } as Req,
    traceId,
  );

  const status = httpStatusFor(res.result);
  logAudit({
    accountId: session.accountId,
    role: session.role,
    action: audit.action,
    target: audit.target,
    result: res.result,
    status,
    traceId,
  });

  if (!isOk(res.result)) throw new AdminError(status, res.message?.trim() || messageFor(res.result));
  return res;
}
