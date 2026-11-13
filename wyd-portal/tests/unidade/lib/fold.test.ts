import { describe, expect, it } from "vitest";

import { fold, foldMatch } from "@/lib/fold";

/**
 * A dobra precisa dar o mesmo resultado de `content_fold()` (Postgres, 0007).
 * Se divergir, o painel acha um item e o site não — ou pior, a busca pública
 * esconde item que existe.
 */
describe("fold", () => {
  it("remove acento e caixa", () => {
    expect(fold("Poção")).toBe("pocao");
    expect(fold("POCAO")).toBe("pocao");
    expect(fold("Calça de Malha")).toBe("calca de malha");
  });

  it("normaliza NFD (macOS/IME) igual a NFC", () => {
    const nfc = "Poção".normalize("NFC");
    const nfd = "Poção".normalize("NFD");

    expect(nfd).not.toBe(nfc);
    expect(fold(nfd)).toBe(fold(nfc));
  });

  it("troca ligaduras e ß como o SQL", () => {
    expect(fold("Æther")).toBe("aether");
    expect(fold("Œuvre")).toBe("oeuvre");
    expect(fold("Straße")).toBe("strasse");
  });

  it("colapsa separador e apara as pontas", () => {
    expect(fold("  Espada__do__Caos  ")).toBe("espada do caos");
  });
});

describe("foldMatch", () => {
  it("aceita termos em qualquer ordem", () => {
    expect(foldMatch("Capa de Akelonia", "akelonia capa")).toBe(true);
  });

  it("exige todos os termos", () => {
    expect(foldMatch("Capa de Akelonia", "capa hekalotia")).toBe(false);
  });

  it("acha com e sem acento", () => {
    expect(foldMatch("Elmo Anão", "anao")).toBe(true);
    expect(foldMatch("Elmo Anao", "anão")).toBe(true);
  });

  it("consulta vazia casa tudo", () => {
    expect(foldMatch("qualquer", "")).toBe(true);
  });
});
