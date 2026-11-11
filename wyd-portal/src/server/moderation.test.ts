import { describe, expect, it } from "vitest";

import {
  MAX_AUDIT_ROWS,
  clampAuditLimit,
  filterAuditTrail,
  loginHistoryFor,
  type AuditEntryView,
} from "./moderation";

const linha = (over: Partial<AuditEntryView>): AuditEntryView => ({
  at: new Date("2026-01-01T12:00:00Z"),
  actorId: 7,
  actorRole: "moderator",
  action: "accounts.block",
  target: "login:alvo",
  detail: "bloqueio: spam",
  ...over,
});

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

describe("filtro da trilha de auditoria", () => {
  const linhas = [
    linha({}),
    linha({ action: "grants.request", target: "char:9", actorId: 8, actorRole: "admin" }),
    linha({ action: "auth.login", target: "Alvo", detail: "ok" }),
  ];

  it("sem filtro devolve tudo", () => {
    expect(filterAuditTrail(linhas, {})).toHaveLength(3);
  });

  it("filtra por ação e alvo com substring case-insensitive", () => {
    expect(filterAuditTrail(linhas, { action: "grants" })).toHaveLength(1);
    expect(filterAuditTrail(linhas, { target: "CHAR:9" })).toHaveLength(1);
    expect(filterAuditTrail(linhas, { action: "inexistente" })).toHaveLength(0);
  });

  it("filtra por autor (id ou papel) e por período", () => {
    expect(filterAuditTrail(linhas, { actor: "admin" })).toHaveLength(1);
    expect(filterAuditTrail(linhas, { actor: "#7" })).toHaveLength(2);
    const depois = new Date("2026-06-01T00:00:00Z").getTime();
    expect(filterAuditTrail(linhas, { sinceMs: depois })).toHaveLength(0);
  });
});

describe("histórico de login da ficha", () => {
  const linhas = [
    linha({ action: "auth.login", target: "alvo", detail: "ok" }),
    linha({ action: "auth.login", target: "outra", detail: "ok" }),
    linha({ action: "accounts.block", target: "login:alvo" }),
  ];

  it("pega só auth.login com o login exato, ignorando caixa", () => {
    expect(loginHistoryFor(linhas, "alvo")).toHaveLength(1);
    expect(loginHistoryFor(linhas, "ALVO")).toHaveLength(1);
  });

  it("login vazio não casa com nada", () => {
    expect(loginHistoryFor(linhas, "  ")).toHaveLength(0);
  });
});
