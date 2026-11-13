# 16 — Estratégia de testes e regressão

> Base: 810 testes medidos, 64 arquivos em `crates/*/tests`, `Makefile` (`make check`,
> `make sistema-test`), `docs/arquitetura/testes.md`, `docs/rust/02-execucao.md` e
> `docs/rust/evidencias/*`. **Nenhum teste foi executado nesta auditoria.**

## 1. Inventário atual

| Camada | Onde | Quantidade | O que prova |
|---|---|---:|---|
| Unitário | dentro de `src` | maioria dos 810 | regra pura (core, sim, protocol) |
| Contrato | `tests/contrato/` | admin, db, launcher | API/HTTP/JSON, migrations, manifesto |
| Integração | `tests/integracao/` | core, net, server | determinismo, autoridade, flood, campo |
| Sistema | `tests/sistema/` | server (28 arquivos) | fio loopback + banco real |
| Golden | `tests/golden/` | protocol, launcher, sim | bytes e fórmulas |
| E2E | `wyd-portal` (Playwright) | 6/6 em três execuções | portal ponta a ponta |
| Carga | `wyd-bots` + `QA-10/11` | 🚧 | ainda sem relatório de 200 jogadores |

Distribuição por crate: `wyd-core` 165 · `wyd-server` 151 · `wyd-launcher` 140 · `wyd-sim` 120 ·
`wyd-admin` 112 · `wyd-protocol` 58 · `wyd-db` 33 · `wyd-net` 21 · `wyd-bots` 10.

**Assimetria crítica:** o eixo **S→C** tem 3 arquivos de teste e 5 fixtures. Onde o conhecimento
falta, o teste também falta — não é coincidência, é causa.

## 2. Camadas recomendadas (do mais barato ao mais caro)

```
1. propriedade/unitário      → regra pura (sem IO, milissegundos)
2. golden de bytes           → serialização/parse (frame real ou sintético declarado)
3. contrato                  → API/JSON/migrations (sem regra de jogo)
4. integração com duplo       → engine + tick + cena (sem socket)
5. sistema loopback           → fio + banco (WYD_TEST_REQUIRED=1, zero SKIP)
6. captura do cliente         → prova de compatibilidade real (CLIENT-06)
7. carga/stress               → 200 bots, P99, rejeições
```

## 3. Testes comparativos legado × Rust — o que é possível

| Abordagem | Viabilidade | Comentário |
|---|---|---|
| Rodar o C++/Go e comparar saída | **baixa** | stack removida em 2026-11; exigiria checkout de `origin/main` + ambiente |
| Comparar contra `game-rules.md` citado | **média** | transformar `arquivo:linha` em caso de teste no `wyd-sim` |
| Comparar contra captura do cliente | **alta** | é o único oráculo de wire real |
| Comparar contra o cliente do Global | **baixa** | structs provavelmente diferentes (`DIV-12`) |

**Decisão recomendada:** o oráculo de paridade é **(a)** fórmula citada do C++ + **(b)** frame
capturado do 7662. Não existe “rodar os dois e diffar” neste projeto sem reconstruir a stack
antiga — e isso não vale o custo.

## 4. Cenários por sistema (o que existe × o que falta)

| Sistema | Normal | Erro | Limite | Concorrência | Desconexão | Recuperação | Abuso | Integração |
|---|---|---|---|---|---|---|---|---|
| Autenticação | ✅ | ✅ | ✅ | 🚧 | ✅ | ✅ | ✅ | ✅ |
| Personagem | ✅ | ✅ | 🚧 | 🚧 | ✅ | 🚧 | ⬜ | ✅ |
| Movimento | 🚧 | ⬜ | ⬜ | ⬜ | 🚧 | ⬜ | ⬜ | ⬜ |
| Combate | ✅ | ✅ | 🚧 | ⬜ | 🚧 | ⬜ | ✅ | ✅ |
| Inventário | ✅ | ✅ | ✅ | ⬜ | 🚧 | 🚧 | ✅ | 🚧 |
| Refino | ✅ | ✅ | ✅ | ⬜ | ⬜ | ✅ | ✅ | ✅ |
| Drop | ✅ | ✅ | ✅ | ⬜ | — | ⬜ | ⬜ | ✅ |
| Trade | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Loja | 🚧 | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | 🚧 |
| Quest | ✅ | 🚧 | 🚧 | — | ⬜ | ⬜ | ⬜ | ✅ |
| Chat | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Guilda | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| Portal | ✅ | ✅ | ✅ | 🚧 | — | ✅ | ✅ | ✅ |
| Launcher | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | ✅ |

Legenda: ✅ existe · 🚧 parcial · ⬜ ausente · — não se aplica.
Fonte: `docs/rust/08-checklist-funcionalidades.md` + `docs/arquitetura/testes.md`.

## 5. Testes de protocolo — o que criar primeiro

| Teste | Entrada | Saída esperada | Onde |
|---|---|---|---|
| Roundtrip de header | 12 B arbitrários válidos | ofuscação/desofuscação idêntica | `wyd-protocol` (existe) |
| `Size` fora de `[12, 8192]` | frame curto/longo | erro tipado, sem panic | `wyd-net` (existe) |
| Opcode não portado | `Trade` | `PayloadError::Unported` | `flood.rs` (existe) |
| Cena delta | entrar/sair NPC | `CreateMob`/`RemoveMob` corretos | `world::tests` (existe) |
| **Struct S→C** | byte capturado do 7662 | campos esperados | **criar após captura** |
| **CreateItem 30 B packed** | frame mínimo | offsets `+0x0C…+0x1D` | **criar** |
| **GetItem 28 B** | `DestType=1`, `DestPos` | `CNFGetItem` correspondente | **criar** |
| **Trade 156/154** | candidato | decisão binária | **criar após captura** |
| Fuzz de payload | bytes aleatórios | nunca panic | parcial (`flood`) |

## 6. Regressão

| Regra | Estado |
|---|---|
| Toda correção nasce com teste que falha antes | exigido em `AGENTS.md` §2.8 |
| Anti-skip no CI | ✅ (`DEVOPS-07`) |
| `WYD_TEST_REQUIRED=1` → zero SKIP | ✅ (`make sistema-test`) |
| Cobertura como baseline, não meta | ✅ (`DEVOPS-08`) |
| Golden de bytes para tabela congelada | ✅ (`opcode_table`, `serverlist`) |
| Teste de tabela congelada impede “correção” silenciosa | ✅ — é o que segura `DIV-02` |

## 7. Carga e stress

| Item | Estado | Meta |
|---|---|---|
| 200 jogadores em loopback | 🚧 `QA-11` | P99 e rejeições medidas |
| Bot de combate/campanha | ✅ `wyd-bots` | 4 raças, 4 etapas |
| Flood de pacotes | ✅ `flood.rs` | rejeita sem panic |
| Carga de banco | 🚧 `QA-10` | integridade sob escrita concorrente |
| Concorrência de save | ⬜ | dois saves da mesma conta |
| Miri/tarpaulin | 🚧 `QA-09` | memória/cobertura |

## 8. O que **não** dá para testar aqui

| Item | Por quê |
|---|---|
| Render do cliente 7662 | exige GUI/VM; tentativa de 2026-11-12 não recebeu input |
| Captura de pacote real | exige cliente aberto + servidor emitindo |
| Cliente em outra resolução/GPU | depende da máquina do usuário |
| Stack antiga Go/C++ | removida; só via `origin/main` |
| Anti-cheat/jlock | não instrumentar sem autorização explícita |

## 9. Recomendação final

1. **Não aumentar a contagem de testes por aumentar**: o gargalo é o eixo S→C.
2. Criar **fixtures de captura** com nome, build e hash — sem isso, o golden vira folclore.
3. Um teste de **cadência de ataque** (`GAME-20`) vale mais que 50 testes de unidade novos.
4. Um teste de **dois saves concorrentes** fecha o maior risco silencioso de integridade.
5. Manter o gate anti-skip: é a única coisa que impede “verde” sem prova.
