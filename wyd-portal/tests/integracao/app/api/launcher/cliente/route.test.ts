import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { GET as manifesto } from "@/app/api/launcher/cliente/manifesto/route";
import { GET as arquivo } from "@/app/api/launcher/cliente/arquivo/[...caminho]/route";
import { caminhoSeguro } from "@/server/launcher-cliente";

/**
 * Rotas do cliente real: servem o manifesto e os bytes que o reparo do launcher
 * baixa, e recusam qualquer caminho que saia da raiz.
 */

let raiz: string;
const anterior = process.env.WYD_CLIENT_DIR;

beforeEach(() => {
  raiz = mkdtempSync(path.join(tmpdir(), "wyd-cliente-rota-"));
  mkdirSync(path.join(raiz, "Mesh"));
  writeFileSync(
    path.join(raiz, "manifest.txt"),
    "build 7662\nprotocol 12000\n7125f0 3 Mesh/a - Cópia.bin\n",
  );
  writeFileSync(path.join(raiz, "Mesh", "a - Cópia.bin"), Buffer.from([1, 2, 3]));
  process.env.WYD_CLIENT_DIR = raiz;
});

afterEach(() => {
  if (anterior === undefined) delete process.env.WYD_CLIENT_DIR;
  else process.env.WYD_CLIENT_DIR = anterior;
});

describe("GET /api/launcher/cliente/manifesto", () => {
  it("serve o manifesto sem cache", async () => {
    const res = manifesto();
    expect(res.status).toBe(200);
    expect(res.headers.get("cache-control")).toBe("no-store");
    expect(await res.text()).toContain("build 7662");
  });

  it("404 quando o cliente ainda não tem manifesto", async () => {
    process.env.WYD_CLIENT_DIR = mkdtempSync(path.join(tmpdir(), "wyd-cliente-vazio-"));
    const res = manifesto();
    expect(res.status).toBe(404);
  });
});

describe("GET /api/launcher/cliente/arquivo", () => {
  it("devolve os bytes exatos de um caminho com espaço e acento", async () => {
    const res = await arquivo(new Request("http://local/x"), {
      params: Promise.resolve({ caminho: ["Mesh", "a - Cópia.bin"] }),
    });
    expect(res.status).toBe(200);
    expect([...new Uint8Array(await res.arrayBuffer())]).toEqual([1, 2, 3]);
  });

  it("404 para arquivo ausente e para caminho inseguro", async () => {
    const ausente = await arquivo(new Request("http://local/x"), {
      params: Promise.resolve({ caminho: ["nao-existe.bin"] }),
    });
    expect(ausente.status).toBe(404);
    const fuga = await arquivo(new Request("http://local/x"), {
      params: Promise.resolve({ caminho: ["..", "manifest.txt"] }),
    });
    expect(fuga.status).toBe(404);
  });
});

describe("caminhoSeguro", () => {
  it("recusa absoluto, drive, barra invertida e ..", () => {
    expect(caminhoSeguro("C:/Windows/win.ini", raiz)).toBeNull();
    expect(caminhoSeguro("/etc/passwd", raiz)).toBeNull();
    expect(caminhoSeguro("Mesh\\a.bin", raiz)).toBeNull();
    expect(caminhoSeguro("../fora.bin", raiz)).toBeNull();
    expect(caminhoSeguro("Mesh/a.bin", raiz)).toBe(path.join(raiz, "Mesh", "a.bin"));
  });
});
