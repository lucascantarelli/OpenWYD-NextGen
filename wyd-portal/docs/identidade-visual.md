# Identidade visual do portal

Documento curto de direção. A **paleta, a escala de espaço e o raio continuam vindo de
`docs/arquitetura/identidade.tokens`** (a mesma fonte que o launcher embute); o que este
documento registra é o desenho que usa esses tokens — tipografia, densidade, molduras,
hierarquia e estados.

## Direção

**“Laca e ouro”: mesa de operação de um mundo oriental medieval, não fantasia de vitrine.**
Superfície de laca escura, filetes de ouro velho, cantos vivos, tipografia editorial e
números tabulares. O portal deve parecer um instrumento de operação — sóbrio, denso,
legível em sessões longas — e não um template de dashboard.

## Referências consultadas

| Fonte | O que foi observado | Adotado | Recusado |
| --- | --- | --- | --- |
| <https://wyd-international.com/> | Portal oficial: faixa de status do servidor, canais, temporada em destaque, tom de comunidade | Status sempre visível no topo; hierarquia de “servidor → canal → personagem” | Banner promocional grande, contagem regressiva e arte de evento ocupando a primeira dobra |
| <https://wydglobal.raidhut.com/en> | Servidor oficial: tabela de guildas/ranking, cores de reino, caixa de login enxuta | Ranking como tabela densa com números tabulares; caixa de entrada curta, sem ruído | Gradientes saturados de reino e imagens de fundo atrás de texto |
| <https://wydglobal.gg/> | Entrada de conta e download em passos | Fluxo de credenciais em cartão único, instrução no lugar de aviso genérico | Página de marketing como entrada do produto |
| <https://portalnewdestiny.com/> | WYD clássico: estética escura, serifa com ar de pedra/livro, ouro como acento | Serifa lapidar nos títulos; ouro só em acento e filete | Textura de pergaminho literal, brasão ornamentado, ouro em excesso |
| <https://nostalgic.gg/pt/with-your-destiny/wyd-international> | Diretório de servidores: comparação lado a lado, status e população | Vocabulário de status (“online”, “indisponível”) em vez de jargão de infraestrutura | Contadores de “players” como métrica principal do portal |
| <https://github.com/pterodactyl/panel> | Painel de operação de servidores: linha de servidor, console, indicadores de estado | Linha de tabela alta e estável, estado com ponto + rótulo, console monoespaçado | Densidade baixa e cartões muito arredondados do tema padrão |
| <https://github.com/MUnique/OpenMU/blob/master/docs-website/docs/admin-panel/overview.md> | Painel de servidor de jogo: cartões de configuração, atalho para logs, configuração de conta | Cartões com cabeçalho curto e ação única; “conta” separada de “personagens” | Menu lateral com dezenas de itens de configuração interna |
| <https://grafana.com/docs/grafana/latest/best-practices/best-practices-for-creating-dashboards> | Golden signals, redução de carga cognitiva, agrupamento por domínio | Poucos números por tela, rótulo antes do número, agrupamento por domínio | Mural de gráficos sem hierarquia e datas de coleta na interface |
| <https://linear.app/now/how-we-redesigned-the-linear-ui> | Elevação por superfície, hierarquia tipográfica, densidade com respiro | Camadas de superfície discreta, títulos serifados contra UI sans, réguas de 1px | Animações decorativas e “glassmorphism” gratuito |

## Sistema aplicado

- **Tipografia** — `Cinzel` (display, títulos, marca, números de destaque) + `Manrope`
  (operação: tabela, formulário, legenda), via `next/font`. Micro-rótulos em caixa alta
  com `0.14–0.26em` de entreletra; corpo em 14 px/1.5 vindo dos tokens.
- **Cor** — obsidiana + ouro velho dos tokens; ouro só em acento, filete, foco e ação
  primária. Status usam `ok`/`warn`/`danger` em ponto + rótulo, nunca cor sozinha.
- **Forma** — raio 2–3 px em cartões, botões, campos e badges; cantos vivos, molduras de
  1 px e filete superior de ouro nos cartões. Pílulas ficaram restritas a chips de filtro.
- **Profundidade** — sem sombras difusas grandes: moldura interna de 1 px, réguas e
  gradiente de 2% na superfície. O fundo tem trama diagonal de seda a 1,5%.
- **Navegação** — rail de 268 px com grupos rotulados por régua; item ativo com barra de
  ouro e fundo em degradê de 14%. Topbar de 64 px com breadcrumb em caixa alta e sessão
  à direita. Área do jogador usa topbar horizontal com sublinhado de ouro no item ativo.
- **Densidade** — tabelas com linha de 46 px, cabeçalho fixo, zebra de 1,6% e números
  tabulares; cartões com cabeçalho de 18 px e corpo de 20 px.
- **Login** — composição editorial em duas colunas, monograma “W” gigante em contorno,
  cartão de entrada com cantos de moldura; no estreito, só o cartão.
- **Estados** — vazio com moldura tracejada de ouro e frase de produto; indisponível
  sempre em linguagem de produto (“não foi possível…”, “tente de novo em instantes”),
  sem nome de serviço, tabela, data de coleta ou mensagem crua do backend.
- **Segunda senha** — três estados explícitos: **não definida** (passos para criar no
  cliente, sem formulário), **definida** (troca no portal) e **indisponível** (consulta
  falhou). O backend expõe `pin_defined` em `panel/self`; o portal não infere estado.

## O que deliberadamente não existe

Datas de coleta, roadmap, nome de serviço/tabela, contagem de “issues”, tema claro,
glassmorphism, ilustração de fantasia, contador de jogadores como hero e qualquer
paleta paralela à de `identidade.tokens`.
