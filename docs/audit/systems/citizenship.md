# Sistema — Cidadania, reinos e canais

> **Paridade: P4 na cidadania derivada do canal, P0 na compra/atribuição de reino.**
> O canal já sabe se é PvP ou “verde”; o jogador ainda não escolhe reino.

## 1. Escopo

Atribuição, remoção, requisitos, permissões, restrições, benefícios, relações com cidades,
guildas, guerras, NPCs, estados, persistência, comunicação.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_{Challange,ChallangeConfirm,ChangeCity}.cpp`, Go `kingdom.go`/`castle.go` |
| Cliente | `0x028E/0x028F` (challange), `0x0291` ChangeCity, `0x0E0E` War |
| Rust | `channels::ChannelRules`, `gameplay::loot*` (bônus por cidadania) |
| Banco | `0041_papel_do_canal` (`channel_role`: 1 normal, 2 verde), `Channel Settings.citizen_bonus_bps` |
| Testes | `integracao/canal_verde.rs` 4 casos, `selchar.rs` |
| Doc | `docs/wyd/canais-e-cidadania.md` |

## 3. Regras confirmadas (no Rust)

| Regra | Valor |
|---|---|
| Papel do canal | dado (`channel_role`) + env por processo |
| PvP por mapa | canal normal segue o mapa; verde **nunca** liga |
| Loja pessoal | política por canal |
| Bônus de cidadania no drop | parâmetro `citizen_bonus_bps` (default 0, teto 50%) |
| Publicação | `/health` expõe `role`/`pvp`/`shop`; boot loga origem (`banco|env`) |
| Cidadania persistida | **não** — é derivada do canal |

## 4. Divergência de vocabulário

| Fonte | O que diz |
|---|---|
| `docs/rust/08-checklist-funcionalidades.md` 2.3 | “Cidadania (**3 reinos**) e canais por reino” |
| Documentação oficial (Kersef) | reinos são **Hekalotia** e **Akeronia**; ingresso nível 220, taxa em safiras |
| Nosso código | cidadania = **papel do canal** (normal/verde), sem reino |

**Classificação: Divergente.** Três nomes de canal não são três reinos. Antes de implementar
“reino”, decidir: (a) reino como dado de personagem, ou (b) reino como papel de canal
(implementação atual). Ver `UNK-31`.

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Cidadania derivada do canal | Implementado e testado | P4 |
| Bônus de drop | Implementado e testado | P4 |
| Compra/atribuição de reino | ausente | P0 |
| Requisito de nível 220 | não modelado | P0 |
| Relação com guerra/castelo | ausente | P0 |
| Persistência de reino | não existe | P0 |

## 6. Ações

1. Decidir modelo de reino (dado × canal) e registrar em ADR/nota.
2. Se for dado: coluna em `characters` + validação + efeito no PvP e na guerra.
3. Se for canal: renomear a documentação para “papel de canal” e remover “3 reinos”.
4. Só então ligar guerra de reinos.

## 7. Riscos

| Risco | Nota |
|---|---|
| Confundir canal com reino | PvP e guerra errados |
| Reino sem persistência | jogador “perde” reino ao relogar |
| Bônus sem teto | economia de drop quebra |
