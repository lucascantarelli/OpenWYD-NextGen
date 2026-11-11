/**
 * Painel do jogador (FASE-07-2) — leitura no `wyd-admin` para quem está logado.
 *
 * Diferença para `adminCall`: aqui qualquer papel autenticado pode ler **a própria**
 * conta. O `account_id` sai do cookie de sessão assinado (`getSession`), nunca de
 * parâmetro do browser; o backend audita a leitura com esse id interno.
 *
 * Nada aqui escreve estado vivo: `requestRecovery` registra uma solicitação
 * auditada e a resposta é a mesma exista ou não o login.
 */
import { ADMIN_RESULT, httpStatusFor, messageFor } from "./admin-result";
import { AdminError, AdminUnavailableError } from "./errors";
import { getSession } from "./session";
import { newTraceId, traceHeaders } from "./trace";

const ADMIN_URL = (process.env.WEB_API_ADDR ?? "http://127.0.0.1:7600").replace(/\/+$/, "");
const ADMIN_TOKEN = process.env.WEB_API_TOKEN ?? "";
const TIMEOUT_MS = 10_000;

export interface PanelAccount {
  id: number;
  login: string;
  role: string;
  /** A conta já tem segunda senha; `false` significa que o jogador ainda vai criá-la no cliente. */
  pin_defined: boolean;
}

export interface PanelCharacter {
  id: number;
  slot: number;
  name: string;
  class: number;
  level: number;
}

export interface RankingEntry {
  rank: number;
  name: string;
  class: number;
  level: number;
}

export interface PanelSession {
  group_id: number;
  channel: number;
  client_version: number;
  character_id: number | null;
  opened_ms: number;
  renewed_ms: number;
  released_ms: number | null;
}

export interface PanelEvent {
  at_ms: number;
  action: string;
  detail: string;
}

export interface PanelHistory {
  sessions: PanelSession[];
  events: PanelEvent[];
}

interface Envelope {
  result: number;
  message?: string;
  trace_id?: string;
  account?: PanelAccount;
  characters?: PanelCharacter[];
  ranking?: RankingEntry[];
  sessions?: PanelSession[];
  events?: PanelEvent[];
  requested?: boolean;
}

async function call<T extends Envelope>(
  path: string,
  params: Record<string, unknown>,
  traceId?: string,
): Promise<T> {
  const id = traceId && traceId.trim() !== "" ? traceId : newTraceId();
  let response: Response;
  try {
    response = await fetch(`${ADMIN_URL}/admin/v1/${path}`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        ...traceHeaders(id),
        ...(ADMIN_TOKEN ? { authorization: `Bearer ${ADMIN_TOKEN}` } : {}),
      },
      body: JSON.stringify(params),
      signal: AbortSignal.timeout(TIMEOUT_MS),
      cache: "no-store",
    });
  } catch {
    throw new AdminUnavailableError("serviço de administração indisponível");
  }

  const text = await response.text();
  let body: Partial<Envelope> = {};
  try {
    body = text ? (JSON.parse(text) as Partial<Envelope>) : {};
  } catch {
    body = {};
  }
  if (!response.ok) {
    if (response.status >= 500) {
      throw new AdminUnavailableError("serviço de administração indisponível");
    }
    throw new AdminError(response.status, body.message ?? "chamada recusada");
  }
  if (typeof body.result !== "number") {
    throw new AdminUnavailableError("resposta inválida do serviço de administração");
  }
  if (body.result !== ADMIN_RESULT.OK) {
    throw new AdminError(httpStatusFor(body.result), body.message ?? messageFor(body.result));
  }
  return body as T;
}

/** Conta da sessão; sem sessão é 401 (nunca lê id de query/body do browser). */
async function ownAccountId(): Promise<number> {
  const session = await getSession();
  if (!session) throw new AdminError(401, "sessão expirada: entre novamente");
  return session.accountId;
}

export async function getPanelAccount(traceId?: string): Promise<PanelAccount> {
  const accountId = await ownAccountId();
  const res = await call<Envelope>("panel/self", { account_id: accountId }, traceId);
  if (!res.account) throw new AdminUnavailableError("conta ausente na resposta");
  return res.account;
}

export async function getPanelCharacters(traceId?: string): Promise<PanelCharacter[]> {
  const accountId = await ownAccountId();
  const res = await call<Envelope>("panel/characters", { account_id: accountId }, traceId);
  return res.characters ?? [];
}

/** Ranking na área do jogador: exige sessão (o ator é a própria conta). */
export async function getPanelRankings(limit = 10, traceId?: string): Promise<RankingEntry[]> {
  await ownAccountId();
  return getPublicRankings(limit, traceId);
}

/**
 * Ranking público (home e `/ranking`).
 *
 * A rota `panel/rankings` do gateway não leva identidade de conta: o BFF chama
 * com o token de processo e o gateway audita com a identidade de serviço. Só
 * nome, classe e nível do personagem saem daqui — nunca id de conta ou login.
 */
export async function getPublicRankings(limit = 10, traceId?: string): Promise<RankingEntry[]> {
  const safe = Number.isInteger(limit) && limit > 0 && limit <= 100 ? limit : 10;
  const res = await call<Envelope>("panel/rankings", { limit: safe }, traceId);
  return res.ranking ?? [];
}

/**
 * Sessões e tentativas de login da própria conta.
 *
 * O `account_id` sai do cookie de sessão; o gateway lê o login no banco. A trilha
 * devolve só a ação e o resultado (`ok`, bloqueio, credencial recusada) — nunca
 * senha, hash ou PIN.
 */
export async function getPanelHistory(traceId?: string): Promise<PanelHistory> {
  const accountId = await ownAccountId();
  const res = await call<Envelope>("panel/history", { account_id: accountId }, traceId);
  return { sessions: res.sessions ?? [], events: res.events ?? [] };
}

export interface ChangePinInput {
  password: string;
  currentPin: string;
  newPin: string;
}

/**
 * Troca o PIN pelo portal — **sempre** pelo servidor de jogo.
 *
 * O gateway executa o `0x0FDE` em duas etapas (`change_numeric` 1 com o PIN atual,
 * 2 com o novo); o portal não grava hash nem decide se o PIN vale. `login` vem de
 * `getPanelAccount` (sessão), nunca do corpo do browser.
 */
export async function changeNumericPin(
  login: string,
  input: ChangePinInput,
  traceId?: string,
): Promise<void> {
  await call<Envelope>(
    "account_web/change_numeric_pin",
    {
      login,
      password: input.password,
      current_pin: input.currentPin,
      new_pin: input.newPin,
    },
    traceId,
  );
}

/** Solicitação de recuperação: resposta idêntica exista ou não o login. */
export async function requestRecovery(login: string, traceId?: string): Promise<void> {
  await call<Envelope>("accounts/recovery", { login }, traceId);
}
