# AGENTS.md — Dolorestec WYD (Rust)

> Instruções **sempre ativas** e curtas. O detalhe operacional vive nos docs e nas skills
> linkados abaixo; não duplique regra aqui. Fonte do planejamento: `roadmap-rust/index.md`.

## 0. Comece aqui

- O repositório é o servidor **WYD2 em Rust** (Hot Path), o portal administrativo Next.js
  (Cold Path) e o launcher; o cliente Windows 7662 é artefato externo.
- **100% Rust**: a stack Go/C++ foi removida em 2026-11. Não use build/testes, caminhos ou
  roadmap da stack antiga como fonte vigente.
- Leia, nesta ordem: esta página → a issue em `roadmap-rust/issues/` → **uma** skill principal
  pelo roteador `.agents/skills/README.md` → o doc normativo da área.
- Trabalhe em `develop`; commit/push só quando o usuário pedir. Não reinicie serviços durante
  um teste humano em andamento: espere o teste terminar.

## 1. Mapa mínimo

| Caminho | Papel |
|---|---|
| `crates/wyd-protocol/` | wire CPSock, offsets explícitos, parsers e goldens |
| `crates/wyd-core/` | engine Hot Path: tick, ECS/DOD, combate, visibilidade |
| `crates/wyd-net/` | transporte async, sessão, rate-limit, hardening |
| `crates/wyd-db/` | persistência, migrations, `delivery_queue` |
| `crates/wyd-admin/` | control plane Cold Path e binário `wyd-gateway` |
| `crates/wyd-server/` | composição do monólito; um processo por canal |
| `crates/wyd-sim/` | fórmulas e simulações determinísticas |
| `crates/wyd-launcher/` + `apps/launcher-ui/` | distribuição/patch e janela Tauri |
| `crates/wyd-bots/` | bots de QA/carga; cliente de protocolo sem GUI |
| `wyd-portal/` | BFF Next.js sobre `wyd-admin` (nunca SQL/estado vivo) |
| `docs/` | norma técnica, cliente, operação e conhecimento do jogo |
| `roadmap-rust/` | fases, issues ativas, changelog e bugs |
| `client/` | cliente proprietário, **ignorado no git**; não versionar |

## 2. Invariantes

1. **Hot Path sem I/O no tick.** Tick, física, combate e visibilidade no mesmo espaço de
   memória (`wyd-core` + composição em `wyd-server`); nada de HTTP/SQL/gRPC, mutex global ou
   alocação quente no loop. Persistência entra por traits/canais async fora do tick.
2. **Wire por offset explícito.** HEADER de 12 bytes LE, `Size` em [12, 8192],
   `INITCODE = 0x1F11F311`, transformação a partir do offset 4 com `pKeyWord[512]`.
   `repr(C)` + offsets provados por teste; nunca confie em `repr(Rust)`.
3. **Servidor é autoridade.** O cliente não dita posição, dano, cooldown ou inventário.
   Pacote autoritativo de cena usa `HEADER.ID = ESCENE_FIELD` (30000).
4. **UNVERIFIED não vira regra.** Comportamento não capturado: tolerar, logar e seguir pelo
   caminho conservador. Não invente layout de pacote nem crie tipo/gate a partir de hipótese.
5. **Segredos nunca vazam.** Senhas/PINs com hash resistente (argon2id); DSN, tokens,
   `SESSION_SECRET` e dumps de conta não entram em log, git ou assets. `client/` não é versionado.
6. **A web não escreve estado vivo.** Grants passam por `delivery_queue`; o portal é BFF,
   autentica staff, deriva `moderator_id` da sessão e audita toda ação administrativa.
7. **Estrutura antes de código.** A árvore normativa está em
   `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002). Arquivo novo fora dela, mudança de
   fronteira, threading ou dependência pesada exige ADR em `docs/arquitetura/adr/`.
8. **Qualidade é gate.** `cargo fmt --check`, `clippy -D warnings`, build e testes limpos;
   entrada de rede nunca causa `panic!`; todo comportamento novo nasce com teste.

## 3. Comandos

```bash
make check                 # gate único: fmt + clippy + build + test + migrations + issues + tools
make test-all              # unitários + integração + doc-tests + benches (espelho do CI)
make bench                 # benchmarks quando a issue exigir números
make stack-config          # valida compose base+prod sem subir
make stack-up              # db + gateway + 2 canais + portal + monitoramento
make portal-test           # tsc --noEmit + vitest
make portal-build          # next build
```

Sem `make` no Windows/Git Bash, rode os comandos `cargo` equivalentes e registre na evidência.
Diagnóstico de stack, portas e banco: skill `wyd-server-ops` + `docs/rust/05-stack-docker.md`.

## 4. Onde está a verdade

| Preciso de… | Leia |
|---|---|
| Estrutura, nomenclatura, SOLID e fronteiras | `docs/arquitetura/estrutura-e-padroes.md` + `software-architecture` |
| Protocolo CPSock, opcodes e payloads | `docs/arquitetura/protocolo.md` + `crates/wyd-protocol/` |
| Banco, ER, migrations e seeds | `docs/arquitetura/modelo-banco.md` + `crates/wyd-db/` |
| Cliente: inventário, RE, pacotes e depuração | `docs/cliente/` + `client/` (local) |
| Execução, gates, DoD e revisão | `docs/rust/02-execucao.md` |
| Stack e operação Docker | `docs/rust/05-stack-docker.md` + `wyd-server-ops` |
| Dependências e toolchain | `docs/rust/01-dependencias.md` + `wyd-rust-toolchain` |
| Progresso, evidências e histórico | `docs/rust/03-progresso.md` + `docs/rust/evidencias/README.md` + `roadmap-rust/changelog.md` |
| Estado e prioridade das issues | `roadmap-rust/index.md` |
| Lore, classes, itens e sistemas | `docs/wyd/` |
| Portal (BFF, telas, sessão) | `wyd-portal/README.md` + `wyd-web-admin-ui` |

O que não está documentado é **hipótese/UNVERIFIED**: não vira tipo, gate ou regra.

## 5. Fluxo de entrega

1. **Análise**: leia a issue, o aceite e os docs/skills citados; confirme o estado real no código.
2. **Design**: liste arquivos e fronteiras; ADR se a mudança for estrutural.
3. **Código mínimo**: uma issue por vez, sem refatoração fora do escopo.
4. **Verificação**: `make check`; testes específicos quando a área exigir; link gate.
5. **Docs**: atualize a issue, `roadmap-rust/index.md`, `docs/rust/03-progresso.md` e o doc
   invalidado na mesma entrega. Ao concluir, remova o arquivo da issue e registre no
   `roadmap-rust/changelog.md` com evidência.

Detalhe do ciclo, branch e Definition of Done: `docs/rust/02-execucao.md` §1–§4.

## 6. Skills

O roteador é `.agents/skills/README.md`: ele mapeia cada tarefa para **uma** skill principal e
lista as transversais de apoio. Não replique o catálogo aqui. As skills são sob demanda
(nome/descrição na descoberta; corpo completo só quando carregadas), por isso o custo fixo é
este `AGENTS.md`, não o conjunto de skills.

Histórico Go→Rust, fusões e justificativa de remoções: `docs/manual-técnico/skills.md`.

## 7. Limites

- Não versione `client/`, `target/`, dumps, `.pcap`, segredos, `serverlist.bin` com IP privado
  ou `.freebuff/`.
- Não faça `docker compose down -v`, `DROP`/`TRUNCATE` ou reset de volume sem pedido explícito.
- Não publique portas do banco fora do loopback em desenvolvimento.
- Não edite `crates/` para “consertar” documentação, nem promova hipótese a contrato.
