# ADR-0011 — `wyd-sim` organizado por sistema de jogo, com unidades e ordem explícitas

- **Estado:** proposta (árvore já materializada a pedido do dono em 2026-11-08; aceite formal pendente)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (a missão "não escreva tudo na raiz, planeje a matemática antes das pastas" abriu a decisão)
- **Governa:** `roadmap-rust/issues/FASE-10-1-equacoes.md`, `FASE-10-2-velocidade.md`, `FASE-10-3-laboratorio.md`, `FASE-03-6-ia-npc.md`
- **Depende de:** ADR-0002 (estrutura de módulos), ADR-0006 (dependências externas diferidas), ADR-0009 (conteúdo no banco), ADR-0010 (IA de NPC)

## Contexto

`crates/wyd-sim` nasceu com três arquivos: `lib.rs` com `damage_after_absorb`,
`formula/damage.rs` e `formula/refine.rs`. A própria `lib.rs` anunciava `scenario/` e
`report/` "na FASE-10-3", e a árvore alvo de `estrutura-e-padroes.md` §14 listava
`formula/{damage,refine,speed,crit}.rs` — ou seja, o plano dizia quais arquivos existiriam,
mas não qual **matemática** cada um representa. O sintoma que o dono apontou ("tudo no
`rott`") é consequência disso: sem sistema definido, todo número novo cai no arquivo que já
está aberto.

Três lacunas de fonte tornam a decisão urgente:

1. `pre-release-rust.md` §10.1 manda "documentar e codificar as equações exatas", mas a ordem
   de mitigação (defesa subtrativa × multiplicativa, absorção antes ou depois do crítico) é
   **UNVERIFIED** segundo `docs/wyd/skills.md` §5 — não existe fonte pública.
2. `docs/wyd/progressao.md` §7 declara que a curva de EXP do build 7662 **não é publicada**;
   qualquer constante no código seria progressão inventada.
3. A dúvida de bots (`ADR-0010`) criou um módulo de política em `wyd-sim/src/bot/` que ainda
   não tem contraparte matemática: a tabela de ameaça não existe em lugar nenhum.

## Decisão

Adotamos **um arquivo por sistema de jogo** em `crates/wyd-sim/src/formula/`, com unidades
newtype, constantes publicadas onde existem, e **tudo que a fonte não prova como entrada de
dado** (nunca constante escondida), porque sem essa separação o laboratório deixa de
distinguir fato de hipótese e passa a "provar" número inventado.

A árvore decorre da matemática, não o contrário:

```
crates/wyd-sim/src/
├── lib.rs          # fachada: declara e reexporta
├── error.rs        # SimError (um erro por fronteira)
├── units.rs        # Bps, Tenths, mul_bps (truncamento único)
├── roll/           # Die + XorShift64 (determinismo com seed explícita)
├── formula/        # um sistema por arquivo
├── scenario/       # ordem do tick e do golpe
└── report/         # saída comparável em texto
```

Regras que passam a valer:

1. **Nenhuma fórmula nova em `lib.rs`, `mod.rs` ou em arquivo de outro sistema.** Fórmula nova
   é módulo novo (OCP); `mod.rs` só declara e reexporta.
2. **Percentual é `Bps` (inteiro).** `f32`/`f64` proibido em `formula/`; relatório é texto.
3. **Arredondamento é um só:** truncamento (piso) na divisão inteira, como o legado C++.
   Quem quiser outra política constrói o número antes de entrar na fórmula.
4. **Sorteio é fronteira, não fórmula.** `formula/` recebe o roll pronto; `roll::Die` é a única
   fonte de aleatoriedade, com seed e estado inspecionáveis.
5. **Ambiguidade de fonte não vira constante.** Vira parâmetro (`GrowthProfile`,
   `ExpTable`, `AttrWeight`, `crit_multiplier_bps`) ou decisão provisória numerada em
   `docs/arquitetura/matematica-sim.md` §4.
6. **Crate nova só se houver I/O a separar.** Não há: `wyd-sim` continua dependendo apenas de
   `wyd-protocol` (nenhuma dependência interna usada hoje) e nenhum crate novo foi criado.

## Consequências

- **Positivas:** cada número tem dono e documento; o ouro de "mesma seed, mesma partida" é
  testável (`tests/determinism.rs`); `wyd-core` pode consumir a mesma função sem duplicar
  regra; a árvore deixa de crescer por acúmulo.
- **Negativas:** mais arquivos pequenos e mais tipos de entrada (`AttrWeight`, `GrowthProfile`)
  que só o banco vai preencher — até lá, os cenários usam perfis explícitos. Duas estruturas
  de efeito de gema continuam existindo (`refine::GemEffect` e o código `GEM_EFFECT_*` do
  core): dívida registrada em `docs/rust/06-matematica.md` §5.
- **Invariantes afetados:** arquivo novo em `crates/wyd-sim/src/` fora desta árvore exige nova
  ADR; `f64` em `formula/` reprova revisão; fórmula que sorteia internamente reprova.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Manter `formula/damage.rs` como arquivo único e crescer por `if` | foi exatamente o problema que motivou a ADR; OCP quebrado e golden único frágil |
| Criar `crates/wyd-rules` para a matemática "de servidor" | duplicaria a regra em vez de compartilhar; `wyd-core` já consome `formula` por cópia presa por golden, e a unificação exige contrato em `wyd-protocol` (dívida da FASE-10-1) |
| Embutir a curva de EXP e os pesos de atributo como constantes | inventaria progressão e escala; `docs/wyd/progressao.md` §7 proíbe |
| Usar `f64` para percentual e arredondar no fim | divergência cliente×servidor por representação; proibido pelo princípio 2 de `06-matematica.md` |
| Trazer `rand`/`fastrand` como dependência | `01-dependencias.md` mantém build offline; 12 linhas de xorshift resolvem e o estado fica auditável |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/FASE-10-1-equacoes.md` (🚧),
  `FASE-10-2-velocidade.md`, `FASE-10-3-laboratorio.md`, `FASE-03-6-ia-npc.md`.
- Docs atualizadas nesta ADR: `docs/arquitetura/matematica-sim.md` (novo, documento
  normativo da matemática), `docs/arquitetura/estrutura-e-padroes.md` §14,
  `docs/rust/06-matematica.md`, `docs/rust/03-progresso.md`, `docs/rust/07-ia-e-bots.md`.
- Evidência: `cargo test -p wyd-sim` (goldens + determinismo + unidade por módulo),
  `docs/rust/03-progresso.md` (registro de 2026-11-08).

## Revisão

- Revisar quando: a FASE-10-1 fechar a ordem de combate com trace real (as decisões D-02..D-04
  caem e viram fato), a FASE-10-2 provar a anomalia de velocidade (o sistema `speed` entra) ou
  o banco passar a fornecer `mob_template_stat` (pesos de atributo deixam de ser hipótese).
- Sinais de que a decisão envelheceu: arquivo de `formula/` com mais de um sistema, cenário
  precisando de `f64` para "funcionar", golden que muda sem fonte nova.
