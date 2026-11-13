#!/usr/bin/env node
// Parser PE32/PE32+ mínimo, sem dependências — base da análise estática do cliente.
// Contrato: entrada = Buffer do arquivo; saída = objeto JSON determinístico.
// Nomenclatura: campos com o nome do formato PE (CamelCase), endereços em VA.
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const SCN = {
  code: 0x0000_0020,
  initializedData: 0x0000_0040,
  uninitializedData: 0x0000_0080,
  execute: 0x2000_0000,
  read: 0x4000_0000,
  write: 0x8000_0000,
};

const MACHINES = { 0x014c: 'i386', 0x8664: 'amd64', 0x01c0: 'arm', 0xaa64: 'arm64' };
const SUBSYSTEMS = { 2: 'windows-gui', 3: 'windows-cui', 9: 'windows-ce' };

const DIR_NAMES = [
  'export', 'import', 'resource', 'exception', 'certificate', 'baseRelocation',
  'debug', 'architecture', 'globalPointer', 'tls', 'loadConfig', 'boundImport',
  'importAddressTable', 'delayImport', 'clrRuntime', 'reserved',
];

function cString(buf, offset, max = 512) {
  if (offset < 0 || offset >= buf.length) return '';
  let end = offset;
  while (end < offset + max && end < buf.length && buf[end] !== 0) end += 1;
  return buf.toString('latin1', offset, end);
}

function sectionFlags(characteristics) {
  return {
    code: Boolean(characteristics & SCN.code),
    initializedData: Boolean(characteristics & SCN.initializedData),
    uninitializedData: Boolean(characteristics & SCN.uninitializedData),
    execute: Boolean(characteristics & SCN.execute),
    read: Boolean(characteristics & SCN.read),
    write: Boolean(characteristics & SCN.write),
  };
}

export function parsePe(buf) {
  if (buf.length < 0x40 || buf.readUInt16LE(0) !== 0x5a4d) throw new Error('não é MZ');
  const peOffset = buf.readUInt32LE(0x3c);
  if (buf.toString('latin1', peOffset, peOffset + 4) !== 'PE\0\0') throw new Error('não é PE');

  const coff = peOffset + 4;
  const machine = buf.readUInt16LE(coff);
  const numberOfSections = buf.readUInt16LE(coff + 2);
  const timeDateStamp = buf.readUInt32LE(coff + 4);
  const sizeOfOptionalHeader = buf.readUInt16LE(coff + 16);
  const characteristics = buf.readUInt16LE(coff + 18);
  const optional = coff + 20;
  const magic = buf.readUInt16LE(optional);
  const pe32plus = magic === 0x20b;
  if (magic !== 0x10b && !pe32plus) throw new Error(`optional header desconhecido: 0x${magic.toString(16)}`);

  const imageBase = pe32plus
    ? Number(buf.readBigUInt64LE(optional + 24))
    : buf.readUInt32LE(optional + 28);
  const dirsBase = optional + (pe32plus ? 112 : 96);
  const numberOfRvaAndSizes = buf.readUInt32LE(optional + (pe32plus ? 108 : 92));

  const directories = [];
  for (let i = 0; i < Math.min(numberOfRvaAndSizes, 16); i += 1) {
    directories.push({
      index: i,
      name: DIR_NAMES[i] || `dir${i}`,
      rva: buf.readUInt32LE(dirsBase + i * 8),
      size: buf.readUInt32LE(dirsBase + i * 8 + 4),
    });
  }

  const sections = [];
  const sectionBase = optional + sizeOfOptionalHeader;
  for (let i = 0; i < numberOfSections; i += 1) {
    const o = sectionBase + i * 40;
    const flags = buf.readUInt32LE(o + 36);
    sections.push({
      name: buf.toString('latin1', o, o + 8).replace(/\0+$/, ''),
      virtualSize: buf.readUInt32LE(o + 8),
      virtualAddress: buf.readUInt32LE(o + 12),
      sizeOfRawData: buf.readUInt32LE(o + 16),
      pointerToRawData: buf.readUInt32LE(o + 20),
      characteristics: flags,
      flags: sectionFlags(flags),
    });
  }

  return {
    kind: pe32plus ? 'PE32+' : 'PE32',
    machine,
    machineName: MACHINES[machine] || `0x${machine.toString(16)}`,
    numberOfSections,
    timeDateStamp,
    timeDateStampIso: new Date(timeDateStamp * 1000).toISOString(),
    characteristics,
    entryPoint: buf.readUInt32LE(optional + 16),
    imageBase,
    imageBaseHex: `0x${imageBase.toString(16)}`,
    sectionAlignment: buf.readUInt32LE(optional + 32),
    fileAlignment: buf.readUInt32LE(optional + 36),
    sizeOfImage: buf.readUInt32LE(optional + 56),
    sizeOfHeaders: buf.readUInt32LE(optional + 60),
    checksum: buf.readUInt32LE(optional + 64),
    subsystem: buf.readUInt16LE(optional + 68),
    subsystemName: SUBSYSTEMS[buf.readUInt16LE(optional + 68)] || 'desconhecido',
    dllCharacteristics: buf.readUInt16LE(optional + 70),
    directories,
    sections,
  };
}

/** Converte RVA em offset de arquivo, ou `null` se a RVA não tem bytes no arquivo. */
export function rvaToOffset(pe, rva) {
  for (const section of pe.sections) {
    const span = Math.max(section.virtualSize, section.sizeOfRawData);
    if (rva >= section.virtualAddress && rva < section.virtualAddress + span) {
      const delta = rva - section.virtualAddress;
      return delta < section.sizeOfRawData ? section.pointerToRawData + delta : null;
    }
  }
  return rva < pe.sizeOfHeaders ? rva : null;
}

/** VA (image base + RVA) → offset. */
export function vaToOffset(pe, va) {
  return rvaToOffset(pe, va - pe.imageBase);
}

/** Offset → VA, atravessando apenas bytes presentes no arquivo. */
export function offsetToVa(pe, offset) {
  for (const section of pe.sections) {
    if (offset >= section.pointerToRawData && offset < section.pointerToRawData + section.sizeOfRawData) {
      return pe.imageBase + section.virtualAddress + (offset - section.pointerToRawData);
    }
  }
  return offset < pe.sizeOfHeaders ? pe.imageBase + offset : null;
}

/** Seção que contém o offset, ou `null`. */
export function sectionOfOffset(pe, offset) {
  return (
    pe.sections.find(
      (s) => offset >= s.pointerToRawData && offset < s.pointerToRawData + s.sizeOfRawData,
    ) || null
  );
}

export function readImports(buf, pe) {
  const dir = pe.directories.find((d) => d.name === 'import');
  if (!dir || !dir.rva) return [];
  const base = rvaToOffset(pe, dir.rva);
  if (base === null) return [];

  const dlls = [];
  for (let i = 0; i < 256; i += 1) {
    const o = base + i * 20;
    if (o + 20 > buf.length) break;
    const originalFirstThunk = buf.readUInt32LE(o);
    const nameRva = buf.readUInt32LE(o + 12);
    const firstThunk = buf.readUInt32LE(o + 16);
    if (!nameRva && !firstThunk) break;

    const nameOffset = rvaToOffset(pe, nameRva);
    const dll = { name: nameOffset === null ? '' : cString(buf, nameOffset, 128), functions: [] };
    const thunkRva = originalFirstThunk || firstThunk;
    const thunkBase = rvaToOffset(pe, thunkRva);
    if (thunkBase !== null) {
      for (let t = 0; t < 4096; t += 1) {
        const at = thunkBase + t * 4;
        if (at + 4 > buf.length) break;
        const value = buf.readUInt32LE(at);
        if (value === 0) break;
        if (value & 0x8000_0000) {
          dll.functions.push({ ordinal: value & 0xffff });
        } else {
          const nameAt = rvaToOffset(pe, value & 0x7fff_ffff);
          dll.functions.push({
            name: nameAt === null ? '' : cString(buf, nameAt + 2, 128),
          });
        }
      }
    }
    dlls.push(dll);
  }
  return dlls;
}

export function readExports(buf, pe) {
  const dir = pe.directories.find((d) => d.name === 'export');
  if (!dir || !dir.rva) return { dllName: '', entries: [] };
  const base = rvaToOffset(pe, dir.rva);
  if (base === null) return { dllName: '', entries: [] };

  const nameRva = buf.readUInt32LE(base + 12);
  const numberOfFunctions = buf.readUInt32LE(base + 20);
  const numberOfNames = buf.readUInt32LE(base + 24);
  const functionsRva = buf.readUInt32LE(base + 28);
  const namesRva = buf.readUInt32LE(base + 32);
  const ordinalsRva = buf.readUInt32LE(base + 36);

  const functionsAt = rvaToOffset(pe, functionsRva);
  const namesAt = rvaToOffset(pe, namesRva);
  const ordinalsAt = rvaToOffset(pe, ordinalsRva);
  const entries = [];
  if (functionsAt !== null && namesAt !== null && ordinalsAt !== null) {
    for (let i = 0; i < Math.min(numberOfNames, 4096); i += 1) {
      const entryNameRva = buf.readUInt32LE(namesAt + i * 4);
      const ordinal = buf.readUInt16LE(ordinalsAt + i * 2);
      const nameOffset = rvaToOffset(pe, entryNameRva);
      const target = ordinal < numberOfFunctions ? buf.readUInt32LE(functionsAt + ordinal * 4) : 0;
      entries.push({
        name: nameOffset === null ? '' : cString(buf, nameOffset, 256),
        ordinal,
        rva: target,
        va: target ? pe.imageBase + target : null,
      });
    }
  }
  return {
    dllName: nameRva && rvaToOffset(pe, nameRva) !== null ? cString(buf, rvaToOffset(pe, nameRva), 128) : '',
    entries,
  };
}

export function readRichHeader(buf, pe) {
  const text = pe.sections.find((s) => s.flags.code);
  const limit = text ? Math.min(text.pointerToRawData, buf.length) : Math.min(pe.sizeOfHeaders, buf.length);
  const signature = buf.indexOf('Rich', 0x40);
  if (signature < 0 || signature > limit) return null;
  const key = buf.readUInt32LE(signature + 4);
  const entries = [];
  for (let o = signature - 8; o > 0x40; o -= 8) {
    const product = buf.readUInt32LE(o);
    if (product === 0x536e_6144) break; // "DanS"
    entries.unshift({ productId: (product >>> 16) & 0xffff, build: product & 0xffff, count: buf.readUInt32LE(o + 4) ^ key });
  }
  return { key: `0x${key.toString(16)}`, entries };
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  const file = process.argv[2] || 'client/WYD.exe';
  const report = parsePe(fs.readFileSync(file));
  report.imports = readImports(fs.readFileSync(file), report);
  report.exports = readExports(fs.readFileSync(file), report);
  console.log(JSON.stringify(report, null, 2));
}
