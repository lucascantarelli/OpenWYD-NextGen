# MATH-02 — Laboratório de simulação e investigação da velocidade/hits

**Fase:** matemática · **Depende de:** MATH-01, BOT-03 · **Bloqueia:** — · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Reproduzir ou descartar, com evidência, o comportamento de hits adicionais em velocidade alta (~200%)
e ter um simulador independente do servidor para comparar fórmula atual × proposta.

## Contexto / evidência

- Relato de jogadores: personagem físico com ~200% de velocidade "começa a dar hits duplicados".
  **Nível C**: nenhuma medição no projeto confirma ou descarta.
- O caminho de dano já é server-authoritative (`MSG_Attack` → handler → `SetHpDam`), então o
  fenômeno, se existir, está no servidor (frequência de ataque) ou na leitura do cliente.
- Ordem de RNG do MSVC é invariante do projeto (`tmserver/internal/rng`): qualquer mudança de ordem
  muda drop, crítico e refino — a simulação precisa reproduzir a mesma sequência.
- O laboratório de bots (`BOT-02`) já dá população determinística; falta a ação de ataque real
  (`BOT-03`) para medir hits no servidor.

## Passos

1. Medir o comportamento atual: N ataques com velocidades 100/150/200/300/400% com seed fixa,
   contando pacotes `MSG_Attack` enviados × dano aplicado × hits desenhados.
2. Instrumentar o servidor (contador por conexão) para separar "hits processados" de "hits exibidos".
3. Definir a hipótese matemática: frequência = f(velocidade) com teto e arredondamento declarados;
   testar pelo menos uma curva linear, uma por faixas e uma com retornos decrescentes.
4. Implementar simulador offline (pacote Go, sem rede) que recebe atributos e devolve DPS, hits/s e
   distribuição de crítico para milhares de combinações.
5. Comparar simulado × medido; publicar a diferença. Sem diferença, o relato é descartado com dado.
6. Só propor mudança de fórmula com golden case e com a economia (drops) reconferida.

## Arquivos

- `tmserver/internal/mathlab/` (novo: simulador)
- `tmserver/internal/mathlab/*_test.go`
- `docs/sistemas/matematica.md` (resultados)
- `tmserver/internal/bot/scenarios/velocidade.json`

## Testes / verificação

```
go test -race ./tmserver/internal/mathlab/...
go run ./tmserver/cmd/botlab run -scenario tmserver/internal/bot/scenarios/velocidade.json
```

Saída esperada: tabela velocidade × hits/s × DPS para a fórmula atual e para a proposta, com a
diferença medida em jogo ao lado.

## Critérios de aceite

- [ ] Hits acima de 200% reproduzidos ou descartados com número
- [ ] Simulador determinístico e independente do servidor
- [ ] Comparação simulado × medido publicada, incluindo divergências
- [ ] Nenhuma mudança de fórmula sem golden case e sem reconferir drop/crítico

## Riscos / rollback

Risco: simulação virar "verdade" e substituir teste real. Mitigação: nenhum número publicado sem
medida em jogo ao lado. Risco: alterar RNG e mudar drop. Mitigação: preservar a ordem das chamadas
e travar com teste.
