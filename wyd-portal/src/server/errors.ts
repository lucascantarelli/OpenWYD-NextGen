/**
 * Erros do BFF: **um** tipo com status HTTP + mensagem pronta para o usuário.
 *
 * Regra da casa: nenhuma action/page monta `{ error }` na mão. Ela lança
 * `AdminError` (ou deixa a falha de transporte subir) e a rota chama
 * `toResponse`. Assim o browser nunca recebe stack, DSN, caminho de arquivo ou
 * nome de tabela.
 */

const NO_STORE = { "cache-control": "no-store" } as const;

export class AdminError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "AdminError";
    this.status = status;
  }
}

/**
 * Backend de administração fora do ar/inacessível (conexão recusada, timeout,
 * 5xx). Não confundir com `AdminResult`, que viaja no corpo com status 200.
 */
export class AdminUnavailableError extends Error {
  constructor(message = "serviço de administração indisponível") {
    super(message);
    this.name = "AdminUnavailableError";
  }
}

/** Falha de infraestrutura do `wyd-admin` (não é recusa de negócio). */
export function isInfraFailure(err: unknown): boolean {
  return err instanceof AdminUnavailableError;
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
