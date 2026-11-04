import { describe, expect, it } from "vitest";

import { ADMIN_RESULT, httpStatusFor, isOk, messageFor } from "./admin-result";

/**
 * O mapa `AdminResult → HTTP` é o ponto onde o bug #257 nasceu: um valor de
 * negócio esperado (`CONTENT_OWNED`) virou 500 e o moderador viu "erro
 * interno" ao tentar apagar um NPC de conteúdo.
 */
describe("AdminResult → HTTP", () => {
  it("OK vira 200", () => {
    expect(httpStatusFor(ADMIN_RESULT.OK)).toBe(200);
    expect(isOk(ADMIN_RESULT.OK)).toBe(true);
  });

  it("FORBIDDEN vira 403 com mensagem de permissão", () => {
    expect(httpStatusFor(ADMIN_RESULT.FORBIDDEN)).toBe(403);
    expect(messageFor(ADMIN_RESULT.FORBIDDEN)).toMatch(/permissão/i);
  });

  it("INVALID vira 422 com mensagem de dados", () => {
    expect(httpStatusFor(ADMIN_RESULT.INVALID)).toBe(422);
    expect(messageFor(ADMIN_RESULT.INVALID)).toMatch(/dados inválidos/i);
  });

  it("NOT_FOUND vira 404", () => {
    expect(httpStatusFor(ADMIN_RESULT.NOT_FOUND)).toBe(404);
    expect(messageFor(ADMIN_RESULT.NOT_FOUND)).toMatch(/não existe/i);
  });

  it("CONTENT_OWNED vira 409 (não 500) e diz para ocultar", () => {
    expect(httpStatusFor(ADMIN_RESULT.CONTENT_OWNED)).toBe(409);
    expect(messageFor(ADMIN_RESULT.CONTENT_OWNED)).toMatch(/oculte em vez de apagar/i);
  });

  it("UNSPECIFIED vira 500 com mensagem genérica", () => {
    expect(httpStatusFor(ADMIN_RESULT.UNSPECIFIED)).toBe(500);
    expect(messageFor(ADMIN_RESULT.UNSPECIFIED)).toMatch(/desconhecido/i);
  });

  it("valor fora do enum vira 500 genérico (sem vazar o código)", () => {
    expect(httpStatusFor(99)).toBe(500);
    expect(httpStatusFor(-1)).toBe(500);
    expect(messageFor(99)).toBe(messageFor(ADMIN_RESULT.UNSPECIFIED));
  });

  it("nenhum resultado de negócio é confundido com sucesso", () => {
    for (const code of [0, 2, 3, 4, 5, 99]) {
      expect(isOk(code)).toBe(false);
      expect(httpStatusFor(code)).not.toBe(200);
    }
  });
});
