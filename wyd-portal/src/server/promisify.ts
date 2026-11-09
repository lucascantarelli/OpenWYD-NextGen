/**
 * Adaptador de chamada admin (ADR-0008).
 *
 * O cliente `admin-client.ts` já devolve `Promise` (fetch), aplica deadline de
 * 10 s (`AbortSignal.timeout`) e propaga `x-trace-id`. Este módulo existe para
 * manter a assinatura usada por `adminCall` (`promisify(fn)(req, traceId)`)
 * sem acoplar o BFF ao transporte.
 */
export const DEFAULT_TIMEOUT_MS = 10_000;

/**
 * Chamada admin unária: request → envelope `AdminResult` + campos do domínio.
 *
 * `actor` (`id:role`) é a conta de staff que está agindo; vai como
 * `x-wyd-actor` para o gateway autenticar por papel, nunca pelo token de
 * serviço. Chamada sem sessão (login) simplesmente omite o terceiro argumento.
 */
export type Unary<Req, Res> = (req: Req, traceId?: string, actor?: string) => Promise<Res>;

/**
 * Devolve a chamada pronta. A função recebida já carrega o deadline do cliente;
 * o parâmetro de timeout fica só por compatibilidade de assinatura.
 */
export function promisify<Req, Res>(fn: Unary<Req, Res>, _timeoutMs = DEFAULT_TIMEOUT_MS) {
  return (req: Req, traceId?: string, actor?: string): Promise<Res> =>
    fn(req, traceId, actor);
}
