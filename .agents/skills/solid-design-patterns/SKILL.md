---
name: solid-design-patterns
description: Aplica SOLID e design patterns idiomáticos em Rust no OpenWYD (traits, composição, newtype, type-state, command/strategy via enums). Use ao modelar domínios (itens, skills, combate, quests), refatorar condicionais espalhadas ou eliminar duplicação entre crates.
---

# SOLID e patterns em Rust

SOLID continua valendo; a forma muda: em Rust, herança vira **composição +
traits**, polimorfismo vira **generics/enums**, e invariantes viram **tipos**.

## Guia por princípio

- **S (responsabilidade única).** Um sistema ECS faz uma coisa; um parser valida
  bytes mas não decide regra de jogo; `wyd-sim` calcula, não persiste.
- **O (aberto/fechado).** Estender via novo tipo/enum variant + trait, não via
  `if` em código alheio. Regras de jogo entram como sistemas/funções puras
  plugáveis, não como flags booleanas.
- **L (substituição).** Qualquer implementador de um trait deve honrar o contrato
  documentado (ex.: todo `fn damage()` nunca retorna negativo — use `u32` +
  `saturating_*` em vez de convenção).
- **I (segregação).** Traits pequenos e coesos (`Validates`, `Tickable`,
  `Persistable`) em vez de trait gigante; dependa só do que usa.
- **D (inversão).** `wyd-core` depende de abstrações (traits/injeção via
  parâmetros), não de `wyd-db` concreto; persistência entra pela borda async.

## Patterns preferidos (idiomáticos)

| Problema | Pattern | Forma Rust |
|---|---|---|
| Unidades/invariantes | Newtype | `struct Tick(u64)`; conversões via `From` |
| Estados válidos | Type-state | transição só por método que consome `self` |
| Variação de comportamento | Strategy via enum | `match` exaustivo, sem `dyn` no quente |
| Comandos GM/admin | Command | struct por ação + `execute()` auditável |
| Construção complexa | Builder | builder com type-state até `build()` validar |
| Erros de domínio | Typed errors | `thiserror`, um erro por fronteira |

## Anti-patterns (recusar em review)

Herança simulada com `dyn Any` + downcast; `bool` como parâmetro de modo;
`match` com `_ =>` engolindo variante nova; lógica de negócio dentro de parser;
cópia de struct gigante por valor no loop.
