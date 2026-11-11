import fs from 'node:fs';

const ctx = {
  'crates/wyd-server/tests/sistema/conta_de_teste.rs': 'provisionamento da conta de teste',
  'crates/wyd-server/tests/sistema/cena_armia.rs': 'prova viva da cena',
  'crates/wyd-server/tests/sistema/lotacao.rs': 'lotação do canal',
  'crates/wyd-server/tests/sistema/portal_pin.rs': 'PIN pelo portal',
  'crates/wyd-server/tests/sistema/loja_viva.rs': 'loja viva',
  'crates/wyd-server/tests/sistema/login.rs': 'login do canal',
  'crates/wyd-server/tests/sistema/selchar.rs': 'selchar',
  'crates/wyd-server/tests/sistema/pin_troca.rs': 'troca de PIN',
  'crates/wyd-server/tests/sistema/refino_piedade.rs': 'refino/piedade',
  'crates/wyd-server/tests/sistema/entrada_mundo.rs': 'entrada no mundo',
  'crates/wyd-server/tests/sistema/gold_inicial.rs': 'gold inicial',
  'crates/wyd-server/tests/sistema/personagens_crud.rs': 'personagens CRUD',
  'crates/wyd-server/tests/sistema/pin_persistencia.rs': 'PIN persistido',
  'crates/wyd-server/tests/sistema/campo_vivo.rs': 'quest do campo',
  'crates/wyd-db/tests/sistema/pg.rs': 'integração com Postgres',
  'crates/wyd-db/tests/sistema/sessoes.rs': 'presença por canal',
  'crates/wyd-db/tests/sistema/stack_caps.rs': 'view item_stack_cap',
  'crates/wyd-admin/tests/contrato/conteudo.rs': 'leitura real de conteúdo',
  'crates/wyd-admin/tests/contrato/operacao.rs': 'operação real',
  'crates/wyd-admin/tests/sistema/seed.rs': 'seed de boot',
};

function removeItem(texto, marcador) {
  const idx = texto.indexOf(marcador);
  if (idx < 0) return texto;
  // remove comentários /// imediatamente acima
  let inicio = idx;
  for (;;) {
    const linhaInicio = texto.lastIndexOf('\n', inicio - 1) + 1;
    const linha = texto.slice(linhaInicio, inicio).trim();
    if (linha.startsWith('///')) {
      inicio = linhaInicio;
    } else {
      break;
    }
  }
  // casamento de chaves a partir do primeiro '{'
  const abre = texto.indexOf('{', idx);
  let nivel = 0;
  let fim = abre;
  for (; fim < texto.length; fim += 1) {
    if (texto[fim] === '{') nivel += 1;
    if (texto[fim] === '}') {
      nivel -= 1;
      if (nivel === 0) {
        fim += 1;
        break;
      }
    }
  }
  return (texto.slice(0, inicio) + texto.slice(fim)).replace(/\n{3,}/g, '\n\n');
}

for (const [arquivo, contexto] of Object.entries(ctx)) {
  let texto = fs.readFileSync(arquivo, 'utf8');
  const original = texto;

  // 1. módulo comum (um nível acima: tests/<tipo>/arquivo.rs)
  if (!texto.includes('mod common;')) {
    const linhas = texto.split('\n');
    let fimDoc = 0;
    while (fimDoc < linhas.length && (linhas[fimDoc].startsWith('//!') || linhas[fimDoc].trim() === '')) {
      if (linhas[fimDoc].startsWith('//!')) fimDoc += 1;
      else if (fimDoc === 0) fimDoc += 1;
      else break;
    }
    linhas.splice(fimDoc, 0, '', '#[path = "../common/mod.rs"]', 'mod common;');
    texto = linhas.join('\n');
  }

  texto = removeItem(texto, 'fn dsn() -> Option<String> {');
  texto = removeItem(texto, 'macro_rules! skip_sem_banco {');

  texto = texto.replaceAll('let dsn = dsn().expect(', 'let dsn = common::dsn().expect(');
  texto = texto.replaceAll('let dsn = dsn()?;', `let dsn = common::dsn_ou_skip("${contexto}")?;`);
  texto = texto.replaceAll('let Some(dsn) = dsn() else', `let Some(dsn) = common::dsn_ou_skip("${contexto}") else`);
  texto = texto.replaceAll(
    'skip_sem_banco!();',
    `let Some(_dsn) = common::dsn_ou_skip("${contexto}") else { return; };`,
  );

  // gate com eprintln local antigo dentro do else
  texto = texto.replace(
    /let Some\(dsn\) = common::dsn_ou_skip\("([^"]*)"\) else \{\s*eprintln!\([\s\S]*?\);\s*return;\s*\};/g,
    'let Some(dsn) = common::dsn_ou_skip("$1") else { return; };',
  );

  // qualquer sobra
  texto = texto.replaceAll('dsn()', `common::dsn_ou_skip("${contexto}")`);

  if (texto !== original) fs.writeFileSync(arquivo, texto);
}
console.log('ok', Object.keys(ctx).length);
