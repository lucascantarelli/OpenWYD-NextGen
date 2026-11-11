# ADR-0014 — Autoridade do servidor mora em `systems/`, com dano injetado por trait

- **Estado:** aceita (2026-11-08)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (FASE-03-5)
- **Governa:** `crates/wyd-core/src/systems/{mod,movement,combat}.rs`, FASE-03-5, FASE-10
- **Depende de:** ADR-0001 (Hot Path), ADR-0002 (estrutura), ADR-0004 (ECS SoA), ADR-0012 (gema)

## Contexto

A FASE-03-5 exige que física, dano, skills e inventário sejam validados pelo servidor, com
"pacote forjado recusado sem alterar estado". O core já tinha validação pontual de inventário
(`ComponentStore::apply_gem`, ADR-0012), mas **nenhum** ponto de validação de movimento ou
combate: não existia caminho pelo qual uma intenção vinda do cliente pudesse ser aceita ou
recusada, nem estado de cooldown para impedir ataque em rajada.

Duas tentações apareceram:

1. aceitar `(posição final, dano)` do chamador e só "conferir faixa" — deixaria o cliente como
   autor do resultado, violando o invariante 3 do `AGENTS.md`;
2. escrever a fórmula de dano aqui — proibido pela própria issue ("Proibido tocar: fórmulas
   finais (FASE-10)") e por `estrutura-e-padroes.md` §10 (`combat.rs` aplica fórmula aprovada,
   não a inventa).

## Decisão

Adotamos **sistemas de validação em `crates/wyd-core/src/systems/`** (árvore já reservada no
§10 de `estrutura-e-padroes.md`), com **dano injetado por trait** e **cooldown como componente
contíguo**, porque isso mantém o servidor como único autor do resultado sem obrigar o core a
conhecer balanceamento.

Regras que decorrem dela:

1. A API pública recebe **intenção**, nunca resultado: `movement::step(store, entity, to,
   elapsed_ms, rules)` e `combat::attack(store, attacker, target, now_ms, rules, model)`.
   Não existe parâmetro de dano nem de posição "confirmada".
2. `MoveRules`/`AttackRules` são configuração explícita do servidor (velocidade em tiles/s,
   retângulo do mundo, alcance, cooldown). O core **não** fixa constantes de jogo: nenhuma
   velocidade mágica inventada (marcadores de balanceamento seguem na FASE-10).
3. `DamageModel` (trait) recebe os `Stats` autoritativos e devolve o dano; a equação aprovada
   da FASE-10 é a implementação. Enquanto não houver fórmula, o core não decide dano — quem
   monta o mundo injeta o modelo (nos testes, um modelo determinístico).
4. `next_attack_ms` é componente do `ComponentStore` (`Vec<u64>` contíguo, `Copy`), alocado no
   spawn e zerado na reciclagem de índice — nada de `HashMap`/mutex no quente.
5. Ordem transacional em ambos os sistemas: validação completa → evento na fila limitada →
   gravação. Recusa não muda posição, HP, cooldown nem fila; só o aceite publica evento
   (`Moved`, `DamageDealt`).
6. Recusas são variantes de `CoreError` (`MoveTooFar`, `NoMoveBudget`, `InvalidMoveRules`,
   `SelfAttack`, `TargetOutOfRange`, `AttackCooldown`, `TargetAlreadyDead`,
   `InvalidAttackRules`), sem `panic!` em nenhum caminho.

## Consequências

- **Positivas:** a borda passa a ter um ponto único de recusa auditável; o teste de pacote
  forjado vira teste de API pública (`tests/integracao_autoridade.rs`); trocar a fórmula da FASE-10 não
  toca validação, e vice-versa.
- **Negativas:** o chamador precisa fornecer as regras (velocidade, alcance, cooldown) e o
  modelo de dano — configuração a mais no boot, deliberada: números de jogo não ficam
  escondidos na engine.
- **Invariantes afetados:** nenhum `unsafe`; sem I/O no tick; fila de eventos continua limitada;
  dano nunca ultrapassa o HP restante (o evento carrega o valor aplicado, não o "bruto").

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Validar na camada de rede (`wyd-net`) | core seria confiável por omissão; a autoridade precisa morar onde o estado vive, e a rede só entrega intenção |
| Aceitar dano do chamador e limitar por faixa | cliente vira autor do número; viola "servidor é autoridade" |
| Escrever fórmula de dano agora | proibido pela FASE-03-5 e pela FASE-10; fórmula entra por trait aprovado |
| Cooldown em `HashMap`/recurso global | alocação no quente e lookup por hash; o SoA já indexa por entidade |
| Guardar `next_attack_ms` em `Stats` | `Stats` é dado de personagem persistido; cooldown é estado de tick, não de ficha |

## Rastreabilidade

- Issue consumidora: `roadmap-rust/issues/FASE-03-5-autoridade-servidor.md`.
- Docs: `docs/arquitetura/autoridade-servidor.md` (mapa das ações validadas),
  `docs/arquitetura/estrutura-e-padroes.md` §10, `docs/rust/03-progresso.md` (entrada datada).
- Evidência: `crates/wyd-core/tests/integracao_autoridade.rs` (7 testes de pacote forjado) +
  `systems/movement.rs`/`systems/combat.rs` (13 testes unitários).

## Adendo (2026-11-08) — `systems/skill.rs` entra na árvore

O eixo **skills** da issue não podia ficar declarado como pendência sem código: a validação
não depende do banco, só do que o servidor já sabe. `systems/skill.rs` (novo arquivo, registrado
no §10 de `estrutura-e-padroes.md`) recebe a lista de skills **aprendidas** (`known`, carregada de
`character_skills` na FASE-04-3) e o `SkillBook` do catálogo (id, custo de MP, cooldown,
alcance), validando na mesma ordem transacional dos demais sistemas. O evento `SkillCast`
registra quem usou e o custo; o **efeito** da skill continua na fórmula aprovada (FASE-10), como
no combate. Componente novo: `next_skill_ms` (contíguo, `Vec<u64>`, zerado no spawn).

## Revisão

- Revisar quando: a FASE-10 entregar a fórmula aprovada (o trait ganha implementação de
  produção), a FASE-03-6 precisar de alvo/ameaça no `combat.rs`, ou o cliente provar que o
  movimento é passo-a-passo em vez de destino (muda a unidade do orçamento).
- Sinais de que envelheceu: sistema aceitando número do chamador, `AttackRules` fixo no core,
  cooldown por `HashMap`, evento publicado antes da validação final.
