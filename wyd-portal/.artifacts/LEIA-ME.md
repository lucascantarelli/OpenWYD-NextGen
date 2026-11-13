# Evidências da reformulação visual

## Limitação do recorte

O painel de preview deste ambiente captura **661 px da largura visível**, mesmo quando o
viewport CSS é maior. As capturas `*-1280.png` foram feitas com viewport CSS real de
`1280×900`, sem zoom de página, e mostram o recorte visível do painel. A prova do layout
de 1280 vem da medição feita na própria página:

- `innerWidth = 1280`, `documentElement.clientWidth = 1280`, `scrollWidth = 1280`;
- `scrollWidth - clientWidth = 0` em painel, contas, itens, monitoramento, conta, guias,
  enciclopédia e login;
- rail de 268 px e topbar de 64 px no desktop;
- login com grid `704px 576px` (marca + formulário), cartão de 424 px em `x=780`.

As capturas `*-390.png` usam viewport CSS `390×844`, também sem zoom, com
`scrollWidth - clientWidth = 0` em todas as rotas.

## Estados da segunda senha

- `pin-nao-definida-1280.png` — conta `admin`, `pin_hash` nulo: badge “Não definida”,
  três passos para criar no cliente, sem formulário de troca.
- `pin-definida-1280.png` / `pin-definida-390.png` — conta descartável `pinprova`
  (id 377), criada com o hash de senha do admin e `pin_hash='prova'`: badge “Definida”
  e formulário completo de troca.
- `pin-indisponivel-1280.png` — gateway parado: badge “Indisponível” e a mensagem
  “Não foi possível consultar o estado da segunda senha agora…”, sem inventar estado.

A conta `pinprova` foi removida (`DELETE 1`) e a conta `admin` terminou com
`pin_defined = false`, como começou. O gateway foi reiniciado e responde 200.

## Console

Sem erro de runtime. O console traz apenas `console.info` do servidor (auditoria e
timeouts esperados de métricas/logs no host), avisos de desenvolvimento do Next e
`[HMR] connected`.
