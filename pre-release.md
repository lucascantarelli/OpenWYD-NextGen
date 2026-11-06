# /mission — ANÁLISE, REORGANIZAÇÃO DO ROADMAP E MODERNIZAÇÃO COMPLETA DO PROJETO

## 1. IDENTIFICAÇÃO DA MISSÃO

**Nome:** Análise de Roadmap, Auditoria Técnica e Modernização Completa do Projeto.

**Objetivo principal:** Realizar uma auditoria técnica abrangente de todo o projeto, revisar e reorganizar integralmente o roadmap, identificar todas as pendências, inconsistências, problemas arquiteturais e oportunidades de melhoria e, posteriormente, executar as atividades necessárias para alcançar um estado de excelência técnica, funcional e operacional.

Esta missão envolve todas as áreas do projeto, incluindo:

- Backend e servidor.
- Cliente do jogo.
- Banco de dados.
- Infraestrutura e DevOps.
- CI/CD.
- Portal web (Next.js).
- Sistemas e funcionalidades do jogo.
- Observabilidade e monitoramento.
- Segurança.
- Testes automatizados.
- Documentação.
- Skills, agentes e ferramentas de IA.
- Arquitetura e padrões de desenvolvimento.
- Launcher e ferramentas auxiliares.
- Engenharia reversa e modernização do cliente.

**Diretriz fundamental:** Não se limite às issues já existentes. Faça uma investigação completa do projeto, identificando problemas, lacunas, funcionalidades incompletas, código legado, implementações ausentes e melhorias necessárias que ainda não estejam registradas no roadmap.

O resultado esperado é um projeto completamente organizado, modernizado, documentado, testado, seguro, observável e preparado para uma futura release.

---

# 2. REGRAS FUNDAMENTAIS DE EXECUÇÃO

Estas regras devem ser seguidas durante toda a missão.

## 2.1. Análise antes da implementação

- Não comece implementando funcionalidades sem antes compreender a arquitetura atual, as dependências e o estado real do projeto.
- Analise os arquivos, implementações, documentação, migrations, testes, configurações e infraestrutura antes de propor mudanças.
- Não presuma que uma issue está concluída apenas porque está em `done/`.
- Não presuma que uma implementação está ausente apenas porque não existe uma issue correspondente.
- Verifique o código real para confirmar o estado de cada implementação.
- Identifique dependências entre funcionalidades e possíveis impactos das alterações.
- Priorize a eliminação de problemas estruturais antes da implementação de novas funcionalidades.

## 2.2. Preservação do conhecimento do projeto

- Utilize integralmente a estrutura existente de documentação, skills, agentes, instruções e arquivos de contexto de IA.
- Sempre consulte as skills pertinentes antes de iniciar uma implementação.
- Identifique quais skills precisam ser atualizadas ou complementadas.
- Mantenha as decisões arquiteturais, ADRs, padrões e convenções documentados.
- Não crie documentação redundante ou desnecessária.
- Não descarte informações técnicas relevantes durante a reorganização.

## 2.3. Integridade e rastreabilidade

- Toda implementação relevante deve estar vinculada a uma issue.
- Toda issue deve possuir critérios de aceitação claros.
- Toda alteração deve respeitar as dependências e os bloqueios identificados.
- Toda conclusão deve possuir evidências verificáveis.
- Não marque issues como `done` sem comprovação da implementação e dos testes aplicáveis.
- Não oculte problemas, testes falhos ou limitações encontradas.
- Não considere uma funcionalidade concluída apenas porque o código foi desenvolvido.

## 2.4. Autonomia e tomada de decisão

Execute autonomamente as atividades que possam ser realizadas com segurança e que estejam suficientemente especificadas.

Quando encontrar decisões arquiteturais relevantes, alternativas com impactos significativos, mudanças irreversíveis ou questões que dependam de aprovação:

1. Investigue profundamente o problema.
2. Desenvolva alternativas tecnicamente fundamentadas.
3. Compare vantagens, desvantagens, custos, riscos e complexidade.
4. Registre uma recomendação técnica.
5. Documente a decisão para análise e aprovação.

Não interrompa toda a missão por causa de uma decisão isolada. Continue com as atividades independentes que possam ser executadas com segurança.

---

# 3. FASE 1 — AUDITORIA COMPLETA E REORGANIZAÇÃO DO ROADMAP

Esta fase é prioritária. Antes de iniciar a execução em larga escala, estabeleça uma visão real e confiável de todo o trabalho existente e necessário.

## 3.1. Auditoria das issues concluídas — `done/`

Analise individualmente todas as issues existentes em `done/`.

Para cada issue:

- Verifique se a implementação realmente existe.
- Confira se está integralmente implementada.
- Verifique se os testes foram executados e estão passando.
- Confira se a documentação correspondente foi atualizada.
- Verifique se os critérios de aceitação foram atendidos.
- Procure implementações parciais ou pendências não registradas.
- Verifique se existem bugs relacionados ainda não resolvidos.
- Identifique issues abertas que tornem a conclusão da issue obsoleta, incompleta ou contraditória.
- Verifique se a implementação foi posteriormente substituída por outra abordagem.
- Identifique funcionalidades que foram removidas, alteradas ou descontinuadas.

Classifique cada issue conforme seu estado real.

Caso uma issue não esteja efetivamente concluída, corrija sua classificação e mova-a para o local adequado.

Não mantenha uma issue em `done/` apenas por conveniência histórica.

## 3.2. Auditoria das issues aguardando testes — `test/`

Analise todas as issues existentes em `test/`.

Elabore um roteiro completo de testes, contemplando todas as funcionalidades e implementações que aguardam validação.

Para cada issue:

- Identifique o que precisa ser testado.
- Analise os cenários positivos e negativos.
- Identifique requisitos funcionais e não funcionais.
- Determine a estratégia de testes adequada.
- Verifique se já existem testes automatizados.
- Identifique testes ausentes ou insuficientes.
- Determine quais testes podem ser executados autonomamente.
- Execute os testes que puderem ser realizados com segurança.

Avalie individualmente a necessidade de:

- Testes unitários.
- Testes de integração.
- Testes de contrato.
- Testes de API.
- Testes end-to-end.
- Testes de regressão.
- Testes de segurança.
- Testes de performance.

Quando a funcionalidade exigir validação de um fluxo real de utilização, implemente os testes E2E utilizando Playwright.

Não crie testes E2E desnecessários para funcionalidades que podem ser validadas de forma mais eficiente em outras camadas.

Organize os testes seguindo as boas práticas e os padrões estabelecidos pelas skills do projeto.

## 3.3. Reorganização completa das issues

Faça uma reorganização integral do roadmap.

Objetivos:

- Identificar issues obsoletas.
- Identificar issues já resolvidas.
- Identificar issues duplicadas.
- Identificar issues parcialmente implementadas.
- Identificar issues com informações desatualizadas.
- Identificar issues sem critérios de aceitação.
- Identificar issues com escopo inadequado.
- Identificar dependências não documentadas.
- Identificar issues que precisam ser divididas.
- Identificar issues que precisam ser agrupadas.
- Identificar problemas que estão documentados em outros locais, mas ainda não possuem issue.
- Corrigir classificações incorretas.
- Atualizar descrições, referências, prioridades e dependências.

Reorganize as issues por categoria, área técnica, prioridade e sequência de execução.

Dê atenção especial a:

1. Refatoração de código.
2. Remoção de código legado.
3. Remoção de implementações obsoletas.
4. Qualidade de código.
5. Padronização.
6. Arquitetura.
7. Segurança.
8. Performance.
9. Infraestrutura.
10. Testes.
11. Documentação.
12. Funcionalidades pendentes.

Construa uma sequência de execução coerente, considerando dependências técnicas e funcionais.

Issues bloqueantes devem ser identificadas explicitamente.

O roadmap deve permitir compreender facilmente:

- O que precisa ser feito.
- Por que precisa ser feito.
- Quais problemas serão resolvidos.
- Quais issues dependem dessa implementação.
- Quais issues estão bloqueadas.
- Qual é a ordem correta de execução.
- Quais resultados são esperados.

## 3.4. Investigação completa do projeto para identificação de gaps

Faça uma varredura completa de todo o repositório.

Investigue:

- Código-fonte.
- Estrutura de diretórios.
- Arquivos de configuração.
- Dependências.
- Scripts.
- Infraestrutura.
- Banco de dados.
- Migrations.
- Seeds.
- Testes.
- Documentação.
- Workflows.
- Dockerfiles.
- Arquivos de deploy.
- Configurações de monitoramento.
- Arquivos de IA.
- Skills.
- Agentes.
- Integrações.
- Componentes do portal.
- Arquivos e recursos do cliente.

Procure especificamente por:

- TODO.
- FIXME.
- HACK.
- XXX.
- Implementações incompletas.
- Código morto.
- Código duplicado.
- Imports não utilizados.
- Dependências desnecessárias.
- Funções sem utilização.
- Arquivos obsoletos.
- Configurações inconsistentes.
- Bugs documentados.
- Bugs conhecidos ainda não corrigidos.
- Workarounds temporários.
- Implementações que não seguem os padrões atuais.
- Funcionalidades planejadas, mas não implementadas.
- Ausência de tratamento de erros.
- Problemas de segurança.
- Falhas de observabilidade.
- Falhas de testes.
- Problemas de performance.
- Inconsistências entre cliente e servidor.

Tudo que for identificado e ainda não estiver registrado deverá ser transformado em issues detalhadas.

**Padrão obrigatório:** Utilize o padrão atual de abertura de issues do projeto. Antes de criar novas issues, estude os modelos existentes e mantenha sua estrutura, nomenclatura, detalhamento, organização e convenções.

Evite criar issues genéricas. Cada issue deve descrever claramente o problema, a justificativa, o escopo, as dependências, a solução esperada, os critérios de aceitação e a estratégia de testes.

## 3.5. Skills e instruções de implementação

Sempre leia as skills relacionadas às atividades que serão executadas.

Consulte, entre outras, as skills de:

- Desenvolvimento.
- Arquitetura.
- Qualidade de código.
- Testes.
- Backend.
- Frontend.
- Go.
- Next.js.
- Banco de dados.
- DevOps.
- CI/CD.
- Segurança.
- Documentação.
- Observabilidade.
- Engenharia reversa.
- Cliente do jogo.
- Agentes de IA.

Nas issues, sempre inclua referências às skills e aos documentos que orientam sua implementação.

O agente responsável pela execução deve conseguir identificar facilmente:

- Qual skill precisa consultar.
- Quais padrões devem ser seguidos.
- Quais convenções precisam ser respeitadas.
- Quais ADRs são relevantes.
- Quais restrições arquiteturais existem.
- Como validar a implementação.

Caso uma skill esteja desatualizada, incompleta ou não contemple determinada tecnologia, pesquise as boas práticas atuais e atualize-a.

## 3.6. Issues de REVIEW — documentos vivos

Identifique todas as issues de review existentes.

Elas são documentos vivos e representam a memória técnica das revisões realizadas no projeto.

Não trate essas issues como tarefas descartáveis ou revisões pontuais.

Elas devem registrar continuamente:

- O que já foi analisado.
- O que foi corrigido.
- O que ainda precisa ser analisado.
- O que ainda precisa ser corrigido.
- Problemas encontrados.
- Bugs identificados.
- Melhorias propostas.
- Inconsistências arquiteturais.
- Violações de padrões.
- Pendências técnicas.
- Novos pontos de investigação.
- Riscos identificados.

Preserve os apontamentos anteriores e seu histórico.

Sempre que identificar novos problemas ou oportunidades de melhoria durante uma revisão, registre-os nas respectivas issues de review.

Organize as reviews por área do projeto.

Caso alguma área ainda não possua uma issue de review, crie-a.

As issues de review devem servir como referência permanente para as próximas análises, evitando que problemas já conhecidos sejam esquecidos ou analisados repetidamente sem evolução.

---

# 4. FASE 2 — EXECUÇÃO DO ROADMAP E CICLO DE QUALIDADE

Após concluir a auditoria e a reorganização do roadmap, inicie a execução das issues.

## 4.1. Execução sequencial

Execute as issues respeitando:

- Prioridade.
- Dependências.
- Bloqueios.
- Sequência arquitetural.
- Impactos entre sistemas.
- Critérios de aceitação.

Priorize a eliminação de problemas estruturais, código legado, inconsistências arquiteturais e dependências desnecessárias antes de expandir funcionalidades.

Não execute issues bloqueadas sem resolver ou tratar previamente suas dependências.

Sempre consulte as skills e ADRs pertinentes antes de implementar.

## 4.2. Validação de cada implementação

Para cada issue:

1. Analise o escopo.
2. Consulte as skills e referências técnicas.
3. Examine a implementação atual.
4. Planeje a solução.
5. Implemente as alterações.
6. Execute os testes adequados.
7. Analise possíveis regressões.
8. Atualize a documentação.
9. Atualize a issue com as evidências.
10. Atualize as issues de review relacionadas.
11. Atualize o roadmap e seus indicadores.

Não marque uma issue como concluída enquanto houver critérios de aceitação pendentes.

## 4.3. Review a cada 10 issues concluídas

A cada 10 issues efetivamente fechadas, execute obrigatoriamente uma revisão técnica abrangente.

Utilize as skills de review, qualidade, arquitetura, testes e demais instruções aplicáveis.

A revisão deverá verificar:

- Aderência às regras do projeto.
- Adoção dos padrões arquiteturais.
- Conformidade com os ADRs.
- Qualidade do código.
- Organização dos diretórios.
- Consistência das implementações.
- Cobertura e qualidade dos testes.
- Atualização da documentação.
- Segurança.
- Performance.
- Observabilidade.
- Regressões.
- Pendências introduzidas durante a execução.

Atualize as issues de review com todos os novos apontamentos.

Caso sejam encontrados problemas, registre-os no roadmap e priorize sua correção.

A revisão não deve ser apenas uma conferência superficial: deve avaliar se as últimas 10 implementações mantiveram a qualidade e a coerência global do projeto.

---

# 5. FASE 3 — METAS ESTRATÉGICAS DE MODERNIZAÇÃO

As metas abaixo representam os resultados que devem orientar a reorganização e a execução do roadmap.

## 5.1. Modernização completa do servidor

**Objetivo:** Eliminar completamente o código legado do servidor.

- Remover integralmente implementações legadas.
- Remover código morto.
- Eliminar arquivos obsoletos.
- Eliminar dependências desnecessárias.
- Modernizar a infraestrutura.
- Melhorar a organização do código.
- Aplicar corretamente os padrões arquiteturais.
- Padronizar as implementações.
- Melhorar legibilidade e manutenibilidade.
- Garantir consistência entre os módulos.
- Eliminar workarounds e soluções temporárias desnecessárias.

A remoção do legado deve ser comprovada por análise de dependências, referências, funcionalidades e testes de regressão.

## 5.2. Eliminação da dependência de arquivos TXT e CSV

O servidor não deve mais depender de arquivos TXT ou CSV para consultar dados operacionais do jogo.

Objetivos:

- Identificar todas as consultas realizadas nesses arquivos.
- Mapear quais sistemas dependem deles.
- Identificar a origem e a finalidade de cada informação.
- Migrar os dados necessários para o banco de dados.
- Criar ou atualizar as estruturas necessárias.
- Implementar consultas utilizando o banco.
- Garantir integridade e consistência dos dados.
- Remover as dependências de leitura desses arquivos no servidor.
- Remover código, scripts e configurações obsoletas relacionados a essa abordagem.

Ao final, o servidor deve obter todas as informações operacionais necessárias diretamente do banco de dados.

Não confunda arquivos de dados operacionais com recursos estáticos necessários ao funcionamento do cliente.

## 5.3. Compatibilidade completa entre cliente e servidor

Faça uma análise integral de todos os arquivos do cliente e compare-os com os arquivos utilizados pelo servidor.

Investigue especialmente:

- Arquivos de mapas.
- Arquivos de configuração.
- Definições de itens.
- Recursos do jogo.
- Estruturas de dados.
- Identificadores.
- Tabelas e registros.
- Arquivos de eventos.
- Arquivos de NPCs.
- Arquivos de cidades.
- Outros recursos compartilhados.

Identifique:

- Arquivos duplicados.
- Arquivos divergentes.
- Versões diferentes.
- Arquivos desatualizados.
- Recursos utilizados apenas pelo cliente.
- Recursos utilizados apenas pelo servidor.
- Dependências implícitas.
- Incompatibilidades de estrutura ou conteúdo.

Crie um mapeamento completo de compatibilidade.

Sempre que possível, implemente mecanismos automatizados de validação para garantir que cliente e servidor permaneçam sincronizados.

---

# 6. FASE 4 — BANCO DE DADOS E ENRIQUECIMENTO DO CONHECIMENTO DO JOGO

## 6.1. Normalização e integridade

O banco de dados deve estar completamente normalizado, coerente e otimizado.

Realize uma auditoria de todas as tabelas, relacionamentos e estruturas.

Verifique:

- Normalização.
- Integridade referencial.
- Chaves primárias e estrangeiras.
- Índices.
- Constraints.
- Tipos de dados.
- Campos obrigatórios.
- Campos nulos indevidos.
- Registros duplicados.
- Registros incompletos.
- Dados incorretos.
- Relacionamentos inconsistentes.
- Performance das consultas.
- Estruturas redundantes.

Garanta que todos os dados estejam corretamente preenchidos, validados e organizados.

## 6.2. Migrations e seeds

Revise integralmente o sistema de migrations e seeds.

- Atualize migrations obsoletas.
- Corrija inconsistências.
- Remova estruturas desnecessárias.
- Garanta que a sequência de migrations seja coerente.
- Verifique a possibilidade de reconstrução completa do banco.
- Garanta consistência entre migrations e modelos.
- Atualize seeds.
- Garanta que os dados iniciais sejam suficientes e corretos.
- Documente os procedimentos de inicialização e atualização.

O banco deve ser independente e reproduzível em relação à estrutura e aos dados: deve ser possível reconstruí-lo de forma confiável utilizando migrations e seeds, sem depender de arquivos TXT ou CSV externos.

## 6.3. Enriquecimento dos registros de itens e skills

Todos os itens e skills cadastrados no banco de dados devem possuir informações descritivas suficientes para compreender sua finalidade, utilização e relação com os sistemas do jogo.

Não mantenha registros contendo apenas nome, identificador e descrição superficial.

Expanda o modelo de informações para permitir documentar:

**Itens:**

- Nome e identificação.
- Descrição detalhada.
- Finalidade.
- Categoria.
- Tipo.
- Sistemas em que pode ser utilizado.
- Funções desempenhadas.
- Formas de utilização.
- Requisitos.
- Restrições.
- Possibilidades de combinação.
- Possibilidades de transformação.
- Utilização em eventos.
- Utilização em trocas.
- Utilização em consumíveis.
- Utilização em evolução de personagens.
- Utilização em quests.
- Utilização em sistemas especiais.
- Relações com outros itens.
- Origem e obtenção.
- Comercialização.
- Utilização em sistemas de donate, quando aplicável.

Exemplo de enriquecimento:

A Pedra Ideal pode ser utilizada no sistema de evolução de personagens, incluindo a transformação de um personagem Arch em Celestial.

Esse exemplo não representa todos os usos do item. É necessário pesquisar e mapear suas demais aplicações.

Pesquise na internet, em fontes relacionadas ao WYD, para identificar as utilizações reais dos itens.

Não invente funcionalidades. Diferencie informações confirmadas, informações encontradas em fontes externas e hipóteses ainda não verificadas.

## 6.4. Expansão do modelo de conhecimento

Analise como o mesmo conceito de enriquecimento pode ser aplicado às demais entidades e tabelas do jogo.

Investigue especialmente:

- Skills.
- Sistemas.
- Quests.
- Eventos.
- NPCs.
- Monstros.
- Mapas.
- Cidades.
- Equipamentos.
- Itens especiais.
- Sistemas de evolução.
- Sistemas de troca.
- Sistemas de crafting.
- Sistemas de guerras.
- Sistemas de comércio.
- Outros sistemas existentes.

O objetivo é construir uma base de conhecimento estruturada, que permita entender como cada elemento do jogo funciona, onde é utilizado, quais sistemas afeta e quais dependências possui.

Proponha melhorias no modelo de dados e registre as alterações necessárias em issues.

---

# 7. FASE 5 — MODERNIZAÇÃO DE DEVOPS E CI/CD

## 7.1. Auditoria completa da infraestrutura

Revise toda a infraestrutura de desenvolvimento, execução, deploy e monitoramento.

Garanta que:

- A infraestrutura esteja coerente com a arquitetura.
- Os serviços estejam corretamente configurados.
- As dependências estejam atualizadas.
- Os containers estejam organizados.
- Os ambientes estejam padronizados.
- As configurações sejam centralizadas.
- Os serviços possuam health checks adequados.
- Os recursos estejam corretamente dimensionados.
- Os procedimentos de inicialização e encerramento sejam confiáveis.
- Não existam serviços desnecessários.
- Não existam configurações legadas.

## 7.2. Modernização completa do CI — GitHub Actions

Faça uma auditoria e modernização completa dos pipelines do GitHub Actions.

Pesquise profundamente as melhores práticas atuais da plataforma e atualize as skills relacionadas.

Objetivos:

- Separar os jobs por responsabilidade.
- Organizar corretamente os estágios do pipeline.
- Reutilizar workflows e actions quando apropriado.
- Evitar duplicação de configurações.
- Implementar estratégias adequadas de cache.
- Otimizar o empacotamento.
- Otimizar o tempo de execução.
- Organizar corretamente os artefatos.
- Implementar validações de qualidade.
- Implementar verificações de segurança.
- Executar os testes adequados.
- Garantir consistência entre ambientes.
- Utilizar corretamente os recursos modernos do GitHub Actions.
- Configurar permissões mínimas necessárias.
- Utilizar corretamente secrets e variáveis.
- Evitar exposição de informações sensíveis.
- Implementar estratégias adequadas de concorrência.
- Implementar cancelamento de execuções redundantes quando aplicável.
- Melhorar a rastreabilidade dos pipelines.

Avalie também:

- Matriz de versões.
- Jobs condicionais.
- Reutilização de workflows.
- Composite actions.
- Dependências entre jobs.
- Upload e download de artefatos.
- Cache de dependências.
- Cache de build.
- Estratégias de release.
- Versionamento.
- Publicação.
- Geração de relatórios.

Não adote recursos apenas por serem recentes. Avalie sua utilidade real para a arquitetura do projeto.

## 7.3. CI e validação final

Após as alterações:

- Execute os pipelines.
- Valide todos os jobs.
- Verifique permissões.
- Identifique warnings.
- Corrija falhas.
- Faça push das alterações no GitHub.
- Utilize GitHub CLI para consultar as execuções.
- Verifique o resultado de todos os workflows.
- Investigue eventuais falhas.
- Garanta que os pipelines estejam funcionando completamente.

Não considere o CI concluído apenas porque os arquivos YAML estão corretos localmente.

---

# 8. FASE 6 — OBSERVABILIDADE, MONITORAMENTO E SEGURANÇA

## 8.1. Prometheus e Grafana

Implemente e configure corretamente Prometheus e Grafana na infraestrutura.

Garanta:

- Coleta de métricas.
- Exportação de métricas.
- Configuração de targets.
- Dashboards.
- Alertas.
- Monitoramento dos serviços.
- Monitoramento do banco.
- Monitoramento da infraestrutura.
- Monitoramento do servidor.
- Monitoramento dos recursos computacionais.
- Identificação de falhas.
- Histórico de métricas.
- Visibilidade operacional.

As informações relevantes devem ser disponibilizadas no portal, permitindo visualizar o estado operacional do projeto.

## 8.2. Integração com o portal

O portal deve apresentar as informações disponíveis de monitoramento de forma clara e organizada.

Investigue quais métricas podem ser expostas com segurança para usuários e administradores.

Separe as informações públicas das informações operacionais restritas.

Implemente visualizações adequadas para:

- Estado dos serviços.
- Disponibilidade.
- Performance.
- Consumo de recursos.
- Métricas do servidor.
- Métricas do banco.
- Infraestrutura.
- Alertas.
- Histórico.
- Indicadores operacionais.

## 8.3. Análise de redundância e arquitetura de observabilidade

Investigue profundamente a relação entre:

- Prometheus.
- Grafana.
- `openwyd-logpipe`.
- `openwyd-node-exporter`.

Responda tecnicamente:

- Prometheus e Grafana substituem nosso sistema de logs `openwyd-logpipe`?
- Prometheus e Grafana substituem `openwyd-node-exporter`?
- Existe redundância entre esses componentes?
- Eles se complementam?
- Estamos aumentando desnecessariamente a complexidade?
- Existem riscos operacionais ou de manutenção?
- Quais componentes devemos manter?
- Quais podem ser removidos?
- Quais devem ser modernizados?
- Estamos utilizando a melhor arquitetura possível?
- Existem alternativas melhores?

Avalie as responsabilidades individuais de cada componente.

Não confunda coleta de métricas, armazenamento de métricas, visualização, exportação e gerenciamento de logs.

Apresente uma arquitetura recomendada, com justificativas técnicas, responsabilidades, fluxos de dados e possíveis simplificações.

## 8.4. Segurança

Faça uma revisão completa da segurança do projeto.

Investigue:

- Autenticação.
- Autorização.
- Gestão de permissões.
- Secrets.
- Variáveis de ambiente.
- Segurança das APIs.
- Segurança do banco.
- Segurança dos containers.
- Segurança dos pipelines.
- Controle de acesso.
- Exposição de serviços.
- Validação de entradas.
- Proteção de dados.
- Dependências vulneráveis.
- Configurações inseguras.
- Monitoramento de eventos de segurança.

Registre todas as vulnerabilidades e melhorias necessárias.

---

# 9. FASE 7 — PORTAL WEB COMPLETO E MODERNIZADO

## 9.1. Auditoria e modernização do portal

Realize uma análise completa da aplicação Next.js.

Revise:

- Arquitetura.
- Componentes.
- Páginas.
- Rotas.
- Layouts.
- Navegação.
- Estado.
- Integração com APIs.
- Autenticação.
- Autorização.
- Formulários.
- Validações.
- Tratamento de erros.
- Performance.
- Responsividade.
- Acessibilidade.
- Segurança.
- Testes.

Aplique boas práticas atuais de desenvolvimento Next.js, React, TypeScript e UI/UX.

## 9.2. Portal de usuário e administrador

O projeto deve possuir um portal completo, funcional e visualmente moderno.

Pesquise referências de portais de servidores privados de WYD e de outros MMORPGs.

Explore detalhadamente:

- Sites.
- Estrutura de navegação.
- Paths.
- Páginas.
- Áreas de usuário.
- Áreas administrativas.
- Lojas.
- Sistemas de contas.
- Notícias.
- Rankings.
- Informações do jogo.
- Painéis administrativos.
- Sistemas de gerenciamento.
- Experiência do usuário.

Utilize essas referências para identificar boas soluções e oportunidades de melhoria.

O portal deverá contemplar as funcionalidades necessárias para usuários e administradores.

## 9.3. Integração com MkDocs

Integre o portal com a documentação publicada pelo MkDocs.

Disponibilize acesso organizado a:

- Guias de usuários.
- Sistemas do jogo.
- Informações sobre guerras.
- Tutoriais.
- Manuais.
- Informações técnicas pertinentes aos usuários.
- Outros conteúdos necessários.

A integração deve ser coerente com a navegação e identidade visual do portal.

## 9.4. Identidade visual e UI/UX

Aplique a identidade visual em todas as páginas.

Garanta:

- Consistência visual.
- Componentes padronizados.
- Tipografia coerente.
- Paleta de cores.
- Temas.
- Iconografia.
- Hierarquia visual.
- Responsividade.
- Acessibilidade.
- Feedbacks visuais.
- Estados de carregamento.
- Estados vazios.
- Mensagens de erro.
- Confirmações.
- Navegação intuitiva.

O portal deve possuir informações claras, recursos visuais ricos, aparência moderna e excelente experiência de utilização.

Não mantenha páginas com layouts provisórios, funcionalidades fictícias ou componentes sem implementação real.

## 9.5. Mapa do jogo no portal

Modernize completamente o mapa do jogo apresentado no portal.

Pesquise bibliotecas modernas para renderização, desenho e interação com mapas.

Avalie alternativas adequadas à representação de mapas de MMORPGs.

Objetivos:

- Melhorar a qualidade gráfica.
- Melhorar a identificação das cidades.
- Melhorar a identificação de regiões.
- Melhorar a navegação.
- Melhorar a experiência visual.
- Implementar recursos interativos.
- Melhorar zoom e movimentação.
- Identificar locais importantes.
- Organizar camadas e informações.
- Permitir navegação intuitiva.

Investigue as possibilidades de utilização dos mapas e recursos gráficos originais do WYD.

## 9.6. Imagens, logos e artes

Faça um levantamento dos recursos visuais disponíveis.

O WYD possui uma biblioteca extensa de imagens e materiais gráficos espalhados pela internet.

Pesquise referências e recursos que possam ser utilizados no projeto.

Quando necessário:

- Localize imagens originais.
- Identifique logos.
- Identifique artes.
- Identifique ícones.
- Identifique mapas.
- Identifique recursos gráficos.
- Avalie a qualidade dos arquivos.
- Verifique compatibilidade e direitos de utilização.

Caso não existam referências adequadas, proponha ou crie novos recursos visuais.

Organize corretamente os assets no projeto.

## 9.7. Dados reais e funcionalidades completas

O portal deve possuir informações atualizadas, coerentes e dados reais cadastrados.

Não deixe funcionalidades importantes utilizando dados fictícios ou placeholders.

Garanta:

- Sistemas efetivamente implementados.
- Produtos reais cadastrados.
- Informações corretas.
- Dados atualizados.
- Integrações funcionais.
- Fluxos completos.
- Validações.
- Tratamento de erros.

### Loja de donate

Revise a loja de donate.

Pesquise referências de lojas de donate de outros servidores de WYD e MMORPGs.

Investigue nossa própria base de dados para identificar os itens que devem ser cadastrados na loja.

Objetivos:

- Identificar itens de donate.
- Mapear seus identificadores.
- Conferir descrições.
- Conferir imagens.
- Conferir categorias.
- Organizar os produtos.
- Definir informações comerciais necessárias.
- Garantir consistência com os dados do jogo.
- Preparar os produtos para utilização real.

Utilize referências externas para melhorar a apresentação e a organização da loja.

## 9.8. Testes E2E do portal

Todas as funcionalidades do portal devem possuir validação adequada.

Utilize Playwright para implementar testes E2E dos principais fluxos de usuário e administrador.

Cubra:

- Navegação.
- Autenticação.
- Cadastro.
- Recuperação de acesso, quando aplicável.
- Área do usuário.
- Área administrativa.
- Loja.
- Produtos.
- Sistemas.
- Formulários.
- Integrações.
- Permissões.
- Navegação da documentação.
- Funcionalidades do mapa.
- Fluxos operacionais.

Garanta que os testes validem o comportamento real da aplicação, não apenas a existência dos elementos visuais.

---

# 10. FASE 8 — CLIENTE WYD: ENGENHARIA REVERSA E MODERNIZAÇÃO

Esta fase deve ser preparada a partir do conhecimento consolidado durante a modernização do servidor.

Antes de realizar alterações extensas no cliente, finalize o mapeamento de compatibilidade entre cliente e servidor e estabeleça uma documentação técnica confiável.

## 10.1. Mapeamento da dependência ClientPatch_v7662

A dependência do cliente sobre `ClientPatch_v7662` precisa ser completamente mapeada.

Investigue:

- Origem da biblioteca.
- Finalidade.
- Funções.
- Dependências.
- Pontos de integração.
- Interfaces.
- Símbolos exportados.
- Funções utilizadas.
- Interações com o executável.
- Impactos sobre o funcionamento do cliente.

Existe a suspeita de que essa biblioteca tenha sido injetada no executável `wyd.exe`, possivelmente utilizando OllyDbg, e que tenha recebido hooks implementados por usuários há alguns anos.

Pesquise profundamente na internet referências sobre:

- `ClientPatch_v7662`.
- Injeção de DLL no `wyd.exe`.
- Hooks.
- Funcionamento da biblioteca.
- Modificações do cliente.
- Técnicas utilizadas pela comunidade WYD.

Investigue como a biblioteca é carregada, como se integra ao executável e quais funcionalidades modifica.

Documente exatamente onde o cliente depende dela e quais funcionalidades seriam afetadas por sua remoção ou substituição.

## 10.2. Engenharia reversa completa do cliente

Realize um levantamento técnico abrangente da estrutura interna do cliente.

Investigue:

- Executável principal.
- Bibliotecas.
- DLLs.
- Dependências externas.
- Comunicação com o servidor.
- Protocolos.
- Estruturas de pacotes.
- Arquivos de configuração.
- Recursos gráficos.
- Mapas.
- Texturas.
- Modelos.
- Animações.
- Interfaces.
- Sistemas de renderização.
- Áudio.
- Scripts.
- Sistemas de carregamento.
- Hooks.
- Extensões.
- Mecanismos de customização.

O objetivo é entender completamente como o cliente funciona internamente e identificar todas as possibilidades de modernização.

Mapeie o que pode ser:

- Modificado.
- Modernizado.
- Substituído.
- Estendido.
- Customizado.
- Automatizado.
- Melhorado visualmente.

## 10.3. Modernização da comunicação cliente/servidor

Mapeie completamente a comunicação atual.

Investigue:

- Protocolo.
- Formato dos pacotes.
- Serialização.
- Autenticação.
- Sessões.
- Criptografia.
- Compatibilidade.
- Dependências.
- Limitações.
- Performance.
- Segurança.

Analise se é possível modernizar a comunicação sem comprometer a compatibilidade com o jogo.

Pesquise alternativas e ferramentas.

Avalie se podemos desenvolver uma biblioteca própria para substituir ou complementar dependências existentes.

Investigue possibilidades de injetar uma dependência desenvolvida por nós, caso tecnicamente viável.

Apresente diferentes abordagens, com riscos, vantagens, limitações e recomendações.

## 10.4. Modernização visual e customização

Faça um mapeamento completo das possibilidades de evolução visual do cliente.

Investigue:

- Interface.
- HUD.
- Menus.
- Janelas.
- Elementos gráficos.
- Resolução.
- Escalabilidade.
- Texturas.
- Modelos.
- Animações.
- Efeitos.
- Mapas.
- Ícones.
- Fontes.
- Recursos visuais.
- Experiência de utilização.

Avalie o que pode ser modernizado sem reconstruir completamente o cliente.

Pesquise ferramentas, bibliotecas, técnicas e boas práticas de modificação de jogos legados.

Elabore um mapa completo das possibilidades de customização e modernização.

## 10.5. Launcher como ferramenta auxiliar

Planeje a criação de um launcher como ferramenta adicional dentro do CLI do projeto.

O launcher deverá ajudar a executar funções auxiliares relacionadas ao cliente.

Investigue e proponha funcionalidades como:

- Gerenciamento do cliente.
- Inicialização.
- Verificação de integridade.
- Atualização de arquivos.
- Verificação de versões.
- Sincronização de recursos.
- Diagnóstico.
- Configurações.
- Logs.
- Verificações de compatibilidade.
- Integração com os serviços do projeto.

Não implemente funcionalidades arbitrariamente. Faça primeiro o mapeamento de necessidades, arquitetura, possibilidades técnicas e dependências.

Crie o planejamento completo da implementação do launcher e registre as issues correspondentes.

---

# 11. FASE 9 — ORGANIZAÇÃO DOS ARQUIVOS DE IA E FREEBUFF

Garanta que o agente do FreeBuff utilize corretamente toda a estrutura de arquivos de IA existente no projeto.

Faça uma auditoria completa de:

- Arquivos de contexto.
- Instruções.
- Skills.
- Agentes.
- Templates.
- Prompts.
- Referências técnicas.
- Documentação.
- Configurações específicas do FreeBuff.

Verifique se os arquivos estão:

- Corretamente organizados.
- Padronizados.
- Atualizados.
- Sem duplicações desnecessárias.
- Coerentes com a arquitetura.
- Compatíveis com o funcionamento do FreeBuff.
- Corretamente referenciados.

Garanta que o agente saiba localizar e utilizar as informações necessárias antes de executar suas tarefas.

Mapeie eventuais limitações da estrutura atual e proponha melhorias.

---

# 12. FASE 10 — DOCUMENTAÇÃO E PADRONIZAÇÃO GLOBAL

## 12.1. Documentação completa

Mantenha toda a documentação atualizada durante a execução da missão.

Revise os documentos existentes e elimine informações obsoletas, inconsistentes ou duplicadas.

O projeto deve possuir documentação real, organizada e útil.

Separe claramente:

**Documentação oficial do MkDocs:**

- Manuais do projeto.
- Documentação técnica de desenvolvimento.
- Arquitetura.
- Diagramas.
- Sistemas do jogo.
- Guias de utilização.
- Manuais de usuários.
- Manuais de administradores.
- Infraestrutura.
- Procedimentos operacionais.
- Referências técnicas.

**Documentação interna de desenvolvimento:**

- Análises técnicas.
- Relatórios de auditoria.
- Investigações.
- Estudos.
- Reviews.
- Propostas de melhoria.
- Planejamentos.
- Documentos temporários.
- Relatórios de implementação.

O MkDocs deve conter apenas documentação oficial de utilização do projeto e documentação técnica de desenvolvimento.

Remova da estrutura oficial do MkDocs documentos gerados durante o desenvolvimento que não pertençam a essas categorias, como análises pontuais e relatórios de investigação.

Não descarte essas informações: organize-as em locais apropriados.

## 12.2. Padronização completa

O projeto deve ser completamente padronizado de ponta a ponta.

Revise:

- Estrutura de pastas.
- Nomenclatura.
- Organização dos módulos.
- Arquitetura.
- Código.
- Comentários.
- Docstrings.
- Logs.
- Tratamento de erros.
- Testes.
- Configurações.
- Infraestrutura.
- DevOps.
- Skills.
- Agentes.
- Documentação.
- Migrations.
- Seeds.
- Workflows.
- Scripts.
- Processos de desenvolvimento.

Aplique os padrões apropriados aos projetos Go e Next.js.

Pesquise profundamente as melhores práticas atuais de ambas as tecnologias.

Avalie as ferramentas, bibliotecas, padrões e arquiteturas mais adequados.

Atualize as skills correspondentes para refletir as práticas recomendadas e as convenções específicas do projeto.

Não padronize tecnologias diferentes utilizando regras inadequadas. Cada stack deve possuir suas próprias convenções, respeitando os princípios arquiteturais globais.

---

# 13. FASE 11 — DASHBOARD DO ROADMAP

Após a auditoria e reorganização das issues, atualize o índice principal do roadmap.

O índice deve funcionar como um dashboard completo de acompanhamento da modernização do projeto.

Não crie apenas uma lista de issues. Desenvolva uma visão gerencial e técnica do andamento de todas as metas.

## 13.1. Indicadores gerais

Apresente:

- Total de issues.
- Issues concluídas.
- Issues aguardando testes.
- Issues em andamento.
- Issues abertas.
- Issues bloqueadas.
- Issues obsoletas identificadas.
- Issues reabertas.
- Issues criadas durante a auditoria.
- Percentual de conclusão.
- Progresso por área.
- Progresso por fase.
- Pendências críticas.

## 13.2. Indicadores por área

Crie indicadores individuais para:

- Servidor.
- Cliente.
- Banco de dados.
- Infraestrutura.
- DevOps.
- CI/CD.
- Segurança.
- Observabilidade.
- Portal.
- Testes.
- Documentação.
- Skills e agentes.
- Engenharia reversa.
- Launcher.
- Modernização geral.

## 13.3. Apresentação visual

Utilize:

- Tabelas.
- Indicadores percentuais.
- Barras de progresso, quando suportadas pelo formato do roadmap.
- Status.
- Cores.
- Emojis.
- Agrupamentos.
- Links para as issues.
- Links para reviews.
- Links para documentação.

A apresentação deve ser intuitiva, organizada e de fácil compreensão.

Utilize uma legenda consistente para os estados.

Exemplo:

- 🟢 Concluído.
- 🔵 Em andamento.
- 🟡 Aguardando testes.
- 🔴 Bloqueado.
- ⚪ Pendente.
- 🟠 Necessita revisão.

## 13.4. Evolução contínua

O dashboard deve ser atualizado continuamente conforme as issues forem modificadas.

As métricas devem refletir o estado real das issues, sem percentuais inventados ou estimativas apresentadas como fatos.

Quando possível, automatize a geração dos indicadores a partir dos dados reais do roadmap.

O índice deve permitir identificar rapidamente:

- Onde estamos.
- O que já foi concluído.
- O que ainda falta.
- Quais são os principais bloqueios.
- Quais áreas estão atrasadas.
- Quais são as próximas prioridades.
- Como está evoluindo a modernização.

Inclua também uma seção específica com o escopo completo desta missão e suas metas estratégicas.

---

# 14. FASE 12 — ANÁLISE TÉCNICA E PROPOSTAS DE MELHORIA

Além das atividades obrigatórias, faça uma análise crítica de todo o projeto.

Não se limite às soluções atualmente utilizadas.

Investigue oportunidades de modernização, simplificação, otimização e evolução arquitetural.

Todas as propostas devem ser documentadas em um documento específico para análise e aprovação.

## 14.1. Observabilidade: Prometheus, Grafana e componentes próprios

Avalie profundamente:

- Prometheus.
- Grafana.
- openwyd-logpipe.
- openwyd-node-exporter.

Responda às questões:

1. Prometheus e Grafana substituem o openwyd-logpipe?
2. Prometheus e Grafana substituem o openwyd-node-exporter?
3. Quais funcionalidades são redundantes?
4. Quais funcionalidades são complementares?
5. Existe complexidade desnecessária?
6. Existem riscos de manutenção?
7. Quais componentes devemos manter?
8. Quais componentes devemos remover?
9. Quais componentes devemos modernizar?
10. Qual seria a arquitetura ideal para nosso projeto?

Apresente uma recomendação fundamentada.

## 14.2. Serviço db-roles

Analise a real necessidade de manter um serviço separado chamado `db-roles`.

Investigue:

- Qual responsabilidade ele possui.
- Quais operações executa.
- Quais dependências possui.
- Como é utilizado pela infraestrutura.
- Quais problemas resolve.
- Quais riscos evita.
- Se existe dependência de inicialização.
- Se pode ser substituído por migrations.
- Se as responsabilidades podem ser incorporadas a outro componente.

Responda:

1. Qual é a real necessidade de utilizar um serviço separado?
2. Isso poderia ser resolvido diretamente pelas migrations?
3. Precisamos realmente manter um serviço dedicado?
4. Por que essa arquitetura foi adotada?
5. Ela representa a melhor prática para nosso cenário?
6. Quais são as alternativas?
7. Quais melhorias podem ser implementadas?

Compare as alternativas considerando segurança, privilégios, inicialização, manutenção, complexidade e confiabilidade.

Não recomende simplesmente remover o serviço sem compreender suas responsabilidades e garantias de segurança.

## 14.3. Propostas gerais de melhoria

Analise o projeto completo e identifique oportunidades adicionais.

Investigue:

- Melhorias arquiteturais.
- Melhorias de performance.
- Melhorias de segurança.
- Simplificação de infraestrutura.
- Redução de dependências.
- Melhorias de automação.
- Modernização de ferramentas.
- Melhorias de desenvolvimento.
- Melhorias de observabilidade.
- Melhorias de UX.
- Melhorias de manutenção.
- Melhorias de escalabilidade.
- Melhorias no processo de release.
- Melhorias no fluxo de trabalho dos agentes de IA.

Crie um documento específico de propostas de melhoria.

Para cada proposta, apresente:

- Identificação.
- Área afetada.
- Problema atual.
- Justificativa.
- Solução proposta.
- Alternativas consideradas.
- Benefícios.
- Custos e complexidade.
- Riscos.
- Dependências.
- Prioridade.
- Impacto esperado.
- Necessidade de aprovação.

Separe claramente:

- Melhorias obrigatórias.
- Melhorias recomendadas.
- Melhorias opcionais.
- Melhorias experimentais.

Não implemente mudanças arquiteturais relevantes que dependam de aprovação sem antes documentar a proposta.

---

# 15. FASE 13 — PESQUISA TÉCNICA E ATUALIZAÇÃO DAS SKILLS

As decisões técnicas devem ser fundamentadas em boas práticas atuais.

Faça pesquisas profundas na internet sempre que a atividade envolver tecnologias, ferramentas, arquiteturas ou bibliotecas cuja evolução possa impactar a decisão.

Pesquise prioritariamente:

- Documentações oficiais.
- Repositórios oficiais.
- Guias de arquitetura.
- Recomendações dos mantenedores.
- Padrões reconhecidos.
- Projetos de referência.
- Estudos técnicos relevantes.
- Experiências reais de implementação.

Evite depender exclusivamente de artigos antigos, tutoriais desatualizados ou opiniões sem fundamentação.

Para cada área relevante:

1. Identifique as práticas atualmente adotadas.
2. Compare com as recomendações atuais.
3. Identifique divergências.
4. Avalie se a mudança é realmente vantajosa.
5. Registre as conclusões.
6. Atualize a skill correspondente.
7. Utilize a skill atualizada nas implementações seguintes.

As skills devem refletir tanto as boas práticas gerais das tecnologias quanto as decisões específicas do nosso projeto.

---

# 16. FASE 14 — REVISÃO TÉCNICA FINAL PRÉ-RELEASE

Após concluir as atividades previstas, execute uma auditoria técnica final antes da release.

Esta revisão deverá abranger o projeto inteiro.

## 16.1. Código e arquitetura

- Código legado.
- Código morto.
- Dependências.
- Organização.
- Padronização.
- Arquitetura.
- Qualidade.
- Tratamento de erros.
- Segurança.
- Performance.

## 16.2. Banco de dados

- Normalização.
- Integridade.
- Dados.
- Migrations.
- Seeds.
- Índices.
- Relacionamentos.
- Consistência.
- Independência de arquivos TXT e CSV.

## 16.3. Servidor e cliente

- Compatibilidade.
- Recursos compartilhados.
- Arquivos.
- Protocolos.
- Dependências.
- Sistemas.
- Funcionalidades.
- Integrações.

## 16.4. Portal

- Funcionalidades.
- Interface.
- Responsividade.
- Segurança.
- Acessibilidade.
- Integrações.
- Dados.
- Loja.
- Área administrativa.
- Mapa.
- Documentação.
- Testes E2E.

## 16.5. Infraestrutura

- Docker.
- Serviços.
- DevOps.
- CI/CD.
- GitHub Actions.
- Monitoramento.
- Prometheus.
- Grafana.
- Logs.
- Segurança.
- Configurações.

## 16.6. Testes

- Unitários.
- Integração.
- E2E.
- Regressão.
- Contrato.
- Segurança.
- Performance.

## 16.7. Documentação e governança

- MkDocs.
- Documentação técnica.
- Manuais.
- ADRs.
- Skills.
- Agentes.
- Issues.
- Reviews.
- Dashboard.
- Padronização.

## 16.8. Validação de release

- Execute todos os testes aplicáveis.
- Execute os pipelines.
- Corrija erros e warnings relevantes.
- Verifique o estado dos serviços.
- Valide migrations.
- Valide os processos de inicialização.
- Verifique a integridade dos dados.
- Verifique a compatibilidade.
- Faça push das alterações.
- Utilize GitHub CLI para acompanhar as execuções.
- Confirme que os workflows finalizaram com sucesso.
- Documente eventuais limitações ou pendências.

A release somente deverá ser considerada tecnicamente pronta quando os critérios de aceitação forem atendidos e as evidências estiverem registradas.

Não esconda pendências para apresentar um resultado artificialmente positivo.

---

# 17. ORDEM DE EXECUÇÃO ESTRATÉGICA

Organize o trabalho respeitando a seguinte sequência lógica:

| Fase | Atividade | Prioridade |
|---|---|---|
| 01 | Auditoria completa do roadmap | Crítica |
| 02 | Validação de issues done/ e test/ | Crítica |
| 03 | Varredura completa do projeto e identificação de gaps | Crítica |
| 04 | Reorganização das issues, reviews e dependências | Crítica |
| 05 | Atualização das skills e instruções | Alta |
| 06 | Auditoria arquitetural do servidor | Alta |
| 07 | Remoção de legado e modernização do servidor | Alta |
| 08 | Migração das dependências TXT/CSV para o banco | Alta |
| 09 | Normalização e enriquecimento do banco | Alta |
| 10 | Compatibilidade cliente/servidor | Alta |
| 11 | Modernização da infraestrutura e DevOps | Alta |
| 12 | Modernização do CI/CD | Alta |
| 13 | Observabilidade e segurança | Alta |
| 14 | Modernização e conclusão do portal | Alta |
| 15 | Documentação e integração MkDocs | Alta |
| 16 | Engenharia reversa do cliente | Média/Alta |
| 17 | Planejamento de modernização do cliente | Média/Alta |
| 18 | Planejamento e desenvolvimento do launcher | Média |
| 19 | Organização dos arquivos de IA e FreeBuff | Alta |
| 20 | Dashboard completo do roadmap | Alta |
| 21 | Propostas de melhoria e decisões arquiteturais | Contínua |
| 22 | Revisão técnica final pré-release | Crítica |

**Importante:** Esta tabela representa uma sequência estratégica, não uma autorização para ignorar dependências técnicas. Durante a auditoria, ajuste a ordem quando necessário, documentando os motivos.

Atividades independentes podem ser executadas em paralelo quando isso não comprometer a integridade do projeto.

---

# 18. CRITÉRIOS GERAIS DE CONCLUSÃO

A missão deverá perseguir os seguintes resultados:

1. Roadmap integralmente auditado e reorganizado.
2. Issues com estados reais e verificáveis.
3. Nenhuma pendência conhecida sem registro.
4. Issues de review funcionando como documentos vivos.
5. Skills atualizadas e referenciadas nas issues.
6. Servidor completamente livre de código legado desnecessário.
7. Servidor independente de consultas a arquivos TXT e CSV operacionais.
8. Compatibilidade cliente/servidor mapeada e validada.
9. Banco normalizado, íntegro, documentado e reproduzível.
10. Registros de itens e skills enriquecidos com informações reais.
11. Infraestrutura modernizada e padronizada.
12. CI/CD completo e funcional.
13. Observabilidade coerente, integrada e segura.
14. Portal de usuário e administrador completo.
15. Portal com dados reais e funcionalidades implementadas.
16. Mapa do jogo modernizado.
17. Testes unitários, integração e E2E organizados e funcionais.
18. Documentação oficial e interna corretamente separadas.
19. Cliente completamente mapeado para modernização.
20. Dependência ClientPatch_v7662 documentada.
21. Launcher planejado e integrado à estratégia do projeto.
22. Estrutura de IA e FreeBuff organizada.
23. Dashboard do roadmap atualizado.
24. Skills e boas práticas atualizadas.
25. Todas as áreas padronizadas.
26. Propostas de melhorias documentadas.
27. Revisão técnica pré-release concluída.

Os resultados devem ser comprovados por implementações, testes, documentação e evidências.

---

# 19. RELATÓRIO E ACOMPANHAMENTO CONTÍNUO

Durante toda a missão, mantenha o roadmap como fonte central de acompanhamento.

Atualize continuamente:

- Status das issues.
- Dependências.
- Prioridades.
- Reviews.
- Documentação.
- Skills.
- Indicadores.
- Dashboard.
- Evidências.
- Propostas.
- Decisões técnicas.

Ao concluir cada etapa relevante, registre um resumo contendo:

- O que foi analisado.
- O que foi implementado.
- O que foi corrigido.
- Quais issues foram concluídas.
- Quais issues foram criadas.
- Quais problemas foram encontrados.
- Quais testes foram executados.
- Quais resultados foram obtidos.
- Quais pendências permanecem.
- Quais são as próximas atividades.

Mantenha o histórico técnico para que qualquer agente consiga compreender o estado atual do projeto e continuar o trabalho sem perder conhecimento.

---

# 20. DIRETRIZ FINAL

Esta não é uma missão de simplesmente reorganizar arquivos ou fechar issues.

É uma missão de engenharia, auditoria, modernização, padronização e evolução completa do projeto.

Atue como um engenheiro de software responsável pela qualidade técnica global da solução, considerando arquitetura, manutenção, segurança, performance, experiência do usuário, infraestrutura e sustentabilidade do desenvolvimento.

Seja crítico com as implementações existentes. Questione decisões arquiteturais quando necessário. Pesquise alternativas. Identifique problemas que não foram percebidos anteriormente.

Não introduza complexidade sem justificativa.

Não mantenha código legado apenas porque já funciona.

Não implemente soluções improvisadas quando existe uma abordagem tecnicamente superior e viável.

Não crie funcionalidades sem compreender suas relações com o restante do sistema.

Não considere uma tarefa concluída sem validação.

Não deixe documentação desatualizada.

Não deixe conhecimento importante restrito ao histórico de conversas dos agentes.

**O objetivo final é entregar um projeto tecnicamente consistente, moderno, organizado, seguro, totalmente documentado, testável, observável, funcional e preparado para uma release de qualidade profissional.**