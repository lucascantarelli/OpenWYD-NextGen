// Lista comentários de linha candidatos a inglês nos diretórios/paths passados.
const fs = require('fs');
const path = require('path');
const ENW = new Set(
  'the and with when from this that which only must never because while before after instead is are was were be been being has have had does did but into than each every same reads holds keeps means its their they then there would should could will not for of to in on by or all any two such also over under about more most less other parses writes builds sits stays makes wrote returns first next where whose these those via an as at if so it we you no'.split(' ')
);
const PTW = new Set(
  'a o os as um uma uns umas de da do das dos que nao não para com sem por em no na nos nas ao aos à às e ou se é eh ja já sobre quando isso essa esse esta este pelo pela como onde porque mais menos muito toda todo todos todas ser está esta são tem há até entre depois antes cada qual quais devolve fica ficam usa usar recebe aplica envia grava lê sobe cai vem vai seu sua seus suas deste desta desse dessa num numa'.split(' ')
);
const out = [];
function walk(d) {
  if (fs.statSync(d).isFile()) {
    report(d);
    return;
  }
  for (const e of fs.readdirSync(d, { withFileTypes: true })) {
    const p = path.join(d, e.name);
    if (e.isDirectory()) {
      if (e.name === 'testdata' || e.name.startsWith('.')) continue;
      walk(p);
      continue;
    }
    if (!p.endsWith('.go')) continue;
    report(p);
  }
}

function report(p) {
    fs.readFileSync(p, 'utf8')
      .split('\n')
      .forEach((l, i) => {
        const m = l.match(/\/\/\s*(.*)$/);
        if (!m) return;
        const t = m[1].trim();
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
        if (flags) out.push(p + ':' + (i + 1) + ': ' + t);
      });
}

process.argv.slice(2).forEach(walk);
console.log(out.join('\n'));
console.error('TOTAL=' + out.length);
