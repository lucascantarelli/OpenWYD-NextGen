# ANÁLISE, REORGANIZAÇÃO DO ROADMAP E MODERNIZAÇÃO COMPLETA DO PROJETO

## 1. IDENTIFICAÇÃO DA MISSÃO

**Nome:** Análise de Roadmap, Auditoria Técnica e Modernização Completa do Projeto.

**Objetivo principal:** Realizar uma auditoria técnica abrangente de todo o projeto, revisar e reorganizar integralmente o roadmap, identificar todas as pendências, inconsistências, problemas arquiteturais e oportunidades de melhoria e, posteriormente, executar as atividades necessárias para alcançar um estado de excelência técnica, funcional e operacional
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

# 1. DIRETRIZES GERAIS E REGRAS DE EXECUÇÃO

## 1.1. Análise antes da implementação

Antes de modificar qualquer componente:

- Analise a estrutura atual do projeto.
- Entenda a arquitetura e as dependências existentes.
- Identifique implementações concluídas, incompletas, redundantes ou obsoletas.
- Investigue o funcionamento real dos componentes antes de propor substituições.
- Consulte a documentação oficial das tecnologias utilizadas.
- Avalie alternativas modernas e tecnicamente adequadas.
- Identifique impactos e riscos de cada alteração.
- Preserve funcionalidades que já estejam corretas.

Não implemente mudanças apenas por preferência pessoal ou por considerar uma tecnologia mais moderna.

Toda decisão deve ter justificativa técnica.

## 1.2. Preservação do conhecimento

O projeto contém conhecimento acumulado em código, Issues, documentação, scripts, arquivos de configuração, implementações existentes e engenharia reversa.

Esse conhecimento não pode ser perdido durante a reorganização.

Antes de remover, substituir ou consolidar qualquer implementação:

- Verifique sua finalidade.
- Identifique suas dependências.
- Verifique se existem funcionalidades que não estão documentadas.
- Preserve informações relevantes.
- Registre decisões e mudanças arquiteturais.
- Atualize a documentação correspondente.

Não exclua arquivos, funcionalidades ou implementações sem compreender suas responsabilidades.

## 1.3. Rastreabilidade

Toda atividade relevante deve possuir rastreabilidade.

Relacionar:

- Problema identificado.
- Issue correspondente.
- Diagnóstico técnico.
- Solução proposta.
- Arquivos e componentes afetados.
- Implementação realizada.
- Testes executados.
- Resultado da validação.
- Documentação atualizada.

Quando uma atividade não possuir Issue, avalie se é necessário criar uma.

## 1.4. Autonomia com responsabilidade

Trabalhe com autonomia para:

- Investigar.
- Auditar.
- Comparar tecnologias.
- Identificar problemas.
- Propor soluções.
- Reorganizar documentação.
- Criar Issues.
- Implementar melhorias de baixo risco.
- Executar testes.
- Corrigir problemas identificados.

Entretanto, não execute alterações destrutivas ou de alto impacto sem avaliação adequada.

Exemplos de ações que exigem cuidado especial:

- Exclusão de dados.
- Alterações irreversíveis no banco.
- Mudanças incompatíveis no protocolo cliente-servidor.
- Alterações em sistemas de autenticação.
- Mudanças em produção.
- Remoção de funcionalidades existentes.
- Alterações que possam prejudicar personagens, itens ou progressão dos jogadores.

Quando necessário, apresente alternativas e solicite aprovação antes de prosseguir.

## 1.5. Qualidade técnica

Todas as implementações devem priorizar:

- Arquitetura clara.
- Baixo acoplamento.
- Alta coesão.
- Responsabilidade única.
- Reutilização adequada.
- Tipagem e contratos explícitos.
- Tratamento consistente de erros.
- Segurança.
- Observabilidade.
- Testabilidade.
- Documentação.
- Manutenibilidade.
- Compatibilidade entre componentes.

Evite soluções improvisadas, duplicação de código, dependências desnecessárias e complexidade sem justificativa.

---

# FASE 1 — AUDITORIA COMPLETA DO ROADMAP E ORGANIZAÇÃO DAS ISSUES

## 1.1. Auditoria do estado atual

Realize uma análise completa do roadmap existente.

Examine:

- Todas as Issues abertas.
- Issues fechadas.
- Issues em andamento.
- Milestones.
- Labels.
- Dependências.
- Projetos e quadros.
- Documentação relacionada.
- Diretórios utilizados para armazenar Issues concluídas ou testes.

Analise especialmente os diretórios:

- `done/`
- `test/`

Verifique se as Issues nesses diretórios realmente representam atividades concluídas, testes realizados, experimentos, implementações temporárias ou atividades que ainda precisam ser incorporadas ao roadmap.

Não considere uma Issue concluída apenas porque foi movida para `done/`.

Verifique se existe evidência concreta da implementação e validação.

## 1.2. Reorganização das Issues

Revise a organização atual e identifique:

- Issues duplicadas.
- Issues obsoletas.
- Issues sem objetivo claro.
- Issues excessivamente abrangentes.
- Issues que misturam responsabilidades.
- Issues sem critérios de aceitação.
- Issues sem dependências identificadas.
- Issues que deveriam ser divididas.
- Issues que deveriam ser agrupadas.
- Issues concluídas que ainda estão abertas.
- Issues abertas que já foram implementadas.
- Issues que precisam ser reabertas.

Organize as Issues por domínio técnico e prioridade.

A estrutura deve facilitar a identificação do estado real do projeto e permitir acompanhar sua evolução.

## 1.3. Identificação de lacunas

Faça uma varredura completa no projeto procurando:

- TODO.
- FIXME.
- XXX.
- HACK.
- Implementações temporárias.
- Código morto.
- Funcionalidades incompletas.
- Tratamentos de erro ausentes.
- Dependências desnecessárias.
- Problemas arquiteturais.
- Problemas de segurança.
- Falhas de testes.
- Débitos técnicos.
- Documentação desatualizada.

Cruze os resultados com as Issues existentes.

Não crie Issues duplicadas para problemas que já estejam adequadamente registrados.

## 1.4. Skills e agentes de IA

Analise as Skills, instruções, agentes e configurações utilizados pelo projeto.

Verifique:

- Se estão atualizados.
- Se possuem responsabilidades claras.
- Se existem instruções conflitantes.
- Se existem regras duplicadas.
- Se faltam conhecimentos especializados.
- Se os agentes conseguem compreender a arquitetura real.
- Se as Skills refletem as tecnologias e convenções atuais.

Crie ou atualize Skills especializadas quando necessário.

## 1.5. Issues de revisão permanentes

Estabeleça Issues de acompanhamento contínuo para os principais domínios.

Essas Issues devem funcionar como pontos de controle técnico, permitindo registrar:

- Evolução.
- Problemas encontrados.
- Pendências.
- Débitos técnicos.
- Melhorias futuras.
- Resultados de auditorias.
- Decisões arquiteturais.

Não feche essas Issues apenas porque uma rodada de revisão terminou.

Elas devem continuar vivas enquanto o domínio correspondente precisar de acompanhamento.

## 1.6. Entregáveis

Ao finalizar esta fase, apresente:

1. Diagnóstico completo do roadmap.
2. Inventário de Issues.
3. Lista de duplicidades e inconsistências.
4. Lista de Issues concluídas que precisam ser regularizadas.
5. Lista de lacunas identificadas.
6. Proposta de reorganização.
7. Nova estrutura de prioridades.
8. Plano de execução.
9. Issues de revisão contínua.
10. Atualização das Skills necessárias.

---

# FASE 2 — EXECUÇÃO CONTROLADA E EVOLUÇÃO SEQUENCIAL DO ROADMAP

Após a auditoria e reorganização, inicie a execução das atividades.

## 2.1. Execução por Issue

Trabalhe preferencialmente uma Issue por vez.

Para cada Issue:

1. Leia integralmente a descrição.
2. Analise o contexto técnico.
3. Identifique dependências.
4. Inspecione o código relacionado.
5. Confirme o problema.
6. Defina a abordagem de implementação.
7. Implemente a solução.
8. Execute os testes necessários.
9. Revise a implementação.
10. Atualize a documentação.
11. Atualize a Issue.
12. Registre o resultado.

Não marque uma Issue como concluída sem validação.

## 2.2. Revisões periódicas

A cada 10 Issues concluídas, execute uma revisão técnica consolidada.

Avalie:

- Qualidade das implementações.
- Consistência arquitetural.
- Regressões.
- Cobertura de testes.
- Débitos técnicos introduzidos.
- Documentação.
- Integração entre componentes.
- Necessidade de reorganizar o roadmap.

A revisão deve identificar problemas antes que eles se propaguem para as próximas atividades.

Se forem encontrados problemas relevantes, corrija-os antes de continuar.

## 2.3. Priorização

A ordem de execução deve considerar:

- Dependências técnicas.
- Criticidade.
- Risco.
- Impacto nos jogadores.
- Impacto operacional.
- Segurança.
- Complexidade.
- Ganhos estruturais.
- Possibilidade de desbloquear outras atividades.

Não execute atividades apenas pela ordem em que aparecem no GitHub.

## 2.4. Atualização contínua

Mantenha o roadmap sincronizado com a realidade do projeto.

Quando uma implementação revelar novos problemas:

- Registre o problema.
- Verifique se já existe Issue.
- Crie uma nova Issue quando necessário.
- Identifique dependências.
- Atualize as prioridades.

---

# FASE 3 — MODERNIZAÇÃO E PROFISSIONALIZAÇÃO DO SERVIDOR

Realize uma auditoria completa da arquitetura do servidor WYD.

## 3.1. Arquitetura

Avalie:

- Organização dos módulos.
- Responsabilidades.
- Dependências.
- Acoplamento.
- Fluxos de execução.
- Concorrência.
- Gerenciamento de memória.
- Gerenciamento de conexões.
- Tratamento de erros.
- Ciclo de vida dos serviços.
- Configurações.
- Logs.
- Comunicação entre componentes.

Identifique oportunidades de modernização sem comprometer o funcionamento do jogo.

## 3.2. Código legado

Procure:

- Código duplicado.
- Funções excessivamente grandes.
- Estruturas de dados inadequadas.
- Dependências desnecessárias.
- Implementações obsoletas.
- Tratamentos inconsistentes.
- Código morto.
- Acoplamento excessivo.
- Responsabilidades misturadas.

Avalie refatorações progressivas.

Não faça reescritas completas sem demonstrar que são necessárias.

## 3.3. Arquivos TXT e CSV

Analise a dependência do servidor em relação a arquivos TXT e CSV.

Identifique quais arquivos são utilizados para:

- Configuração.
- Dados de itens.
- Skills.
- NPCs.
- Mobs.
- Quests.
- Balanceamento.
- Sistemas auxiliares.
- Dados persistentes.
- Informações estáticas.

Avalie a migração dos dados que precisam de gerenciamento centralizado para soluções mais adequadas, especialmente banco de dados.

Não confunda arquivos de dados operacionais com recursos estáticos necessários ao cliente.

O objetivo é eliminar dependências inadequadas, não remover arquivos que sejam parte legítima do funcionamento do jogo.

## 3.4. Compatibilidade

Todas as modernizações devem considerar:

- Compatibilidade com o cliente.
- Persistência dos dados.
- Integridade dos personagens.
- Sistemas de combate.
- Itens.
- Skills.
- NPCs.
- Mobs.
- Quests.
- Eventos.
- Sistemas de progressão.

Identifique as mudanças que exigem atualização simultânea do cliente e servidor.

---

# FASE 4 — MODERNIZAÇÃO E EVOLUÇÃO DO BANCO DE DADOS

Realize uma auditoria completa da estrutura atual do banco de dados.

## 4.1. Modelagem

Analise:

- Tabelas.
- Colunas.
- Relacionamentos.
- Chaves primárias.
- Chaves estrangeiras.
- Índices.
- Constraints.
- Tipos de dados.
- Normalização.
- Redundâncias.
- Integridade referencial.
- Consultas.
- Performance.
- Estratégias de persistência.

Identifique problemas de modelagem e oportunidades de otimização.

## 4.2. Migrações

Estabeleça uma estratégia consistente para:

- Migrations.
- Versionamento de schema.
- Seeds.
- Dados iniciais.
- Atualizações.
- Rollback.
- Reprodutibilidade.
- Ambientes de desenvolvimento e produção.

Evite scripts manuais e alterações de schema sem rastreabilidade.

## 4.3. Enriquecimento dos dados do jogo

Analise os dados existentes e identifique oportunidades de enriquecimento.

O banco deve representar adequadamente os sistemas do jogo.

Considere:

- Itens.
- Equipamentos.
- Armas.
- Armaduras.
- Acessórios.
- Pedras.
- Refinamentos.
- Skills.
- Magias.
- Classes.
- Personagens.
- NPCs.
- Mobs.
- Quests.
- Eventos.
- Mapas.
- Sistemas de progressão.
- Sistemas de fabricação.
- Sistemas de evolução.
- Regras de combate.

Exemplo: analisar a estrutura de evolução de uma Pedra Ideal, desde Arch até Celestial, verificando atributos, requisitos, modificadores e regras de transformação.

Não limite o enriquecimento a esse exemplo.

Investigue todos os sistemas relevantes.

## 4.4. Consistência

Verifique se os dados:

- São coerentes com as regras do jogo.
- Estão completos.
- Não possuem duplicidades indevidas.
- Possuem relacionamentos corretos.
- São compatíveis com o servidor.
- São compatíveis com o cliente.
- Podem ser utilizados por ferramentas externas e pelo portal.

---

# FASE 5 — MODERNIZAÇÃO DO DEVOPS E CI/CD

Realize uma auditoria completa dos processos de build, integração, validação e entrega.

## 5.1. GitHub Actions

Analise todos os workflows existentes.

Verifique:

- Estrutura.
- Duplicidades.
- Jobs.
- Stages.
- Dependências.
- Condições de execução.
- Estratégias de cache.
- Artifacts.
- Reutilização de workflows.
- Paralelismo.
- Tempo de execução.
- Segurança.
- Permissões.
- Secrets.
- Variáveis.
- Versionamento de Actions.

Reorganize os pipelines para que sejam previsíveis, eficientes e fáceis de manter.

## 5.2. Estratégia de pipelines

Defina uma estrutura adequada para:

- Lint.
- Formatação.
- Análise estática.
- Build.
- Testes unitários.
- Testes de integração.
- Testes E2E.
- Security scanning.
- Análise de dependências.
- Validação de migrations.
- Empacotamento.
- Publicação.
- Deploy.

Não execute tarefas redundantes em diferentes workflows sem necessidade.

## 5.3. Segurança do CI/CD

Avalie:

- Permissões mínimas.
- Proteção de branches.
- Secrets.
- Tokens.
- Dependências de terceiros.
- Supply chain.
- Proveniência dos artifacts.
- Scans de vulnerabilidade.
- Proteção contra exposição de credenciais.

## 5.4. Git e GitHub CLI

Analise a utilização de Git e GitHub CLI.

Melhore os processos de:

- Criação de Issues.
- Atualização de Issues.
- Pull Requests.
- Releases.
- Labels.
- Milestones.
- Automação de tarefas.
- Consulta do estado do roadmap.

## 5.5. Resultado esperado

O CI/CD deve fornecer confiança sobre a qualidade do código e impedir que implementações inconsistentes avancem para os próximos estágios.

---

# FASE 6 — OBSERVABILIDADE, MONITORAMENTO E SEGURANÇA

Realize uma revisão completa da infraestrutura de observabilidade e segurança.

## 6.1. Observabilidade

Avalie a arquitetura atual e as ferramentas existentes.

Considere:

- Prometheus.
- Grafana.
- Logs centralizados.
- Métricas.
- Traces.
- Alertas.
- Health checks.
- Monitoramento de infraestrutura.
- Monitoramento dos serviços.
- Monitoramento do banco.
- Monitoramento do servidor do jogo.
- Monitoramento de aplicações auxiliares.

Avalie especialmente os componentes:

- `openwyd-logpipe`
- `openwyd-node-exporter`

Investigue se existem responsabilidades redundantes, sobreposição de funcionalidades ou oportunidades de consolidação.

Não remova componentes sem compreender suas funções e dependências.

## 6.2. Métricas

Identifique métricas relevantes para:

- Disponibilidade.
- CPU.
- Memória.
- Disco.
- Rede.
- Conexões.
- Banco de dados.
- Consultas.
- Latência.
- Erros.
- Processamento.
- Filas.
- Sessões.
- Jogadores conectados.
- Atividade do servidor.

Diferencie métricas técnicas, operacionais e métricas relacionadas ao jogo.

## 6.3. Segurança

Faça uma auditoria abrangente.

Avalie:

- Autenticação.
- Autorização.
- Gerenciamento de usuários.
- Gerenciamento de permissões.
- Banco de dados.
- Credenciais.
- Segredos.
- APIs.
- Comunicação.
- Configurações.
- Dependências.
- Containers.
- Infraestrutura.
- CI/CD.
- Superfície de ataque.
- Proteção contra abuso.
- Auditoria de operações administrativas.

## 6.4. db-roles e migrations

Investigue a relação entre `db-roles` e o mecanismo de migrations.

Determine:

- Responsabilidades de cada componente.
- Possíveis sobreposições.
- Dependências.
- Estratégia correta de permissões.
- Separação entre usuários administrativos e usuários de aplicação.
- Estratégia de execução das migrations.

Proponha uma arquitetura coerente e segura.

---

# FASE 7 — PORTAL WEB, EXPERIÊNCIA DO USUÁRIO E ECOSSISTEMA DIGITAL

Realize uma auditoria completa do portal web desenvolvido com Next.js.

O objetivo é transformar o portal em uma aplicação profissional, integrada ao ecossistema do jogo.

## 7.1. Arquitetura Next.js

Analise:

- Estrutura de diretórios.
- Organização de componentes.
- Rotas.
- Layouts.
- Server Components.
- Client Components.
- API.
- Autenticação.
- Autorização.
- Estado.
- Cache.
- Performance.
- Acessibilidade.
- Segurança.
- Responsividade.

Proponha melhorias estruturais e implemente-as de forma consistente.

## 7.2. Portal do usuário

Avalie e desenvolva as funcionalidades necessárias para os jogadores.

Considere:

- Cadastro.
- Login.
- Perfil.
- Conta.
- Personagens.
- Informações do jogo.
- Notícias.
- Eventos.
- Rankings.
- Histórico.
- Informações de itens.
- Skills.
- Guias.
- Suporte.
- Área do jogador.

As funcionalidades devem utilizar dados reais sempre que houver integração disponível.

## 7.3. Portal administrativo

Analise e melhore a área administrativa.

Considere:

- Gestão de usuários.
- Gestão de personagens.
- Gestão de conteúdo.
- Gestão de itens.
- Gestão de eventos.
- Gestão de notícias.
- Monitoramento.
- Logs.
- Auditoria.
- Métricas.
- Configurações.

As operações administrativas devem respeitar permissões e registrar ações relevantes.

## 7.4. Identidade visual

Revise a identidade visual do portal.

Analise:

- Paleta de cores.
- Tipografia.
- Componentes.
- Ícones.
- Espaçamentos.
- Grid.
- Layouts.
- Temas.
- Dark mode.
- Light mode.
- Responsividade.
- Consistência visual.

Estabeleça um Design System reutilizável.

## 7.5. Mapa do jogo

Analise a implementação do mapa.

Verifique:

- Estrutura.
- Dados geográficos.
- Mapas disponíveis.
- Regiões.
- Coordenadas.
- NPCs.
- Mobs.
- Pontos de interesse.
- Rotas.
- Interações.
- Performance.

Avalie como integrar os dados do servidor e do banco ao mapa.

## 7.6. Imagens, logos e artes

Faça um levantamento dos assets utilizados pelo portal.

Identifique:

- Imagens ausentes.
- Imagens inadequadas.
- Logos.
- Ícones.
- Artes.
- Banners.
- Imagens de itens.
- Imagens de NPCs.
- Imagens de mobs.
- Elementos visuais do jogo.

Organize os assets e padronize sua utilização.

Não utilize imagens aleatórias ou produtos fictícios como se fossem recursos oficiais.

## 7.7. Loja e sistema de donate

Revise a loja e o sistema de donate.

Avalie:

- Produtos.
- Pacotes.
- Preços.
- Moedas.
- Entrega de benefícios.
- Integração com o jogo.
- Histórico de compras.
- Segurança.
- Auditoria.
- Fluxo de pagamento.
- Confirmação de transações.
- Prevenção contra duplicidade.
- Tratamento de falhas.

A entrega de produtos deve ser rastreável e consistente com o estado do personagem e da conta.

## 7.8. Testes E2E com Playwright

Implemente ou modernize testes E2E para os principais fluxos.

Considere:

- Cadastro.
- Login.
- Logout.
- Recuperação de conta.
- Navegação.
- Perfil.
- Área administrativa.
- Loja.
- Donate.
- Gestão de conteúdo.
- Permissões.
- Responsividade.
- Fluxos críticos.

---

# FASE 8 — ENGENHARIA REVERSA E MODERNIZAÇÃO DO CLIENTE WYD

Realize uma investigação aprofundada do cliente do jogo.

Esta fase trata especificamente do funcionamento interno do cliente, dos executáveis, das bibliotecas, dos recursos visuais e da comunicação com o servidor.

A matemática e o balanceamento das regras do jogo serão tratados separadamente na FASE 10.

## 8.1. Auditoria do cliente

Investigue:

- Executáveis.
- DLLs.
- Bibliotecas.
- Arquivos de configuração.
- Assets.
- Estrutura de diretórios.
- Recursos gráficos.
- Scripts.
- Protocolos.
- Inicialização.
- Atualização.
- Comunicação.
- Dependências externas.

Documente a arquitetura identificada.

## 8.2. ClientPatch_v7662

Investigue especificamente o `ClientPatch_v7662`.

Verifique:

- Estrutura.
- Funcionalidades.
- Alterações introduzidas.
- Compatibilidade.
- Dependências.
- Integração com o cliente.
- Mecanismos de carregamento.
- Possíveis problemas.

Investigue a hipótese de utilização de DLL Injection no `wyd.exe`.

Avalie ferramentas de análise e depuração, incluindo OllyDbg, quando tecnicamente apropriado.

Não assuma que a hipótese de DLL Injection está correta: comprove o comportamento por meio de análise.

## 8.3. Engenharia reversa

Investigue o funcionamento do cliente para compreender:

- Inicialização.
- Carregamento de recursos.
- Renderização.
- Interface.
- Eventos.
- Processamento de pacotes.
- Comunicação com o servidor.
- Tratamento de erros.
- Sistemas de atualização.
- Integração com bibliotecas.

Preserve evidências, documentação e resultados dos experimentos.

## 8.4. Comunicação cliente-servidor

Mapeie a comunicação entre cliente e servidor.

Identifique:

- Estrutura dos pacotes.
- Identificadores.
- Campos.
- Tipos.
- Tamanhos.
- Codificação.
- Serialização.
- Fluxos de requisição e resposta.
- Estados.
- Validações.
- Compatibilidade de versões.

Documente os contratos identificados.

Não altere protocolos sem avaliar o impacto sobre clientes existentes.

A investigação matemática dos dados transmitidos, dos atributos e das regras de combate será aprofundada na FASE 10.

## 8.5. Personalizações visuais

Avalie possibilidades de customização:

- Interface.
- HUD.
- Menus.
- Fontes.
- Ícones.
- Texturas.
- Sprites.
- Animações.
- Efeitos.
- Recursos gráficos.

Identifique o que pode ser modificado sem comprometer a estabilidade ou compatibilidade.

## 8.6. Launcher

Revise o launcher do jogo.

Avalie a possibilidade de organizá-lo como uma ferramenta independente, com interface CLI e arquitetura reutilizável.

Considere:

- Inicialização.
- Atualização.
- Verificação de arquivos.
- Integridade.
- Versionamento.
- Download.
- Configurações.
- Logs.
- Diagnóstico.
- Compatibilidade.
- Segurança.

---

# FASE 9 — INTELIGÊNCIA ARTIFICIAL, FREEBUFF, SKILLS E AGENTES

Realize uma revisão completa da estrutura de IA e das ferramentas auxiliares do projeto.

## 9.1. Estrutura de arquivos de IA

Analise todos os arquivos relacionados a:

- Agents.
- Skills.
- Instructions.
- Prompts.
- Templates.
- Configurações.
- Ferramentas.
- Contextos.
- Documentação.
- Automação.

Identifique duplicidades, inconsistências e instruções obsoletas.

## 9.2. Skills especializadas

Organize Skills por domínio.

Considere especializações para:

- Arquitetura do servidor.
- Banco de dados.
- DevOps.
- CI/CD.
- Observabilidade.
- Segurança.
- Next.js.
- Portal.
- Engenharia reversa.
- Cliente WYD.
- Matemática do jogo.
- Combate e balanceamento.
- Testes.
- Documentação.
- Release engineering.

As Skills devem possuir instruções objetivas, contexto suficiente e critérios de qualidade.

## 9.3. Agentes especializados

Avalie a criação ou reorganização de agentes especializados.

Cada agente deve possuir:

- Responsabilidade definida.
- Escopo.
- Contexto.
- Ferramentas permitidas.
- Regras de execução.
- Critérios de validação.
- Limites de atuação.

Evite agentes redundantes ou com responsabilidades conflitantes.

## 9.4. FreeBuff

Analise a compatibilidade e a organização dos recursos relacionados ao FreeBuff.

Investigue:

- Arquivos.
- Configurações.
- Funcionalidades.
- Dependências.
- Integração.
- Compatibilidade.
- Manutenção.

Preserve o conhecimento existente e identifique oportunidades de melhoria.

## 9.5. Padronização

Estabeleça uma estrutura consistente para:

- Prompts.
- Skills.
- Agentes.
- Templates.
- Instruções.
- Documentação.
- Versionamento.
- Manutenção.

---

# FASE 10 — ENGENHARIA REVERSA MATEMÁTICA DO WYD/WYD2, BUG HUNTING E BALANCEAMENTO COMPLETO

## 10.1. Objetivo geral

Realize uma investigação matemática e funcional completa do WYD, incluindo as particularidades do WYD2 e das versões relacionadas que sejam relevantes para o projeto.

O objetivo é compreender como o jogo realmente funciona, identificar as fórmulas utilizadas, encontrar bugs matemáticos e inconsistências, validar os cálculos e propor um modelo de balanceamento tecnicamente fundamentado.

Esta fase deve ir além da simples leitura do código.

É necessário combinar:

- Análise do código-fonte disponível.
- Engenharia reversa.
- Análise de dados.
- Pesquisa histórica.
- Documentação.
- Testes controlados.
- Simulações matemáticas.
- Observação do comportamento real do jogo.
- Comparação entre cliente e servidor.

Não presuma que as fórmulas atuais estejam corretas apenas porque estão implementadas há muito tempo.

## 10.2. Pesquisa profunda sobre WYD e WYD2

Pesquise referências disponíveis na internet sobre:

- WYD.
- WYD2.
- Versões antigas.
- Versões posteriores.
- Mecânicas de combate.
- Atributos.
- Classes.
- Builds.
- Skills.
- Magias.
- Itens.
- Equipamentos.
- Velocidade.
- Crítico.
- Dano.
- Defesa.
- Resistências.
- NPCs.
- Mobs.
- Sistemas de evolução.
- Sistemas do jogo
- Sistemas de guerra
- Sistemas de cidades
- Sistemas de guild
- Sistemas de quests
- Sistemas de montarias, amagos e criar
- Sistemas de evocação
- Quests executáveis
- Bugs conhecidos.
- Correções históricas.
- Mudanças de balanceamento.

Investigue fóruns, comunidades, documentação, wikis, repositórios públicos, discussões técnicas e materiais históricos.

Procure informações que ajudem a reconstruir as regras originais e identificar diferenças entre versões.

Diferencie claramente:

1. Comportamento confirmado no código.
2. Comportamento observado no jogo.
3. Informação encontrada em documentação.
4. Relato de jogadores.
5. Hipótese ainda não validada.

Não transforme relatos de fóruns em verdades técnicas sem validação.

## 10.3. Mapeamento matemático dos atributos

Mapeie todos os atributos utilizados pelo jogo.

Investigue, entre outros:

- Força.
- Inteligência.
- Destreza.
- Constituição.
- HP.
- MP.
- Ataque físico.
- Ataque mágico.
- Defesa.
- Precisão.
- Evasão.
- Velocidade de ataque.
- Velocidade de movimento.
- Chance crítica.
- Dano crítico.
- Resistências.
- Absorção.
- Regeneração.
- Penetração.
- Bloqueio.
- Modificadores percentuais.
- Atributos derivados.

Para cada atributo, identifique:

- Origem.
- Tipo.
- Unidade.
- Faixa de valores.
- Limites.
- Fórmula.
- Dependências.
- Arredondamento.
- Conversões.
- Aplicação no servidor.
- Representação no cliente.
- Persistência no banco.

Documente as equações completas.

## 10.4. Matemática de combate e cálculo de dano

Reconstrua o cálculo de dano de ponta a ponta.

Investigue:

- Dano base.
- Atributos do personagem.
- Arma.
- Equipamentos.
- Refinamentos.
- Buffs.
- Debuffs.
- Skills.
- Crítico.
- Defesa do alvo.
- Resistências.
- Absorção.
- Redução percentual.
- Multiplicadores.
- Modificadores condicionais.
- Arredondamentos.
- Limites mínimos e máximos.

Identifique a ordem em que cada modificador é aplicado.

Exemplo de investigação:

1. Cálculo do ataque base.
2. Aplicação dos atributos.
3. Aplicação dos bônus de equipamentos.
4. Aplicação dos modificadores de skill.
5. Aplicação dos efeitos temporários.
6. Cálculo de crítico.
7. Aplicação da defesa do alvo.
8. Aplicação das resistências.
9. Aplicação das reduções.
10. Resultado final.

Essa sequência é apenas um roteiro de investigação, não uma afirmação sobre a ordem real do WYD.

Descubra a sequência efetivamente implementada.

## 10.5. Velocidade de ataque, quantidade de hits e crítico

Esta é uma das áreas prioritárias da investigação.

Analise detalhadamente a relação entre:

- Velocidade de ataque.
- Percentual de velocidade.
- Intervalo entre ataques.
- Quantidade de hits.
- Quantidade de animações.
- Dano por segundo.
- Críticos.
- Multiplicadores.
- Limites internos.
- Arredondamento.
- Frequência de processamento.

Existe um comportamento observado que precisa ser investigado:

Personagens de dano físico, ao atingirem aproximadamente 200% de velocidade, aparentemente começam a realizar hits duplicados ou apresentar aumento anormal de hits/críticos.

Investigue se isso é:

- Um comportamento previsto pelo jogo.
- Uma mecânica de escalonamento.
- Um limite interno.
- Um erro de cálculo.
- Um problema de arredondamento.
- Uma falha de sincronização.
- Um problema de animação.
- Um problema de processamento de pacotes.
- Um bug no cliente.
- Um bug no servidor.
- Uma combinação desses fatores.

Não considere o comportamento como correto ou incorreto antes de identificar sua origem.

### Modelo matemático a investigar

Avalie a possibilidade de existir uma curva de escalonamento da velocidade que aumente gradualmente a quantidade de ataques.

Exemplo hipotético de comportamento desejado:

| Velocidade | Hits esperados |
|---|---:|
| 100% | 1 |
| 200% | 2 |
| 300% | 2 |
| 400% | 3 |
| 500% | 3 |
| 600% | 4 |

Essa tabela é apenas uma referência para investigação, não uma fórmula definitiva.

Analise diferentes funções matemáticas que possam representar esse comportamento.

Considere:

- Funções lineares.
- Funções por intervalos.
- Funções com retornos decrescentes.
- Curvas progressivas.
- Limites máximos.
- Curvas com suavização.
- Fórmulas com frequência de ataque.
- Relação entre velocidade e tempo de execução.

Compare os resultados e identifique qual modelo oferece melhor coerência matemática e jogabilidade.

### Investigações obrigatórias

Verifique:

- Se o aumento de velocidade está produzindo ataques adicionais corretamente.
- Se os críticos são contabilizados de forma independente.
- Se existe duplicação indevida de dano.
- Se o cliente mostra uma quantidade de hits diferente da processada pelo servidor.
- Se a animação influencia o processamento real.
- Se existe limite de velocidade no servidor.
- Se o cliente consegue informar valores incompatíveis.
- Se há possibilidade de exploração por manipulação de pacotes.
- Se a frequência de ataques é consistente com o tempo real.
- Se o DPS aumenta de forma proporcional ou descontrolada.

Crie testes que permitam reproduzir os cenários.

## 10.6. Engenharia reversa de skills e magias

Investigue todas as skills e magias.

Para cada uma, documente:

- Nome.
- Classe.
- Tipo.
- Requisitos.
- Nível.
- Custo de MP.
- Cooldown.
- Alcance.
- Área de efeito.
- Alvos.
- Dano.
- Cura.
- Buffs.
- Debuffs.
- Duração.
- Escalonamento.
- Modificadores.
- Efeitos secundários.
- Interações com atributos.
- Interações com equipamentos.

Reconstrua as fórmulas utilizadas.

Investigue também:

- Skills sem implementação.
- Skills parcialmente implementadas.
- Skills com parâmetros inconsistentes.
- Skills com comportamento divergente entre cliente e servidor.
- Skills com escalonamento desproporcional.
- Skills com efeitos duplicados.
- Skills com problemas de cooldown.
- Skills que podem ser exploradas.

## 10.7. Equipamentos, itens e modificadores

Analise como cada tipo de item influencia o personagem.

Considere:

- Armas.
- Armaduras.
- Elmos.
- Luvas.
- Botas.
- Escudos.
- Acessórios.
- Anéis.
- Colares.
- Pedras.
- Refinamentos.
- Itens especiais.
- Itens de evolução.
- Itens temporários.
- Buffs associados.

Identifique como os atributos são somados, multiplicados, limitados ou convertidos.

Verifique se existem bônus aplicados duas vezes ou ignorados.

## 10.8. NPCs e mobs

Realize uma análise completa dos NPCs e mobs.

Para cada entidade, investigue:

- Identificador.
- Nome.
- Tipo.
- Classe.
- Nível.
- HP.
- MP.
- Ataque.
- Defesa.
- Velocidade.
- Alcance.
- Precisão.
- Evasão.
- Resistências.
- Skills.
- Magias.
- Comportamento.
- Inteligência artificial.
- Agressividade.
- Perseguição.
- Área de atuação.
- Recompensas.
- Experiência.
- Drops.
- Respawn.

Reconstrua as fórmulas de combate utilizadas por essas entidades.

Identifique mobs com atributos desproporcionais, comportamento incorreto, dano anormal ou recompensas inconsistentes.

Analise também o comportamento da IA e sua relação com as regras de combate.

## 10.9. Classes e balanceamento de builds

Realize uma análise comparativa de todas as classes.

Investigue:

- Potencial ofensivo.
- Potencial defensivo.
- Dano físico.
- Dano mágico.
- Velocidade.
- Mobilidade.
- Resistência.
- Sustentação.
- Eficiência em PvE.
- Eficiência em PvP.
- Dependência de equipamentos.
- Dependência de atributos.
- Eficiência de diferentes builds.

Compare as classes em condições equivalentes.

Considere diferentes níveis, equipamentos, atributos, buffs e situações de combate.

Identifique:

- Builds dominantes.
- Builds inviáveis.
- Classes subdimensionadas.
- Classes excessivamente fortes.
- Combinações exploráveis.
- Dependências exageradas de equipamentos.
- Desbalanceamento entre PvE e PvP.

Proponha melhorias fundamentadas em simulações e evidências.

Não faça balanceamento baseado apenas em opinião ou percepção individual.

## 10.10. Progressão e evolução

Investigue os sistemas de progressão.

Considere:

- Experiência.
- Níveis.
- Evoluções.
- Classes.
- Requisitos.
- Atributos por nível.
- Pontos de distribuição.
- Equipamentos.
- Refinamentos.
- Sistemas especiais.
- Recompensas.
- Quests.
- Progressão econômica.

Avalie se a progressão é coerente e se existem pontos em que a dificuldade ou a recompensa cresce de maneira desproporcional.

## 10.11. Comunicação cliente-servidor sob a perspectiva matemática

Cruze os resultados desta fase com a engenharia reversa realizada na FASE 8.

Investigue como os atributos, estados e resultados de combate são transmitidos.

Identifique:

- Quais cálculos são realizados no cliente.
- Quais cálculos são realizados no servidor.
- Quais valores são enviados.
- Quais valores são recebidos.
- Quais valores são persistidos.
- Quais validações são executadas.
- Quais informações são apenas visuais.
- Quais informações representam o estado autoritativo do jogo.

O servidor deve ser a referência autoritativa para cálculos sensíveis de combate, progressão e integridade do jogo.

Identifique possíveis vulnerabilidades de manipulação de atributos, velocidade, dano, cooldown e resultados de combate.

## 10.12. Bug Hunter matemático

Realize uma busca sistemática por bugs.

Investigue:

- Overflow.
- Underflow.
- Divisão por zero.
- Arredondamento incorreto.
- Conversão de tipos.
- Perda de precisão.
- Multiplicação de percentuais.
- Ordem incorreta de operações.
- Limites inconsistentes.
- Valores negativos indevidos.
- Multiplicadores duplicados.
- Condições de corrida.
- Inconsistência entre cliente e servidor.
- Falhas de sincronização.
- Exploração de pacotes.
- Resultados determinísticos incorretos.

Classifique cada problema por:

- Severidade.
- Impacto.
- Reprodutibilidade.
- Causa raiz.
- Componentes afetados.
- Risco de exploração.
- Correção proposta.

## 10.13. Laboratório matemático e simulações

Crie uma abordagem sistemática para validar as fórmulas.

Quando apropriado, implemente um simulador ou conjunto de ferramentas independentes do servidor de produção.

O laboratório deve permitir:

- Simular atributos.
- Simular builds.
- Simular ataques.
- Simular críticos.
- Simular velocidade.
- Simular dano.
- Simular defesa.
- Simular resistências.
- Simular skills.
- Simular equipamentos.
- Simular NPCs e mobs.
- Comparar classes.
- Comparar versões de fórmulas.
- Executar milhares de combinações.

Crie cenários reproduzíveis.

Os resultados devem permitir comparar o comportamento atual com o comportamento proposto.

Não utilize o simulador como substituto dos testes de integração com o jogo real.

## 10.14. Documentação matemática

Crie documentação técnica organizada contendo:

- Catálogo de atributos.
- Fórmulas.
- Equações.
- Unidades.
- Limites.
- Regras de arredondamento.
- Cálculos de combate.
- Cálculos de velocidade.
- Skills.
- Magias.
- NPCs.
- Mobs.
- Classes.
- Builds.
- Progressão.
- Bugs identificados.
- Resultados dos experimentos.
- Referências externas.
- Hipóteses ainda não comprovadas.

Toda fórmula deve indicar sua origem e nível de confiança.

## 10.15. Integração com Skills, arquitetura e roadmap

Atualize as Skills de IA para que os agentes consigam compreender:

- Matemática do WYD.
- Fórmulas.
- Combate.
- Balanceamento.
- Atributos.
- Skills.
- NPCs.
- Mobs.
- Engenharia reversa.
- Testes matemáticos.

Crie Issues específicas para:

- Correção de bugs.
- Implementação de fórmulas.
- Testes de regressão.
- Simulações.
- Balanceamento.
- Documentação.
- Atualizações de banco.
- Alterações no cliente.
- Alterações no servidor.

Não misture todas essas atividades em uma única Issue abrangente.

## 10.16. Critérios de conclusão

Esta fase só poderá ser considerada concluída quando:

1. Os principais atributos estiverem documentados.
2. As fórmulas críticas estiverem identificadas ou formalmente classificadas como desconhecidas.
3. O cálculo de dano estiver mapeado.
4. A velocidade de ataque estiver investigada.
5. O comportamento de hits duplicados estiver reproduzido ou descartado com evidências.
6. O sistema de críticos estiver compreendido.
7. Skills e magias estiverem catalogadas.
8. NPCs e mobs tiverem seus parâmetros analisados.
9. As classes e builds tiverem sido comparadas.
10. Os bugs identificados estiverem registrados.
11. As hipóteses estiverem separadas dos fatos.
12. Existirem testes e simulações reproduzíveis.
13. As correções propostas tiverem critérios de validação.
14. A documentação estiver atualizada.
15. O roadmap refletir todas as atividades pendentes.

Não é obrigatório inventar uma fórmula quando não houver evidências suficientes.

É preferível documentar uma lacuna de conhecimento do que apresentar uma fórmula incorreta como se fosse verdadeira.

---

# FASE 11 — DOCUMENTAÇÃO E PADRONIZAÇÃO COMPLETA

Realize uma revisão de toda a documentação do projeto.

## 11.1. MkDocs

Analise a documentação oficial construída com MkDocs.

Verifique:

- Estrutura.
- Navegação.
- Organização.
- Conteúdo.
- Links.
- Imagens.
- Exemplos.
- Diagramas.
- Referências.
- Atualização.
- Consistência.

A documentação deve refletir o estado real do projeto.

## 11.2. Documentação interna

Separe claramente:

- Documentação oficial.
- Documentação de desenvolvimento.
- Documentação de arquitetura.
- Documentação operacional.
- Documentação de engenharia reversa.
- Documentação matemática.
- Documentação de troubleshooting.
- Documentação de deploy.
- Documentação de segurança.

Evite duplicar conteúdos sem necessidade.

## 11.3. Padronização tecnológica

Revise a padronização ponta a ponta das tecnologias utilizadas, especialmente:

- Go.
- Next.js.
- TypeScript.
- JavaScript.
- SQL.
- Configurações.
- Scripts.
- Containers.
- CI/CD.

Estabeleça convenções consistentes de:

- Estrutura.
- Nomenclatura.
- Formatação.
- Organização.
- Testes.
- Logs.
- Erros.
- Configurações.
- Documentação.

## 11.4. Diagramas

Crie ou atualize diagramas para:

- Arquitetura.
- Infraestrutura.
- Comunicação.
- Banco de dados.
- Fluxos operacionais.
- CI/CD.
- Sistemas do jogo.
- Cliente-servidor.
- Matemática de combate.
- Dependências.

Os diagramas devem representar o funcionamento real.

---

# FASE 12 — DASHBOARD E VISUALIZAÇÃO DO ESTADO DO PROJETO

Revise a estrutura de dashboards existentes e proponha uma visão consolidada do ecossistema.

## 12.1. Métricas do projeto

Considere métricas de:

- Roadmap.
- Issues.
- Pull Requests.
- CI/CD.
- Builds.
- Testes.
- Cobertura.
- Segurança.
- Dependências.
- Infraestrutura.
- Banco.
- Observabilidade.
- Portal.
- Servidor.
- Cliente.
- Releases.
- Débitos técnicos.

## 12.2. Métricas do jogo

Quando houver dados disponíveis, considere:

- Jogadores conectados.
- Sessões.
- Personagens.
- Atividade.
- Eventos.
- Desempenho.
- Combate.
- Erros.
- NPCs.
- Mobs.
- Sistemas operacionais.

## 12.3. Visualização

Avalie a criação de dashboards separados por domínio.

A interface deve permitir identificar:

- Estado atual.
- Evolução.
- Problemas.
- Alertas.
- Tendências.
- Gargalos.
- Atividades pendentes.

Não invente números, métricas ou resultados.

Utilize dados reais e identifique explicitamente informações indisponíveis.

---

# FASE 13 — PROPOSTAS TÉCNICAS E MELHORIAS ESTRUTURAIS

Após as auditorias, consolide propostas de evolução que não estejam contempladas adequadamente no roadmap.

## 13.1. Avaliação de oportunidades

Investigue melhorias em:

- Arquitetura.
- Servidor.
- Cliente.
- Banco.
- DevOps.
- CI/CD.
- Segurança.
- Observabilidade.
- Portal.
- Performance.
- Escalabilidade.
- Testes.
- Automação.
- Engenharia reversa.
- Matemática do jogo.

## 13.2. Avaliação de tecnologias

Para cada oportunidade, compare:

- Tecnologia atual.
- Alternativas.
- Maturidade.
- Compatibilidade.
- Custo operacional.
- Complexidade.
- Benefícios.
- Riscos.
- Impacto da migração.
- Manutenibilidade.

Não introduza tecnologias apenas por serem populares.

## 13.3. ADRs e decisões arquiteturais

Quando uma decisão tiver impacto arquitetural significativo, registre:

- Contexto.
- Problema.
- Alternativas.
- Decisão.
- Justificativa.
- Consequências.
- Riscos.

## 13.4. Priorização das propostas

Classifique as propostas por:

- Obrigatórias.
- Alta prioridade.
- Média prioridade.
- Baixa prioridade.
- Experimentais.
- Futuras.

Crie Issues para as propostas aprovadas e incorpore-as ao roadmap.

Não execute automaticamente mudanças de alto risco sem aprovação.

---

# FASE 14 — PESQUISA TÉCNICA CONTÍNUA E ATUALIZAÇÃO DO CONHECIMENTO

Realize pesquisas aprofundadas para identificar soluções modernas que possam melhorar o ecossistema.

## 14.1. Pesquisa de tecnologias

Investigue tecnologias, bibliotecas, ferramentas, padrões e arquiteturas relevantes para:

- Go.
- Next.js.
- TypeScript.
- Banco de dados.
- DevOps.
- CI/CD.
- Observabilidade.
- Segurança.
- Testes.
- Automação.
- Engenharia reversa.
- Desenvolvimento de jogos.
- Matemática computacional.

## 14.2. Pesquisa de WYD/WYD2

Continue a pesquisa de referências históricas e técnicas do jogo.

Busque informações que possam ajudar a validar:

- Fórmulas.
- Atributos.
- Combate.
- Skills.
- Velocidade.
- Classes.
- Itens.
- Mobs.
- NPCs.
- Bugs.
- Balanceamento.
- Diferenças entre versões.

Cruze as informações externas com as evidências encontradas no projeto.

## 14.3. Atualização das Skills

Atualize as Skills e agentes sempre que novas informações relevantes forem confirmadas.

Evite registrar hipóteses como conhecimento consolidado.

## 14.4. Relatório de pesquisa

Para cada pesquisa relevante, registre:

- Tema.
- Fontes.
- Data.
- Evidências.
- Conclusões.
- Limitações.
- Aplicabilidade.
- Recomendações.
- Issues relacionadas.

---

# FASE 15 — AUDITORIA FINAL E PREPARAÇÃO PARA RELEASE

Realize uma revisão consolidada antes de considerar o projeto pronto para release.

## 15.1. Auditoria técnica

Verifique:

- Arquitetura.
- Código.
- Banco.
- Cliente.
- Servidor.
- Portal.
- DevOps.
- CI/CD.
- Observabilidade.
- Segurança.
- Testes.
- Documentação.
- Skills.
- Agentes.
- Matemática do jogo.
- Compatibilidade.

## 15.2. Validação funcional

Execute testes de:

- Inicialização.
- Integração.
- Persistência.
- Comunicação.
- Autenticação.
- Autorização.
- Combate.
- Skills.
- Itens.
- Personagens.
- NPCs.
- Mobs.
- Portal.
- Launcher.
- Sistemas administrativos.

Verifique possíveis regressões.

## 15.3. Validação de segurança

Execute as verificações necessárias para identificar:

- Vulnerabilidades.
- Dependências inseguras.
- Configurações incorretas.
- Credenciais expostas.
- Falhas de autorização.
- Problemas de comunicação.
- Riscos de manipulação do cliente.
- Problemas de integridade.

## 15.4. Release engineering

Revise:

- Versionamento.
- Changelog.
- Artifacts.
- Migrations.
- Configurações.
- Deploy.
- Rollback.
- Documentação.
- Compatibilidade.
- Procedimentos operacionais.

## 15.5. Relatório final

Produza um relatório consolidado contendo:

1. Estado inicial.
2. Estado atual.
3. Atividades concluídas.
4. Issues encerradas.
5. Issues pendentes.
6. Melhorias implementadas.
7. Problemas encontrados.
8. Riscos conhecidos.
9. Resultados dos testes.
10. Resultados de segurança.
11. Estado da infraestrutura.
12. Estado do cliente.
13. Estado do servidor.
14. Estado do banco.
15. Estado do portal.
16. Estado da matemática e do balanceamento.
17. Débitos técnicos restantes.
18. Recomendações futuras.
19. Situação do release.

---

# 16. ORDEM ESTRATÉGICA DE EXECUÇÃO

As fases representam domínios de trabalho e não necessariamente uma sequência rígida em que uma fase precisa estar 100% concluída antes de iniciar a seguinte.

A execução deve respeitar dependências técnicas.

Ordem estratégica recomendada:

1. Auditoria completa e organização do roadmap.
2. Definição do plano de execução.
3. Modernização do servidor.
4. Modernização do banco de dados.
5. Modernização do CI/CD e DevOps.
6. Observabilidade e segurança.
7. Portal web e experiência do usuário.
8. Engenharia reversa e modernização do cliente.
9. Organização de IA, FreeBuff, Skills e agentes.
10. Engenharia reversa matemática, Bug Hunting e balanceamento do WYD.
11. Documentação e padronização.
12. Dashboard e métricas.
13. Propostas técnicas e melhorias estruturais.
14. Pesquisa técnica contínua.
15. Auditoria final e preparação do release.

A FASE 10 deve começar com a coleta e análise das evidências disponíveis, podendo avançar em paralelo com as demais fases quando não houver dependências impeditivas.

A documentação deve ser atualizada durante todas as fases, não apenas ao chegar à FASE 11.

Da mesma forma, a segurança, os testes e a observabilidade devem ser considerados continuamente.

---

# 17. PADRÃO DE ENTREGA E CRITÉRIOS GERAIS DE QUALIDADE

Toda atividade deve respeitar os seguintes critérios:

## 17.1. Implementação

- Código consistente.
- Arquitetura adequada.
- Ausência de duplicação desnecessária.
- Tratamento de erros.
- Segurança.
- Testabilidade.
- Documentação.
- Compatibilidade.

## 17.2. Testes

Executar os testes adequados ao escopo:

- Unitários.
- Integração.
- Contratos.
- E2E.
- Regressão.
- Performance.
- Segurança.
- Testes matemáticos.
- Testes de compatibilidade.

Não declare uma implementação validada sem apresentar evidências.

## 17.3. Issues

Toda Issue deve conter:

- Título objetivo.
- Contexto.
- Problema.
- Objetivo.
- Escopo.
- Requisitos.
- Critérios de aceitação.
- Dependências.
- Testes.
- Documentação relacionada.

Evite Issues genéricas que não permitam identificar quando o trabalho estará realmente concluído.

## 17.4. Relatórios de progresso

Durante a missão, mantenha relatórios com:

- Fase atual.
- Atividades concluídas.
- Atividades em andamento.
- Issues afetadas.
- Problemas encontrados.
- Decisões tomadas.
- Testes realizados.
- Riscos.
- Próximas atividades.

## 17.5. Critério de conclusão global

A missão não estará concluída apenas porque todas as Issues foram fechadas.

É necessário comprovar que:

- O roadmap representa o estado real do projeto.
- A arquitetura está documentada.
- Os principais débitos técnicos foram tratados ou registrados.
- O servidor e o cliente possuem compatibilidade validada.
- O banco está versionado e consistente.
- Os pipelines são confiáveis.
- A segurança foi revisada.
- A observabilidade permite acompanhar o ambiente.
- O portal está funcional e validado.
- O conhecimento de engenharia reversa está documentado.
- A matemática do jogo foi investigada com evidências.
- Os bugs críticos estão corrigidos ou formalmente registrados.
- A documentação está atualizada.
- As ferramentas de IA estão organizadas.
- Os testes são reproduzíveis.
- O processo de release é confiável.

---

# INSTRUÇÃO FINAL AO AGENTE

Não trate esta missão como uma simples lista de tarefas.

Trate-a como um programa completo de auditoria, engenharia, modernização e evolução de um ecossistema de software e de um jogo online.

Comece pela compreensão profunda do estado atual.

Não implemente mudanças sem entender suas consequências.

Não descarte conhecimento existente.

Não invente resultados, métricas, fórmulas ou funcionalidades.

Não considere hipóteses como fatos.

Não encerre Issues sem evidências.

Não deixe a documentação divergir da implementação.

Não sacrifique compatibilidade, segurança ou integridade dos dados por velocidade de execução.

Quando encontrar problemas, investigue a causa raiz.

Quando encontrar oportunidades, compare alternativas.

Quando identificar melhorias necessárias, registre-as no roadmap.

Quando implementar, valide.

Quando validar, documente.

Quando concluir um conjunto de 10 Issues, execute uma revisão técnica consolidada antes de prosseguir.

Mantenha o roadmap, o código, os testes, a documentação, as Skills e os agentes alinhados durante toda a missão.

**O resultado esperado é um ecossistema WYD tecnicamente profissional, moderno, sustentável, seguro, observável, documentado e com suas regras internas — incluindo a matemática do jogo — compreendidas, testáveis e evoluíveis.**