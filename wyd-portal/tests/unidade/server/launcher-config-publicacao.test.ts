import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import {
  configuracaoAtual,
  historicoConfiguracao,
  publicarConfiguracao,
  revisaoDe,
} from "@/server/launcher-dist";

/**
 * Publicação da config assinada: o portal preserva bytes, exige assinatura na
 * última linha e revisão estritamente crescente, e registra o histórico.
 * Cada teste usa o seu próprio diretório temporário.
 */
const ASSINATURA = "a".repeat(64);

const corpo = (revisao: number, extra = ""): string =>
  `# wyd-launcher-config v1\nrevisao = ${revisao}\n` +
  `portal.base = http://127.0.0.1:3000\n${extra}assinatura = ${ASSINATURA}\n`;

let dir: string;

beforeEach(() => {
  dir = mkdtempSync(path.join(tmpdir(), "wyd-launcher-publicacao-"));
});

afterEach(() => {
  rmSync(dir, { recursive: true, force: true });
});

describe("revisão declarada", () => {
  it("lê a primeira linha `revisao = N`", () => {
    expect(revisaoDe(corpo(7))).toBe(7);
    expect(revisaoDe("# sem revisão\nportal.base = x\n")).toBeNull();
    expect(revisaoDe(corpo(0))).toBeNull();
  });
});

describe("publicarConfiguracao", () => {
  it("grava os bytes exatos e registra o histórico com autor", () => {
    const texto = corpo(3);
    const entrada = publicarConfiguracao(texto, "portal#42", dir);

    expect(entrada.revisao).toBe(3);
    expect(entrada.bytes).toBe(Buffer.byteLength(texto, "utf8"));
    expect(entrada.autor).toBe("portal#42");
    expect(readFileSync(path.join(dir, "launcher.conf"), "utf8")).toBe(texto);

    const historico = historicoConfiguracao(dir);
    expect(historico).toHaveLength(1);
    expect(historico[0]).toEqual(entrada);
  });

  it("mantém o histórico append-only e faz backup da revisão anterior", () => {
    publicarConfiguracao(corpo(1), "portal#1", dir);
    publicarConfiguracao(corpo(2), "portal#2", dir);

    expect(historicoConfiguracao(dir).map((item) => item.revisao)).toEqual([1, 2]);
    expect(existsSync(path.join(dir, "launcher.conf.bak"))).toBe(true);
    expect(readFileSync(path.join(dir, "launcher.conf"), "utf8")).toBe(corpo(2));

    const atual = configuracaoAtual(dir);
    expect(atual?.revisao).toBe(2);
    expect(atual?.autor).toBe("portal#2");
    expect(atual?.sha256).toHaveLength(64);
  });

  it("recusa revisão repetida ou menor que a publicada", () => {
    publicarConfiguracao(corpo(5), "portal#1", dir);
    expect(() => publicarConfiguracao(corpo(5), "portal#1", dir)).toThrow(/já foi publicada/);
    expect(() => publicarConfiguracao(corpo(4), "portal#1", dir)).toThrow(
      /não é maior que a publicada/,
    );
    expect(readFileSync(path.join(dir, "launcher.conf"), "utf8")).toBe(corpo(5));
    expect(historicoConfiguracao(dir)).toHaveLength(1);
  });

  it("recusa texto sem assinatura na última linha", () => {
    const semAssinatura = "# wyd-launcher-config v1\nrevisao = 2\n";
    expect(() => publicarConfiguracao(semAssinatura, "portal#1", dir)).toThrow(
      /última linha precisa ser/,
    );
    expect(existsSync(path.join(dir, "launcher.conf"))).toBe(false);
  });

  it("recusa assinatura fora de 64 hex, sem revisão e corpo vazio", () => {
    const curta = corpo(2).replace(ASSINATURA, "abc");
    expect(() => publicarConfiguracao(curta, "portal#1", dir)).toThrow(/64 hex/);

    expect(() => publicarConfiguracao(`assinatura = ${ASSINATURA}\n`, "portal#1", dir)).toThrow(
      /sem `revisao = N`/,
    );
    expect(() => publicarConfiguracao("", "portal#1", dir)).toThrow(/vazia/);
    expect(() => publicarConfiguracao(corpo(2).trimEnd(), "portal#1", dir)).toThrow(
      /terminar com/,
    );
  });

  it("só considera publicado o que está no diretório pedido", () => {
    publicarConfiguracao(corpo(1), "portal#1", dir);
    const outro = mkdtempSync(path.join(tmpdir(), "wyd-launcher-vazio-"));
    expect(configuracaoAtual(outro)).toBeNull();
    expect(historicoConfiguracao(outro)).toEqual([]);
    rmSync(outro, { recursive: true, force: true });
  });

  it("ignora linha corrompida no histórico sem esconder as válidas", () => {
    publicarConfiguracao(corpo(1), "portal#1", dir);
    writeFileSync(
      path.join(dir, "launcher.conf.historico.jsonl"),
      `${readFileSync(path.join(dir, "launcher.conf.historico.jsonl"), "utf8")}nao-e-json\n`,
      "utf8",
    );
    expect(historicoConfiguracao(dir)).toHaveLength(1);
  });
});
