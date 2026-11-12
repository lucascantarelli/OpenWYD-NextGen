import { describe, expect, it } from "vitest";

import { POST } from "@/app/api/launcher/telemetria/route";
import { resumo } from "@/server/launcher-telemetria";

/**
 * Rota pública de telemetria: aceita só o contrato, limita corpo e nunca devolve
 * corpo em sucesso (204) — o launcher não pode depender dela para abrir o jogo.
 */

const versao = 7301;

function req(body: unknown, headers: Record<string, string> = {}): Request {
  return new Request("http://127.0.0.1:3000/api/launcher/telemetria", {
    method: "POST",
    headers: { "content-type": "application/json", ...headers },
    body: typeof body === "string" ? body : JSON.stringify(body),
  });
}

const valido = {
  versao,
  build_cliente: 7662,
  operacao: "patch",
  ok: true,
  duracao_ms: 900,
  arquivos: 2,
  falhas: [{ arquivo: "jogo.dat", motivo: "rede" }],
};

describe("POST /api/launcher/telemetria", () => {
  it("aceita evento válido, responde 204 sem corpo e sem cache", async () => {
    const antes = resumo().chaves.find((c) => c.versao === versao)?.eventos ?? 0;

    const res = await POST(req(valido));

    expect(res.status).toBe(204);
    expect(res.headers.get("cache-control")).toBe("no-store");
    expect(await res.text()).toBe("");
    const depois = resumo().chaves.find((c) => c.versao === versao)!;
    expect(depois.eventos - antes).toBe(1);
    expect(depois.falhas).toBeGreaterThanOrEqual(1);
  });

  it("recusa campo extra (PII) com 400 e sem gravar", async () => {
    const antes = resumo().totalEventos;

    const res = await POST(req({ ...valido, login: "jogador", ip: "10.0.0.1" }));

    expect(res.status).toBe(400);
    expect(await res.json()).toEqual({ error: "telemetria fora do contrato" });
    expect(resumo().totalEventos).toBe(antes);
  });

  it("recusa JSON inválido com 400", async () => {
    const res = await POST(req("{nao-e-json"));
    expect(res.status).toBe(400);
  });

  it("recusa content-type que não é JSON com 415", async () => {
    const res = await POST(req(valido, { "content-type": "text/plain" }));
    expect(res.status).toBe(415);
  });

  it("recusa corpo acima do limite com 413", async () => {
    const grande = JSON.stringify({ ...valido, falhas: [] }) + " ".repeat(5000);
    const res = await POST(req(grande));
    expect(res.status).toBe(413);
  });
});
