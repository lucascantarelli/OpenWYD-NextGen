# Definition of Done e processo de trabalho

Documento único de processo do Dolorestec WYD2. Substitui as regras espalhadas em `AGENTS.md` §8 e
em `docs/analysis/qualidade-e-processo.md` §4: quando houver divergência, **vale esta página**.

## 1. O que conta como "pronto"

Uma entrega só está pronta quando **as seis linhas abaixo têm evidência**, não quando o código foi
escrito:

| # | Linha | Evidência mínima |
|---|-------|------------------|
| 1 | Código compila | `go build ./...` (ou `make portal-build` no portal) com exit 0 |
| 2 | Teste cobre a mudança | teste novo/ajustado que **falha sem a correção** e passa com ela |
| 3 | Verificação executada | o comando citado na issue rodou, com a saída registrada na própria issue |
| 4 | Documentação atualizada | doc do sistema/regra que mudou, ou `roadmap/index.md` quando o estado mudou |
| 5 | Legado removido | o caminho antigo saiu na mesma entrega (`AGENTS.md` §8, regra da migração) |
| 6 | Rollback descrito | como voltar atrás se a mudança quebrar em uso |

"Não testei, mas deve funcionar" não é evidência. "Passou no meu ambiente" sem o comando não é
evidência. Um print de tela do cliente **é** evidência para comportamento de cliente; um teste
headless **é** evidência para servidor.

## 2. Regra da issue

- Uma issue = **um resultado verificável**. Se o título precisa de "e", provavelmente são duas.
- Issue fechada cita: comando ou cenário, saída, e o que ficou **fora** do escopo.
- Issue bloqueada por humano (cliente Windows, decisão de produto) recebe um bloco **Evidência** com
  o que já foi provado e o passo exato que falta — não fica "em progresso" indefinidamente.
- Ao fechar, atualizar `roadmap/index.md` **na mesma entrega** e, se o comportamento observável
  mudou, `docs/migration/SESSION-PRIMER.md`.

## 3. Regra do comentário de código

- Comentário e docstring em **português**, começando pelo nome do símbolo quando for `doc`.
- **Proibido** citar número de issue, caminho de `roadmap/` ou de `docs/` dentro de `.go`: o
  histórico vive no git e no roadmap, não no código.
- Comentário explica **por que**, não o que a linha já diz. Comentário que descreve comportamento
  errado é pior que comentário ausente: se não dá para garantir, apague.
- Referência ao legado C++ é permitida só quando o comportamento foi **medido** nele
  (`arquivo.cpp:linha`), e nesse caso precisa dizer o que foi medido.

## 4. Revisão de código

- PR/entrega com **menos de 400 linhas** de diff lógico (excluindo gerados e migrações de dados).
- Um revisor humano ou agente independente do autor; o autor não aprova a própria entrega.
- O que **reprova** sem discussão: segredo no diff, teste desligado/afrouxado para passar, migração
  sem `down`, escrita em tabela de personagem vivo vinda da web, lock novo no estado do mundo.

## 5. Commits

- Mensagem em português, imperativo, uma frase de **por que** e uma de **o que**.
- Commit pequeno e temático; nada de "ajustes" misturando três assuntos.
- Nunca versionar: segredo/`.env`, `client/`, `serverlist.bin` com IP privado, dump de conta,
  binário gerado que o build recria.

## 6. Definition of Done por tipo de mudança

| Tipo | Além do padrão |
|------|----------------|
| Protocolo/wire | vetor de bytes no teste (`protocol/*_test.go`) + tamanho do frame conferido |
| Regra de jogo | golden case com o número esperado (dano, drop, exp, refino) + doc da regra |
| Persistência | migração `up` **e** `down`, teste em base efêmera, contagem por tabela |
| Portal/BFF | teste de autorização (401 sem sessão, 403 por papel) + auditoria da ação |
| Conteúdo (`tmserver/content/`, seeds) | contagem antes/depois e o comando que reimporta |
| Observabilidade | métrica/log com nome estável e o alerta correspondente |

## 7. Quando a mudança é grande demais

Divida em: (1) medição que prova o problema, (2) mecanismo mínimo que fecha o caso principal,
(3) generalização. A entrega 1 é obrigatória e barata; sem ela, a 2 vira aposta.

## 8. Pendências e dívida

Dívida assumida (temporária, com saída) registrada como issue, nunca como comentário `TODO` solto.
`TODO` no código só é aceito com issue aberta e prazo; caso contrário, apague.
