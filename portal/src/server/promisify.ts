/**
 * Único caminho de promisificação de RPC do portal.
 *
 * Além de transformar callback em `Promise`, aplica **deadline** de 10s via
 * `CallOptions`. Sem isso uma RPC pendurada deixa a request do browser aberta
 * até o proxy do Next desistir — e o usuário acha que a UI travou.
 *
 * O segundo argumento é o `trace_id` (OBS-02): vai no metadata `x-trace-id`, e
 * sem ele um id novo é gerado, então nenhuma chamada sai sem correlação.
 */
import * as grpc from "@grpc/grpc-js";

import { traceMetadata } from "./trace";

export const DEFAULT_TIMEOUT_MS = 10_000;

/**
 * Assinatura real de um unary do grpc-js (request, metadata, options, callback).
 * O gRPC aceita as formas curtas, mas aqui sempre passamos metadata e options.
 */
export type Unary<Req, Res> = (
  req: Req,
  metadata: grpc.Metadata,
  options: grpc.CallOptions,
  cb: (err: grpc.ServiceError | null, res: Res) => void,
) => void;

export function promisify<Req, Res>(fn: Unary<Req, Res>, timeoutMs = DEFAULT_TIMEOUT_MS) {
  return (req: Req, traceId?: string) =>
    new Promise<Res>((resolve, reject) => {
      let settled = false;

      // Rede de segurança para o caso do canal sumir sem RST: o deadline
      // deveria disparar antes, este timer garante que a promise sempre fecha.
      const timer = setTimeout(() => {
        if (settled) return;
        settled = true;
        reject(new Error("web-api: tempo esgotado"));
      }, timeoutMs + 1_000);
      timer.unref?.();

      fn(req, traceMetadata(traceId), { deadline: Date.now() + timeoutMs }, (err, res) => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        if (err) reject(err);
        else resolve(res);
      });
    });
}
