# Issue 08-7 — Customização sustentável e integração com o launcher

**Fase:** 8 · **Depende de:** 08-5, 08-6 · **Bloqueia:** 08-3 (distribuição) e FASE-07 · **Esforço:** L · **Status:** ✅

## Objetivo

Transformar o conhecimento de RE em **camada de customização sustentável** do cliente
(hooks/patch por versão, UI, comandos) e no contrato que o `wyd-launcher` usa para validar,
atualizar e iniciar o jogo.

## Contexto / evidência

A customização legada é o `ClientPatch_v7662` (GPL-3.0, offsets do build 7662 — `docs/cliente/superficie-conhecida.md`);
os 3 sites de checksum já vêm desativados no disco (`analise-estatica.md` §4). O updater
legado aponta para endpoint não mapeado (`settings.txt`). Sem registry crates.io utilizável
(decisão offline).

## Resultados (2026-11-07)

| Entrega | Evidência |
|---|---|
| Estratégia decidida | ADR-0003: camada própria por tabela de offsets presa a hash + manifesto versionado + aplicação atômica, sem herdar GPL |
| Especificação | `docs/cliente/customizacao.md`: formato do manifesto, exemplo 7662 com hashes medidos, fluxo patch/rollback, bootstrap, CLI futura |
| Implementação | `crates/wyd-launcher/src/{sha256,manifest,patch,bootstrap}.rs` + CLI `check`/`launch` em `main.rs`; zero dependência nova (`std` puro) |
| Testes | 19/19 verdes: vetores NIST, manifesto (parse/verify/falhas), patch (novo/troca/rollback/fonte corrompida/fuga de diretório), bootstrap (erro/dry-run) |
| Prova ponta a ponta | binário real `check client/` com o manifesto do doc: 6/6 OK; adulteração detectada (tamanho); `launch --dry-run` exibe o comando sem executar |

Limitação registrada (não escondida): o spawn real do `WYD.exe` não foi executado —
abriria o jogo de verdade neste ambiente. Cobertos: plano, erro tipado e dry-run.
Validação com jogo roda em VM na FASE-07.

## Arquivos

- Criados: `docs/cliente/customizacao.md`, `docs/arquitetura/adr/0003-*.md`,
  `crates/wyd-launcher/src/{sha256,manifest,patch,bootstrap}.rs`.
- Alterados: `crates/wyd-launcher/src/{lib,main}.rs`, `docs/cliente/{README,mapa}.md`,
  `engenharia-reversa.md`, `roadmap-rust/index.md`, `docs/rust/03-progresso.md`.
- Proibido e respeitado: nenhum binário do cliente no git; `serverlist.bin`/`sn.bin`
  fora do manifesto de build (por ambiente).

## Testes / verificação

```bash
cargo fmt --all -- --check && cargo clippy --workspace --all-targets -- -D warnings && cargo test --workspace
./target/debug/wyd-launcher check client/ <manifesto-7662>   # 6/6 OK, exit 0
./target/debug/wyd-launcher launch client/ <manifesto> --dry-run
```

Saída obtida: fmt 0, clippy limpo, workspace todo verde (launcher 19/19, protocolo 28/28);
`check` recusou arquivo adulterado com motivo e tamanho esperado/encontrado.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-client-reverso - offsets e limites de patch.
- .agents/skills/wyd-rust-toolchain - distribuição e assinatura do binário.
- .agents/skills/rust-code-quality - testes e erros tipados.

## Documentação de referencia

- docs/cliente/customizacao.md - especificação seguida pela implementação.
- docs/arquitetura/adr/0003-estrategia-patch-cliente.md - decisão.
- docs/cliente/inventario.md - base do manifesto.
- docs/cliente/customizacao.md - mecânica de patch em runtime.

## Subtasks

- [x] ADR de estratégia
- [x] Especificação + exemplo de manifesto
- [x] Codificar issue (`manifest/patch/bootstrap.rs` + CLI)
- [x] Testar issue (19 testes + prova ponta a ponta)
- [x] Revisar issue (clippy `-D warnings`, sem `unwrap` fora de teste)
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] ADR de estratégia de patch aceita e rastreável.
- [x] Manifesto validável por hash e versão, com exemplo do build 7662.
- [x] Patch atômico com rollback comprovado por teste.
- [x] Launcher recusa `verify` sujo e inicia só com tudo verde (dry-run + erro tipado
      verificados; spawn real pendente de ambiente com jogo — ver Limitação).
- [x] Nenhum binário do cliente versionado ou redistribuído.

## Riscos / rollback

- Risco: patch quebrar o build em produção → mitigação: manifesto + backup + aplicação atômica.
- Risco: obrigação GPL por herdar ClientPatch → mitigação: ADR-0003 antes de codar (feita).
- Rollback: `git checkout -- crates/wyd-launcher docs/cliente/customizacao.md`.
