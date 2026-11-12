// Markdown determinístico dos documentos de RE.
//
// Regras: nenhum timestamp, nenhuma ordem dependente de sistema de arquivos
// (tudo ordenado), nenhuma afirmação sem a fonte medida. Cada documento traz o
// comando que o regenera.
const esc = (texto) =>
  String(texto)
    .replace(/[\u0000-\u001f\u007f]/g, (c) => `\\x${c.charCodeAt(0).toString(16).padStart(2, "0")}`)
    .replaceAll("|", "\\|")
const hex = (n) => (n === null || n === undefined ? "—" : `0x${Number(n).toString(16)}`);
const corta = (lista, limite) => (lista.length > limite ? [...lista.slice(0, limite), `… (+${lista.length - limite})`] : lista);

function cabecalho(titulo, comando, aviso) {
  return [
    `# ${titulo}`,
    "",
    `> Gerado por \`${comando}\` — não editar à mão.`,
    aviso ? `> ${aviso}` : null,
    "",
  ].filter((l) => l !== null);
}

function tabela(cab, linhas) {
  return [
    `| ${cab.join(" | ")} |`,
    `| ${cab.map(() => "---").join(" | ")} |`,
    ...linhas.map((l) => `| ${l.map(esc).join(" | ")} |`),
  ].join("\n");
}

/** `docs/cliente/re-binarios.md` */
export function renderBinarios(binarios) {
  const linhas = cabecalho(
    "RE do cliente — binários PE",
    "node apps/tools/bin/wyd-tools.mjs binarios client",
    "Seções/imports/exports/strings/entrypoint são **medidos**; candidatos a função são **inferência** pela heurística declarada.",
  );
  const totalImportadas = binarios.reduce((n, b) => n + b.imports.reduce((s, d) => s + d.total, 0), 0);
  const dlls = new Set(binarios.flatMap((b) => b.imports.map((d) => d.dll))).size;
  linhas.push("## Resumo executivo", "");
  linhas.push(`- **${binarios.length}** binários PE analisados (${binarios.map((b) => b.caminho).join(", ")}).`);
  linhas.push(`- **${dlls}** DLLs importadas distintas; **${totalImportadas}** funções importadas.`);
  linhas.push(`- **${binarios.reduce((n, b) => n + b.exports.entradas.length, 0)}** exports nomeados.`);
  linhas.push(`- **${binarios.reduce((n, b) => n + b.strings.totais.relevant, 0)}** strings classificadas como relevantes (de ${binarios.reduce((n, b) => n + b.strings.totais.all, 0)} extraídas).`);
  linhas.push("");
  for (const bin of binarios) {
    linhas.push(`## ${bin.caminho}`, "");
    linhas.push(`- Arquitetura: **${bin.arquitetura}** · subsystem **${bin.subsystem}**`);
    linhas.push(`- EntryPoint (RVA/VA): **${bin.entryPoint.rvaHex} / ${bin.entryPoint.vaHex}** · ImageBase ${bin.imageBaseHex}`);
    linhas.push(`- Timestamp PE: ${bin.timestampIso ?? "—"} · PDB: ${bin.pdb ? `\`${bin.pdb}\`` : "ausente"}`, "");
    linhas.push("### Seções (medido)", "");
    linhas.push(tabela(["Seção", "VA", "VirtualSize", "RawSize"], bin.secoes.map((s) => [s.nome, s.vaHex, s.virtualSize, s.rawSize])), "");
    linhas.push("### Imports por DLL (medido)", "");
    if (bin.imports.length === 0) linhas.push("_Nenhum import no diretório PE._", "");
    for (const dll of bin.imports) {
      linhas.push(`**${dll.dll || "(sem nome)"}** — ${dll.total} função(ões):`);
      linhas.push("");
      linhas.push(corta(dll.funcoes, 40).map((f) => `\`${f}\``).join(", "), "");
    }
    linhas.push("### Exports (medido)", "");
    if (bin.exports.entradas.length === 0) linhas.push("_Sem tabela de export._", "");
    else linhas.push(tabela(["Nome", "Ordinal", "VA"], bin.exports.entradas.map((e) => [e.name, e.ordinal, hex(e.va)])), "");
    linhas.push("### Strings (medido)", "");
    linhas.push(`Totais: ${bin.strings.totais.all} extraídas (ascii ${bin.strings.totais.ascii}, utf16le ${bin.strings.totais.utf16le}), ${bin.strings.totais.relevant} classificadas. Top por categoria:`);
    linhas.push("");
    linhas.push(tabela(["Offset", "VA", "Seção", "Tags", "Texto"], bin.strings.relevant.slice(0, 120).map((s) => [hex(s.offset), hex(s.va), s.section ?? "—", (s.tags ?? []).join(","), s.text])), "");
    linhas.push("### Variáveis globais candidatas (inferência)", "");
    linhas.push(`Referências absolutas a seções de dados (\`A1/A3\` e \`8B/89/FF modrm=00 r/m=101\`): **${bin.variaveis.total}** endereços distintos. Nomes não são recuperados sem PDB/RTTI.`);
    linhas.push("");
    linhas.push(tabela(["VA (inferido)", "Seção", "Referências"], bin.variaveis.top.slice(0, 60).map((v) => [hex(v.va), v.secao, v.refs])), "");
    linhas.push("### Candidatos a função (inferência)", "");
    linhas.push(`Heurística: destinos de \`call\`/\`jmp rel32\` (E8/E9) dentro de \`${bin.funcoes.secao ?? "?"}\` + prólogos x86 (\`55 8B EC\`, \`55 89 E5\`, \`8B FF 55 8B EC\`). Prólogos encontrados: **${bin.funcoes.prologos}**; alvos distintos: **${bin.funcoes.totalAlvos}**.`);
    linhas.push("");
    linhas.push(tabela(["VA alvo (inferido)", "Chamadas"], bin.funcoes.alvos.slice(0, 60).map((f) => [hex(f.va), f.chamadas])), "");
    linhas.push("Nenhum nome de função é inventado: só exports têm nome medido.", "");
  }
  linhas.push("## Limitações registradas", "");
  linhas.push("- Candidatos a função são subconjunto: destinos diretos (E8/E9) e prólogos. Se a densidade for baixa em relação ao tamanho de `.text`, isso é compatível com packing/proteção (JProtect/XTrap); confirmar exige dump em memória, que este comando não faz.");
  linhas.push("- Nomes de função/variável só existem quando o binário tem PDB/export; a ausência é mostrada por binário e não é preenchida com suposição.");
  linhas.push("- Strings são extraídas por comprimento mínimo 5 e classificadas por regex; o total bruto inclui ruído de dados compactados.");
  linhas.push("");
  linhas.push("## Como regenerar", "", "```bash", "node apps/tools/bin/wyd-tools.mjs binarios client", "```", "");
  return `${linhas.join("\n")}\n`;
}

/** `docs/cliente/re-strings.md` (um binário, tabela completa das relevantes). */
export function renderStrings(bin) {
  const linhas = cabecalho(
    `RE do cliente — strings de ${bin.caminho}`,
    `node apps/tools/bin/wyd-tools.mjs strings client --bin ${bin.caminho}`,
    "Offset/VA/seção são medidos pelo parser PE; a classificação por tag é heurística de regex declarada em `tools/client/strings.mjs`.",
  );
  linhas.push("## Totais", "");
  linhas.push(`- ${bin.strings.totais.all} strings extraídas (ascii ${bin.strings.totais.ascii}, utf16le ${bin.strings.totais.utf16le}).`);
  linhas.push(`- ${bin.strings.totais.relevant} classificadas como relevantes (rede, dados, proteção, protocolo, versão, UI).`, "");
  linhas.push(tabela(["Offset", "VA", "Seção", "Enc", "Tags", "Texto"], bin.strings.relevant.map((s) => [hex(s.offset), hex(s.va), s.section ?? "—", s.encoding, (s.tags ?? []).join(","), s.text])), "");
  linhas.push("## Como regenerar", "", "```bash", `node apps/tools/bin/wyd-tools.mjs strings client --bin ${bin.caminho}`, "```", "");
  return `${linhas.join("\n")}\n`;
}

/** `docs/cliente/re-pacotes.md` */
export function renderPacotes(analise, rede = null) {
  const linhas = cabecalho(
    "RE do cliente — pacotes e protocolo",
    "node apps/tools/bin/wyd-tools.mjs pacotes",
    "Catálogo e handlers vêm do código Rust; capturas de `tmp/` entram como evidência e são marcadas quando não decodificadas.",
  );
  linhas.push("## Resumo executivo", "");
  linhas.push(`- **${analise.linhas.length}** opcodes no catálogo (\`opcode_table.rs\`).`);
  linhas.push(`- **${analise.totalComHandler}** com referência de handler em \`crates/wyd-server\`; **${analise.totalSemHandler}** sem referência.`);
  linhas.push(`- **${analise.totalComEstrutura}** com estrutura de campos declarada em \`crates/wyd-protocol/src/payload\`.`);
  linhas.push(`- **${analise.totalComCaptura}** com opcode confirmado por captura em \`tmp/\`.`);
  linhas.push(`- **${analise.capturas.length}** arquivo(s) de captura em \`tmp/\` (${analise.capturas.reduce((n, c) => n + c.frames, 0)} frames declarados).`);
  linhas.push("");
  linhas.push("## Opcodes", "");
  linhas.push("Handler = qualquer referência `Opcode::X` em `crates/wyd-server` (inclui o envio de respostas CNF, não só o `match` de entrada).");
  linhas.push("");
  linhas.push(
    tabela(
      ["Wire", "Opcode", "Direção", "Handler no servidor", "Estrutura (payload)", "Captura"],
      analise.linhas.map((l) => [
        l.wire,
        l.nome,
        l.direcao,
        l.handlers.length ? l.handlers.join(", ") : "não encontrado",
        l.estrutura ? `${l.estrutura.struct}${l.estrutura.wireLen ? ` (${l.estrutura.wireLen} B)` : ""}` : "—",
        l.capturas.join(", ") || "—",
      ]),
    ),
    "",
  );
  const semHandler = analise.linhas.filter((l) => l.handlers.length === 0);
  linhas.push("## Lacunas de handler", "");
  if (semHandler.length === 0) linhas.push("_Nenhuma lacuna no catálogo._", "");
  else {
    linhas.push(`${semHandler.length} opcode(s) do catálogo sem referência em \`crates/wyd-server\`:`);
    linhas.push("");
    linhas.push(semHandler.map((l) => `\`${l.wire}\` ${l.nome} (${l.direcao})`).join(" · "), "");
  }
  const comCaptura = analise.linhas.filter((l) => l.capturas.length > 0);
  linhas.push("## Opcodes vistos em captura", "");
  if (comCaptura.length === 0) linhas.push("_Nenhum opcode decodificado em captura._", "");
  else {
    linhas.push("Confirmados pelo diff de captura (opcode já decodificado no artefato; os frames crus seguem cifrados):");
    linhas.push("");
    linhas.push(tabela(["Wire", "Opcode", "Arquivo"], comCaptura.map((l) => [l.wire, l.nome, l.capturas.join(", ")])), "");
  }
  linhas.push("## Estruturas conhecidas", "");
  const comEstrutura = analise.linhas.filter((l) => l.estrutura);
  if (comEstrutura.length === 0) linhas.push("_Nenhuma estrutura mapeada._", "");
  else {
    for (const l of comEstrutura) {
      linhas.push(`### ${l.nome} (${l.wire})`, "");
      linhas.push(`\`struct ${l.estrutura.struct}\`${l.estrutura.wireLen ? ` — ${l.estrutura.wireLen} B de fio` : ""}`, "");
      linhas.push(l.estrutura.campos.length ? l.estrutura.campos.map((c) => `- \`${c}\``).join("\n") : "- (sem campos públicos)", "");
    }
  }
  linhas.push("## Evidência no cliente", "");
  if (!rede) linhas.push("_Sem binário analisado._", "");
  else {
    const redeImports = rede.imports.filter((d) => /ws2_32|wsock|wininet|winhttp/i.test(d.dll));
    linhas.push(`Imports de rede em ${rede.caminho}: ${redeImports.length ? redeImports.map((d) => `**${d.dll}** (${d.total})`).join(", ") : "nenhum"}.`, "");
    const stringsRede = rede.strings.relevant.filter((s) => s.tags.includes("rede") || s.tags.includes("protocolo"));
    linhas.push(`Strings de rede/protocolo classificadas: **${stringsRede.length}**; amostra:`, "");
    linhas.push(corta(stringsRede, 40).map((s) => `- \`${s.text}\` (${hex(s.va)})`).join("\n"), "");
  }
  linhas.push("## Capturas em `tmp/`", "");
  if (analise.capturas.length === 0) linhas.push("_Nenhuma captura encontrada; nada foi inferido._", "");
  else {
    linhas.push(tabela(["Arquivo", "Bytes", "Frames/tokens", "Tipos textuais"], analise.capturas.map((c) => [c.arquivo, c.bytes, c.frames, c.tiposTextuais.join(", ") || "—"])), "");
    linhas.push("Os frames crus guardam bytes cifrados; decodificar o `Type` exige a keyword da sessão, que não está no arquivo. Por isso a coluna de tipos só mostra o que já está em texto/diff (ex.: `diff-selchar.json`, que traz `opcode: 0x010a`) — o restante é **não decodificado**, não inferido.", "");
  }
  linhas.push("## Como regenerar", "", "```bash", "node apps/tools/bin/wyd-tools.mjs pacotes", "```", "");
  return `${linhas.join("\n")}\n`;
}
