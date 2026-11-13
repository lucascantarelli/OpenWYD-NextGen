// Protocolo medido: catálogo do `wyd-protocol`, handlers no `wyd-server`,
// estruturas de campo nos payloads e evidência de capturas em `tmp/`.
//
// Nada é inventado: cada linha da tabela vem de um `match` Rust, de uma
// referência `Opcode::X` no servidor, de um `struct` de payload ou de um arquivo
// de captura lido. Captura não decodificada é marcada como tal (motivo incluso).
import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";

const ANDAR = (dir, filtro, saida = []) => {
  for (const entrada of readdirSync(dir, { withFileTypes: true })) {
    const completo = path.join(dir, entrada.name);
    if (entrada.isDirectory()) ANDAR(completo, filtro, saida);
    else if (filtro(completo)) saida.push(completo);
  }
  return saida;
};

/** `Opcode::Nome => 0x0101` do catálogo gerado + direção pelos bits FLAG. */
export function lerCatalogo(raiz) {
  const fonte = path.join(raiz, "crates/wyd-protocol/src/wire/generated/opcode_table.rs");
  const texto = readFileSync(fonte, "utf8");
  const opcodes = [];
  const re = /Opcode::(\w+)\s*=>\s*0x([0-9a-fA-F]+)/g;
  for (const m of texto.matchAll(re)) {
    const valor = Number.parseInt(m[2], 16);
    const direcao =
      valor & 0x0100 ? "S→C" : valor & 0x0200 ? "C→S" : valor & 0x0400 ? "DB→S" : valor & 0x0800 ? "S→DB" : "outra";
    opcodes.push({ nome: m[1], wire: `0x${m[2].toLowerCase().padStart(4, "0")}`, valor, direcao });
  }
  return opcodes.sort((a, b) => a.valor - b.valor || a.nome.localeCompare(b.nome));
}

/** Arquivos do `wyd-server` que mencionam `Opcode::Nome` (evidência de handler). */
export function lerHandlers(raiz) {
  const dir = path.join(raiz, "crates/wyd-server/src");
  const mapa = new Map();
  for (const arquivo of ANDAR(dir, (f) => f.endsWith(".rs"))) {
    const texto = readFileSync(arquivo, "utf8");
    for (const m of texto.matchAll(/Opcode::(\w+)/g)) {
      const rel = path.relative(raiz, arquivo).replaceAll("\\", "/");
      if (!mapa.has(m[1])) mapa.set(m[1], new Set());
      mapa.get(m[1]).add(rel);
    }
  }
  return mapa;
}

/** `struct` de payload → opcode → campos públicos (estrutura conhecida). */
export function lerEstruturas(raiz) {
  const dir = path.join(raiz, "crates/wyd-protocol/src/payload");
  const mapa = new Map();
  for (const arquivo of ANDAR(dir, (f) => f.endsWith(".rs"))) {
    const texto = readFileSync(arquivo, "utf8");
    const blocos = texto.split(/pub struct /).slice(1);
    for (const bloco of blocos) {
      const nome = /^(\w+)/.exec(bloco)?.[1];
      const corpo = /^[^{]*\{([\s\S]*?)\n\}/.exec(bloco)?.[1] ?? "";
      const campos = [...corpo.matchAll(/pub\s+(\w+)\s*:\s*([^,]+),/g)].map((m) => `${m[1]}: ${m[2].trim()}`);
      const opcode = /const OPCODE:\s*Opcode\s*=\s*Opcode::(\w+)/.exec(bloco)?.[1];
      const wireLen = /const WIRE_LEN:\s*usize\s*=\s*(\d+)/.exec(bloco)?.[1];
      if (opcode) mapa.set(opcode, { struct: nome, campos, wireLen: wireLen ? Number(wireLen) : null });
    }
  }
  return mapa;
}

/** Capturas em `tmp/`: arquivos, frames e tokens de Tipo já textuais. */
export function lerCapturas(raiz, dir = "tmp") {
  const base = path.join(raiz, dir);
  const arquivos = [];
  try {
    // Só capturas: ignora JSON de build em tmp/hashgen/target e afins.
    const filtro = (f) => {
      const rel = path.relative(base, f).replaceAll("\\", "/");
      if (rel.split("/").includes("target")) return false;
      return (rel.includes("captura") || /(?:^|\/)diff[^/]*\.(?:json|txt)$/.test(rel)) && (f.endsWith(".json") || f.endsWith(".txt"));
    };
    for (const arquivo of ANDAR(base, filtro)) {
      const texto = readFileSync(arquivo, "utf8");
      let frames = 0;
      let opcodes = [];
      try {
        const json = JSON.parse(texto);
        if (Array.isArray(json.frames)) frames = json.frames.length;
        // `diff-*.json` já traz o opcode decodificado do par (medido pela captura).
        opcodes = (json.com_diferenca ?? []).map((d) => ({
          opcode: String(d.opcode).toLowerCase(),
          direcao: d.direcao ?? null,
          size: d.size ?? null,
        }));
      } catch {
        frames = (texto.match(/\b(S2C|C2G|G2C)\s+0x[0-9a-fA-F]+/g) ?? []).length;
      }
      const tipos = [...new Set((texto.match(/\b(?:S2C|C2G|G2C)\s+0x[0-9a-fA-F]+/g) ?? []).map((t) => t.toUpperCase()))];
      arquivos.push({
        arquivo: path.relative(raiz, arquivo).replaceAll("\\", "/"),
        bytes: statSync(arquivo).size,
        frames,
        opcodes,
        tiposTextuais: tipos.slice(0, 20),
      });
    }
  } catch {
    // sem tmp/: a doc registra "nenhuma captura" em vez de inventar.
  }
  return arquivos.sort((a, b) => a.arquivo.localeCompare(b.arquivo));
}

/** Junta catálogo + handlers + estruturas + capturas. */
export function analisarPacotes(raiz) {
  const catalogo = lerCatalogo(raiz);
  const handlers = lerHandlers(raiz);
  const estruturas = lerEstruturas(raiz);
  const capturas = lerCapturas(raiz);
  const capturaPorOpcode = new Map();
  for (const arquivo of capturas) {
    for (const visto of arquivo.opcodes) {
      if (!capturaPorOpcode.has(visto.opcode)) capturaPorOpcode.set(visto.opcode, new Set());
      capturaPorOpcode.get(visto.opcode).add(arquivo.arquivo);
    }
  }
  const linhas = catalogo.map((op) => ({
    ...op,
    handlers: [...(handlers.get(op.nome) ?? [])].sort(),
    estrutura: estruturas.get(op.nome) ?? null,
    capturas: [...(capturaPorOpcode.get(op.wire) ?? [])].sort(),
  }));
  return {
    catalogo,
    linhas,
    capturas,
    totalComHandler: linhas.filter((l) => l.handlers.length > 0).length,
    totalSemHandler: linhas.filter((l) => l.handlers.length === 0).length,
    totalComEstrutura: linhas.filter((l) => l.estrutura).length,
    totalComCaptura: linhas.filter((l) => l.capturas.length > 0).length,
  };
}
