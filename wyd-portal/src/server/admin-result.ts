/**
 * `AdminResult` → HTTP + mensagem. Módulo **puro** (nenhum import): é o mapa
 * que o bug #257 mostrou ser perigoso quando incompleto — `CONTENT_OWNED` é um
 * resultado esperado de `DeleteNpc`, não um erro interno.
 *
 * Fonte do enum: `api/web/v1/web.proto:253`.
 */
export const ADMIN_RESULT = {
  UNSPECIFIED: 0,
  OK: 1,
  FORBIDDEN: 2,
  INVALID: 3,
  NOT_FOUND: 4,
  CONTENT_OWNED: 5,
} as const;

const HTTP: Record<number, number> = {
  [ADMIN_RESULT.UNSPECIFIED]: 500,
  [ADMIN_RESULT.OK]: 200,
  [ADMIN_RESULT.FORBIDDEN]: 403,
  [ADMIN_RESULT.INVALID]: 422,
  [ADMIN_RESULT.NOT_FOUND]: 404,
  [ADMIN_RESULT.CONTENT_OWNED]: 409,
};

const MESSAGE: Record<number, string> = {
  [ADMIN_RESULT.UNSPECIFIED]: "resultado desconhecido do web-api",
  [ADMIN_RESULT.FORBIDDEN]: "sua conta não tem permissão para esta ação",
  [ADMIN_RESULT.INVALID]: "dados inválidos para esta ação",
  [ADMIN_RESULT.NOT_FOUND]: "o registro não existe mais",
  [ADMIN_RESULT.CONTENT_OWNED]: "este item é conteúdo versionado: oculte em vez de apagar",
};

/** Status HTTP do resultado. `UNSPECIFIED` e valor desconhecido caem em 500. */
export function httpStatusFor(result: number): number {
  return HTTP[result] ?? 500;
}

/** Mensagem acionável do resultado, genérica quando o código é desconhecido. */
export function messageFor(result: number): string {
  return MESSAGE[result] ?? MESSAGE[ADMIN_RESULT.UNSPECIFIED];
}

export function isOk(result: number): boolean {
  return result === ADMIN_RESULT.OK;
}
