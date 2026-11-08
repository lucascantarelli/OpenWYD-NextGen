#!/usr/bin/env node
/**
 * Captura do primeiro pacote C→S do cliente 7662 em loopback (FASE-08-6).
 *
 * O cliente faz, em ordem (`docs/cliente/depuracao.md`, skill `wyd-client-connect`):
 *
 *   1. `GET http://IP:PORTA/serv00.htm` — status/versão do servidor;
 *   2. TCP `IP:PORTA` no formato CPSock: 4 bytes `INITCODE` (0x1F11F311) e depois
 *      `HEADER` de 12 bytes + corpo (`AccountLogin` = 0x020D quando o jogador envia o login).
 *
 * Este harness escuta em loopback, responde o `serv00.htm` mínimo e grava **os bytes
 * crus** de cada conexão que não for HTTP, com hex, tamanho, peer e o cabeçalho CPSock já
 * decodificado quando der (Size/KeyWord/CheckSum/Type/ID/Tick). Nada aqui inventa byte:
 * o que não chegar não entra no artefato.
 *
 * Uso:
 *   node tools/client/captura-live.mjs --timeout 90000 \
 *     --out fixtures/cliente/captura-handshake.json
 *
 * Saída: exit 0 quando pelo menos uma conexão trouxe bytes; exit 1 quando o cliente não
 * conectou (com o resumo do que aconteceu), exit 2 em erro de bind/config.
 */
import dgram from "node:dgram";
import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import process from "node:process";

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

const conexoes = [];
const datagramas = [];
let recebeuBytes = false;

// Sonda de status dos canais: o cliente marca "Inacessível" sem receber resposta, e o
// TCP 8281 não abre enquanto os canais não ficam acessíveis. Capturamos o datagrama e
// devolvemos o mesmo payload, que é o comportamento de um servidor vivo (echo).
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
    // INITCODE (4 B) + HEADER + possível AccountLogin: guarda tudo até o fim da janela.
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
  console.log(`captura-live: comando = node tools/client/captura-live.mjs ${process.argv.slice(2).join(" ")}`);
});

setTimeout(() => {
  servidor.close();
  try {
    udp.close();
  } catch {
    // socket já fechado
  }
  const artefato = {
    origem: "execução do cliente oficial client/WYD.exe apontado para loopback",
    aviso:
      "bytes crus de rede; contas/senhas digitadas no login NÃO foram preenchidas na execução registrada",
    comando: `node tools/client/captura-live.mjs --host ${HOST} --port ${PORT} --timeout ${TIMEOUT_MS} --out ${OUT}`,
    endereco_loopback: `${HOST}:${PORT}`,
    capturado_em: new Date().toISOString(),
    conexoes,
    datagramas_udp: datagramas,
  };
  const destino = path.resolve(OUT);
  fs.mkdirSync(path.dirname(destino), { recursive: true });
  fs.writeFileSync(destino, `${JSON.stringify(artefato, null, 2)}\n`, "utf8");
  console.log(`captura-live: ${conexoes.length} conexão(ões); artefato escrito em ${destino}`);
  if (!recebeuBytes) {
    console.error(
      "captura-live: nenhuma conexão CPSock com bytes — o cliente não chegou ao handshake",
    );
    process.exit(1);
  }
  process.exit(0);
}, TIMEOUT_MS);
