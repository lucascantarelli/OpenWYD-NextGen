// Aplica um mapa path<TAB>texto_antigo<TAB>texto_novo, trocando apenas o conteúdo do
// comentário (tudo depois do primeiro "//") e exigindo exatamente uma ocorrência do
// texto antigo no arquivo (comparação no texto do comentário já aparado). Nunca toca em código.
const fs = require('fs');

const file = process.argv[2];
const raw = fs.readFileSync(file, 'utf8').split('\n').filter((l) => l.trim() !== '');
const files = new Map();
let applied = 0;
const refused = [];

for (const entry of raw) {
  const parts = entry.split('\t');
  if (parts.length < 3) {
    refused.push('MALFORMADA: ' + entry.slice(0, 60));
    continue;
  }
  const [path, oldText, newText] = parts;
  if (!files.has(path)) files.set(path, fs.readFileSync(path, 'utf8').split('\n'));
  const lines = files.get(path);
  const hits = [];
  for (let i = 0; i < lines.length; i++) {
    const idx = lines[i].indexOf('//');
    if (idx < 0) continue;
    if (lines[i].slice(idx + 2).trim() === oldText.trim()) hits.push(i);
  }
  if (hits.length !== 1) {
    refused.push(`${path} [${hits.length} ocorrencias]: ${oldText.slice(0, 50)}`);
    continue;
  }
  const i = hits[0];
  const idx = lines[i].indexOf('//');
  lines[i] = lines[i].slice(0, idx + 2) + ' ' + newText;
  applied++;
}

for (const [path, lines] of files) fs.writeFileSync(path, lines.join('\n'));
console.log(`aplicadas=${applied} recusadas=${refused.length}`);
for (const r of refused) console.log('RECUSADA ' + r);
