# ADR-0003 — Estratégia de patch do cliente e manifesto do launcher

- **Estado:** aceita
- **Data:** 2026-11-07
- **Decisores:** dono do projeto + agente (registrado na issue FASE-08-7)
- **Governa:** `roadmap-rust/issues/FASE-08-3-launcher.md`, `FASE-08-7-cliente-customizacao.md`

## Contexto

O `wyd-launcher` precisa verificar, atualizar e iniciar o `WYD.exe` 7662 (fechado,
`04c9d2a1…27ddf8`). Existem três pressões reais: (1) o `ClientPatch_v7662` resolve o patch
com offsets fixos mas é GPL-3.0 e não tem manifesto nem rollback (`docs/cliente/superficie-conhecida.md`);
(2) os três sites de checksum já vêm desativados no disco (`analise-estatica.md` §4), então
o patch inicial é idempotente; (3) não há registry crates.io utilizável neste ambiente
(`docs/rust/01-dependencias.md` — decisão offline), logo SHA-256 e CLI não podem vir de
dependência externa. Hipótese marcada como hipótese: o updater legado fala HTTP com
`http://54.207.102.145/update` (`settings.txt`) — protocolo não mapeado, tratado como
**não suportado**.

## Decisão

Adotamos camada própria de patch por tabela de offsets presa a hash de build, com manifesto
de integridade versionado e aplicação atômica (tmp + rename + verificação pós-patch), sem
herdar código do ClientPatch.

## Consequências

- **Positivas:** sem contaminação GPL no código novo; patch reproduzível por build; rollback
  sempre possível; `serverlist.bin`/`sn.bin` (por ambiente) ficam fora do manifesto de build.
- **Negativas:** reimplementar verificação SHA-256 em `std` puro + vetores NIST (custo pago
  uma vez, testado).
- **Invariantes afetados:** nenhum offset de patch vale sem hash de build publicado;
  nenhum patch sem backup + verificação pós-patch; nenhum binário do cliente no git;
  manifesto é a única fonte de verdade do que o launcher aceita.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Reusar `ClientPatch.dll` como patcher | GPL-3.0 contamina o launcher; offsets fixos sem manifesto, sem rollback, sem auditoria |
| Diff binário genérico (estilo xdelta) | complexidade e opacidade para um problema que hoje é “mesmos bytes por build” |
| SHA-256/CLI via crates externas | sem registry utilizável; viola a decisão offline |
| Falar o protocolo do updater legado | endpoint não mapeado; alvo móvel de terceiro |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/FASE-08-3-launcher.md`, `FASE-08-7-cliente-customizacao.md`
- Docs atualizadas nesta ADR: `docs/cliente/customizacao.md`, `docs/cliente/mapa.md`
- Evidência: hashes medidos em `client/` (2026-11-07, todos conferem com `inventario.md`)

## Revisão

- **2026-11-11 — assinatura do pacote trocada de HMAC para Ed25519.** O registry passou a
  estar disponível (a decisão "offline" do contexto histórico não vale mais) e o HMAC
  simétrico exigia distribuir a mesma chave que assina: quem espelhasse o binário também
  podia publicar pacote. `ed25519-dalek` 2.x entra verify-only (`default-features = false`,
  `std`), a chave pública é embutida (`distribuicao::CHAVE_PUBLICA_HEX`) e a privada fica
  no CI/secret (`tools/launcher/sign-package.mjs`). Sem mudança no formato do manifesto nem
  no pipeline patch/rollback; `assinatura` passa de 64 para 128 hex. Custo: rotação de
  chave exige launcher novo; benefício: o cliente não guarda segredo. Evidência:
  `cargo test -p wyd-launcher` + golden cross-language (`tests/golden/assinatura_ed25519.rs`).
- Revisar quando: mudar o build de referência do cliente, ou surgir registry offline.
- Sinais de que a decisão envelheceu: patch real divergir do manifesto em produção, ou
  offsets por build virarem manutenção maior que um diff binário versionado.
