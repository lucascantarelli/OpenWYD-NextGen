// Markdown do inventário: só tabelas com valores medidos no cliente real.
// O documento é gerado, nunca editado à mão (cabeçalho diz o comando).

const mb = (bytes) => `${(bytes / (1024 * 1024)).toFixed(1)} MB`;

function tabela(cabecalho, linhas) {
  return [
    `| ${cabecalho.join(" | ")} |`,
    `| ${cabecalho.map(() => "---").join(" | ")} |`,
    ...linhas.map((linha) => `| ${linha.join(" | ")} |`),
  ].join("\n");
}

export function renderInventario(inventario) {
  const linhas = [];
  linhas.push("# Inventário do cliente (gerado)");
  linhas.push("");
  linhas.push("> Gerado por `node apps/tools/bin/wyd-tools.mjs docs client`.");
  // Só o nome da pasta: o documento é versionado e não deve carregar caminho de máquina.
  const raiz = inventario.raiz.replace(/[\\/]+$/, "").split(/[\\/]/).pop();
  linhas.push(`> Raiz: \`${raiz}\``);
  linhas.push("");
  linhas.push("## Resumo");
  linhas.push("");
  linhas.push(`- **${inventario.arquivos}** arquivos de jogo`);
  linhas.push(`- **${mb(inventario.bytes)}** (${inventario.bytes} bytes)`);
  linhas.push(`- **${inventario.executaveis.length}** binários PE (.exe/.dll)`);
  linhas.push("");
  linhas.push("Artefatos do launcher (`manifest.txt`, `canais.txt`, `serverlist.bin`, `sn.bin`,");
  linhas.push("`WYDLauncher.exe`, `*.bak`) ficam fora do inventário: são ambiente, não cliente.");
  linhas.push("");
  linhas.push("## Por extensão");
  linhas.push("");
  linhas.push(
    tabela(
      ["Extensão", "Arquivos", "Bytes"],
      inventario.porExtensao.map(([ext, v]) => [ext, v.arquivos, v.bytes]),
    ),
  );
  linhas.push("");
  linhas.push("## Por diretório (topo)");
  linhas.push("");
  linhas.push(
    tabela(
      ["Diretório", "Arquivos", "Bytes"],
      inventario.porDiretorio.map(([dir, v]) => [dir, v.arquivos, v.bytes]),
    ),
  );
  linhas.push("");
  linhas.push("## Binários PE");
  linhas.push("");
  linhas.push("| Arquivo | Bytes | Machine | Subsystem | EntryPoint | Seções | SHA-256 |");
  linhas.push("| --- | --- | --- | --- | --- | --- | --- |");
  for (const exe of inventario.executaveis) {
    const pe = exe.pe ?? {};
    const secoes = Array.isArray(pe.sections) ? pe.sections.map((s) => s.name).join(", ") : (pe.erro ?? "—");
    linhas.push(
      `| ${exe.caminho} | ${exe.bytes} | ${pe.machine ?? "—"} | ${pe.subsystem ?? "—"} | ${pe.entryPoint ?? "—"} | ${secoes} | ${exe.sha256} |`,
    );
  }
  linhas.push("");
  linhas.push("## O que este documento NÃO cobre");
  linhas.push("");
  linhas.push("O inventário é a camada estática (arquivos, tamanhos, PE). Superfície de");
  linhas.push("protocolo/funções/variáveis continua nas ferramentas de RE por captura");
  linhas.push("(`tools/client/analyse.mjs`, `captura-live.mjs`) e no dossiê `docs/cliente/`;");
  linhas.push("nada aqui é inferido além do que os bytes mostram.");
  linhas.push("");
  return `${linhas.join("\n")}`;
}
