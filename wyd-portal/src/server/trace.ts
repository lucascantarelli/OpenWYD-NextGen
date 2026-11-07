/**
 * `trace_id` do BFF (OBS-02).
 *
 * O portal gera um id por ação e o manda no metadata gRPC `x-trace-id`; o
 * web-api devolve o mesmo id no header e o inclui em cada linha de log. Com
 * isso `docker compose logs | grep <id>` liga a ação do moderador à RPC e ao SQL,
 * sem procurar por conta/horário.
 *
 * O id **não** vira label de métrica: cardenalidade por requisição estoura a
 * série temporal.
 */
import * as grpc from "@grpc/grpc-js";
import { randomBytes } from "node:crypto";

export const TRACE_HEADER = "x-trace-id";

/** 32 dígitos hex: 16 bytes aleatórios, o mesmo formato do lado Go. */
const TRACE_ID_RE = /^[0-9a-f]{32}$/;

export function newTraceId(): string {
  return randomBytes(16).toString("hex");
}

/** Aceita só o formato esperado; qualquer outra coisa é descartada. */
export function validTraceId(id: string | null | undefined): id is string {
  return typeof id === "string" && TRACE_ID_RE.test(id.toLowerCase());
}

/**
 * Metadata da chamada com o trace id: o recebido (normalizado) ou um novo. Um id
 * forjado/ inválido nunca entra no metadata nem no log.
 */
export function traceMetadata(traceId?: string): grpc.Metadata {
  const md = new grpc.Metadata();
  md.set(TRACE_HEADER, validTraceId(traceId) ? traceId.toLowerCase() : newTraceId());
  return md;
}
