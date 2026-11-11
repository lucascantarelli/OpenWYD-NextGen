import { describe, expect, it } from "vitest";

import { scheduleLabel, weekdayLabel } from "./events";

/**
 * Rótulos da grade de eventos: dia da semana e horário vêm do banco
 * (`event_schedule`, com `timezone` explícita); o portal só formata.
 */
describe("rótulos da grade de eventos", () => {
  it("nomeia os dias 0..6", () => {
    expect(weekdayLabel(0)).toBe("dom");
    expect(weekdayLabel(5)).toBe("sex");
    expect(weekdayLabel(6)).toBe("sáb");
  });

  it("devolve o número quando o dia sai da faixa", () => {
    expect(weekdayLabel(9)).toBe("dia 9");
  });

  it("formata horário e duração", () => {
    expect(scheduleLabel("20:00", 60)).toBe("20:00 · 60 min");
  });

  it("mostra travessão para grade incompleta", () => {
    expect(scheduleLabel("", 60)).toBe("—");
    expect(scheduleLabel("20:00", 0)).toBe("—");
    expect(scheduleLabel("20:00", Number.NaN)).toBe("—");
  });
});
