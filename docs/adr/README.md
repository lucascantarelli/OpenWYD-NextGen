# ADRs — decisões arquiteturais

Registro curto do **porquê** de decisões que custam caro para reverter. Não é tutorial: cada ADR diz
o contexto, a decisão e o que ela impede.

| # | Decisão | Estado |
|---|---|---|
| [0001](0001-go-no-servidor.md) | Go como linguagem do servidor | aceita |
| [0002](0002-goroutine-unica-do-mundo.md) | Goroutine única dona do estado do mundo | aceita |
| [0003](0003-portal-bff.md) | Portal como BFF sobre o web-api | aceita |
| [0004](0004-banco-fonte-de-conteudo.md) | Banco como fonte de catálogo e spawn | aceita |
| [0005](0005-delivery-queue.md) | `delivery_queue` para grants da web | aceita |
| [0006](0006-arquivamento-legado-cpp.md) | Arquivar o legado C++ em `_legacy` | aceita |

## Como escrever um ADR

Numeração sequencial, um arquivo por decisão, no máximo uma página. Se a decisão for revertida, não
se apaga o ADR: ele ganha `Substituído por 00XX` no cabeçalho. Histórico que se apaga deixa a próxima
pessoa repetindo a mesma investigação.
