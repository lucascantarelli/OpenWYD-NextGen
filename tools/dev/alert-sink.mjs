#!/usr/bin/env node
/**
 * Receptor local de notificação do Alertmanager (DEVOPS-02).
 *
 * Não é observabilidade caseira: é o "humano" do diagrama
 * `Prometheus → Alertmanager → destino`, no formato mais simples que dá prova —
 * cada notificação entregue vira uma linha JSON em `ALERT_SINK_DIR/alertas.jsonl`
 * (volume `alerts-out` no compose).
 *
 * Para notificar de verdade (e-mail/Slack/webhook da equipe), troque a URL do receiver
 * em `docker/monitoring/alertmanager/alertmanager.yml`; este serviço pode então sair do
 * compose sem tocar em regra, roteamento ou runbook.
 *
 * Endpoints: `POST /alerts` (webhook do Alertmanager), `GET /healthz`, `GET /alertas`.
 */
import { appendFile, mkdir, readFile } from "node:fs/promises";
import { createServer } from "node:http";
import path from "node:path";

const porta = Number(process.env.ALERT_SINK_PORT ?? 9099);
const dir = process.env.ALERT_SINK_DIR ?? "/var/log/wyd-alerts";
const arquivo = path.join(dir, "alertas.jsonl");
const limite = 2 * 1024 * 1024; // 2 MiB: o arquivo é prova, não histórico

let sequencia = 0;

async function registrar(corpo) {
  await mkdir(dir, { recursive: true });
  sequencia += 1;
  const linha =
    JSON.stringify({
      recebido_em: new Date().toISOString(),
      sequencia,
      tamanho: corpo.length,
      payload: JSON.parse(corpo),
    }) + "\n";
  await appendFile(arquivo, linha, "utf8");
  return sequencia;
}

function responde(res, status, corpo = "", tipo = "text/plain; charset=utf-8") {
  res.writeHead(status, { "content-type": tipo });
  res.end(corpo);
}

const servidor = createServer((req, res) => {
  let corpo = "";
  req.on("data", (parte) => {
    corpo += parte;
    // Alertmanager com payload absurdo é sinal de erro de configuração, não de carga.
    if (corpo.length > limite) {
      req.destroy();
      responde(res, 413, "payload grande demais");
    }
  });
  req.on("end", async () => {
    if (req.method === "POST" && req.url?.startsWith("/alerts")) {
      try {
        const n = await registrar(corpo);
        console.log(`notificacao recebida #${n} bytes=${corpo.length}`);
        responde(res, 200, "ok");
      } catch (erro) {
        console.error(`falha ao gravar notificacao: ${erro.message}`);
        responde(res, 500, erro.message);
      }
      return;
    }
    if (req.method === "GET" && req.url === "/healthz") {
      responde(res, 200, "ok");
      return;
    }
    if (req.method === "GET" && req.url === "/alertas") {
      try {
        const linhas = await readFile(arquivo, "utf8");
        responde(res, 200, linhas, "application/x-ndjson");
      } catch (erro) {
        if (erro.code === "ENOENT") return responde(res, 200, "", "application/x-ndjson");
        responde(res, 500, erro.message);
      }
      return;
    }
    responde(res, 404, "nao encontrado");
  });
});

servidor.listen(porta, "0.0.0.0", () => {
  console.log(`alert-sink ouvindo em 0.0.0.0:${porta} -> ${arquivo}`);
});
