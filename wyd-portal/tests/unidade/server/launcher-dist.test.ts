import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

/**
 * O módulo lê `LAUNCHER_DIST_DIR` no import, então cada teste troca o env,
 * limpa o registro de módulos e importa de novo.
 */
const CONF =
  "# wyd-launcher-config v1\n" +
  "revisao = 1\n" +
  "portal.base = http://127.0.0.1:3000\n" +
  `assinatura = ${"0".repeat(64)}\n`;

let dir: string;

beforeEach(() => {
  dir = mkdtempSync(path.join(tmpdir(), "wyd-launcher-dist-"));
  vi.stubEnv("LAUNCHER_DIST_DIR", dir);
  vi.resetModules();
});

afterEach(() => {
  vi.unstubAllEnvs();
  rmSync(dir, { recursive: true, force: true });
});

describe("configuração publicada do launcher", () => {
  it("sem launcher.conf devolve nulo", async () => {
    const { lerConfiguracao } = await import("../../../src/server/launcher-dist");
    expect(lerConfiguracao()).toBeNull();
  });

  it("devolve o texto assinado byte a byte", async () => {
    writeFileSync(path.join(dir, "launcher.conf"), CONF, "utf8");
    const { lerConfiguracao } = await import("../../../src/server/launcher-dist");
    expect(lerConfiguracao()).toBe(CONF);
  });

  it("sem manifesto a publicação continua não publicada", async () => {
    writeFileSync(path.join(dir, "launcher.conf"), CONF, "utf8");
    const { getLauncherPublicacao } = await import("../../../src/server/launcher-dist");
    expect(getLauncherPublicacao().publicada).toBe(false);
  });
});
