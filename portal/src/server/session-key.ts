/** Chave HS256 do cookie de sessão, compartilhada por session.ts e o middleware. */

const MIN_LENGTH = 32;

/**
 * Lê `SESSION_SECRET` do ambiente. Falha alto e cedo: um segredo curto ou vazio
 * permitiria forjar o cookie de sessão (issue 19, riscos).
 */
export function sessionKey(): Uint8Array {
  const raw = process.env.SESSION_SECRET ?? "";
  if (raw.length < MIN_LENGTH) {
    throw new Error(
      `SESSION_SECRET ausente ou com menos de ${MIN_LENGTH} caracteres (gere com: openssl rand -base64 48)`,
    );
  }
  return new TextEncoder().encode(raw);
}

export const SESSION_COOKIE = "wyd_session";
export const SESSION_TTL_SECONDS = 8 * 60 * 60;
