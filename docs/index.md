# Documentação do OpenWYD

Servidor local de WYD2 em Go, portal administrativo em Next.js e cliente Windows — num
repositório só. Esta é a porta de entrada; cada seção aponta para o documento que manda
naquele assunto.

## Por onde começar

| Preciso de… | Leia |
|---|---|
| Subir a stack e logar | `migration/SESSION-PRIMER.md` |
| Saber o que funciona hoje | `analysis/estado-atual.md` |
| Entender a arquitetura | `analysis/README.md` |
| Mexer em wire/formato | `migration/protocol-spec.md`, `migration/data-formats.md` |
| Mexer em regra de jogo | `migration/game-rules.md` |
| Escrever um handler | `migration/handlers/README.md` |
| Construir tela do portal | `integrations/` |
| Operar (portas, configs, banco) | `migration/config-ops.md` |

## Como esta documentação se organiza

- **`migration/`** — engenharia reversa do legado: o que foi medido, com que evidência, e o que
  continua **UNVERIFIED** (não é gate: comportamento não confirmado é tolerado e logado).
- **`analysis/`** — diagnóstico do estado real: banco, cliente, legado, processo.
- **`integrations/`** — contrato de cada tela do portal com o web-api.
- **`sistemas/`** — uma página por sistema de jogo, ligando handler e regra.
- **`adr/`** — decisões caras, com o contexto de quando foram tomadas.

## Regra de manutenção

Documento que descreve comportamento precisa citar a evidência (`arquivo:linha` do legado, teste,
medição). Quando o código mudar e o doc não, o doc está errado — não o código. Doc superado vai para
`roadmap/doc_legacy/` com README dizendo o que o substituiu, nunca é apagado no meio da migração.
