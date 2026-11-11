import fs from 'node:fs';
import path from 'node:path';

const map = {
  campo_treinamento: 'sistema_campo_treinamento',
  campo_vivo: 'sistema_campo_vivo',
  canal_verde: 'integracao_canal_verde',
  cena_armia: 'sistema_cena_armia',
  cena_mobs: 'sistema_cena_mobs',
  config_gameplay: 'integracao_config_gameplay',
  config_recarga: 'integracao_config_recarga',
  conta_de_teste: 'sistema_conta_de_teste',
  entrada_mundo: 'sistema_entrada_mundo',
  gold_inicial: 'sistema_gold_inicial',
  handshake: 'sistema_handshake',
  limite_ip: 'sistema_limite_ip',
  loja_viva: 'sistema_loja_viva',
  lotacao: 'sistema_lotacao',
  monitoramento_contrato: 'contrato_monitoramento',
  personagens_crud: 'sistema_personagens_crud',
  pin_persistencia: 'sistema_pin_persistencia',
  pin_troca: 'sistema_pin_troca',
  portal_pin: 'sistema_portal_pin',
  refino_piedade: 'sistema_refino_piedade',
  selchar: 'sistema_selchar',
  selecao: 'sistema_selecao',
  sessao_unica: 'sistema_sessao_unica',
  status_udp: 'sistema_status_udp',
  tick_processo: 'integracao_tick_processo',
  autoridade: 'integracao_autoridade',
  compose: 'integracao_compose',
  content: 'integracao_content',
  determinism: 'integracao_determinism',
  gem: 'integracao_gem',
  gema_seed: 'golden_gema_seed',
  inventario: 'integracao_inventario',
  npc: 'integracao_npc',
  telemetry: 'sistema_telemetry',
  goldens: 'golden_formulas',
  refino_seed: 'golden_refino_seed',
  golden: 'golden_pacotes',
  roundtrip: 'integracao_roundtrip',
  flood: 'integracao_flood',
  metrics: 'integracao_metrics',
  catalogo: 'contrato_catalogo',
  constraints: 'contrato_migrations',
  pg: 'sistema_pg',
  sessions: 'sistema_sessoes',
  stack_caps: 'sistema_stack_caps',
  conteudo: 'contrato_conteudo',
  contract: 'contrato_transporte',
  operacao: 'contrato_operacao',
  painel: 'contrato_painel',
  seed: 'sistema_seed',
  unauthorized: 'contrato_autorizacao',
  identidade: 'contrato_identidade',
  serverlist_paridade: 'golden_serverlist_paridade',
  cena_campo: 'sistema_cena_campo',
};

const alvos = ['docs', 'Makefile', 'makefile', '.github', 'tools'];
const arquivos = [];
function walk(dir) {
  if (!fs.existsSync(dir)) return;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(p);
    else if (/\.(md|yml|yaml|mjs|js|json)$|Makefile|makefile/.test(entry.name)) arquivos.push(p);
  }
}
for (const a of alvos) {
  if (fs.statSync(a).isFile()) arquivos.push(a);
  else walk(a);
}

let mudados = 0;
for (const arquivo of arquivos) {
  const antes = fs.readFileSync(arquivo, 'utf8');
  let texto = antes;
  for (const [antigo, novo] of Object.entries(map)) {
    texto = texto.split(`tests/${antigo}.rs`).join(`tests/${novo}.rs`);
    texto = texto.split(`--test ${antigo}`).join(`--test ${novo}`);
  }
  // Referências sem pasta em estrutura-e-padroes (lista da árvore).
  for (const [antigo, novo] of Object.entries(map)) {
    texto = texto.split(`${antigo}.rs /`).join(`${novo}.rs /`);
  }
  if (texto !== antes) {
    fs.writeFileSync(arquivo, texto);
    mudados += 1;
    console.log('atualizado', arquivo);
  }
}
console.log('arquivos atualizados:', mudados);
