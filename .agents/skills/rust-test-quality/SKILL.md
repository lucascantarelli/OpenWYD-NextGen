---
name: rust-test-quality
description: Garante tipo, nome e higiene dos testes Rust do WYD (unidade/integração/contrato/golden/sistema), com gate explícito para Postgres/socket/processo, sem skip silencioso, sleep de sincronização ou monólito. Use ao criar, revisar ou reorganizar qualquer teste em crates/ e antes de fechar uma issue.
---

# Qualidade e organização dos testes

Norma canônica: `docs/arquitetura/testes.md` — taxonomia §2, nomenclatura/localização §3,
higiene §4, matriz por crate §5 e checklist §8. Não repita a tabela aqui.

## Critério de revisão

- **Tipo pelo que o teste exige para rodar.** Unidade não abre socket/banco/processo/FS;
  integração usa só API pública; contrato compara doc/schema/TS/assets; golden usa fixture
  versionada; sistema é opt-in por Postgres, socket, processo ou FS.
- **Gate explícito.** Teste de sistema sem DSN imprime `SKIP:` com todas as variáveis e o
  que não foi provado. Nunca `#[ignore]`, `return` mudo ou panic por env ausente; `SKIP`
  não é prova.
- **Sem `sleep` fixo.** Poll com prazo e falha diagnóstica; exceção é sessão viva
  proposital e documentada.
- **Isolamento e asserção concreta.** Dados únicos, limpeza, porta efêmera, escrita só em
  temp/flag; asserte valor/erro/byte, caso negativo e determinismo (seed/relógio injetados).
  Não enfraqueça asserção para o CI passar.
- **Um tema por arquivo.** Passou de ~400 linhas ou misturou infraestruturas, divida; helper
  em `tests/common/mod.rs`; nome em português descreve o estado esperado.

## Prova

O comando específico da issue + `make check`; para sistema, registre `WYD_TEST_DSN`/serviço
usado e o que ficou em `SKIP`.
