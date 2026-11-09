#!/usr/bin/env node
// Decodificador dos pacotes de UI do cliente 7662 (`.wyt`, magic `WT10`).
//
// O formato é **raster puro**, não um contêiner de controles:
//
//   offset  tamanho  campo
//   0       4        magic "WT10"
//   4       4        versão (0x00020000 em todas as 259 amostras de `client/NUI` + `client/UI`)
//   8       4        zero nas 259 amostras (reservado)
//   12      4        zero nas 259 amostras (reservado)
//   16      2        largura em pixels
//   18      2        altura em pixels
//   20      1        bits por pixel (24 ou 32)
//   21      1        0x08 quando 32 bpp, 0x00 quando 24 bpp
//   22      w*h*bpp  pixels; 24 bpp = BGR, 32 bpp = BGRA
//   fim-8   8        marcador final, `00`×8 nas 259 amostras
//
// Prova da fórmula: `22 + w*h*(bits/8) + 8 == tamanho do arquivo` em **259/259** `.wyt`
// de `client/NUI` e `client/UI` (`--scan .`). Antes desta passada o doc dizia que
// "controles, fontes e sprites seguem opacos"; o que era opaco era o *desenho*, não os
// bytes — cada arquivo é uma única imagem sem compressão.
//
// Uso:
//   node tools/client/wyt.mjs client/NUI/ServerList.wyt --png tmp/serverlist.png
//   node tools/client/wyt.mjs client/NUI --json docs/cliente/gerado/wyt.json
//   node tools/client/wyt.mjs client --scan .            # valida a fórmula em todos
import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';

const MAGIC = 'WT10';
const HEADER = 22;
const TRAILER = 8;
const BITS_VALIDOS = new Set([24, 32]);

/** Erro de formato: nunca "melhor esforço" — bytes que não fecham a conta param aqui. */
function erro(mensagem) {
  const e = new Error(mensagem);
  e.code = 'WYT_INVALIDO';
  return e;
}

export function parseWyt(buf, arquivo = '<memória>') {
  if (buf.length < HEADER + TRAILER) throw erro(`${arquivo}: menor que o cabeçalho (${buf.length} B)`);
  const magic = buf.toString('latin1', 0, 4);
  if (magic !== MAGIC) throw erro(`${arquivo}: magic "${magic}" (esperado "${MAGIC}")`);
  const largura = buf.readUInt16LE(16);
  const altura = buf.readUInt16LE(18);
  const bits = buf[20];
  const marcador21 = buf[21];
  if (!BITS_VALIDOS.has(bits)) throw erro(`${arquivo}: ${bits} bits por pixel (esperado 24 ou 32)`);
  if (largura === 0 || altura === 0) throw erro(`${arquivo}: dimensão zero (${largura}x${altura})`);
  const bpp = bits / 8;
  const esperado = HEADER + largura * altura * bpp + TRAILER;
  if (esperado !== buf.length) {
    throw erro(`${arquivo}: tamanho ${buf.length} B não fecha ${largura}x${altura}x${bits}bpp (esperado ${esperado} B)`);
  }
  const fim = buf.length - TRAILER;
  return {
    arquivo,
    versao: buf.readUInt32LE(4),
    reservado8: buf.readUInt32LE(8),
    reservado12: buf.readUInt32LE(12),
    largura,
    altura,
    bits,
    bpp,
    marcador21,
    pixelsOffset: HEADER,
    pixelsBytes: largura * altura * bpp,
    trailer: buf.subarray(fim).toString('hex'),
    pixels: buf.subarray(HEADER, fim),
  };
}

/** Pixels BGRA (32 bpp) ou BGR (24 bpp) na ordem em que estão no arquivo. */
export function pixelBitmap(wyt, x, y) {
  const base = (y * wyt.largura + x) * wyt.bpp;
  const p = wyt.pixels;
  return { b: p[base], g: p[base + 1], r: p[base + 2], a: wyt.bpp === 4 ? p[base + 3] : 0xff };
}

/** Estatísticas que dizem se o raster é imagem de verdade (e não ruído/cripto). */
export function analisarRaster(wyt) {
  const cores = new Map();
  let alfaOpaco = 0;
  let alfaZero = 0;
  const total = wyt.largura * wyt.altura;
  for (let i = 0; i < total; i += 1) {
    const base = i * wyt.bpp;
    const chave = (wyt.pixels[base] << 16) | (wyt.pixels[base + 1] << 8) | wyt.pixels[base + 2];
    cores.set(chave, (cores.get(chave) ?? 0) + 1);
    if (wyt.bpp === 4) {
      if (wyt.pixels[base + 3] === 0xff) alfaOpaco += 1;
      else if (wyt.pixels[base + 3] === 0x00) alfaZero += 1;
    }
  }
  const top = [...cores.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 8)
    .map(([rgb, n]) => ({ cor: `#${rgb.toString(16).padStart(6, '0')}`, pixels: n, fracao: Number((n / total).toFixed(4)) }));
  return {
    pixels: total,
    coresDistintas: cores.size,
    alfaOpaco: wyt.bpp === 4 ? Number((alfaOpaco / total).toFixed(4)) : 1,
    alfaZero: wyt.bpp === 4 ? Number((alfaZero / total).toFixed(4)) : 0,
    topCores: top,
  };
}

function crc32(buf) {
  let c = ~0;
  for (const byte of buf) {
    c ^= byte;
    for (let k = 0; k < 8; k += 1) c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
  }
  return (~c) >>> 0;
}

function chunkPng(tipo, dados) {
  const tam = Buffer.alloc(4);
  tam.writeUInt32BE(dados.length);
  const corpo = Buffer.concat([Buffer.from(tipo, 'latin1'), dados]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(corpo));
  return Buffer.concat([tam, corpo, crc]);
}

/** PNG RGBA8 sem filtro — o raster do cliente é BGRA, então só trocamos os canais. */
export function toPng(wyt) {
  const { largura, altura, bpp, pixels } = wyt;
  const linhas = Buffer.alloc((largura * 4 + 1) * altura);
  for (let y = 0; y < altura; y += 1) {
    const destino = y * (largura * 4 + 1);
    linhas[destino] = 0;
    for (let x = 0; x < largura; x += 1) {
      const o = (y * largura + x) * bpp;
      const d = destino + 1 + x * 4;
      linhas[d] = pixels[o + 2];
      linhas[d + 1] = pixels[o + 1];
      linhas[d + 2] = pixels[o];
      linhas[d + 3] = bpp === 4 ? pixels[o + 3] : 0xff;
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(largura, 0);
  ihdr.writeUInt32BE(altura, 4);
  ihdr[8] = 8;
  ihdr[9] = 6;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunkPng('IHDR', ihdr),
    chunkPng('IDAT', zlib.deflateSync(linhas, { level: 9 })),
    chunkPng('IEND', Buffer.alloc(0)),
  ]);
}

/** BMP 24 bpp bottom-up (o formato que o `Launcher.bmp`/Windows abrem sem codec extra). */
export function toBmp(wyt) {
  const { largura, altura, bpp, pixels } = wyt;
  const linhaBytes = Math.ceil((largura * 3) / 4) * 4;
  const tamanhoPixels = linhaBytes * altura;
  const out = Buffer.alloc(54 + tamanhoPixels);
  out.write('BM', 0, 'latin1');
  out.writeUInt32LE(54 + tamanhoPixels, 2);
  out.writeUInt32LE(54, 10);
  out.writeUInt32LE(40, 14);
  out.writeInt32LE(largura, 18);
  out.writeInt32LE(altura, 22);
  out.writeUInt16LE(1, 26);
  out.writeUInt16LE(24, 28);
  out.writeUInt32LE(tamanhoPixels, 34);
  for (let y = 0; y < altura; y += 1) {
    const destino = 54 + (altura - 1 - y) * linhaBytes;
    for (let x = 0; x < largura; x += 1) {
      const o = (y * largura + x) * bpp;
      out[destino + x * 3] = pixels[o];
      out[destino + x * 3 + 1] = pixels[o + 1];
      out[destino + x * 3 + 2] = pixels[o + 2];
    }
  }
  return out;
}

export function listarWyt(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir)
    .filter((nome) => nome.toLowerCase().endsWith('.wyt'))
    .sort()
    .map((nome) => path.join(dir, nome));
}

function descrever(arquivo) {
  const buf = fs.readFileSync(arquivo);
  const wyt = parseWyt(buf, arquivo.split(path.sep).join('/'));
  const { pixels, ...semPixels } = wyt;
  return { ...semPixels, tamanho: buf.length, raster: analisarRaster(wyt) };
}

function main(argv) {
  const args = argv.slice(2);
  if (args.length === 0) {
    console.error('uso: wyt.mjs <arquivo|dir> [--png saída.png] [--bmp saída.bmp] [--json saída.json] [--scan rótulo]');
    return 2;
  }
  const alvo = args[0];
  const flag = (nome) => {
    const i = args.indexOf(nome);
    return i >= 0 ? args[i + 1] : null;
  };

  if (flag('--scan')) {
    const rotulo = flag('--scan');
    const arquivos = fs.statSync(alvo).isDirectory() ? listarWyt(alvo) : [alvo];
    let ok = 0;
    const falhas = [];
    for (const arquivo of arquivos) {
      try {
        const wyt = parseWyt(fs.readFileSync(arquivo), arquivo.split(path.sep).join('/'));
        ok += 1;
        if (rotulo !== '.') console.log(`${wyt.arquivo}: ${wyt.largura}x${wyt.altura} ${wyt.bits}bpp`);
      } catch (e) {
        falhas.push(e.message);
      }
    }
    console.log(`scan ${rotulo}: ${ok}/${arquivos.length} fecham 22 + w*h*bpp + 8`);
    for (const f of falhas.slice(0, 10)) console.error(`  ${f}`);
    return falhas.length ? 1 : 0;
  }

  const arquivos = fs.statSync(alvo).isDirectory() ? listarWyt(alvo) : [alvo];
  const relatorio = arquivos.map(descrever);
  const png = flag('--png');
  const bmp = flag('--bmp');
  const json = flag('--json');
  if (png) {
    fs.mkdirSync(path.dirname(png), { recursive: true });
    fs.writeFileSync(png, toPng(parseWyt(fs.readFileSync(arquivos[0]), arquivos[0])));
    console.log(`png: ${png}`);
  }
  if (bmp) {
    fs.mkdirSync(path.dirname(bmp), { recursive: true });
    fs.writeFileSync(bmp, toBmp(parseWyt(fs.readFileSync(arquivos[0]), arquivos[0])));
    console.log(`bmp: ${bmp}`);
  }
  if (json) {
    fs.mkdirSync(path.dirname(json), { recursive: true });
    fs.writeFileSync(json, `${JSON.stringify({ formato: 'WT10', amostras: relatorio.length, telas: relatorio }, null, 2)}\n`);
    console.log(`json: ${json} (${relatorio.length} tela(s))`);
  }
  for (const tela of relatorio.slice(0, json ? 0 : 12)) {
    console.log(
      `${tela.arquivo}: ${tela.largura}x${tela.altura} ${tela.bits}bpp · ${tela.raster.coresDistintas} cores · alfa opaco ${tela.raster.alfaOpaco}`,
    );
  }
  return 0;
}

if (import.meta.url === `file://${process.argv[1]?.replace(/\\/g, '/')}` || process.argv[1]?.endsWith('wyt.mjs')) {
  process.exitCode = main(process.argv);
}
