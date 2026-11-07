/**
 * Erros do BFF: **um** tipo com status HTTP + mensagem pronta para o usuário.
 *
 * Regra da casa (issue 21): nenhuma action/page monta `{ error }` na mão. Ela
 * lança `AdminError` (ou deixa o erro gRPC subir) e a rota chama `toResponse`.
 * Assim o browser nunca recebe stack, DSN, caminho de arquivo ou nome de tabela.
 */

/** Códigos do grpc-js que significam "o web-api não está aí / não respondeu". */
const GRPC_DEADLINE_EXCEEDED = 4;
const GRPC_UNAVAILABLE = 14;

const NO_STORE = { "cache-control": "no-store" } as const;

export class AdminError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "AdminError";
    this.status = status;
  }
}

function grpcCode(err: unknown): number | null {
  if (typeof err !== "object" || err === null) return null;
  const code = (err as { code?: unknown }).code;
  return typeof code === "number" ? code : null;
}

/**
 * Falha de infraestrutura do web-api: erro gRPC de transporte (`UNAVAILABLE`,
 * `DEADLINE_EXCEEDED`) — não confundir com `AdminResult`, que viaja no corpo.
 */
export function isInfraFailure(err: unknown): boolean {
  const code = grpcCode(err);
  return code === GRPC_UNAVAILABLE || code === GRPC_DEADLINE_EXCEEDED;
}

/** Converte qualquer falha em `Response` JSON sem detalhe interno. */
export function toResponse(err: unknown): Response {
  if (err instanceof AdminError) {
    return Response.json({ error: err.message }, { status: err.status, headers: NO_STORE });
  }

  if (isInfraFailure(err)) {
    return Response.json(
      { error: "serviço de administração indisponível" },
      { status: 502, headers: NO_STORE },
    );
  }

  // Só a mensagem vai para o log do servidor; o cliente recebe o genérico.
  console.error("[bff] falha inesperada:", err instanceof Error ? err.message : String(err));
  return Response.json({ error: "erro interno do portal" }, { status: 500, headers: NO_STORE });
}
