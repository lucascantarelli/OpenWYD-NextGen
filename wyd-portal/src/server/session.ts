/**
 * Sessão do portal em cookie httpOnly assinado (HS256, 8h).
 *
 * O `role` guardado aqui é **dica de UI**: quem autoriza de verdade é o web-api,
 * que relê `account.role` em toda RPC admin (web.proto:VerifyCredentialsResponse).
 * Nunca coloque senha, hash ou token de jogo no cookie.
 */
import { cookies } from "next/headers";
import { SignJWT, jwtVerify } from "jose";

import { AdminError } from "./errors";
import { SESSION_COOKIE, SESSION_TTL_SECONDS, sessionKey } from "./session-key";

export type Role = "player" | "moderator" | "admin";

export interface Session {
  accountId: number;
  role: Role;
}

const ROLES: readonly Role[] = ["player", "moderator", "admin"];

/** Normaliza o papel vindo do banco; valor desconhecido cai em `player`. */
export function normalizeRole(raw: string): Role {
  const value = raw.trim().toLowerCase();
  return (ROLES as readonly string[]).includes(value) ? (value as Role) : "player";
}

/** Só `moderator` e `admin` abrem a UI de administração. */
export function isStaff(role: Role): boolean {
  return role === "moderator" || role === "admin";
}

export async function setSession(accountId: number, role: Role): Promise<void> {
  const token = await new SignJWT({ role })
    .setProtectedHeader({ alg: "HS256" })
    .setSubject(String(accountId))
    .setIssuedAt()
    .setExpirationTime(`${SESSION_TTL_SECONDS}s`)
    .sign(sessionKey());

  (await cookies()).set(SESSION_COOKIE, token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: SESSION_TTL_SECONDS,
  });
}

/** Sessão válida (assinatura + expiração) ou `null`. Nunca lança. */
export async function getSession(): Promise<Session | null> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) return null;

  try {
    const { payload } = await jwtVerify(token, sessionKey(), { algorithms: ["HS256"] });
    const accountId = Number(payload.sub);
    if (!Number.isInteger(accountId) || accountId <= 0) return null;
    return { accountId, role: normalizeRole(String(payload.role ?? "")) };
  } catch {
    return null;
  }
}

/**
 * Sessão de staff ou erro tipado (401 sem sessão, 403 para `player`).
 *
 * Usado **só** no server, via `adminCall`: o papel guardado no cookie é dica de
 * UI; o web-api revalida `account.role` em toda RPC e pode responder
 * `FORBIDDEN` mesmo com cookie válido.
 */
export async function requireStaff(): Promise<Session> {
  const session = await getSession();
  if (!session) throw new AdminError(401, "sessão expirada: entre novamente");
  if (!isStaff(session.role)) {
    throw new AdminError(403, "sua conta não tem permissão para esta ação");
  }
  return session;
}

export async function clearSession(): Promise<void> {
  (await cookies()).delete(SESSION_COOKIE);
}
