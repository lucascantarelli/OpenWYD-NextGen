#!/usr/bin/env node
// E2E real da janela do launcher (CLIENT-08) com tauri-driver + msedgedriver.
//
// Sobe o portal falso (notícias), inicia o tauri-driver apontando para o
// `WYDLauncher.exe` que está DENTRO da pasta do jogo e conduz a janela real:
// confere o HTML injetado, espera a notícia do portal, clica no botão Jogar,
// prova que o jogo abriu exatamente uma vez e que manifesto ausente não abre
// nada nem reescreve `serverlist.bin`.
//
// Uso (com tauri-driver e msedgedriver instalados):
//   node tools/launcher/e2e-janela.mjs
//   node tools/launcher/e2e-janela.mjs --exe target/debug/WYDLauncher.exe \
//     --jogo tmp/e2e-janela/jogo --driver tmp/drivers/msedgedriver.exe
//
// Requisitos: Windows com Edge/WebView2, `tauri-driver` no PATH (ou ~/.cargo/bin).
import { execFileSync, spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { copyFileSync, existsSync, readFileSync, renameSync, writeFileSync } from "node:fs";
import http from "node:http";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const RAIZ = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

function arg(nome, padrao) {
  const i = process.argv.indexOf(nome);
  return i > 0 && process.argv[i + 1] ? process.argv[i + 1] : padrao;
}

const EXE = path.resolve(RAIZ, arg("--exe", "target/debug/WYDLauncher.exe"));
// `--cliente client` conduz a janela na PASTA REAL do cliente: o manifesto e os
// canais são os bytes reais, e só o jogo aberto é falso (`e2e-jogo.exe`), nunca
// o `WYD.exe` do jogador.
const CLIENTE = arg("--cliente", "");
let JOGO = path.resolve(RAIZ, arg("--jogo", "tmp/e2e-janela/jogo"));
let GAME_EXE = "WYD.exe";
let LAUNCHER = null;
let LAUNCHER_ARGS = [];
const MANIFESTO = () => path.join(JOGO, "manifest.txt");
const DRIVER = path.resolve(RAIZ, arg("--driver", "tmp/drivers/msedgedriver.exe"));
const TAURI_DRIVER = arg(
  "--tauri-driver",
  path.join(os.homedir(), ".cargo", "bin", "tauri-driver.exe"),
);
let driver = null;
let portal = null;
let portaPortal = 0;
let sessao = null;
let falhou = null;

const log = (texto) => console.log(`[e2e-janela] ${texto}`);

function exigir(condicao, mensagem) {
  if (!condicao) throw new Error(mensagem);
}

function sha256(arquivo) {
  return createHash("sha256").update(readFileSync(arquivo)).digest("hex");
}

/** Conta processos com o nome exato (tasklist devolve CSV). */
function processos(nome) {
  const saida = execFileSync(
    "tasklist",
    ["/FI", `IMAGENAME eq ${nome}`, "/FO", "CSV", "/NH"],
    { encoding: "utf8" },
  );
  return saida
    .split(/\r?\n/)
    .filter((linha) => linha.toLowerCase().includes(`"${nome.toLowerCase()}"`))
    .map((linha) => Number(linha.split(",")[1]?.replaceAll('"', "")))
    .filter((pid) => Number.isInteger(pid) && pid > 0);
}

function jogoFalso() {
  // Jogo falso: um binário mínimo que dorme, compilado com a toolchain do
  // projeto. Fica vivo (dá para contar processo e provar que não duplicou) e
  // não depende de binário de sistema renomeado — que no Windows pode sair na hora.
  const fonte = path.join(RAIZ, "tmp", "e2e-janela", "jogo-falso.rs");
  const binario = path.join(RAIZ, "tmp", "e2e-janela", "jogo-falso.exe");
  writeFileSync(fonte, "fn main() { std::thread::sleep(std::time::Duration::from_secs(120)); }\n");
  execFileSync("rustc", [fonte, "-O", "-o", binario], { stdio: "inherit" });
  return binario;
}

function prepararClienteReal() {
  JOGO = path.resolve(RAIZ, CLIENTE);
  LAUNCHER = path.join(JOGO, "WYDLauncher.exe");
  for (const obrigatorio of [LAUNCHER, path.join(JOGO, "manifest.txt"), path.join(JOGO, "canais.txt")]) {
    if (!existsSync(obrigatorio)) {
      throw new Error(`pasta do cliente incompleta (falta ${obrigatorio}); rode o gerador de manifesto/canais`);
    }
  }
  if (!existsSync(DRIVER)) throw new Error(`msedgedriver ausente: ${DRIVER}`);
  if (!existsSync(TAURI_DRIVER)) throw new Error(`tauri-driver ausente: ${TAURI_DRIVER}`);
  GAME_EXE = "e2e-jogo.exe";
  copyFileSync(jogoFalso(), path.join(JOGO, GAME_EXE));
  // Sem `--instalar`: o binário da pasta real já é o launcher instalado.
  // O tauri-driver não encaminha `args` de forma confiável no app empacotado:
  // as mesmas escolhas vão por ambiente (`WYD_LAUNCHER_DIR`/`WYD_LAUNCHER_EXE`).
  LAUNCHER_ARGS = [];
  process.env.WYD_LAUNCHER_DIR = JOGO;
  process.env.WYD_LAUNCHER_EXE = GAME_EXE;
}

function prepararJogo() {
  if (!existsSync(EXE)) throw new Error(`launcher não construído: ${EXE} (cargo build -p wyd-launcher-ui)`);
  if (!existsSync(DRIVER)) throw new Error(`msedgedriver ausente: ${DRIVER}`);
  if (!existsSync(TAURI_DRIVER)) throw new Error(`tauri-driver ausente: ${TAURI_DRIVER}`);

  execFileSync("mkdir", ["-p", JOGO]);
  LAUNCHER = path.join(JOGO, "WYDLauncher.exe");
  if (!existsSync(path.join(JOGO, "jogo.txt"))) {
    writeFileSync(path.join(JOGO, "jogo.txt"), "cliente v1\n");
  }
  const conteudo = readFileSync(path.join(JOGO, "jogo.txt"));
  const hash = createHash("sha256").update(conteudo).digest("hex");
  writeFileSync(
    MANIFESTO(),
    `build 3\nprotocol 7662\n${hash} ${conteudo.byteLength} jogo.txt\n`,
  );
  writeFileSync(
    path.join(JOGO, "canais.txt"),
    "# e2e\n1 0 http://192.168.100.3/serv00.htm\n1 1 192.168.100.3\n",
  );
  copyFileSync(jogoFalso(), path.join(JOGO, GAME_EXE));
  // O executável que o jogador usa fica na pasta do jogo, sem `--dir`.
  execFileSync(EXE, ["--instalar", JOGO], { stdio: "pipe" });
}

/** Portal falso: notícias estáveis + telemetria 204 (o launcher também manda). */
function subirPortal() {
  const servidor = http.createServer((req, res) => {
    if (req.method === "GET" && req.url === "/api/launcher/noticias") {
      res.writeHead(200, { "content-type": "application/json" });
      res.end(
        JSON.stringify({
          noticias: [
            {
              slug: "e2e",
              titulo: "Notícias do E2E",
              data: "2026-11-12",
              tag: "Evento",
              resumo: "notícia injetada pelo teste da janela",
              url: "/noticias/e2e",
            },
          ],
        }),
      );
      return;
    }
    if (req.method === "POST" && req.url === "/api/launcher/telemetria") {
      req.resume();
      res.writeHead(204);
      res.end();
      return;
    }
    res.writeHead(404, { "content-type": "application/json" });
    res.end(JSON.stringify({ error: "nao encontrado" }));
  });
  return new Promise((resolve, rejeitar) => {
    servidor.once("error", rejeitar);
    servidor.listen(0, "127.0.0.1", () => {
      portaPortal = servidor.address().port;
      resolve(servidor);
    });
  });
}

async function subirDriver() {
  const porta = 4444 + Math.floor(Math.random() * 500);
  driver = spawn(
    TAURI_DRIVER,
    ["--port", String(porta), "--native-driver", DRIVER],
    {
      cwd: RAIZ,
      env: {
        ...process.env,
        WYD_LAUNCHER_PORTAL: `http://127.0.0.1:${portaPortal}`,
      },
      stdio: ["ignore", "pipe", "pipe"],
    },
  );
  let saida = "";
  driver.stdout.on("data", (pedaco) => (saida += pedaco));
  driver.stderr.on("data", (pedaco) => (saida += pedaco));
  for (let i = 0; i < 120; i += 1) {
    try {
      const resposta = await fetch(`http://127.0.0.1:${porta}/status`);
      if (resposta.ok) return porta;
    } catch {
      // ainda subindo
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`tauri-driver não respondeu:\n${saida}`);
}

async function wd(metodo, caminho, corpo) {
  const url = `http://127.0.0.1:${driver.porta}${caminho}`;
  const resposta = await fetch(url, {
    method: metodo,
    headers: corpo ? { "content-type": "application/json" } : undefined,
    body: corpo ? JSON.stringify(corpo) : undefined,
  });
  const texto = await resposta.text();
  const json = texto ? JSON.parse(texto) : {};
  if (!resposta.ok) throw new Error(`${metodo} ${caminho} → ${resposta.status}: ${texto}`);
  return json.value ?? json;
}

async function novoSession() {
  const criado = await wd("POST", "/session", {
    capabilities: {
      alwaysMatch: {
        "tauri:options": { application: LAUNCHER, args: LAUNCHER_ARGS },
      },
    },
  });
  sessao = criado.sessionId ?? criado["sessionId"];
  exigir(sessao, "sessão WebDriver sem sessionId");
  await wd("POST", `/session/${sessao}/timeouts`, {
    // O Jogar confere 7.131 arquivos do cliente real (344 MB): no build debug
    // isso passa de 30 s, então o script precisa de folga para não estourar.
    script: 300_000,
    implicit: 5_000,
    pageLoad: 60_000,
  });
}

const js = (script) =>
  wd("POST", `/session/${sessao}/execute/sync`, { script, args: [] });

async function esperar(script, descricao, tentativas = 80) {
  for (let i = 0; i < tentativas; i += 1) {
    if (await js(script)) return;
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`tempo esgotado esperando: ${descricao}`);
}

async function clicarSubmit() {
  const achado = await wd("POST", `/session/${sessao}/element`, {
    using: "css selector",
    value: "form button[type=submit]",
  });
  const id = achado["element-6066-11e4-a52e-4f735466cecf"] ?? achado.ELEMENT;
  exigir(id, "botão Jogar não encontrado no DOM");
  await wd("POST", `/session/${sessao}/element/${id}/click`, {});
}

async function principal() {
  // `--cliente` usa a pasta real pronta; sem ele, o harness monta o jogo sintético.
  if (CLIENTE) prepararClienteReal();
  else prepararJogo();
  portal = await subirPortal();
  log(`portal falso em http://127.0.0.1:${portaPortal}`);
  const portaDriver = await subirDriver();
  driver.porta = portaDriver;
  log(`tauri-driver em http://127.0.0.1:${driver.porta}`);

  await novoSession();
  log("sessão WebDriver criada");

  await esperar(
    "return !!(document.querySelector('form') && document.querySelector('#status'));",
    "formulário de entrada no DOM",
  );

  const titulo = await js("return document.title;");
  exigir(/WYD/i.test(titulo), `título inesperado: ${titulo}`);
  const html = await js("return document.documentElement.outerHTML;");
  exigir(html.includes("Canais"), "painel Canais não renderizou");
  exigir(html.includes("Jogar"), "botão Jogar não renderizou");
  exigir(html.includes("data:image/jpeg;base64,"), "fundo da marca não foi injetado");
  log(`DOM ok: título "${titulo}", ${html.length} caracteres de HTML`);

  await esperar(
    "return document.documentElement.outerHTML.includes('Notícias do E2E');",
    "notícia vinda do portal",
  );
  log("notícia do portal apareceu no DOM");

  // Ação Jogar: o clique no submit da própria página chama o IPC `jogar`.
  const antes = processos(GAME_EXE);
  exigir(antes.length === 0, `já havia ${GAME_EXE} rodando: ${antes}`);
  // O formulário da identidade tem conta/senha `required` (a validação do
  // navegador bloqueia o submit vazio). O jogador digita antes de clicar; o
  // teste faz o mesmo — o launcher não guarda nem usa a senha.
  await js(`
    const form = document.querySelector('form');
    form.querySelector('[name=conta]').value = 'e2e';
    form.querySelector('[name=senha]').value = 'e2e-segredo';
    return true;
  `);
  await clicarSubmit();
  let abertos = [];
  const esperarJogo = async (tentativas) => {
    for (let i = 0; i < tentativas; i += 1) {
      abertos = processos(GAME_EXE);
      if (abertos.length > 0) return;
      await new Promise((r) => setTimeout(r, 250));
    }
  };
  await esperarJogo(24);
  if (abertos.length === 0) {
    // WebDriver às vezes entrega o clique sem acionar o submit (coordenadas);
    // um clique real no DOM prova o mesmo caminho da página.
    await js("document.querySelector('form button[type=submit]').click(); return true;");
    await esperarJogo(24);
  }
  if (abertos.length !== 1) {
    const status = await js("return document.querySelector('#status')?.textContent ?? '';");
    throw new Error(
      `esperava 1 jogo aberto, veio ${abertos.length}; status da janela: ${status}`,
    );
  }
  exigir(abertos.length === 1, `esperava 1 jogo aberto, veio ${abertos.length}`);
  await new Promise((r) => setTimeout(r, 750));
  const depois = processos(GAME_EXE);
  exigir(
    depois.length === 1 && depois[0] === abertos[0],
    `jogo duplicou ou reiniciou: ${JSON.stringify({ antes, abertos, depois })}`,
  );
  log(`Jogar abriu exatamente um processo ${GAME_EXE} (pid ${depois[0]})`);
  execFileSync("taskkill", ["/PID", String(depois[0]), "/F"], { stdio: "ignore" });

  // Fail-closed: sem manifesto, nada é escrito e a janela mostra o erro tipado.
  // Na pasta real (`--cliente`) o manifesto é do jogador: não se renomeia.
  if (CLIENTE) {
    log("fail-closed da pasta real coberto pelo teste Rust (manifesto ausente/vermelho)");
  } else {
    const serverlist = path.join(JOGO, "serverlist.bin");
    const hashAntes = existsSync(serverlist) ? sha256(serverlist) : null;
    renameSync(MANIFESTO(), `${MANIFESTO()}.off`);
    try {
      await clicarSubmit();
      await esperar(
        "return document.documentElement.outerHTML.includes('manifesto');",
        "mensagem de manifesto no DOM",
      );
      const hashDepois = existsSync(serverlist) ? sha256(serverlist) : null;
      exigir(hashAntes === hashDepois, "serverlist.bin mudou mesmo com manifesto ausente");
      exigir(processos(GAME_EXE).length === 0, "jogo abriu com manifesto ausente");
      log("fail-closed ok: manifesto ausente não abriu nada nem reescreveu serverlist.bin");
    } finally {
      renameSync(`${MANIFESTO()}.off`, MANIFESTO());
    }
  }

  await wd("DELETE", `/session/${sessao}`);
  sessao = null;
  log("SUCESSO: janela renderizou a identidade, notícia apareceu e Jogar abriu o jogo uma vez");
}

try {
  await principal();
} catch (erro) {
  falhou = erro;
  console.error(`[e2e-janela] FALHA: ${erro.message}`);
} finally {
  try {
    if (sessao) await wd("DELETE", `/session/${sessao}`);
  } catch {
    // sessão já morreu junto com o driver
  }
  for (const pid of processos(GAME_EXE)) {
    try {
      execFileSync("taskkill", ["/PID", String(pid), "/F"], { stdio: "ignore" });
    } catch {
      // já saiu
    }
  }
  if (CLIENTE && GAME_EXE === "e2e-jogo.exe") {
    // O jogo falso é o único arquivo criado na pasta real; sai junto com o teste.
    try {
      execFileSync("rm", ["-f", path.join(JOGO, GAME_EXE)], { stdio: "ignore" });
    } catch {
      // já removido
    }
  }
  driver?.kill?.();
  portal?.close?.();
}
process.exit(falhou ? 1 : 0);
