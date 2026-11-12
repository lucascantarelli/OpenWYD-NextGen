# Captura ao vivo de `STRUCT_SELCHAR` / `STRUCT_MOB` — runbook

O que falta para a tela de personagens mostrar dados reais não é decisão, é **byte**: os 14 campos
de `STRUCT_SELCHAR` e os 6 de `STRUCT_MOB` (tabela na issue
`FASE-08-6` (issue fora do site: `roadmap-rust/issues/FASE-08-6-cliente-reverso-dinamico.md`) continuam sem offset
confirmado. Este runbook é o caminho curto para produzi-los, com os instrumentos que já existem.

Pré-requisitos: stack de pé (`make stack-up`), cliente `client/WYD.exe` apontando para o proxy,
conta de teste com senha e **senha numérica** cadastradas.

## 1. Ferramentas

| Peça | Papel |
|---|---|
| `tools/client/captura-live.mjs --alvo <ip:porta>` | proxy CPSock entre o cliente e o canal real; grava cada frame com direção |
| `tools/client/captura-live.mjs --from <json> --fixtures crates/wyd-protocol/tests/fixtures/cliente/real` | converte a captura em fixtures (corpo do PIN mascarado) |
| `tools/client/diff-captura.mjs --a <a.json> --b <b.json>` | diff de bytes entre duas capturas e candidatos a offset |
| `tools/client/serverlist.mjs` | aponta/restaura o `serverlist.bin` do cliente |

## 2. Apontar o cliente para o proxy (e restaurar)

```bash
cp client/serverlist.bin tmp/serverlist.bin.bak
node tools/client/serverlist.mjs --print client/serverlist.bin | head -5   # estado atual
# edite tools/client/serverlist-servers.txt trocando o host do canal por 127.0.0.1
node tools/client/serverlist.mjs --in tools/client/serverlist-servers.txt --out client/serverlist.bin
```

Ao terminar: `cp tmp/serverlist.bin.bak client/serverlist.bin`. Nunca versione o `serverlist.bin`
alterado (`client/` é ignorado no git).

## 3. Capturar o estado A

```bash
node tools/client/captura-live.mjs --alvo 192.168.100.4:8281 \
  --host 127.0.0.1 --port 8281 --timeout 300000 --out tmp/captura-selchar-a.json
```

Na janela do cliente: escolher canal → login (senha) → **senha numérica** → tela de personagem →
selecionar um personagem. O proxy imprime cada frame; o artefato fica em `tmp/`.

O que precisa aparecer no log do proxy: `#n c2s 116` (login), `#n c2s 32` (senha numérica),
`#n s2c 2008` (`_MSG_CNFAccountLogin`, corpo de `STRUCT_SELCHAR`) e, ao entrar, `#n s2c <tamanho>`
(`_MSG_CNFCharacterLogin`, corpo de `STRUCT_MOB`). Se `0x010A` não aparecer, o servidor ainda está
respondendo só cabeçalho — é o item 1.12 do mapa de funcionalidades.

## 4. Variar **um** campo e capturar o estado B

```bash
docker compose exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "UPDATE characters SET level = level + 1 WHERE name = 'QA00000001'"
```

Repita o passo 3 em `tmp/captura-selchar-b.json`. Uma variação por captura: dois campos mudando
juntos tornam o diff ambíguo. Ordem sugerida (cada uma isola um campo da tabela da issue):

1. `level` → campos 3 e (por consequência) HP/MP do `STRUCT_SCORE`;
2. `exp` → campo 4 (offset e largura de 32 bits);
3. `hp`/`mp` → campo 5 (atuais × máximos);
4. `class` → campo 2 (byte × short e índice da classe);
5. `citizen`/canal → campo 7 (derivação `ServerIndex + 1`);
6. nome do personagem → campo 1 (encoding e limite de 16 B);
7. guilda → campo 8 (par id/nome);
8. `pos_x`/`pos_y` → campo 9 no selchar × campo 6 do mob.

## 5. Diff e correlação

```bash
node tools/client/diff-captura.mjs --a tmp/captura-selchar-a.json --b tmp/captura-selchar-b.json \
  --json tmp/diff-selchar.json
```

O relatório lista os frames pareados por direção/opcode/tamanho e, em cada par, as **faixas de bytes
que mudaram** com o valor dos dois lados. A regra de leitura:

- faixa que muda em **um** frame e cujo valor bate com a variação do banco → **candidato forte**;
- faixa que muda em vários frames ou não acompanha o valor → descartar (pode ser timestamp/tick);
- nada mudou → o campo não está naquele pacote (registrar "ausente neste build", não "não existe").

## 6. Congelar o achado

```bash
node tools/client/captura-live.mjs --from tmp/captura-selchar-b.json \
  --fixtures crates/wyd-protocol/tests/fixtures/cliente/real --canal 2
node tools/client/captura.mjs --check
```

Regras de higiene (`docs/cliente/depuracao.md` §5), reforçadas aqui:

- a conversão **zera o corpo** de `0x0FDE`/`0x0FDF` (senha numérica) e marca `mascarado: true`;
- conta/senha do login continuam por sua conta: a fixture só é versionada depois de trocar os
  campos por `GOLDEN`/zeros;
- `.pcap` e dumps nunca entram no repositório; a captura crua fica em `tmp/` (ignorado).

## 7. Promover para tipo Rust

Só depois do diff: abrir a issue de corpo de pacote, criar o tipo em
`crates/wyd-protocol/src/payload/character.rs` com os offsets provados, teste de ida-e-volta e
golden em `crates/wyd-protocol/tests/fixtures/cliente/`. Campo sem diff continua `UNVERIFIED` — invariante 5 do `AGENTS.md`.

## Layout provisório `rust-v1` (o que já é verdade hoje)

`crates/wyd-server/src/personagens.rs` monta o corpo de `CNFAccountLogin` (4 × 144 B +
`count`/versão) e de `CNFCharacterLogin` (`STRUCT_MOB`, 256 B) a partir do banco. Os offsets
são **nossos** (`rust-v1`) e estão em constantes no topo do módulo; os do cliente 7662 seguem
UNVERIFIED. Quando o diff do cliente aparecer, só as constantes mudam.

Prova automatizada (roda sem cliente Windows, exige Postgres da stack):

```bash
cargo test -p wyd-server --test sistema_selchar
# grava tmp/captura/selchar-a.json (level 41) e selchar-b.json (level 42)
node tools/client/diff-captura.mjs --a tmp/captura/selchar-a.json --b tmp/captura/selchar-b.json --json tmp/captura/diff-selchar.json
```

O diff é a correlação real: a faixa que muda entre A e B acompanha o `level` no banco.
O único passo que depende da máquina do usuário é o `captura-live.mjs` com o cliente 7662
apontado para `127.0.0.1` — o resto do roteiro roda aqui.

### Offset → campo (`rust-v1`, provisório)

| Offset | Campo | Fonte real |
|--------|-------|------------|
| +0 | `name[16]` | `characters.name` |
| +16 / +17 | `class` / `citizen` | `characters.class` / canal + 1 |
| +18 / +20 | `level` / `exp` | `characters.level` / `characters.exp` |
| +24 / +28 | `hp` / `hp_max` | `characters.hp` (máximo espelha o atual: coluna não existe) |
| +32 / +36 | `mp` / `mp_max` | `characters.mp` (idem) |
| +40 | `gold` | sem coluna → 0 |
| +44 / +46 / +47 | `guild_id` / `slot` / `tier` | guilda 0 (sem tabela ligada) / `characters.slot` / 0 |
| +48 / +52 / +54 | `pvp` / `pos_x` / `pos_y` | pvp 0 / `characters.pos_x` / `characters.pos_y` |
| +60 / +68 | `char_id` / `account_id` | `characters.id` / `characters.account_id` |

### O que o primeiro diff (A × B) mostrou

Variação feita no banco: `level` 41 → 42 (o resto do personagem idêntico; o `char_id` muda porque
o estado B é uma linha nova, não um `UPDATE`). O artefato `tmp/captura/diff-selchar.json` traz:

```
S2C 0x010a 592B
  +34 (0x22) 1B  a=29  b=2a     # level (41 → 42)
  +76 (0x4c) 1B  a=10  b=11     # char_id (linha nova)
```

Lido contra o `rust-v1`: `+34` = 12 (cabeçalho) + 4 (`count`/versão) + 18 (`level` no registro);
`+76` = 12 + 4 + 60 (`char_id`). O diff confirma a álgebra do layout, mas **não** confirma os
offsets do 7662 — isso só o frame do cliente real faz.
