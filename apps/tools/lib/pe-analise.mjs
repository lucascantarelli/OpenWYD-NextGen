// Análise PE usada pelos comandos `binarios`/`strings` do `wyd-tools`.
//
// Tudo aqui é leitura de bytes com o parser único (`tools/client/pe.mjs`):
// seções, entrypoint, imports/exports, PDB e strings. "Candidato a função" é
// **inferência** declarada, nunca nome de função: alvo de `call`/`jmp rel32`
// dentro da seção de código, prólogo reconhecido e export nomeado.
import { parsePe, readExports, readImports, rvaToOffset } from "../../../tools/client/pe.mjs";
import { indexStrings } from "../../../tools/client/strings.mjs";

const hex = (n) => (n === null || n === undefined ? "—" : `0x${n.toString(16)}`);

/** Caminho do PDB no diretório de debug (CodeView/RSDS), quando presente. */
export function pdbPath(buf, pe) {
  const dir = (pe.directories ?? []).find((d) => d.name === "debug");
  if (!dir || !dir.rva) return null;
  const base = rvaToOffset(pe, dir.rva);
  if (base === null) return null;
  for (let i = 0; i < 16; i += 1) {
    const at = base + i * 28;
    if (at + 28 > buf.length) break;
    const tipo = buf.readUInt32LE(at + 12);
    const tamanho = buf.readUInt32LE(at + 16);
    const endereco = buf.readUInt32LE(at + 20);
    if (tipo !== 2) continue; // 2 = CODEVIEW
    const dados = rvaToOffset(pe, endereco);
    if (dados === null || dados + tamanho > buf.length || tamanho < 25) continue;
    if (buf.toString("latin1", dados, dados + 4) !== "RSDS") continue;
    const fim = buf.indexOf(0, dados + 24);
    return buf.toString("latin1", dados + 24, fim < 0 ? dados + tamanho : fim);
  }
  return null;
}

/**
 * Candidatos a função na seção de código.
 *
 * Heurísticas (todas determinísticas):
 * 1. `E8 rel32` (call) / `E9 rel32` (jmp) cujo destino cai na própria seção —
 *    conta chamadas por destino;
 * 2. prólogos clássicos x86: `55 8B EC`, `55 89 E5`, `8B FF 55 8B EC`;
 * 3. exports nomeados (rótulo real, não inferência).
 */
export function candidatosFuncao(buf, pe) {
  const texto =
    (pe.sections ?? []).find((s) => s.flags?.code) ??
    (pe.sections ?? []).find((s) => s.name === ".text");
  if (!texto) return { alvos: [], totalAlvos: 0, prologos: 0, secao: null };
  const inicio = texto.pointerToRawData;
  const fim = Math.min(inicio + texto.sizeOfRawData, buf.length);
  const vaBase = texto.virtualAddress;
  const vaFim = vaBase + Math.max(texto.virtualSize, texto.sizeOfRawData);
  const contagem = new Map();
  for (let i = inicio; i + 5 <= fim; i += 1) {
    const op = buf[i];
    if (op !== 0xe8 && op !== 0xe9) continue;
    const destino = vaBase + (i - inicio) + 5 + buf.readInt32LE(i + 1);
    if (destino < vaBase || destino >= vaFim) continue;
    contagem.set(destino, (contagem.get(destino) ?? 0) + 1);
  }
  let prologos = 0;
  for (let i = inicio; i + 5 <= fim; i += 1) {
    if (buf[i] !== 0x55) continue;
    if ((buf[i + 1] === 0x8b && buf[i + 2] === 0xec) || (buf[i + 1] === 0x89 && buf[i + 2] === 0xe5)) {
      prologos += 1;
    } else if (buf[i + 1] === 0x8b && buf[i + 2] === 0xff && buf[i + 3] === 0x55 && buf[i + 4] === 0x8b && buf[i + 5] === 0xec) {
      prologos += 1;
    }
  }
  const alvos = [...contagem.entries()]
    .map(([va, chamadas]) => ({ va, chamadas }))
    .sort((a, b) => b.chamadas - a.chamadas || a.va - b.va);
  return { alvos, totalAlvos: contagem.size, prologos, secao: texto.name };
}

/**
 * Variáveis globais candidatas: referências absolutas (disp32) do código para
 * seções de dados.
 *
 * Heurística medida (sem desmontagem): `A1/A3 imm32` e `8B/89/FF modrm=00 r/m=101`
 * com deslocamento absoluto. O destino é inferência de que existe um global ali;
 * o nome não é recuperado (só PDB/RTTI traria nome).
 */
export function candidatosVariaveis(buf, pe) {
  const texto =
    (pe.sections ?? []).find((s) => s.flags?.code) ??
    (pe.sections ?? []).find((s) => s.name === ".text");
  if (!texto) return { total: 0, top: [] };
  const dados = (pe.sections ?? []).filter(
    (s) => !(s.flags?.code) && s.virtualSize > 0,
  );
  const inicio = texto.pointerToRawData;
  const fim = Math.min(inicio + texto.sizeOfRawData, buf.length);
  const base = texto.virtualAddress;
  const mapa = new Map();
  const considerar = (va, i) => {
    const secao = dados.find(
      (s) => va >= s.virtualAddress && va < s.virtualAddress + s.virtualSize,
    );
    if (!secao) return;
    const chave = `${va}`;
    const atual = mapa.get(chave) ?? { va, secao: secao.name, refs: 0, primeiroOffset: i };
    atual.refs += 1;
    mapa.set(chave, atual);
  };
  for (let i = inicio; i + 6 <= fim; i += 1) {
    const op = buf[i];
    if (op === 0xa1 || op === 0xa3) {
      considerar(buf.readUInt32LE(i + 1), i);
      continue;
    }
    if (op !== 0x8b && op !== 0x89 && op !== 0xff) continue;
    const modrm = buf[i + 1];
    if ((modrm & 0xc7) !== 0x05) continue;
    considerar(buf.readUInt32LE(i + 2), i);
  }
  const top = [...mapa.values()]
    .sort((a, b) => b.refs - a.refs || a.va - b.va)
    .map((v) => ({ va: v.va, secao: v.secao, refs: v.refs }));
  return { total: top.length, top };
}

/** Resumo completo de um PE para a documentação gerada. */
export function analisarBinario(buf, caminho) {
  const pe = parsePe(buf);
  const imports = readImports(buf, pe);
  const exports = readExports(buf, pe);
  const strings = indexStrings(buf, pe);
  const funcoes = candidatosFuncao(buf, pe);
  const variaveis = candidatosVariaveis(buf, pe);
  return {
    caminho,
    arquitetura: `${pe.machineName} (${hex(pe.machine)})`,
    subsystem: `${pe.subsystemName} (${pe.subsystem})`,
    entryPoint: { rva: pe.entryPoint, rvaHex: hex(pe.entryPoint), vaHex: hex(pe.imageBase + pe.entryPoint) },
    imageBaseHex: hex(pe.imageBase),
    timestampIso: pe.timeDateStampIso ?? null,
    pdb: pdbPath(buf, pe),
    secoes: pe.sections.map((s) => ({
      nome: s.name,
      vaHex: hex(s.virtualAddress),
      virtualSize: s.virtualSize,
      rawSize: s.sizeOfRawData,
    })),
    imports: imports.map((dll) => ({
      dll: dll.name,
      total: dll.functions.length,
      funcoes: dll.functions.map((f) => f.name ?? `#${f.ordinal}`),
    })),
    exports: { dllName: exports.dllName, entradas: exports.entries },
    strings: {
      totais: strings.totals,
      relevant: strings.relevant,
    },
    variaveis,
    funcoes: {
      secao: funcoes.secao,
      prologos: funcoes.prologos,
      totalAlvos: funcoes.totalAlvos,
      alvos: funcoes.alvos.slice(0, 200),
    },
  };
}

