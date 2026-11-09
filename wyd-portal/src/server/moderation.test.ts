import { describe, expect, it } from "vitest";

import { MAX_AUDIT_ROWS, clampAuditLimit } from "./moderation";

/**
 * O gateway recusa limite fora de 1..=200 (`api::moderation::recent_audit`).
 * O portal precisa pedir dentro da faixa: um `limit` inventado na query string
 * viraria 422 e a tela de auditoria mostraria "erro" em vez da trilha.
 */
describe("limite da trilha de auditoria", () => {
  it("mantém valor dentro da faixa", () => {
    expect(clampAuditLimit(50)).toBe(50);
    expect(clampAuditLimit(1)).toBe(1);
    expect(clampAuditLimit(MAX_AUDIT_ROWS)).toBe(MAX_AUDIT_ROWS);
  });

  it("corta acima do teto e abaixo do piso", () => {
    expect(clampAuditLimit(0)).toBe(1);
    expect(clampAuditLimit(-10)).toBe(1);
    expect(clampAuditLimit(1000)).toBe(MAX_AUDIT_ROWS);
  });

  it("trunca fração e cai no padrão para valor não finito", () => {
    expect(clampAuditLimit(12.9)).toBe(12);
    expect(clampAuditLimit(Number.NaN)).toBe(50);
    expect(clampAuditLimit(Number.POSITIVE_INFINITY)).toBe(50);
  });
});
