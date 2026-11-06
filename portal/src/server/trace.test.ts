import * as grpc from "@grpc/grpc-js";
import { describe, expect, it } from "vitest";

import { promisify } from "./promisify";
import { TRACE_HEADER, newTraceId, traceMetadata, validTraceId } from "./trace";

describe("trace", () => {
  it("gera ids de 32 hex, distintos entre chamadas", () => {
    const ids = new Set<string>();
    for (let i = 0; i < 32; i++) {
      const id = newTraceId();
      expect(id).toMatch(/^[0-9a-f]{32}$/);
      ids.add(id);
    }
    expect(ids.size).toBe(32);
  });

  it("aceita só o formato esperado", () => {
    expect(validTraceId("0123456789abcdef0123456789abcdef")).toBe(true);
    expect(validTraceId("0123456789ABCDEF0123456789ABCDEF")).toBe(true);
    expect(validTraceId("")).toBe(false);
    expect(validTraceId("not-a-trace-id")).toBe(false);
    expect(validTraceId("0123456789abcdef0123456789abcde")).toBe(false);
    expect(validTraceId(undefined)).toBe(false);
  });

  it("preserva o id válido (normalizado) no metadata", () => {
    const md = traceMetadata("0123456789ABCDEF0123456789ABCDEF");
    expect(md.get(TRACE_HEADER)).toEqual(["0123456789abcdef0123456789abcdef"]);
  });

  it("substitui um id forjado em vez de propagá-lo", () => {
    const md = traceMetadata("texto livre\ncom quebra");
    const got = md.get(TRACE_HEADER);
    expect(got).toHaveLength(1);
    // get() devolve MetadataValue (string | Buffer); o header é sempre texto.
    expect(validTraceId(got?.[0]?.toString())).toBe(true);
  });

  it("promisify manda o trace_id recebido no metadata da RPC", async () => {
    let seen: grpc.Metadata | undefined;
    let sent: unknown;
    const fn = (
      req: { value: number },
      md: grpc.Metadata,
      _opts: grpc.CallOptions,
      cb: (err: grpc.ServiceError | null, res: { result: number }) => void,
    ) => {
      sent = req;
      seen = md;
      cb(null, { result: 1 });
    };

    const id = newTraceId();
    const res = await promisify(fn)({ value: 7 }, id);

    expect(res).toEqual({ result: 1 });
    expect(sent).toEqual({ value: 7 });
    expect(seen?.get(TRACE_HEADER)).toEqual([id]);
  });

  it("promisify gera um id quando não recebe nenhum", async () => {
    let seen: grpc.Metadata | undefined;
    const fn = (
      _req: object,
      md: grpc.Metadata,
      _opts: grpc.CallOptions,
      cb: (err: grpc.ServiceError | null, res: { result: number }) => void,
    ) => {
      seen = md;
      cb(null, { result: 1 });
    };

    await promisify(fn)({});
    const got = seen?.get(TRACE_HEADER);
    expect(got).toHaveLength(1);
    // get() devolve MetadataValue (string | Buffer); o header é sempre texto.
    expect(validTraceId(got?.[0]?.toString())).toBe(true);
  });
});
