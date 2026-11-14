// Núcleo do harness do cliente: separa frames CPSock, grava sessão e reproduz.
//
// O HEADER tem `Size` em u16 LE no offset 0 e ele NÃO é ofuscado (a transformação começa no
// offset 4). Por isso o proxy consegue delimitar frames sem conhecer a `pKeyWord` — é a mesma
// leitura que `cpsock::handshake` faz antes de `decode_in_place`.
import { createWriteStream, readFileSync } from "node:fs";
import net from "node:net";

export const HEADER_SIZE = 12;
export const MAX_SIZE = 8192;
export const INITCODE = Buffer.from([0x11, 0xf3, 0x11, 0x1f]); // 0x1F11F311 LE

/** Delimita frames de um stream CPSock. O lado cliente começa com INITCODE (4 B). */
export class FrameSplitter {
  constructor({ initcode = false } = {}) {
    this.buf = Buffer.alloc(0);
    this.initcode = initcode;
    this.initcodeVisto = false;
  }

  push(chunk) {
    this.buf = Buffer.concat([this.buf, chunk]);
    const saida = [];
    if (this.initcode && !this.initcodeVisto && this.buf.length >= 4) {
      const prefixo = this.buf.subarray(0, 4);
      if (!prefixo.equals(INITCODE)) {
        throw new Error(`INITCODE inesperado: ${prefixo.toString("hex")}`);
      }
      saida.push({ initcode: true });
      this.initcodeVisto = true;
      this.buf = this.buf.subarray(4);
    }
    while (this.buf.length >= 2) {
      const size = this.buf.readUInt16LE(0);
      if (size < HEADER_SIZE || size > MAX_SIZE) {
        throw new Error(`Size fora da faixa [12, 8192]: ${size}`);
      }
      if (this.buf.length < size) break;
      saida.push({ frame: this.buf.subarray(0, size) });
      this.buf = this.buf.subarray(size);
    }
    return saida;
  }
}

/** Proxy TCP que grava as duas direções em JSONL. */
export async function startProxy({ listenPort = 0, listenHost = "127.0.0.1", targetHost, targetPort, outPath }) {
  const registro = createWriteStream(outPath, { flags: "w" });
  const inicio = Date.now();
  const anotar = (dir, item) => {
    const hex = item.frame ? item.frame.toString("hex") : INITCODE.toString("hex");
    const linha = { t: Date.now() - inicio, dir, size: hex.length / 2, hex };
    if (!item.frame) linha.init = true;
    registro.write(`${JSON.stringify(linha)}\n`);
  };

  const servidor = net.createServer((cliente) => {
    const alvo = net.connect({ host: targetHost, port: targetPort });
    const c2s = new FrameSplitter({ initcode: true });
    const s2c = new FrameSplitter();

    cliente.on("data", (chunk) => {
      try {
        for (const item of c2s.push(chunk)) anotar("C", item);
      } catch (erro) {
        console.error(`[proxy] C→S: ${erro.message}`);
      }
      alvo.write(chunk);
    });
    alvo.on("data", (chunk) => {
      try {
        for (const item of s2c.push(chunk)) anotar("S", item);
      } catch (erro) {
        console.error(`[proxy] S→C: ${erro.message}`);
      }
      cliente.write(chunk);
    });

    const encerrar = () => {
      cliente.destroy();
      alvo.destroy();
    };
    cliente.on("error", encerrar);
    alvo.on("error", encerrar);
    cliente.on("close", () => {
      alvo.end();
      registro.write(`${JSON.stringify({ t: Date.now() - inicio, dir: "FIM" })}\n`);
    });
    alvo.on("close", () => cliente.end());
  });

  await new Promise((resolve, reject) => {
    servidor.once("error", reject);
    servidor.listen(listenPort, listenHost, resolve);
  });
  const { port } = servidor.address();
  return {
    port,
    async close() {
      await new Promise((resolve) => servidor.close(resolve));
      registro.end();
    },
  };
}

/** Lê a sessão gravada em frames ordenados. */
export function readSession(caminho) {
  const linhas = [];
  const bruto = readFileSync(caminho, "utf8");
  for (const linha of bruto.split("\n")) {
    if (!linha.trim()) continue;
    linhas.push(JSON.parse(linha));
  }
  return linhas.filter((linha) => linha.dir === "C" || linha.dir === "S");
}

/**
 * Reenvia os frames C→S gravados e compara a sequência S→C do servidor com a da sessão.
 *
 * Compara `size` (contrato de framing) e, com `exact`, os bytes. Bytes exatos raramente batem:
 * a resposta carrega checksum/keyword e ids variáveis; o padrão é comparar tamanhos + opcode
 * quando a sessão tiver sido decodificada.
 */
export async function replaySession({ frames, host, port, idleMs = 600, maxDelayMs = 200, exact = false }) {
  const clienteFrames = frames.filter((f) => f.dir === "C");
  const servidorEsperado = frames.filter((f) => f.dir === "S");
  const recebidos = [];

  const socket = net.connect({ host, port });
  await new Promise((resolve, reject) => {
    socket.once("connect", resolve);
    socket.once("error", reject);
  });

  const splitter = new FrameSplitter();
  socket.on("data", (chunk) => {
    for (const item of splitter.push(chunk)) recebidos.push(item.frame);
  });

  let anterior = clienteFrames[0]?.t ?? 0;
  for (const frame of clienteFrames) {
    const espera = Math.min(Math.max((frame.t ?? anterior) - anterior, 0), maxDelayMs);
    if (espera) await new Promise((r) => setTimeout(r, espera));
    anterior = frame.t ?? anterior;
    if (frame.initcode !== undefined) {
      socket.write(Buffer.from(frame.initcode, "hex"));
    } else {
      socket.write(Buffer.from(frame.hex, "hex"));
    }
  }

  await new Promise((resolve) => {
    let ultimo = Date.now();
    const timer = setInterval(() => {
      if (Date.now() - ultimo > idleMs) {
        clearInterval(timer);
        resolve();
      }
    }, 50);
    socket.on("data", () => {
      ultimo = Date.now();
    });
    socket.on("close", () => {
      clearInterval(timer);
      resolve();
    });
  });
  socket.destroy();

  const divergencias = [];
  const total = Math.max(servidorEsperado.length, recebidos.length);
  for (let i = 0; i < total; i += 1) {
    const esperado = servidorEsperado[i];
    const obtido = recebidos[i];
    if (!esperado) {
      divergencias.push(`#${i}: servidor enviou frame extra (size=${obtido.length})`);
      continue;
    }
    if (!obtido) {
      divergencias.push(`#${i}: servidor não enviou o frame esperado (size=${esperado.size})`);
      continue;
    }
    if (esperado.size !== obtido.length) {
      divergencias.push(`#${i}: size esperado ${esperado.size}, recebido ${obtido.length}`);
      continue;
    }
    if (exact && esperado.hex !== obtido.toString("hex")) {
      divergencias.push(`#${i}: bytes divergem (size=${obtido.length})`);
    }
  }
  return { enviados: clienteFrames.length, recebidos: recebidos.length, esperados: servidorEsperado.length, divergencias };
}
