import { describe, expect, it } from "vitest";

import {
  bucketForDays,
  fmtBucketLabel,
  fmtMoment,
  ordersCsv,
  resolveWindow,
  type TopupOrderRowWire,
} from "./revenue";

const NOW = new Date("2026-11-04T15:00:00-03:00");

describe("resolveWindow", () => {
  it("conta o preset para trás a partir de agora", () => {
    const result = resolveWindow({ period: "7d" }, NOW);
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.to - result.from).toBe(7 * 86_400);
    expect(result.days).toBe(7);
  });

  it("interpreta as datas no fuso de Brasília e inclui o dia final", () => {
    const result = resolveWindow({ period: "custom", desde: "2026-10-01", ate: "2026-10-31" }, NOW);
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    // 01/10 00:00 -03:00 → 2026-10-01T03:00Z
    expect(result.from).toBe(Date.UTC(2026, 9, 1, 3, 0, 0) / 1000);
    // 31/10 inclusive → 01/11 00:00 -03:00 (exclusivo)
    expect(result.to).toBe(Date.UTC(2026, 10, 1, 3, 0, 0) / 1000);
    expect(result.days).toBe(31);
  });

  it("recusa data inválida, intervalo invertido e janela acima de 366 dias", () => {
    expect(resolveWindow({ period: "custom", desde: "01/10/2026", ate: "2026-10-31" }, NOW)).toEqual({
      ok: false,
      error: expect.stringContaining("AAAA-MM-DD") as unknown as string,
    });
    const inverted = resolveWindow({ period: "custom", desde: "2026-11-01", ate: "2026-10-01" }, NOW);
    expect(inverted.ok).toBe(false);
    const wide = resolveWindow({ period: "custom", desde: "2024-01-01", ate: "2026-12-31" }, NOW);
    expect(wide.ok).toBe(false);
    if (!wide.ok) expect(wide.error).toContain("366");
  });
});

describe("bucketForDays", () => {
  it("escolhe a granularidade pela largura da janela", () => {
    expect(bucketForDays(7)).toBe(1);
    expect(bucketForDays(45)).toBe(1);
    expect(bucketForDays(90)).toBe(2);
    expect(bucketForDays(200)).toBe(2);
    expect(bucketForDays(365)).toBe(3);
  });
});

describe("rótulos", () => {
  it("formata bucket em Brasília", () => {
    const unix = Date.UTC(2026, 10, 1, 3, 0, 0) / 1000; // 01/11/2026 00:00 -03:00
    expect(fmtBucketLabel(unix, 1)).toBe("01/11");
    expect(fmtBucketLabel(unix, 2)).toBe("sem. 01/11");
    expect(fmtBucketLabel(unix, 3)).toBe("nov/26");
  });

  it("marca instante ausente com travessão", () => {
    expect(fmtMoment(0)).toBe("—");
    expect(fmtMoment(Date.UTC(2026, 10, 1, 17, 30, 0) / 1000)).toBe("01/11/2026, 14:30");
  });
});

function order(overrides: Partial<TopupOrderRowWire> = {}): TopupOrderRowWire {
  return {
    id: 7,
    external_reference: "abc-123",
    account_id: 3,
    account_name: "test",
    account_email: "test@example.com",
    payer_name: "Ana; Souza",
    payer_cpf_masked: "***.456.789-**",
    credits: 250,
    amount_cents: 5000,
    payment_method: 1,
    provider: "",
    status: 2,
    created_at_unix: Date.UTC(2026, 10, 1, 17, 30, 0) / 1000,
    confirmed_at_unix: Date.UTC(2026, 10, 1, 17, 35, 0) / 1000,
    ...overrides,
  };
}

describe("ordersCsv", () => {
  it("usa ; e vírgula decimal, com aspas quando o campo tem separador", () => {
    const csv = ordersCsv([order()]);
    const [, line] = csv.split("\r\n");
    expect(csv.startsWith("\ufeff")).toBe(true);
    expect(line).toContain('"Ana; Souza"');
    expect(line).toContain(";5000;50,00;");
    expect(line).toContain(";PIX;Pago;");
    expect(line).toContain("01/11/2026, 14:30");
  });

  it("deixa pago_em vazio quando o pedido está pendente", () => {
    const csv = ordersCsv([order({ status: 1, confirmed_at_unix: 0 })]);
    const [, line] = csv.split("\r\n");
    expect(line.endsWith(";Pendente;" + "01/11/2026, 14:30;")).toBe(true);
  });
});
