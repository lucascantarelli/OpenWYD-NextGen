// Fluxo por arquivo: `node scripts/_seq.js dump <arquivo.go>` lista os comentários
// candidatos (mesma heurística do _det.js) numerados; `node scripts/_seq.js apply
// <arquivo.go> <mapa.txt>` aplica um mapa com UMA linha por candidato, na mesma ordem.
// Linha "=" mantém o comentário como está. Só o texto após o primeiro "//" é trocado.
const fs = require('fs');

const ENW = new Set(
  'the and with when from this that which only must never because while before after instead is are was were be been being has have had does did but into than each every same reads holds keeps means its their they then there would should could will not for of to in on by or all any two such also over under about more most less other parses writes builds sits stays makes wrote returns first next where whose these those via an as at if so it we you no'.split(' ')
);
const PTW = new Set(
  'a o os as um uma uns umas de da do das dos que nao não para com sem por em no na nos nas ao aos à às e ou se é eh ja já sobre quando isso essa esse esta este pelo pela como onde porque mais menos muito toda todo todos todas ser está esta são tem há até entre depois antes cada qual quais devolve fica ficam usa usar recebe aplica envia grava lê sobe cai vem vai seu sua seus suas deste desta desse dessa num numa'.split(' ')
);

function candidates(p) {
  const out = [];
  fs.readFileSync(p, 'utf8')
    .split('\n')
    .forEach((l, i) => {
      const m = l.match(/\/\/\s*(.*)$/);
      if (!m) return;
      const t = l.slice(commentIndex(l) + 2).trim();
      if (!t || t.length < 4) return;
      if (/^(go:|nolint|\+build|Code generated|Copyright)/.test(t)) return;
      const toks = t.split(/[^A-Za-zÀ-ÿ0-9_]+/).filter((w) => /^[A-Za-zÀ-ÿ][A-Za-zÀ-ÿ0-9_]*$/.test(w));
      const alpha = toks.filter((w) => /^[A-Za-zÀ-ÿ]{2,}$/.test(w));
      if (alpha.length < 2) return;
      const low = alpha.map((w) => w.toLowerCase());
      const en = low.filter((w) => ENW.has(w)).length;
      const pt = low.filter((w) => PTW.has(w)).length;
      const diac = /[áàâãéêíóôõúüçÁÀÂÃÉÊÍÓÔÕÚÜÇ]/.test(t);
      const flags = en >= 2 || (en >= 1 && pt === 0 && alpha.length >= 3) || (alpha.length >= 6 && pt === 0 && !diac);
      if (flags) out.push({ line: i, text: t });
    });
  return out;
}

const mode = process.argv[2];
const target = process.argv[3];

// lista TODAS as linhas de comentário, numeradas (mapa: uma linha por comentário;
// "=" mantém o texto atual).
// posição do "//" que inicia comentário de linha, ignorando strings/runs.
function commentIndex(l) {
  let inD = false, inB = false, inR = false;
  for (let i = 0; i < l.length; i++) {
    const c = l[i];
    if (inD) {
      if (c === '\\') { i++; continue; }
      if (c === '"') inD = false;
      continue;
    }
    if (inB) {
      if (c === '`') inB = false;
      continue;
    }
    if (inR) {
      if (c === '\\') { i++; continue; }
      if (c === "'") inR = false;
      continue;
    }
    if (c === '"') { inD = true; continue; }
    if (c === '`') { inB = true; continue; }
    if (c === "'") { inR = true; continue; }
    if (c === '/' && l[i + 1] === '/') return i;
  }
  return -1;
}

function allComments(p) {
  const out = [];
  fs.readFileSync(p, 'utf8')
    .split('\n')
    .forEach((l, i) => {
      const idx = commentIndex(l);
      if (idx < 0) return;
      out.push({ line: i, text: l.slice(idx + 2).trim() });
    });
  return out;
}

if (mode === 'dumpc') {
  const cs = allComments(target);
  cs.forEach((c, n) => console.log(`${n + 1}\t${c.text}`));
  console.error('COMENTARIOS=' + cs.length);
} else if (mode === 'applyc') {
  const mapFile = process.argv[4];
  const cs = allComments(target);
  const map = fs.readFileSync(mapFile, 'utf8').split('\n');
  const last = map[map.length - 1] === '' ? map.length - 1 : map.length;
  const repl = map.slice(0, last);
  if (repl.length !== cs.length) {
    console.error(`ERRO: mapa tem ${repl.length} linhas, arquivo tem ${cs.length} comentarios`);
    process.exit(1);
  }
  const lines = fs.readFileSync(target, 'utf8').split('\n');
  let applied = 0;
  cs.forEach((c, n) => {
    const novo = repl[n];
    if (novo === '=') return;
    const i = commentIndex(lines[c.line]);
    lines[c.line] = lines[c.line].slice(0, i + 2) + ' ' + novo;
    applied++;
  });
  fs.writeFileSync(target, lines.join('\n'));
  console.log(`arquivo=${target} aplicadas=${applied} mantidas=${cs.length - applied}`);
} else if (mode === 'dump') {
  const cs = candidates(target);
  cs.forEach((c, n) => console.log(`${n + 1}\t${c.text}`));
  console.error('CANDIDATOS=' + cs.length);
} else if (mode === 'apply') {
  const mapFile = process.argv[4];
  const cs = candidates(target);
  const map = fs.readFileSync(mapFile, 'utf8').split('\n');
  const last = map[map.length - 1] === '' ? map.length - 1 : map.length;
  const repl = map.slice(0, last);
  if (repl.length !== cs.length) {
    console.error(`ERRO: mapa tem ${repl.length} linhas, arquivo tem ${cs.length} candidatos`);
    process.exit(1);
  }
  const lines = fs.readFileSync(target, 'utf8').split('\n');
  let applied = 0;
  cs.forEach((c, n) => {
    const novo = repl[n];
    if (novo === '=') return;
    const i = commentIndex(lines[c.line]);
    lines[c.line] = lines[c.line].slice(0, i + 2) + ' ' + novo;
    applied++;
  });
  fs.writeFileSync(target, lines.join('\n'));
  console.log(`arquivo=${target} aplicadas=${applied} mantidas=${cs.length - applied}`);
} else {
  console.error('uso: node scripts/_seq.js dump|apply <arquivo.go> [mapa.txt]');
  process.exit(1);
}
