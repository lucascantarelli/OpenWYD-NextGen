import fs from 'node:fs';
import path from 'node:path';

const rows = [];
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(p);
    else if (entry.name.endsWith('.rs')) {
      const src = fs.readFileSync(p, 'utf8');
      const tests = [...src.matchAll(/#\[(?:tokio::)?test\][^\n]*\n(?:\s*#\[[^\]]*\]\n)*\s*(?:async\s+)?fn\s+(\w+)/g)].map((m) => m[1]);
      if (!tests.length) continue;
      rows.push({
        path: p.replace(/\\/g, '/'),
        loc: src.split('\n').length,
        tests,
        db: /wyd_db|PgBackend|DATABASE_URL/.test(src),
        net: /TcpStream|TcpListener/.test(src),
        proc: /Command::new|docker/.test(src),
        env: (src.match(/env::var/g) || []).length,
        sleep: (src.match(/sleep\(/g) || []).length,
        ignore: (src.match(/#\[ignore\]/g) || []).length,
        unsafe: (src.match(/\bunsafe\b/g) || []).length,
        unit: p.includes(`${path.sep}src${path.sep}`),
      });
    }
  }
}
walk('crates');
rows.sort((a, b) => a.path.localeCompare(b.path));
fs.writeFileSync('tmp-testes.json', JSON.stringify(rows, null, 1));
for (const r of rows) {
  const flags = `${r.unit ? 'U' : 'I'}${r.db ? ' db' : ''}${r.net ? ' net' : ''}${r.proc ? ' proc' : ''}`;
  console.log(`${r.tests.length}\t${r.loc}\t${flags}\tenv=${r.env} sleep=${r.sleep}\t${r.path}`);
}
console.log('TOTAL', rows.length, rows.reduce((a, r) => a + r.tests.length, 0));
