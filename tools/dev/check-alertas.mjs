#!/usr/bin/env node
/**
 * Valida a configuração do Alertmanager **renderizada** (DEVOPS-02).
 *
 * O arquivo versionado é um template (`__DESTINO__`, `__SLACK_CHANNEL__`, `__SMTP_*__`):
 * validá-lo cru faria o `amtool` recusar um receiver que não existe. Este script roda o
 * MESMO `render-config.sh` que o boot usa, num container descartável, e valida o resultado
 * com `amtool check-config` — logo, o que é validado aqui é exatamente o que o
 * Alertmanager carrega.
 *
 * Uso: `node tools/dev/check-alertas.mjs` (ou `make monitoring-rules`), sem a stack no ar.
 */
import { spawnSync } from "node:child_process";
import path from "node:path";

const dir = path.resolve("docker/monitoring/alertmanager");
const imagem = "prom/alertmanager:v0.28.1";

const r = spawnSync(
  "docker",
  [
    "run",
    "--rm",
    "-v",
    `${dir}:/etc/alertmanager:ro`,
    // Destino de e-mail também é validado: o Alertmanager exige to/from/smarthost
    // mesmo quando o receiver não é o da rota padrão.
    "-e",
    "ALERTMANAGER_DESTINO=slack",
    "--entrypoint",
    "sh",
    imagem,
    "/etc/alertmanager/render-config.sh",
    "--check",
  ],
  { encoding: "utf8", timeout: 60000 },
);

process.stdout.write(r.stdout ?? "");
process.stderr.write(r.stderr ?? "");
if (r.error) {
  console.error(`check-alertas: falha ao executar docker: ${r.error.message}`);
  process.exit(2);
}
if (r.status !== 0) process.exit(r.status ?? 1);

// O check-config valida o parse; ele não prova que o template RENDERIZA com um alerta
// completo (campo faltando quebra em tempo de execução, não na validação). Renderizamos
// os dois estados que precisam chegar ao Slack: disparo e resolução.
const TEXTOS = '{{ template "wyd.titulo" . }}\n{{ template "wyd.texto" . }}';

const CASOS = [
  {
    fixture: "slack-exemplo.json",
    exigidos: ["[FIRING]", ":warning:", "canal: `canal-1`", "runbook:", "desde "],
    proibidos: ["encerrado"],
  },
  {
    fixture: "slack-exemplo-resolvido.json",
    exigidos: ["[RESOLVED]", "encerrado ", "runbook:"],
    proibidos: [],
  },
];

for (const caso of CASOS) {
  const render = spawnSync(
    "docker",
    [
      "run",
      "--rm",
      "-v",
      `${dir}:/etc/alertmanager:ro`,
      "--entrypoint",
      "amtool",
      imagem,
      "template",
      "render",
      "--template.glob=/etc/alertmanager/templates/*.tmpl",
      `--template.text=${TEXTOS}`,
      `--template.data=/etc/alertmanager/${caso.fixture}`,
    ],
    { encoding: "utf8", timeout: 60000 },
  );

  const saida = render.stdout ?? "";
  if (render.status !== 0) {
    process.stderr.write(render.stderr ?? "");
    console.error(`check-alertas: render de ${caso.fixture} falhou (status ${render.status})`);
    process.exit(render.status ?? 1);
  }
  for (const exigido of caso.exigidos) {
    if (!saida.includes(exigido)) {
      console.error(`check-alertas: ${caso.fixture} sem \`${exigido}\` — mensagem perderia contexto`);
      process.exit(1);
    }
  }
  for (const proibido of caso.proibidos) {
    if (saida.includes(proibido)) {
      console.error(`check-alertas: ${caso.fixture} não deveria conter \`${proibido}\``);
      process.exit(1);
    }
  }
  if (saida.includes("<no value>")) {
    console.error(`check-alertas: ${caso.fixture} tem \`<no value>\` — label/anotação ausente`);
    process.exit(1);
  }
  console.log(`render ok (${caso.fixture}): ${saida.trim().split(/\r?\n/)[0]}`);
}
