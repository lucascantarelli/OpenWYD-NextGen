# Economia — moedas, fontes, drenos e mercado

## 1. As quatro moedas

| Moeda | Onde circula | Para que serve |
|---|---|---|
| **Ouro** | mundo | refino (+10 custa 50kk), apostas de guerra, NPC, trade |
| **Safira** | reino / drop | entrar no reino (nível 220); **30 para respec** de status |
| **Fama** | personagem/guilda | skill de topo (1.000/4.000/12.000); aposta de cidade (100) |
| **Wyden Coin / NP** | evento / cash | NPC premium, conveniência |

Existem ainda **medalhas** (verde/roxa/dourada) e **cupons** como moeda de evento — são
itens, não moedas de sistema. Ver `itens.md` §2.

**CONFIRMADO** para ouro, safira, fama e Wyden Coin; **E** para o valor 30 de respec
(`statsapphire = 30` em `gameconfig`).

---

## 2. De onde vem o ouro (fontes)

| Fonte | Ordem de grandeza | Observação |
|---|---|---|
| **Barra de prata** (1kk / 100kk / 1bi) | alto | ouro **como item**: vende-se a NPC ou troca-se |
| **Mobs e baús** | médio | Calabouço é a fonte clássica de early game |
| **Baú de ouro / Caixa de EXP** | médio | recompensa de quest por faixa |
| **Auto-venda premiada** (lojinha) | variável | vender item a NPC com bônus |
| **Guerra** | alto e concentrado | imposto de cidade e castelo |
| **Evento** | alto e pontual | teleport event, invasão, caixa |

**CONFIRMADO** para barra de prata e impostos; **C** para auto-venda (sistema de servidor).

> O ouro em WYD é **item**, não contador abstrato, em várias versões: `Barra de Prata(1bi)` é
> um item negociável. Isso significa que "saldo" e "item de valor" convivem — e um bug de
> duplicação de barra é um bug de economia inteira.

---

## 3. Para onde vai o ouro (drenos)

| Dreno | Custo | Por que existe |
|---|---|---|
| **Refino +10** (Aylin) | **50.000.000** por tentativa | principal sorvedouro do jogo |
| **Composição +12..+15** (Odin) | pedras + insumos | endgame |
| **Anct** | arma +9 + jóia + reforços | alto risco, alto custo |
| **Refino de amuleto** (benção) | +10 500kk · +11 700kk · +12 900kk · +13 1,2bi · +14 1,6bi · Supremo 1,8bi | **C** |
| **Aposta de cidade** | ouro (vai para imposto) | guerra |
| **Entrada de guerra de reinos** (atacante) | **1.000.000** | guerra |
| **Respec de status** | **30 safiras** | não é ouro, mas é dreno |
| **Poção/pergaminho** | barato e constante | consumo |

### O gargalo real

O par **Pedra do Sábio + 50kk** por tentativa de +10 é o que define a economia: cada tentativa
consome dois itens +9 (que custaram refino) e uma pedra que **só cai no Deserto**. Por isso:

1. o **Deserto** é a zona econômica mais disputada;
2. a **guerra de Noatun** acontece ao lado dele;
3. quem controla a oferta de jóia controla o preço de +10 no mercado.

Isso é **design**, não acidente. Qualquer balanceamento de servidor que mexa em drop de Pedra
do Sábio mexe no preço de todo o endgame.

---

## 4. Imposto — a economia de guilda

| Regra | Valor |
|---|---|
| Dono do castelo (Noatun) | **40% do imposto das 3 cidades** |
| Transações nos NPCs do castelo | vão para o master do castelo |
| Atacante em guerra de reinos | 1kk acumula no imposto |
| Defensor em guerra de reinos | 40% do imposto de Noatun do canal |
| Perder a única cidade | **perde o castelo** |

**CONFIRMADO** (Kersef "Siege Warfare" e "Guerra de Reinos"; WYD Global).

### Modelagem correta

```
tax_ledger (id, at, channel, source_kind, source_id, city_id, gross, guild_cut, guild_id)
```

- **Ledger imutável.** Saldo de guilda é `SUM(guild_cut)`, nunca um contador mutável.
- **Rateio no fechamento**, não a cada venda (senão a guerra vira gargalo de escrita).
- **Auditoria**: o portal lê o ledger; não escreve (regra do BFF).

---

## 5. Loja de NPC

`npc_shop_item` (6.706 linhas) já modela: `npc_id`, `slot`, `item_index`, `eff1..3`,
`effv1..3`, `quantity`.

| Observação do dado | Consequência |
|---|---|
| 549 NPCs com `merchant > 0` | são lojas; o resto não vende |
| `item_index` negativo (`-25426`, 2 linhas) | sentinela de "dinheiro" ou serviço — **UNVERIFIED**, sem FK |
| 22 itens de loja fora do catálogo | loja pode vender item que não está no `ItemList` |

**Não** criar FK em `npc_shop_item.item_index`: o dado legado tem item fora do catálogo, e a
FK quebraria o seed (decisão registrada em `modelo-banco.md` §7).

---

## 6. Mercado entre jogadores

- **Trade direto** com confirmação dos dois lados.
- **Lojinha** (loja pessoal) em algumas versões — item exposto com preço.
- **Itens não negociáveis**: `EF_NOTRADE` (156 no catálogo). Ex.: recompensa de evento,
  Coração do Kefra **é** negociável mas **não vendável a NPC**.
- **Regra de preço**: guias recomendam pesquisar preço antes de transação grande — reflexo de
  que não há mercado central com histórico.

### Riscos que o servidor precisa tratar

| Risco | Mitigação |
|---|---|
| duplicação por trade | transação única, lock por `item_instance.id` |
| item de quest negociado | `bound_to` no item de quest |
| trade com inventário cheio | validar espaço **antes** de debitar |
| desconexão no meio | estado `TRADE_PENDING` com timeout no tick |

---

## 7. Números de referência (para o balanceamento)

| Item | Valor |
|---|---|
| Barra de prata | 1kk / 100kk / 1bi |
| Entrada de guerra (atacante) | 1.000.000 |
| Refino +10 | 50.000.000 |
| Anct | 4 reforços +10 = 53% de chance |
| Skill 10ª / 11ª / 12ª (fama) | 4.000 / 1.000 / 12.000 |
| Respec | 30 safiras |
| Aposta de cidade | 100 de fama + 1.000 de influência |
| Influência por mob (Lan A) | 63 |

## 8. O que o servidor precisa

1. **Ouro como inteiro de 64 bits**, com overflow testado (barra de 1bi × 9.000 já estoura
   `i32`).
2. **`wallet` separada do inventário** ou barra como item — escolher **uma** representação e
   documentar; as duas juntas geram duplicação.
3. **Ledger de imposto** imutável e auditável.
4. **Preço de NPC no banco** (`npc_shop_item`), não em constante.
5. **Log de trade** com `item_instance_id`, para responder disputa.
6. **Anti-flood de loja** (rate-limit por sessão), porque loja é superfície de exploit.

## 9. Lacunas

| Ponto | Status |
|---|---|
| preço de NPC por item no 7662 | **parcial** — `item_catalog.price` existe, semântica de moeda não |
| drop de ouro por monstro | **UNVERIFIED** |
| taxa de imposto por cidade | 40% do total é confirmado; rateio por cidade não |
| inflação alvo do servidor | decisão de operação, não de engenharia |
