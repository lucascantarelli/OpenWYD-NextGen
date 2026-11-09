#!/usr/bin/env node
// Gera e inspeciona `serverlist.bin` — o arquivo que o cliente 7662 lê na tela
// inicial para montar a coluna Servidor e a coluna Canal.
//
// Porte fiel de `client-tools/serverlist/main.go` (branch `main`), que substituiu o
// "serverlist editor.exe" fechado. O formato é o de `BASE_InitializeServerList`
// (`Source/Code/Basedef.cpp:6874`):
//
//   MAX_SERVERGROUP(10) × MAX_SERVERNUMBER(11) × 64 bytes = 7040 B
//
// Cada célula é um endereço em ASCII terminado em NUL, com cada byte somado de
// `obfuscationKey[63 - i]` (o legado soma na leitura e subtrai na escrita da mesma
// tabela). A célula 0 de um grupo carrega a URL de status (`http://IP/serv00.htm`);
// as células 1..10 carregam os canais daquele servidor — é delas que o cliente
// desenha a coluna Canal, e a lista só aparece com mais de um endereço preenchido.
//
// Uso:
//   node tools/client/serverlist.mjs --in tools/client/serverlist-servers.txt --out client/serverlist.bin
//   node tools/client/serverlist.mjs --print client/serverlist.bin
import fs from 'node:fs';
import path from 'node:path';

/** MAX_SERVERGROUP (`Basedef.h:109`): quantos servidores o cliente lista. */
export const GROUP_COUNT = 10;
/** MAX_SERVERNUMBER = MAX_SERVER + 1 (`Basedef.h:108-110`): canais por servidor. */
export const CHANNEL_COUNT = 11;
/** Largura fixa de uma célula (`g_pServerList[..][..][64]`). */
export const ADDR_LEN = 64;
/** sizeof(g_pServerList) — tamanho exato do arquivo. */
export const BIN_SIZE = GROUP_COUNT * CHANNEL_COUNT * ADDR_LEN;
/** Limite de caracteres úteis da URL de status (célula 0). Medido no cliente. */
export const STATUS_URL_MAX = 32;

/** Tabela estática que o legado soma a cada byte (`Basedef.cpp:6927`). */
export const OBFUSCATION_KEY = Buffer.from([
  0xa4, 0xa1, 0xa4, 0xa4, 0xa4, 0xa7, 0xa4, 0xa9, 0xa4, 0xb1, 0xa4, 0xb2, 0xa4, 0xb5, 0xa4, 0xb7,
  0xa4, 0xb8, 0xa4, 0xba, 0xa4, 0xbb, 0xa4, 0xbc, 0xa4, 0xbd, 0xa4, 0xbe, 0xa4, 0xbf, 0xa4, 0xc1,
  0xa4, 0xc3, 0xa4, 0xc5, 0xa4, 0xc7, 0xa4, 0xcb, 0xa4, 0xcc, 0xa4, 0xd0, 0xa4, 0xd1, 0xa4, 0xd3,
  0xa4, 0xbf, 0xa4, 0xc4, 0xa4, 0xd3, 0xa4, 0xc7, 0xa4, 0xcc, 0xb0, 0xa1, 0xb3, 0xaa, 0xb4, 0xd9,
]);

/** Lê `servers.txt`: `<grupo> <canal> <endereço>`, `#` comenta, vazio ignora. */
export function parseList(text) {
  const entries = [];
  text.split(/\r?\n/).forEach((raw, index) => {
    const line = raw.trim();
    if (line === '' || line.startsWith('#')) return;
    const fields = line.split(/\s+/);
    if (fields.length !== 3) {
      throw new Error(`linha ${index + 1}: esperado "<grupo> <canal> <endereço>", veio "${line}"`);
    }
    const group = Number.parseInt(fields[0], 10);
    const channel = Number.parseInt(fields[1], 10);
    if (!Number.isInteger(group) || group < 0 || group >= GROUP_COUNT) {
      throw new Error(`linha ${index + 1}: grupo "${fields[0]}" fora de 0..${GROUP_COUNT - 1}`);
    }
    if (!Number.isInteger(channel) || channel < 0 || channel >= CHANNEL_COUNT) {
      throw new Error(`linha ${index + 1}: canal "${fields[1]}" fora de 0..${CHANNEL_COUNT - 1}`);
    }
    const addr = fields[2];
    if (Buffer.byteLength(addr, 'latin1') >= ADDR_LEN) {
      throw new Error(`linha ${index + 1}: endereço maior que ${ADDR_LEN - 1} bytes`);
    }
    // Mesmos dois limites medidos que o Rust cobra (crates/wyd-launcher/src/serverlist.rs):
    // erro nomeado, nunca truncar em silêncio.
    if (channel === 0) {
      if (!addr.startsWith('http://')) {
        throw new Error(
          `linha ${index + 1}: a célula 0 é a URL de status e precisa de "http://" (veio "${addr}")`,
        );
      }
      if ([...addr].length > STATUS_URL_MAX) {
        throw new Error(
          `linha ${index + 1}: URL de status com ${[...addr].length} caracteres ("${addr}"); ` +
            `o cliente só desenha a lista até ${STATUS_URL_MAX} — use "http://host/serv00.htm" sem porta`,
        );
      }
    } else {
      if (addr.includes(':') || addr.includes('/')) {
        throw new Error(
          `linha ${index + 1}: endereço de canal "${addr}" inválido; o cliente fixa a porta 8281, use um IP por canal`,
        );
      }
      if (entries.some((e) => e.group === group && e.channel !== 0 && e.addr === addr)) {
        throw new Error(
          `grupo ${group}: dois canais com o mesmo endereço "${addr}" — um IP por canal`,
        );
      }
    }
    entries.push({ group, channel, addr });
  });
  return entries;
}

/** Distribui as entradas na grade 10×11×64 e ofusca cada célula. */
export function build(entries) {
  if (entries.length === 0) throw new Error('nenhuma entrada: a serverlist ficaria vazia');
  const bin = Buffer.alloc(BIN_SIZE);
  for (const { group, channel, addr } of entries) {
    if (!addr) throw new Error(`endereço vazio em grupo ${group} canal ${channel}`);
    if (Buffer.byteLength(addr, 'latin1') >= ADDR_LEN) {
      throw new Error(`endereço inválido em grupo ${group} canal ${channel}: "${addr}"`);
    }
    const slot = (group * CHANNEL_COUNT + channel) * ADDR_LEN;
    // strcpy do carregador legado: conteúdo + NUL implícito do buffer zerado.
    bin.write(addr, slot, ADDR_LEN - 1, 'latin1');
  }
  for (let slot = 0; slot < BIN_SIZE; slot += ADDR_LEN) {
    for (let i = 0; i < ADDR_LEN; i += 1) {
      bin[slot + i] = (bin[slot + i] + OBFUSCATION_KEY[ADDR_LEN - 1 - i]) & 0xff;
    }
  }
  return bin;
}

/** Desfaz `build`: devolve a grade `[grupo][canal] = endereço`. */
export function decode(bin) {
  if (bin.length !== BIN_SIZE) {
    throw new Error(`tamanho ${bin.length}, esperado ${BIN_SIZE}`);
  }
  const grid = [];
  for (let group = 0; group < GROUP_COUNT; group += 1) {
    const channels = [];
    for (let channel = 0; channel < CHANNEL_COUNT; channel += 1) {
      const slot = (group * CHANNEL_COUNT + channel) * ADDR_LEN;
      const bytes = Buffer.alloc(ADDR_LEN);
      for (let i = 0; i < ADDR_LEN; i += 1) {
        bytes[i] = (bin[slot + i] - OBFUSCATION_KEY[ADDR_LEN - 1 - i]) & 0xff;
      }
      const end = bytes.indexOf(0);
      channels.push(bytes.toString('latin1', 0, end === -1 ? ADDR_LEN : end));
    }
    grid.push(channels);
  }
  return grid;
}

/** Imprime só as células preenchidas, no formato do `servers.txt`. */
export function filled(grid) {
  const lines = [];
  grid.forEach((channels, group) => {
    channels.forEach((addr, channel) => {
      if (addr !== '') lines.push(`${group} ${channel} ${addr}`);
    });
  });
  return lines;
}

/**
 * Lê `sn.bin` — quem decide QUAIS linhas o server select desenha.
 *
 * Leiaute medido (143 B, cliente 7662):
 * - 11 nomes ASCII de 9 B a partir do offset 0;
 * - 11 contadores `int32` little-endian no offset 99;
 * - slot 0 é o cabeçalho reservado (`Reserved` com o `R` sobrescrito por `0xd6`)
 *   e `contadores[0]` é quantas linhas de canal o cliente tenta desenhar;
 * - slots 1..10 são servidores: `contadores[slot] - 1` é o grupo do `.bin`.
 *
 * A inversão é do cliente, não do arquivo: em `0x46d6ce`–`0x46dc76` a lista
 * percorre as células `i = 1..N` do grupo exibido, mas rotula a linha com o
 * índice `last - i` quando esse grupo é o último preenchido (`0x46db0a`–
 * `0x46db83`), consumindo os nomes em ordem decrescente (`[2]`, `[1]`, `[0]`).
 * A linha que cai no slot reservado fica com contador `-1` e o cliente desenha
 * `Inexistente` (`0x46ddd0`).
 */
export function parseSn(buf) {
  const NAME_LEN = 9;
  const SLOTS = 11;
  const COUNTER_OFFSET = 99;
  const names = [];
  // Nomes começam no offset 0 com passo de 9 B: o primeiro caractere do primeiro
  // nome é o byte opaco `0xd6` (o `R` de `Reserved` sobrescrito), não um cabeçalho.
  for (let slot = 0; slot < SLOTS; slot += 1) {
    const start = slot * NAME_LEN;
    names.push(buf.subarray(start, start + NAME_LEN).toString('latin1').split('\0')[0]);
  }
  const counters = [];
  for (let slot = 0; slot < SLOTS; slot += 1) {
    counters.push(buf.readInt32LE(COUNTER_OFFSET + slot * 4));
  }
  return { firstByte: buf[0], names, counters };
}

/** Linhas de canal que o cliente tenta desenhar: `contadores[0] - 1` (o `-1` é o cabeçalho). */
export function channelRowCount(sn) {
  return Math.max(0, (sn.counters[0] ?? 0) - 1);
}

/**
 * Nomes de servidor da identidade única (`docs/arquitetura/identidade.tokens`).
 *
 * O gerador **não** tem nome próprio: a chave `servidor.<slot>` é a única fonte e é a mesma
 * que o `wyd-launcher` (Rust) embute para escrever o `sn.bin`. Divergência entre os dois lados
 * quebra `crates/wyd-launcher/tests/identidade.rs`.
 */
export function identidadeServers(file = 'docs/arquitetura/identidade.tokens') {
  const nomes = {};
  if (!fs.existsSync(file)) return nomes;
  for (const raw of fs.readFileSync(file, 'utf8').split(/\r?\n/)) {
    const linha = raw.trim();
    if (!linha || linha.startsWith('#')) continue;
    const [chave, ...resto] = linha.split('=');
    if (!chave || resto.length === 0) continue;
    const achado = chave.trim().match(/^servidor\.(\d+)$/);
    if (achado) nomes[Number(achado[1])] = resto.join('=').trim();
  }
  return nomes;
}

/**
 * Gera `sn.bin` no leiaute medido: slot 0 preservado como cabeçalho reservado
 * (nome e `0xd6` iniciais, contador = canais + 1) e um slot por grupo usado, com
 * `contador = grupo + 1` — é o que faz aquele slot desenhar o grupo no `.bin`.
 *
 * O nome de cada slot vem de `names` (identidade única); slot com grupo e sem nome é erro,
 * nunca um `NextGen` inventado. Sem o slot 0 intacto o cliente monta `"Canal N Grupp"`, e
 * com o contador do cabeçalho errado sobra `Inexistente` no fim da lista.
 */
export function buildSn(groups, previous, names = {}) {
  const SLOTS = 11;
  const NAME_LEN = 9;
  const COUNTER_OFFSET = 99;
  const buf = Buffer.alloc(143, 0);
  // Slot 0 é o cabeçalho reservado: primeiro byte medido (`0xd6`) e o resto de `Reserved`.
  buf[0] = previous?.firstByte ?? 0xd6;
  buf.write('eserved', 1, 'latin1');
  const canais = previous ? channelRowCount(previous) : 2;
  buf.writeInt32LE(canais + 1, COUNTER_OFFSET);
  for (const group of [...groups].sort((a, b) => a - b)) {
    // Slot do servidor = grupo do `.bin`; o contador é `grupo + 1` (medido).
    const slot = group;
    if (slot < 1 || slot >= SLOTS) continue;
    const nome = names[slot];
    if (!nome) {
      throw new Error(`identidade sem servidor.${slot} (docs/arquitetura/identidade.tokens)`);
    }
    if (Buffer.byteLength(nome, 'latin1') > NAME_LEN - 1) {
      throw new Error(`servidor.${slot} "${nome}" não cabe em 8 caracteres`);
    }
    buf.write(nome, slot * NAME_LEN, 'latin1');
    buf.writeInt32LE(group + 1, COUNTER_OFFSET + slot * 4);
  }
  return buf;
}

/** Mapa slot → grupo desenhado → canais (só as linhas que o cliente tenta desenhar). */
export function describeSn(sn, grid) {
  const linhas = channelRowCount(sn);
  return sn.counters.map((counter, slot) => ({
    slot,
    name: sn.names[slot],
    counter,
    group: counter - 1,
    channels:
      counter >= 1 && grid && Array.isArray(grid[counter - 1])
        ? grid[counter - 1]
            .map((addr, channel) => ({ channel, addr }))
            .filter(({ channel, addr }) => channel >= 1 && channel <= linhas && addr !== '')
        : [],
  }));
}

/**
 * O que o server select desenha: uma linha por servidor (slots 1..10 com contador
 * > 0). O slot 0 é o cabeçalho reservado, não um servidor; é ele que aparece no fim
 * da lista quando o cliente inverte os índices (§3 de `docs/cliente/formatos.md`).
 */
export function renderSelect(sn, grid) {
  return describeSn(sn, grid)
    .filter((row) => row.slot > 0 && row.counter > 0)
    .map((row) => ({ server: row.name, group: row.group, channels: row.channels }));
}

/**
 * Confere os arquivos **instalados** contra o esperado (uma linha de servidor,
 * canais 1..N na ordem, nenhum host vazio). Devolve a lista de problemas — vazia
 * significa que o que o cliente vai desenhar é exatamente o esperado.
 */
/**
 * Hosts esperados, lidos do registro que gera o `.bin` (`serverlist-servers.txt`), na ordem dos
 * canais do grupo desenhado. Endereço de canal é **dado de ambiente** (o host muda com a rede):
 * fixar IP aqui só produz `FALHA` falso quando a máquina troca de endereço.
 */
export function expectedHosts(sn, grid, registryPath) {
  if (!registryPath || !fs.existsSync(registryPath)) return [];
  const grupo = renderSelect(sn, grid)[0]?.group;
  if (grupo === undefined) return [];
  return parseList(fs.readFileSync(registryPath, 'utf8'))
    .filter((e) => e.group === grupo && e.channel > 0)
    .sort((a, b) => a.channel - b.channel)
    .map((e) => e.addr);
}

export function checkInstalled(sn, grid, { server = null, hosts = [] } = {}) {
  const problemas = [];
  const linhas = renderSelect(sn, grid);
  if (linhas.length !== 1) {
    problemas.push(`esperada 1 linha de servidor, vieram ${linhas.length}`);
  }
  const linha = linhas[0];
  if (linha) {
    if (server && linha.server !== server) {
      problemas.push(`nome do servidor "${linha.server}" != "${server}"`);
    }
    const canais = linha.channels.map((c) => c.channel);
    const esperado = hosts.map((_, i) => i + 1);
    if (canais.join(',') !== esperado.join(',')) {
      problemas.push(`canais desenhados [${canais.join(', ')}] != [${esperado.join(', ')}]`);
    }
    hosts.forEach((host, i) => {
      const canal = linha.channels.find((c) => c.channel === i + 1);
      if (!canal) problemas.push(`canal ${i + 1} ausente`);
      else if (canal.addr !== host) problemas.push(`canal ${i + 1} = "${canal.addr}" != "${host}"`);
    });
    const vazios = linha.channels.filter((c) => c.addr === '');
    if (vazios.length > 0) {
      problemas.push(`${vazios.length} canal(is) sem host (o cliente desenha Inexistente)`);
    }
  }
  return { problemas, linhas };
}

const isMain = process.argv[1] && path.resolve(process.argv[1]) === path.resolve(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1'));

if (isMain) {
  const args = process.argv.slice(2);
  const flag = (name, fallback = null) => {
    const index = args.indexOf(name);
    return index === -1 ? fallback : args[index + 1];
  };
  const printPath = flag('--print');
  const snPath = flag('--sn');
  const checkDir = flag('--check');
  try {
    if (checkDir) {
      // Caminho real: lê os arquivos que estão instalados no diretório do cliente.
      const sn = parseSn(fs.readFileSync(path.join(checkDir, 'sn.bin')));
      const grid = decode(fs.readFileSync(path.join(checkDir, 'serverlist.bin')));
      const hosts = flag('--hosts', '').split(',').filter(Boolean);
      const nomes = identidadeServers();
      const primeira = renderSelect(sn, grid)[0];
      const { problemas, linhas } = checkInstalled(sn, grid, {
        server: flag('--server') ?? (primeira ? (nomes[primeira.group] ?? null) : null),
        hosts:
          hosts.length > 0
            ? hosts
            : expectedHosts(sn, grid, flag('--registry', 'tools/client/serverlist-servers.txt')),
      });
      for (const linha of linhas) {
        console.log(`servidor "${linha.server}" (grupo ${linha.group})`);
        for (const canal of linha.channels) {
          console.log(`  canal ${canal.channel} → ${canal.addr}`);
        }
      }
      if (problemas.length > 0) {
        for (const problema of problemas) console.error(`FALHA ${problema}`);
        process.exit(1);
      }
      console.log(`ok: ${checkDir} desenha 1 servidor "${linhas[0].server}" com ${linhas[0].channels.length} canais, nenhum Inexistente`);
      process.exit(0);
    }
    if (snPath) {
      const sn = parseSn(fs.readFileSync(snPath));
      const binPath = flag('--bin', 'client/serverlist.bin');
      const grid = fs.existsSync(binPath) ? decode(fs.readFileSync(binPath)) : null;
      console.log(`${snPath}: primeiro byte 0x${sn.firstByte.toString(16)} (UNVERIFIED)`);
      for (const row of describeSn(sn, grid)) {
        const canais = row.channels.map((c) => `${c.channel}=${c.addr}`).join(' ') || '(sem host — Inexistente)';
        console.log(
          `slot ${row.slot}: nome="${row.name}" contador=${row.counter} → grupo ${row.group < 0 ? '—' : row.group} · ${canais}`,
        );
      }
      console.log('lembrete: o cliente desenha os slots na ordem 0..10 (o grupo vem do contador, não do slot).');
      process.exit(0);
    }
    if (printPath) {
      const grid = decode(fs.readFileSync(printPath));
      for (const line of filled(grid)) console.log(line);
      process.exit(0);
    }
    const input = flag('--in', 'tools/client/serverlist-servers.txt');
    const output = flag('--out', 'client/serverlist.bin');
    const entries = parseList(fs.readFileSync(input, 'utf8'));
    const bin = build(entries);
    fs.writeFileSync(output, bin);
    const snOut = flag('--sn-out');
    if (snOut) {
      const grupos = [...new Set(entries.map((e) => e.group))].sort((a, b) => a - b);
      const anterior = fs.existsSync(snOut) ? parseSn(fs.readFileSync(snOut)) : null;
      fs.writeFileSync(snOut, buildSn(grupos, anterior, identidadeServers()));
      console.log(`${snOut}: ${grupos.length} servidor(es) → grupos ${grupos.join(', ')}`);
    }
    console.log(`${output}: ${entries.length} entradas, ${bin.length} bytes`);
    for (const { group, channel, addr } of entries) {
      console.log(`  grupo ${group} canal ${channel} → ${addr}`);
    }
  } catch (error) {
    console.error('erro:', error.message);
    process.exit(1);
  }
}
