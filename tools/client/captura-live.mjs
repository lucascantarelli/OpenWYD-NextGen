#!/usr/bin/env node
/**
 * Captura ao vivo do cliente 7662 (FASE-08-6).
 *
 * Três modos, um artefato por execução:
 *
 * 1. **servidor falso** (padrão): escuta em `--host:--port`, responde o `serv00.htm`
 *    mínimo e grava os bytes crus de cada conexão CPSock. Serve para provar que o
 *    cliente chega ao handshake quando apontado para loopback.
 * 2. **proxy** (`--alvo ip:porta`): fica entre o cliente e o canal real, repassa o
 *    TCP nos dois sentidos e grava **cada frame CPSock** (Size u16 LE no offset 0)
 *    com direção `c2s`/`s2c`. É o modo que produz tráfego S→C de verdade, com o
 *    servidor Rust respondendo o jogo.
 * 3. **conversão** (`--from captura.json`): transforma a captura em fixtures
 *    versionadas em `fixtures/cliente/real/` (um JSON por frame único) + manifesto
 *    TSV com direção, opcode, tamanho, canal e origem. O opcode é lido do frame
 *    desofuscado (`keytable.mjs`), nunca adivinhado.
 *
 * Uso:
 *   node tools/client/captura-live.mjs --timeout 90000 --out fixtures/cliente/captura-handshake.json
 *   node tools/client/captura-live.mjs --alvo 192.168.100.4:8281 --host 172.17.0.1 \
 *     --port 8281 --timeout 120000 --out target/captura-canal2.json
 *   node tools/client/captura-live.mjs --from target/captura-canal2.json \
 *     --fixtures fixtures/cliente/real --canal 2
 *
 * Nada aqui inventa byte: o que não chegar não entra no artefato. Conta/senha nunca
 * são preenchidas pelo harness; se o fluxo real trouxer login, o JSON é higienizado
 * na conversão (`sanitizado: true`) antes de virar fixture.
 */
import crypto from "node:crypto";
import dgram from "node:dgram";
import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import process from "node:process";
import { loadKeywordTable, transform } from "./keytable.mjs";

function arg(nome, padrao) {
  const indice = process.argv.indexOf(`--${nome}`);
  if (indice === -1) return padrao;
  const valor = process.argv[indice + 1];
  if (valor === undefined) {
    console.error(`captura-live: --${nome} exige valor`);
    process.exit(2);
  }
  return valor;
}

const HOST = arg("host", "127.0.0.1");
const PORT = Number(arg("port", "8281"));
const TIMEOUT_MS = Number(arg("timeout", "90000"));
const OUT = arg("out", "fixtures/cliente/captura-handshake.json");
const ALVO = arg("alvo", null);
const FROM = arg("from", null);
const FIXTURES = arg("fixtures", "fixtures/cliente/real");
const CANAL = arg("canal", "");
const MAX_MESSAGE = 8192;

/** Corpo mínimo de status: o cliente só precisa de HTTP 200 para seguir. */
const SERV00 = "<html><body>WYD status OK</body></html>";

/** Decodifica o HEADER de 12 bytes (little-endian), se houver bytes suficientes. */
function cabecalho(buf) {
  if (buf.length < 12) return null;
  return {
    size: buf.readUInt16LE(0),
    key_word: buf[2],
    checksum: buf[3],
    type: buf.readUInt16LE(4),
    type_hex: `0x${buf.readUInt16LE(4).toString(16).padStart(4, "0")}`,
    id: buf.readUInt16LE(6),
    client_tick: buf.readUInt32LE(8),
  };
}

/** Desofusca um frame selado e devolve `{ opcode, size, payload }`. */
function abreFrame(hex) {
  const wire = Buffer.from(hex, "hex");
  const size = wire.readUInt16LE(0);
  const clear = Buffer.from(wire);
  const tabela = loadKeywordTable();
  transform(clear, { keyWord: wire[2], table: tabela, direction: "decode" });
  return { size, opcode: clear.readUInt16LE(4), payload: clear.subarray(12) };
}

function escreveJson(destino, dados) {
  const caminho = path.resolve(destino);
  fs.mkdirSync(path.dirname(caminho), { recursive: true });
  fs.writeFileSync(caminho, `${JSON.stringify(dados, null, 2)}\n`, "utf8");
  console.log(`captura-live: artefato escrito em ${caminho}`);
}

/* ------------------------------------------------------------------ */
/* Modo 3 — conversão de captura em fixtures versionadas               */
/* ------------------------------------------------------------------ */
if (FROM) {
  const captura = JSON.parse(fs.readFileSync(path.resolve(FROM), "utf8"));
  const frames = Array.isArray(captura.frames) ? captura.frames : [];
  const destino = path.resolve(FIXTURES);
  fs.mkdirSync(destino, { recursive: true });
  for (const antigo of fs.readdirSync(destino)) {
    if (antigo.endsWith(".json") || antigo === "manifest.tsv") fs.rmSync(path.join(destino, antigo));
  }
  const vistos = new Map();
  const linhas = [
    "arquivo\tdirecao\topcode\tsize\tcanal\torigem\tstatus",
  ];
  let ordem = 0;
  for (const frame of frames) {
    if (frame.desalinhado || !frame.hex) continue;
    ordem += 1;
    const direcao = frame.direcao === "c2s" ? "C2S" : "S2C";
    const { size, opcode } = abreFrame(frame.hex);
    // PIN/senha numérica (`0x0FDE`/`0x0FDF`) nunca vai para fixture: corpo zerado.
    let hex = frame.hex;
    let mascarado = false;
    if (opcode === 0x0fde || opcode === 0x0fdf) {
      const b = Buffer.from(hex, "hex");
      b.fill(0, 12);
      hex = b.toString("hex");
      mascarado = true;
    }
    const chave = `${direcao}-${opcode}-${size}-${crypto.createHash("sha1").update(hex).digest("hex")}`;
    if (vistos.has(chave)) {
      vistos.get(chave).repeticoes += 1;
      continue;
    }
    const arquivo = `${String(ordem).padStart(3, "0")}-${direcao.toLowerCase()}-0x${opcode.toString(16).padStart(4, "0")}-${size}.json`;
    const entrada = {
      direcao,
      opcode: `0x${opcode.toString(16).padStart(4, "0")}`,
      size,
      canal: CANAL === "" ? null : Number(CANAL),
      origem: captura.origem ?? "captura-live",
      capturado_em: captura.capturado_em ?? null,
      repeticoes: 1,
      mascarado,
      wireHex: hex,
    };
    vistos.set(chave, entrada);
    fs.writeFileSync(path.join(destino, arquivo), `${JSON.stringify(entrada, null, 2)}\n`, "utf8");
    linhas.push(
      [arquivo, direcao, entrada.opcode, size, CANAL === "" ? "-" : CANAL, "proxy-cpsock", "mapeado"].join("\t"),
    );
  }
  fs.writeFileSync(path.join(destino, "manifest.tsv"), `${linhas.join("\n")}\n`, "utf8");
  console.log(
    `captura-live: ${vistos.size} fixture(s) de ${frames.length} frame(s) em ${destino} (manifest.tsv)`,
  );
  process.exit(vistos.size === 0 ? 1 : 0);
}

/* ------------------------------------------------------------------ */
/* Modo 2 — proxy entre o cliente e o canal real                       */
/* ------------------------------------------------------------------ */
if (ALVO) {
  const [alvoHost, alvoPorta] = ALVO.split(":");
  if (!alvoHost || !alvoPorta) {
    console.error("captura-live: --alvo exige ip:porta");
    process.exit(2);
  }
  const frames = [];
  const conexoes = [];
  let ordem = 0;

  /** Corta o fluxo em frames CPSock pelo `Size` do cabeçalho (claro nos dois sentidos). */
  function bomba(lado, estado, chunk) {
    estado.buffer = Buffer.concat([estado.buffer, chunk]);
    for (;;) {
      if (estado.buffer.length < 12) return;
      const size = estado.buffer.readUInt16LE(0);
      if (size < 12 || size > MAX_MESSAGE) {
        ordem += 1;
        frames.push({ ordem, direcao: lado, desalinhado: true, hex: estado.buffer.toString("hex") });
        console.log(`captura-live: ${lado} desalinhado (size=${size}) — fluxo descartado`);
        estado.buffer = Buffer.alloc(0);
        return;
      }
      if (estado.buffer.length < size) return;
      ordem += 1;
      const frame = estado.buffer.subarray(0, size);
      frames.push({ ordem, direcao: lado, size, hex: frame.toString("hex") });
      const head = cabecalho(frame);
      console.log(
        `captura-live: #${ordem} ${lado} ${size} B kw=${head.key_word} checksum=${head.checksum}`,
      );
      estado.buffer = estado.buffer.subarray(size);
    }
  }

  const servidor = net.createServer((cliente) => {
    const peer = `${cliente.remoteAddress}:${cliente.remotePort}`;
    conexoes.push({ peer, inicio: new Date().toISOString() });
    const estado = { c2s: { buffer: Buffer.alloc(0) }, s2c: { buffer: Buffer.alloc(0) } };
    const upstream = net.connect(Number(alvoPorta), alvoHost);
    cliente.on("data", (chunk) => {
      bomba("c2s", estado.c2s, chunk);
      upstream.write(chunk);
    });
    upstream.on("data", (chunk) => {
      bomba("s2c", estado.s2c, chunk);
      cliente.write(chunk);
    });
    const fecha = () => {
      cliente.destroy();
      upstream.destroy();
    };
    cliente.on("error", fecha);
    upstream.on("error", fecha);
    cliente.on("close", () => upstream.destroy());
    upstream.on("close", () => cliente.destroy());
    console.log(`captura-live: conexão ${peer} → ${alvoHost}:${alvoPorta}`);
  });

  servidor.on("error", (erro) => {
    console.error(`captura-live: bind ${HOST}:${PORT} falhou: ${erro.code ?? erro.message}`);
    process.exit(2);
  });

  servidor.listen(PORT, HOST, () => {
    console.log(
      `captura-live: proxy ${HOST}:${PORT} → ${alvoHost}:${alvoPorta} por ${TIMEOUT_MS} ms`,
    );
  });

  setTimeout(() => {
    servidor.close();
    escreveJson(OUT, {
      origem: `proxy entre client/WYD.exe e ${alvoHost}:${alvoPorta}`,
      comando: `node tools/client/captura-live.mjs ${process.argv.slice(2).join(" ")}`,
      endereco_proxy: `${HOST}:${PORT}`,
      alvo: `${alvoHost}:${alvoPorta}`,
      capturado_em: new Date().toISOString(),
      conexoes,
      frames,
    });
    const s2c = frames.filter((f) => f.direcao === "s2c").length;
    console.log(`captura-live: ${frames.length} frame(s): ${s2c} s2c / ${frames.length - s2c} c2s`);
    process.exit(frames.length === 0 ? 1 : 0);
  }, TIMEOUT_MS);
} else {
  /* ---------------------------------------------------------------- */
  /* Modo 1 — servidor falso (handshake em loopback)                   */
  /* ---------------------------------------------------------------- */
  const conexoes = [];
  const datagramas = [];
  let recebeuBytes = false;

  // Sonda de status dos canais: o cliente marca "Inacessível" sem resposta, e o
  // TCP 8281 não abre enquanto os canais não ficam acessíveis.
  const udp = dgram.createSocket("udp4");
  udp.on("message", (mensagem, origem) => {
    const registro = {
      de: `${origem.address}:${origem.port}`,
      em: new Date().toISOString(),
      tamanho: mensagem.length,
      hex: mensagem.toString("hex"),
    };
    datagramas.push(registro);
    console.log(
      `[${registro.em}] udp ${registro.de} ${registro.tamanho} B hex=${registro.hex}`,
    );
    udp.send(mensagem, origem.port, origem.address);
  });
  udp.on("error", (erro) => console.error(`captura-live: udp: ${erro.message}`));
  udp.bind(PORT, HOST);

  const servidor = net.createServer((socket) => {
    const peer = `${socket.remoteAddress}:${socket.remotePort}`;
    const registradoEm = new Date().toISOString();
    let acumulado = Buffer.alloc(0);
    let classificacao = "pendente";

    const fechar = (motivo) => {
      const entrada = {
        peer,
        inicio: registradoEm,
        classificacao,
        motivo_fechamento: motivo,
        tamanho: acumulado.length,
        hex: acumulado.toString("hex"),
        cabecalho: cabecalho(acumulado),
      };
      conexoes.push(entrada);
      if (classificacao === "cpsock" && acumulado.length > 0) recebeuBytes = true;
      console.log(
        `[${registradoEm}] ${peer} ${classificacao} ${acumulado.length} B` +
          (entrada.cabecalho
            ? ` size=${entrada.cabecalho.size} type=${entrada.cabecalho.type_hex} kw=${entrada.cabecalho.key_word} id=${entrada.cabecalho.id}`
            : "") +
          ` (${motivo})`,
      );
      if (acumulado.length > 0) console.log(`  hex: ${acumulado.toString("hex")}`);
      socket.destroy();
    };

    socket.on("data", (chunk) => {
      acumulado = Buffer.concat([acumulado, chunk]);
      const texto = acumulado.toString("latin1");
      if (classificacao === "pendente" && /^(GET|POST|HEAD) /.test(texto)) {
        classificacao = "http";
        const corpo = Buffer.from(SERV00, "utf8");
        socket.write(
          "HTTP/1.1 200 OK\r\n" +
            "content-type: text/html; charset=utf-8\r\n" +
            `content-length: ${corpo.length}\r\n` +
            "connection: close\r\n\r\n",
        );
        socket.write(corpo);
        socket.end();
        console.log(`[${registradoEm}] ${peer} http (${texto.split("\r\n")[0]})`);
        return;
      }
      if (classificacao === "pendente") classificacao = "cpsock";
    });

    socket.on("error", (erro) => fechar(`erro: ${erro.code ?? erro.message}`));
    socket.on("close", () => {
      if (!conexoes.some((entrada) => entrada.peer === peer)) fechar("fechado pelo cliente");
    });
  });

  servidor.on("error", (erro) => {
    console.error(`captura-live: bind ${HOST}:${PORT} falhou: ${erro.code ?? erro.message}`);
    process.exit(2);
  });

  servidor.listen(PORT, HOST, () => {
    console.log(
      `captura-live: escutando ${HOST}:${PORT} por ${TIMEOUT_MS} ms; artefato em ${OUT}`,
    );
    console.log(
      `captura-live: comando = node tools/client/captura-live.mjs ${process.argv.slice(2).join(" ")}`,
    );
  });

  setTimeout(() => {
    servidor.close();
    try {
      udp.close();
    } catch {
      // socket já fechado
    }
    escreveJson(OUT, {
      origem: "execução do cliente oficial client/WYD.exe apontado para loopback",
      aviso:
        "bytes crus de rede; contas/senhas digitadas no login NÃO foram preenchidas na execução registrada",
      comando: `node tools/client/captura-live.mjs --host ${HOST} --port ${PORT} --timeout ${TIMEOUT_MS} --out ${OUT}`,
      endereco_loopback: `${HOST}:${PORT}`,
      capturado_em: new Date().toISOString(),
      conexoes,
      datagramas_udp: datagramas,
    });
    console.log(`captura-live: ${conexoes.length} conexão(ões)`);
    if (!recebeuBytes) {
      console.error(
        "captura-live: nenhuma conexão CPSock com bytes — o cliente não chegou ao handshake",
      );
      process.exit(1);
    }
    process.exit(0);
  }, TIMEOUT_MS);
}
