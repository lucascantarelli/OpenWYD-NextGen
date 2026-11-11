# Organização da documentação do projeto

## Arquivos de documentos

Temos diversos arquivos de documentos, fizemos muitas buscas, registros
Leia as skills e garregue as que devem ser usadas nessa solicitação

precisamos sintetizar todos esses dados e remover duplicidades, informações divergentes, limpar informações

vamos organizar, deixar apenas as informações relevantes, deixar informações reais, temos exemlos de vários servidores, vamos deixar apenas o que vamos usar realmente no servidor.

## Documentos e skills de IA

- Atualizar as informações em skills e documentos de IA, garantindo que estejam alinhadas com as necessidades do projeto.
- Remover informações desatualizadas ou irrelevantes, mantendo apenas o que é essencial para o desenvolvimento do projeto.
- Garantir que os documentos estejam claros e compreensíveis para todos os membros da equipe, facilitando a colaboração e o compartilhamento de conhecimento.
- organizar a pasta docs seguindo os padrões definidos

## Roadmap e issues

Temos vários caminhos desenvolvidos, temos issues para quests, sistemas, client, launcher, portal, etc

eu quero que vc mapeie esses caminhos e atualize tudo que ainda precisa ser feito, gerar todas as issues reais, seguindo o padrão

- Analisar qual o caminho conseguimos finalizar com mais celeridade
- Qual o caminho correto de seguir para uma execução fluida de todas as atividades, sem bloqueios.
- analisar qual a maior entrega real que conseguimos fazer primeiro.
- Organizar roadmap,issues,checklist e outraos documentos
- Organize as issues do roadmap de forma natual, com uma curva de crescimento com o servidor, para que possamos ter entregas consistentes e contínuas, sem sobrecarregar a equipe ou comprometer a qualidade do projeto.
- primeiramente , focar em issues que são básicas, servicos de comunicação, login, acesso ao mundo, mensagens entre cliente/servidor, notificações do cliente, devemos desenvolver o projeto com acessos ao mundo, testes manuais com conta de admin, times de bots para testes massivos, carga, performance, sistemas completos do jogo, quests, skills, drops, etc, para que possamos ter uma base sólida e confiável para o desenvolvimento do projeto.
- bots para testar drops, quests, guerras, pvp, skills, etc, para que possamos identificar problemas e otimizar o desempenho do servidor de forma eficiente.
- fazer  gestão dos logs de testes dos bots para identificar problemas , ter dados para análises, comparações, balanceamento, etc, para que possamos melhorar continuamente o desempenho e a confiabilidade do servidor.
- Os bots devem ter todas as funcionalidades de um jogador real, analisar projeto de implementação de llm documentado no projeto, bots com comportamentos reais, vivendo como uma população real do servidor, evoluindo, caçando, comercializando, interagindo, montando grupos, fazendo quests, trocas, lojas, evoluindo personagens, etc, para que possamos testar o servidor de forma realista e identificar problemas de forma eficiente.

## Padrões, adrs, decisões e documentaçã ode referencia.

Devemos levar em consideração todos os apontamentos nas documentações de referencia, elas possuem dados sobre o jogo,arquitetura, boas praticas, etc

- wyd2.md ( esta documentação descreve sistemas do jogo,itens e outras informações, atualize com o que não está documentado ainda)
- start-rust.md
- pre-release-rust.md


# Documentação do projeto

a pasta docs é nossa pasta de documentação para usuários, moderadores, administradores, desenvolvedores,arquitetos, engenheiros, etc
a pasta docs deve ser um projeto mkdocs(ou similar, com index, navegação, links internos, plugins, recursos extras,etc)


- Devemos ter todos os manuais(administraivo/jogadores)
- Seguir padrões de documentação, como ADRs, decisões de arquitetura, padrões de código, etc.
- Garantir que a documentação esteja sempre atualizada e acessível para todos os membros da equipe.
- documentação deve ser clara, objetiva e de fácil compreensão, evitando ambiguidades e informações desnecessárias.
- Usar todos os recursos ricos do markdown, como links, imagens, tabelas, listas, etc., para tornar a documentação mais visual e interativa.
- cores, emojis, ícones, imagens, gifs, vídeos, etc. podem ser usados para melhorar a experiência do usuário e facilitar a compreensão das informações.
- Garantir que a documentação esteja organizada de forma lógica e estruturada, facilitando a navegação e a busca por informações específicas.
- Manter um índice atualizado e links internos para facilitar a navegação entre os diferentes documentos e seções.
- Analisar as referencias pesquisas da ongame, que era a empresa oficial no brasil, pegar referencias de portal, documentação, manuais, guias, etc, e atualizar a documentação do projeto com essas informações.

## Gaps,Implementações e melhorias

- Sugerir melhorias e implementações com base nas análises realizadas, buscando otimizar processos e aumentar a eficiência do projeto.
- criar issues padronizadas para todas as atividades 
- Garantir que todas as melhorias e implementações sejam documentadas de forma clara e detalhada, facilitando a compreensão e a execução das tarefas.
- Referenciar skills nas issues.

## Portal 

- Pesquise o antigo portal da ongame, do wyd2 para pegar referencia de um portal de wyd, pesquise outros servidores privados
- Normalmente, a pagina principal do portal deve ter informações sobre o jogo, notícias, eventos, downloads, suporte, com a pagina inicial no /
- a parte administrativa do portal deve ter informações sobre o servidor, estatísticas, logs, ferramentas de administração, etc
- o acesso a parte administrativa deve ser restrito a administradores e moderadores, com autenticação segura e controle de acesso.
- o acessa da pagina administrativa deve ser liberado, após o usuário fazer login na aréa publica do portal, caso o usuário seja um administrador ou moderador, ele aparecer um botão, no perfil do usuário na navbar, ex: um votão acessar atéa administrativa, que redireciona para a pagina administrativa do portal, o acesso pode ser o melhor para encaixar no layout, não obrigatório ser um botão

### Monitoramento
- dentro da pagina de monitoramento, nos logs, devemos ver os logs ao vivo, com filtros, e também ter a opção de baixar os logs em formato de arquivo, para análise posterior.
- no monitoramento na area de servidor, expandir com mais informações, como uso de CPU, memória, rede, processos em execução, dividir numeros de player e mobs, não juntar tudo em entidades, incluir gráficos e estatísticas em tempo real, com histórico de uso, para facilitar a análise do desempenho do servidor.
- area de paineis, eu acho que vc está importando uma pagina completa do grafana, está bom, mas, vc não conseguiria pegas os dados via api do grafana ? e fazer uma página customizada, com gráficos e informações mais relevantes para o monitoramento do servidor, sem precisar importar uma página completa do grafana, que pode ser pesada e não tão flexível para personalização, usando os componentes do portal, como cards, gráficos, tabelas, etc, para criar uma página de monitoramento mais leve e customizada, com informações mais relevantes para o servidor.
- expandir o monitoramento para incluir informações sobre o banco de dados, como número de conexões, queries em execução, tempo de resposta, etc, para facilitar a análise do desempenho do banco de dados.
- analisar o que ainda pode ser incluso no monitoramento, como informações sobre o sistema operacional, logs de eventos, alertas de segurança, etc, para ter uma visão mais completa do estado do servidor.
- prever a possibilidade de criar alertas e notificações para eventos críticos, como falhas no servidor, uso excessivo de recursos, etc, para que os administradores possam tomar medidas preventivas e corretivas rapidamente.
- implementar boas praticas, baseadas nas skills
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do monitoramento do servidor.

### Contas

- expandir informações sobre as contas, como histórico de login, atividades recentes, permissões, etc, para facilitar a gestão das contas de usuários.
- melhorar a interface de gerenciamento de contas, tornando-a mais intuitiva e fácil de usar, com filtros, buscas e opções de ordenação.
- implementar funcionalidades de segurança, como autenticação de dois fatores, recuperação de senha, bloqueio de contas suspeitas, etc, para proteger as contas de usuários e garantir a integridade do sistema.
- implementar boas práticas de gestão de contas, baseadas nas skills, para garantir a segurança e a confiabilidade do sistema.
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do gerenciamento de contas.
- implementar funcionalidades de auditoria e registro de atividades, para monitorar e rastrear ações realizadas nas contas de usuários, facilitando a identificação de problemas e a tomada de decisões.

### Entregas

- Creio isso deve ser removido, seria para dá itens a jogadores, mas, creio que isso possa ser implementado de outra forma
- essas entregas, creio que serão implementadas com sistema de premium mail, cash, etc
- Altere para o sistema de premium mail, cash, etc, para que os jogadores possam receber itens e recompensas de forma segura e eficiente, sem a necessidade de um sistema de entregas separado.
- Implementar funcionalidades de gerenciamento de entregas, como histórico de entregas, status de entrega, notificações, etc, para facilitar a gestão das entregas e garantir que os jogadores recebam seus itens e recompensas corretamente.
- implemente a gestão financeira, para que os jogadores possam acompanhar suas transações, saldo, histórico de compras, etc, de forma clara e transparente.
- implementar boas práticas de gestão financeira, baseadas nas skills, para garantir a segurança e a confiabilidade do sistema financeiro do portal.
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do sistema financeiro do portal.
- implementar recursos visuais e interativos, como gráficos, tabelas, listas, etc., para tornar a gestão das configurações mais visual e intuitiva.
- faça pesquisas dos sistemas de premium mail, cash, etc, para pegar referencias de outros servidores privados, docs, ongame oficial, etc, para implementar um sistema de entregas eficiente e confiável no portal.

### Catálogo de itens

- expandir informações sobre os itens, como descrição detalhada, atributos, requisitos, etc, para facilitar a compreensão e a utilização dos itens pelos jogadores.
- melhorar a interface de gerenciamento de itens, tornando-a mais intuitiva e fácil de usar, com filtros, buscas e opções de ordenação.
- implementar funcionalidades de segurança, como validação de itens, controle de acesso, etc, para proteger os itens e garantir a integridade do sistema.
- implementar boas práticas de gestão de itens, baseadas nas skills, para garantir a segurança e a confiabilidade do sistema de catálogo de itens do portal.
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do gerenciamento de itens do portal.
- implementar funcionalidades de auditoria e registro de atividades, para monitorar e rastrear ações realizadas no catálogo de itens, facilitando a identificação de problemas e a tomada de decisões.
- implementar funcionalidades de integração com outros sistemas, como o sistema de entregas, sistema financeiro, etc, para garantir a consistência e a integridade dos dados entre os diferentes sistemas do portal.
- implementar funcionalidades de personalização e customização do catálogo de itens, permitindo que os administradores possam adaptar o catálogo às necessidades específicas do servidor e dos jogadores.
- implementar funcionalidades de análise e relatórios, permitindo que os administradores possam gerar relatórios detalhados sobre o catálogo de itens, como popularidade dos itens, vendas, etc, para facilitar a tomada de decisões estratégicas.
- implementar recursos visuais e interativos, como gráficos, tabelas, listas, etc., para tornar a gestão das configurações mais visual e intuitiva.

### Configurações

- expandir informações sobre as configurações do portal, como opções de personalização, preferências de usuário, etc, para facilitar a gestão das configurações e garantir que o portal atenda às necessidades dos usuários.
- melhorar a interface de gerenciamento de configurações, tornando-a mais intuitiva e fácil de usar, com filtros, buscas e opções de ordenação.
- implementar funcionalidades de segurança, como controle de acesso, validação de configurações, etc, para proteger as configurações e garantir a integridade do sistema.
- implementar boas práticas de gestão de configurações, baseadas nas skills, para garantir a segurança e a confiabilidade do sistema de configurações do portal.
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do gerenciamento de configurações do portal.
- implementar funcionalidades de auditoria e registro de atividades, para monitorar e rastrear ações realizadas nas configurações do portal, facilitando a identificação de problemas e a tomada de decisões.
- o servidor deve ser completamente configurável via portal, sem necessidade de alterar arquivos de configuração manualmente, garantindo que todas as alterações sejam refletidas em tempo real no servidor.
- implementar funcionalidades de integração com outros sistemas, como o sistema de monitoramento, sistema financeiro, etc, para garantir a consistência e a integridade dos dados entre os diferentes sistemas do portal.
- implementar funcionalidades de personalização e customização das configurações, permitindo que os administradores possam adaptar as configurações às necessidades específicas do servidor e dos jogadores.
- implementar funcionalidades de análise e relatórios, permitindo que os administradores possam gerar relatórios detalhados sobre as configurações do portal, como alterações realizadas, preferências dos usuários, etc, para facilitar a tomada de decisões estratégicas.
- implementar recursos visuais e interativos, como gráficos, tabelas, listas, etc., para tornar a gestão das configurações mais visual e intuitiva.
- implementar recursos visuais e interativos, como gráficos, tabelas, listas, etc., para tornar a gestão das configurações mais visual e intuitiva.

### Auditoria

- implementar funcionalidades de auditoria e registro de atividades, para monitorar e rastrear ações realizadas no portal, facilitando a identificação de problemas e a tomada de decisões.
- implementar funcionalidades de análise e relatórios, permitindo que os administradores possam gerar relatórios detalhados sobre as atividades realizadas no portal, como alterações realizadas, ações dos usuários, etc, para facilitar a tomada de decisões estratégicas.
- implementar funcionalidades de alertas e notificações, permitindo que os administradores possam ser notificados sobre eventos críticos ou ações suspeitas no portal, para que possam tomar medidas preventivas e corretivas rapidamente.
- implementar funcionalidades de integração com outros sistemas, como o sistema de monitoramento, sistema financeiro, etc, para garantir a consistência e a integridade dos dados entre os diferentes sistemas do portal.
- implementar boas práticas de auditoria, baseadas nas skills, para garantir a segurança e a confiabilidade do sistema de auditoria do portal.
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do sistema de auditoria do portal.
- implementar recursos visuais e interativos, como gráficos, tabelas, listas, etc., para tornar a gestão das configurações mais visual e intuitiva.

### Economida do servidor

- implementar funcionalidades de análise e relatórios, permitindo que os administradores possam gerar relatórios detalhados sobre a economia do servidor, como transações realizadas, saldo dos jogadores, etc, para facilitar a tomada de decisões estratégicas.
- implementar funcionalidades de alertas e notificações, permitindo que os administradores possam ser notificados sobre eventos críticos ou ações suspeitas na economia do servidor, para que possam tomar medidas preventivas e corretivas rapidamente.
- implementar funcionalidades de integração com outros sistemas, como o sistema financeiro, sistema de monitoramento, etc, para garantir a consistência e a integridade dos dados entre os diferentes sistemas do portal.
- implementar boas práticas de gestão da economia do servidor, baseadas nas skills, para garantir a segurança e a confiabilidade do sistema de economia do portal.
- implementar melhorias e otimizações com base nas análises realizadas, buscando aumentar a eficiência e a confiabilidade do sistema de economia do portal.
- implementar recursos visuais e interativos, como gráficos, tabelas, listas, etc., para tornar a gestão das configurações mais visual e intuitiva.
- implementar funcionalidades de auditoria e registro de atividades, para monitorar e rastrear ações realizadas na economia do servidor, facilitando a identificação de problemas e a tomada de decisões.

### Aréa publica / aréa de administrativa

- implementar a real separação entre a área pública e a área administrativa do portal, garantindo que apenas usuários autorizados possam acessar as funcionalidades administrativas.
- implementar funcionalidades de controle de acesso, permitindo que os administradores possam definir quais usuários têm permissão para acessar a área administrativa do portal, garantindo a segurança e a integridade do sistema.
- implementar funcionalidades de autenticação segura, como autenticação de dois fatores, para garantir que apenas usuários autorizados possam acessar a área administrativa do portal, protegendo o sistema contra acessos não autorizados.
- implementar funcionalidades de auditoria e registro de atividades, para monitorar e rastrear ações realizadas na área administrativa do portal, facilitando a identificação de problemas e a tomada de decisões.
- implementar funcionalidades de análise e relatórios, permitindo que os administradores possam gerar relatórios detalhados sobre as atividades realizadas na área administrativa do portal, como alterações realizadas, ações dos usuários, etc, para facilitar a tomada de decisões estratégicas.
- implementar funcionalidades de alertas e notificações, permitindo que os administradores possam ser notificados sobre eventos críticos ou ações suspeitas na área administrativa do portal, para que possam tomar medidas preventivas e corretivas rapidamente.
- implementar funcionalidades de integração com outros sistemas, como o sistema de monitoramento, sistema financeiro, etc, para garantir a consistência e a integridade dos dados entre os diferentes sistemas do portal.
- implementar sistemas completos de gerenciamento de usuários, permitindo que os administradores possam gerenciar as contas dos usuários, como criar, editar, excluir, bloquear, desbloquear, etc., garantindo a segurança e a confiabilidade do sistema.
- implementar todas as funcionalidades da área publica para os jogadores,historia,drops, classes,skills, como manuais, guias, suporte, notícias, eventos, downloads, etc., garantindo que os jogadores tenham acesso a todas as informações e recursosnecessários para aproveitar o jogo.

### Launcher - Portal

- implementar funcionalidades de integração entre o launcher e o portal, permitindo que os jogadores possam acessar informações do portal diretamente pelo launcher, como notícias, eventos, suporte, etc.
- implementar funcionalidades de autenticação e login unificado entre o launcher e o portal, permitindo que os jogadores possam acessar suas contas e informações de forma integrada e segura.
- implementar funcionalidades de sincronização de dados entre o launcher e o portal, garantindo que as informações estejam sempre atualizadas e consistentes entre os dois sistemas.
- implementar funcionalidades de personalização e customização do launcher, permitindo que os administradores possam adaptar o launcher às necessidades específicas do servidor e dos jogadores.
- implementar funcionalidade de geração de patches e atualizações automáticas do jogo, garantindo que os jogadores sempre tenham a versão mais recente do jogo, sem a necessidade de downloads manuais.
- os patches devem ser degerados dentro do portal administrativo, com opções de configuração, como versão, arquivos a serem atualizados, etc., garantindo que os administradores possam gerenciar as atualizações de forma eficiente e segura.
- implementar funcionalidades de monitoramento e análise de desempenho do launcher, permitindo que os administradores possam identificar problemas e otimizar o desempenho do launcher para garantir uma experiência de jogo fluida e satisfatória para os jogadores.

## Launcher

- implementar funcionalidades de atualização automática do launcher, garantindo que os jogadores sempre tenham a versão mais recente do launcher e do jogo, sem a necessidade de downloads manuais.
- implementar funcionalidades de personalização e customização do launcher, permitindo que os administradores possam adaptar o launcher às necessidades específicas do servidor e dos jogadores.
- implementar funcionalidades de monitoramento e análise de desempenho do launcher, permitindo que os administradores possam identificar problemas e otimizar o desempenho do launcher para garantir uma experiência de jogo fluida e satisfatória para os jogadores.
- implementar funcionalidades de integração entre o launcher e o portal, permitindo que os jogadores possam acessar informações do portal diretamente pelo launcher, como notícias, eventos, suporte, etc.
- implementar funcionalidades completas do launcher, configurações do jogo, atualizações, patches, downloads, etc., garantindo que os jogadores tenham acesso a todas as funcionalidades necessárias para jogar o jogo de forma eficiente e satisfatória.
- implementar funcionalidades de autenticação e login unificado entre o launcher e o portal, permitindo que os jogadores possam acessar suas contas e informações de forma integrada e segura.
- implementar funcionalidades de sincronização de dados entre o launcher e o portal, garantindo que as informações estejam sempre atualizadas e consistentes entre os dois sistemas.


## Bugs e problemas

- ao tentar logar com a senha incorreta, o cliente não retorna a mensagem de senha incorreta, apenas não mostra nada e não loga, pesquise mais mensagens que estão sem ser exibidas dentro do cliente, e faça uma lista de todas as mensagens que não estão sendo exibidas corretamente, para que possamos corrigir esses problemas e melhorar a experiência do usuário.
- o char ainda não entra no mundo