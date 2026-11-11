import { describe, expect, it, vi } from "vitest";

import { promisify } from "@/server/promisify";
import { TRACE_HEADER, newTraceId, traceHeaders, validTraceId } from "@/server/trace";

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

  it("preserva o id válido (normalizado) no header", () => {
    const headers = traceHeaders("0123456789ABCDEF0123456789ABCDEF");
    expect(headers[TRACE_HEADER]).toBe("0123456789abcdef0123456789abcdef");
  });

  it("substitui um id forjado em vez de propagá-lo", () => {
    const headers = traceHeaders("texto livre\ncom quebra");
    expect(validTraceId(headers[TRACE_HEADER])).toBe(true);
  });

  it("promisify encaminha request e trace_id para a chamada", async () => {
    const fn = vi.fn(async (req: { value: number }) => ({ result: req.value }));

    const id = newTraceId();
    const res = await promisify(fn)({ value: 7 }, id);

    expect(res).toEqual({ result: 7 });
    expect(fn).toHaveBeenCalledWith({ value: 7 }, id, undefined);
  });

  it("promisify encaminha o ator do BFF como terceiro argumento", async () => {
    const fn = vi.fn(async () => ({ result: 1 }));

    await promisify(fn)({ value: 1 }, undefined, "7:moderator");

    expect(fn).toHaveBeenCalledWith({ value: 1 }, undefined, "7:moderator");
  });

  it("promisify gera chamada sem trace quando não recebe nenhum", async () => {
    const fn = vi.fn(async () => ({ result: 1 }));

    await promisify(fn)({});

    expect(fn).toHaveBeenCalledWith({}, undefined, undefined);
  });
});
