# META EXTRA — ECOSSISTEMA DE BOTS INTELIGENTES, SIMULAÇÃO DE POPULAÇÃO, LOAD TESTING E BUG HUNTING AUTOMATIZADO

## OBJETIVO GERAL

Desenvolver um ecossistema independente de bots automatizados, integrado ao ambiente de desenvolvimento e testes do WYD, capaz de simular uma população realista de jogadores conectados ao servidor.

O objetivo é criar uma infraestrutura de simulação que permita testar o comportamento do jogo em diferentes condições, validar sistemas, investigar problemas, reproduzir bugs, realizar testes de capacidade e avaliar o funcionamento da economia virtual.

A proposta consiste em criar um ambiente derivado do ecossistema atual do projeto, com uma população artificial de personagens controlados por bots, capazes de executar atividades semelhantes às realizadas por jogadores reais.

Esses bots devem interagir diretamente com o servidor por meio do protocolo de comunicação do WYD, preferencialmente sem precisar executar o cliente gráfico.

O ecossistema deve ser independente, modular, escalável e integrado aos processos de CI/CD, observabilidade, testes e documentação do projeto.

**Prioridade arquitetural: implementar primeiro uma solução determinística, baseada em algoritmos tradicionais, sem dependência obrigatória de LLMs.**

A utilização de Inteligência Artificial generativa deve ser opcional e complementar, não uma dependência para o funcionamento dos bots.

---

## 1. VIABILIDADE E ARQUITETURA SEM LLM

Investigue e implemente uma arquitetura de bots que funcione integralmente sem modelos de linguagem.

A utilização de LLMs não é necessária para executar a maioria das atividades de simulação de um MMORPG.

O comportamento dos bots pode ser controlado por algoritmos tradicionais, com baixo custo computacional, execução previsível e alta capacidade de paralelização.

### 1.1. Máquinas de Estados Finitos (FSM)

Implemente uma arquitetura baseada em Finite State Machines (FSM) ou Behavior Trees.

Cada bot deverá possuir estados e transições claramente definidos.

Exemplos de estados:

- `IDLE`: aguardando uma atividade.
- `CONNECTING`: estabelecendo conexão.
- `AUTHENTICATING`: realizando autenticação.
- `SELECTING_CHARACTER`: selecionando personagem.
- `ENTERING_WORLD`: entrando no mundo.
- `MOVING_TO_HUNT`: deslocando-se para uma área de caça.
- `HUNTING`: executando atividades de combate.
- `ATTACKING`: atacando um alvo.
- `LOOTING`: coletando itens.
- `HEALING`: recuperando HP ou MP.
- `RETURNING_TO_TOWN`: retornando à cidade.
- `SELLING_ITEMS`: vendendo itens.
- `RESTOCKING`: reabastecendo consumíveis.
- `TRADING`: realizando negociações.
- `PARTY_FORMING`: formando grupos.
- `QUESTING`: executando quests.
- `GUILD_ACTIVITY`: participando de atividades de guilda.
- `DEAD`: personagem morto.
- `RESPAWNING`: aguardando ou executando respawn.
- `DISCONNECTING`: encerrando conexão.

Os estados devem ser configuráveis e extensíveis.

Cada transição precisa possuir condições explícitas, validações e tratamento de falhas.

### 1.2. Behavior Trees

Avalie a utilização de Behavior Trees para comportamentos mais complexos.

Considere árvores específicas para:

- Caça.
- Exploração.
- Sobrevivência.
- Comércio.
- Formação de grupos.
- Participação em eventos.
- Execução de quests.
- Guerra.
- Retorno à cidade.
- Gerenciamento de inventário.

Compare FSM e Behavior Trees e escolha a abordagem mais adequada para cada tipo de comportamento.

Não introduza complexidade desnecessária. As duas abordagens podem coexistir quando houver justificativa arquitetural.

### 1.3. Heurísticas e regras determinísticas

Implemente regras de decisão baseadas em condições e prioridades.

Exemplos:

- Se HP estiver abaixo de determinado limite, utilizar poção.
- Se MP estiver insuficiente, interromper o uso de habilidades.
- Se inventário estiver cheio, retornar à cidade.
- Se não houver alvos disponíveis, procurar outra área.
- Se o personagem morrer, executar o fluxo de respawn.
- Se houver um item de interesse, avaliar sua coleta.
- Se os recursos estiverem abaixo do mínimo, reabastecer.
- Se o personagem estiver em perigo, recuar.
- Se o objetivo for concluído, selecionar a próxima atividade.

Os parâmetros devem ser configuráveis por perfil de bot.

### 1.4. Pathfinding e navegação

Investigue a implementação de algoritmos de navegação.

Considere:

- A*.
- Dijkstra.
- Waypoints.
- Grafos de navegação.
- Áreas navegáveis.
- Obstáculos.
- Regiões.
- Portais.
- Cidades.
- Áreas de caça.
- Pontos de interesse.
- Rotas de retorno.

Utilize os mapas e dados disponíveis no projeto para construir uma representação navegável do mundo.

Investigue locais relevantes, como Kefra, Noatun, Arzan e outras regiões existentes na versão utilizada.

Os bots devem conseguir:

- Navegar entre pontos.
- Encontrar rotas alternativas.
- Evitar obstáculos.
- Identificar áreas inacessíveis.
- Reconhecer portais.
- Detectar falhas de navegação.
- Recalcular rotas.
- Registrar situações em que não conseguem alcançar um destino.

Não presuma que o mapa visual seja suficiente para reconstruir a navegação. Investigue as informações efetivamente utilizadas pelo servidor e pelo cliente.

---

## 2. ARQUITETURA DE IA HÍBRIDA E LLMs OPCIONAIS

Embora o funcionamento principal não dependa de LLMs, avalie a possibilidade de incorporar modelos locais compactos para comportamentos de alto nível.

A arquitetura deve separar claramente execução determinística e inteligência generativa.

### 2.1. Camada determinística

Responsável pela grande maioria das operações.

Exemplos:

- Movimento.
- Combate.
- Coleta.
- Utilização de poções.
- Gerenciamento de inventário.
- Comércio.
- Navegação.
- Login.
- Reconexão.
- Execução de quests.
- Formação de grupos.
- Participação em eventos.

Essa camada deve ser rápida, previsível, reproduzível e independente de serviços externos de IA.

### 2.2. Camada opcional de LLM

Avalie modelos compactos que possam executar localmente, inclusive em containers Docker.

Investigue alternativas como:

- Phi Mini.
- Qwen 3 em variantes compactas.
- Gemma em variantes adequadas ao hardware.
- Llama em variantes compactas e quantizadas.
- Outros modelos locais com boa capacidade de seguir instruções e executar tarefas estruturadas.

Verifique cuidadosamente a disponibilidade, versões, licenciamento, requisitos de memória, suporte a quantização e desempenho real de cada modelo antes de selecioná-lo.

Considere ferramentas como:

- Ollama.
- llama.cpp.
- APIs locais compatíveis.
- Servidores de inferência em containers.

Não assuma que todos os modelos oferecem suporte nativo e equivalente a function calling. Valide os recursos de cada implementação.

### 2.3. Aplicações possíveis de LLM

Utilize IA generativa somente quando agregar valor.

Exemplos:

- Conversas mais naturais no chat.
- Personalidade de personagens.
- Negociações simuladas.
- Comportamento social.
- Estratégias de guilda.
- Interações contextuais.
- Decisões estratégicas de alto nível.
- Geração de cenários de teste.
- Análise de relatórios de bugs.

Evite utilizar LLM para decisões frequentes e sensíveis à latência, como movimento pixel a pixel, ataques e processamento de pacotes.

### 2.4. Arquitetura recomendada

Adote uma arquitetura híbrida:

- Camada principal: FSM, Behavior Trees, heurísticas e scripts determinísticos.
- Camada de navegação: pathfinding e waypoints.
- Camada de comunicação: cliente headless e protocolo WYD.
- Camada opcional de IA: decisões estratégicas e comportamento social.
- Camada de análise: métricas, eventos, relatórios e identificação de anomalias.

A ausência do serviço de IA não deve impedir a execução dos testes.

O LLM deve ser desacoplado dos processos críticos dos bots.

---

## 3. CLIENTE HEADLESS E EMULAÇÃO DO PROTOCOLO WYD

Desenvolva clientes headless capazes de se conectar diretamente ao servidor.

O objetivo é simular conexões reais sem precisar abrir uma instância gráfica do jogo para cada personagem.

### 3.1. Engenharia reversa do protocolo

Utilize os resultados da FASE 8 para compreender e documentar o protocolo.

Investigue:

- Handshake.
- Autenticação.
- Seleção de personagem.
- Entrada no mundo.
- Movimento.
- Ataques.
- Skills.
- Inventário.
- Comércio.
- Chat.
- Grupos.
- Guildas.
- Eventos.
- Morte.
- Respawn.
- Desconexão.

Documente a estrutura dos pacotes, identificadores, campos, tamanhos, codificação, validações e estados envolvidos.

Não invente formatos de pacotes.

Utilize implementações existentes e evidências obtidas por análise do cliente e servidor.

### 3.2. Implementação do cliente headless

Avalie linguagens adequadas para a implementação.

Considere especialmente:

- Go: concorrência, eficiência de memória, gerenciamento de conexões e execução de milhares de tarefas.
- Python: prototipação, automação, análise de dados e integração com ferramentas de testes.
- Node.js: operações assíncronas e integração com ferramentas web.

Compare as alternativas e escolha a arquitetura que melhor se integra ao projeto.

Uma possibilidade é implementar o núcleo de comunicação e execução em Go, deixando ferramentas de análise, simulação e automação em Python.

### 3.3. Características obrigatórias

O cliente headless deverá:

- Estabelecer conexões.
- Autenticar personagens de teste.
- Interpretar mensagens do servidor.
- Gerar ações válidas.
- Manter estado local do personagem.
- Identificar desconexões.
- Recuperar conexões quando permitido.
- Registrar eventos.
- Medir latência.
- Medir erros.
- Permitir execução de cenários.
- Suportar múltiplas instâncias.

O cliente headless deve respeitar os contratos reais do protocolo.

Não utilize o cliente headless para contornar validações do servidor. O objetivo é testar o comportamento do sistema por meio de interações controladas e reproduzíveis.

---

## 4. POPULAÇÃO ARTIFICIAL E PERFIS DE JOGADORES

Crie um sistema para representar uma população artificial de jogadores.

Os bots não devem ser todos idênticos.

Cada personagem deve possuir um perfil comportamental configurável.

### 4.1. Perfis de bots

Considere os seguintes perfis:

**Casual**
- Tempo de atividade reduzido.
- Movimentação limitada.
- Atividades simples.
- Progressão lenta.

**Farmer**
- Foco em caça.
- Coleta de recursos.
- Venda de itens.
- Acúmulo de gold.

**Combatente**
- Busca por combates.
- Utilização de skills.
- Troca de equipamentos.
- Participação em PvP controlado.

**Comerciante**
- Compra e venda.
- Avaliação de preços.
- Gestão de inventário.
- Participação no mercado.

**Explorador**
- Navegação entre regiões.
- Descoberta de áreas.
- Interação com NPCs.
- Investigação de mapas.

**Social**
- Formação de grupos.
- Interação com outros personagens.
- Chat.
- Participação em guildas.

**Quest**
- Execução de missões.
- Coleta de objetivos.
- Interação com NPCs.
- Validação de recompensas.

**Stress Tester**
- Execução de operações intensivas.
- Conexões simultâneas.
- Movimento.
- Combate.
- Consultas e operações repetitivas.

**Bug Hunter**
- Execução de sequências específicas.
- Testes de limites.
- Reprodução de situações anormais.
- Busca por inconsistências.

### 4.2. Personalização

Cada bot deve permitir configuração de:

- Classe.
- Nível.
- Atributos.
- Equipamentos.
- Build.
- Inventário.
- Skills.
- Região.
- Objetivos.
- Perfil comportamental.
- Tempo de atividade.
- Frequência de ações.
- Estratégia.
- Limites de recursos.

Utilize os dados reais do jogo para configurar os personagens.

### 4.3. População heterogênea

Crie cenários com distribuições diferentes de personagens.

Exemplos:

- 20% combatentes.
- 30% farmers.
- 15% comerciantes.
- 15% casuais.
- 10% exploradores.
- 10% sociais.

Essas proporções são exemplos configuráveis, não valores obrigatórios.

O sistema deve permitir simular diferentes composições de população.

---

## 5. CENÁRIO A — STRESS TESTING E TESTES DE CAPACIDADE

Implemente um ambiente para avaliar os limites operacionais do servidor.

### 5.1. Objetivo

Simular centenas ou milhares de conexões simultâneas para avaliar:

- Capacidade de conexão.
- Estabilidade.
- Processamento de pacotes.
- Concorrência.
- Consumo de CPU.
- Consumo de memória.
- Latência.
- Throughput.
- Vazamentos de memória.
- Gargalos.
- Estabilidade do banco.
- Comportamento sob carga.

### 5.2. Cenários de carga

Implemente diferentes perfis:

**Carga progressiva**

Aumentar gradualmente a quantidade de bots conectados.

Exemplo:

- 10 bots.
- 50 bots.
- 100 bots.
- 250 bots.
- 500 bots.
- 1.000 bots.

**Carga constante**

Manter determinada população conectada e executando atividades durante um período definido.

**Pico de conexões**

Simular uma quantidade elevada de conexões em um curto intervalo.

**Stress test**

Aumentar progressivamente a carga até identificar limites, degradação ou falhas.

**Soak test**

Manter o sistema sob carga prolongada para investigar vazamentos, degradação e instabilidade acumulada.

**Teste de recuperação**

Avaliar o comportamento após interrupções, desconexões, reinicializações e recuperação de serviços.

### 5.3. Métricas

Colete:

- Conexões simultâneas.
- Conexões estabelecidas por segundo.
- Desconexões.
- Pacotes enviados.
- Pacotes recebidos.
- Bytes transmitidos.
- Latência média.
- P95.
- P99.
- Taxa de erros.
- CPU.
- Memória.
- Consumo de rede.
- Uso do banco.
- Tempo de consultas.
- Filas.
- Tempo de resposta.

Identifique o ponto em que a qualidade do serviço começa a degradar.

### 5.4. Segurança e isolamento

Os testes de carga devem ser executados exclusivamente em ambientes autorizados e controlados.

Não execute stress tests em produção sem autorização explícita e planejamento específico.

Estabeleça limites de carga, mecanismos de interrupção e proteção contra geração acidental de tráfego excessivo.

---

## 6. CENÁRIO B — SIMULAÇÃO DA ECONOMIA DO JOGO

Desenvolva um sistema para simular a economia virtual do WYD.

O objetivo é compreender como a interação de uma população artificial afeta os recursos, os preços e a progressão econômica.

### 6.1. Atividades econômicas

Os bots devem ser capazes de:

- Caçar.
- Coletar drops.
- Obter baús.
- Adquirir equipamentos.
- Refinar itens.
- Utilizar Ori e Ethereal, conforme as regras da versão.
- Comprar consumíveis.
- Vender equipamentos.
- Negociar itens.
- Abrir lojas.
- Comprar de outros personagens.
- Acumular gold.
- Consumir recursos.
- Participar de atividades econômicas.

### 6.2. Simulação de mercado

Implemente mecanismos para representar:

- Oferta.
- Demanda.
- Preços.
- Estoque.
- Volume de negociação.
- Liquidez.
- Concentração de riqueza.
- Inflação.
- Deflação.
- Acúmulo de recursos.
- Desvalorização de itens.
- Escassez.

Os bots comerciantes devem utilizar estratégias configuráveis, sem depender obrigatoriamente de IA generativa.

### 6.3. Indicadores econômicos

Avalie:

- Gold gerado.
- Gold consumido.
- Gold acumulado.
- Quantidade de itens produzidos.
- Quantidade de itens destruídos.
- Volume de vendas.
- Preços médios.
- Distribuição de riqueza.
- Taxa de refinamento.
- Recursos por hora.
- Progressão econômica.
- Inflação ao longo do tempo.

### 6.4. Balanceamento econômico

Utilize os resultados para identificar:

- Drops excessivamente lucrativos.
- Itens sem demanda.
- Recursos escassos.
- Sistemas de geração excessiva de gold.
- Sistemas de consumo insuficientes.
- Refinamentos desbalanceados.
- Progressão econômica acelerada.
- Possíveis mecanismos de exploração.

Não altere automaticamente a economia do servidor com base em uma simulação isolada.

Primeiro valide os cenários e documente as conclusões.

---

## 7. CENÁRIO C — BUG HUNTING E TESTES FUNCIONAIS AUTOMATIZADOS

Crie bots especializados em executar sequências de ações que ajudem a descobrir falhas funcionais e problemas de concorrência.

### 7.1. Sistemas a testar

Considere:

- Login.
- Seleção de personagens.
- Movimentação.
- Ataques.
- Skills.
- Inventário.
- Equipamentos.
- Troca de itens.
- Trade.
- Refinamento.
- Montarias.
- Grupos.
- Guildas.
- Quests.
- Eventos.
- Guerra de Torres.
- Guerra de Cidades.
- NPCs.
- Respawn.
- Morte.
- Drops.
- Recompensas.
- Persistência.
- Desconexões.

### 7.2. Cenários simultâneos

Implemente testes que executem diferentes operações em paralelo.

Exemplos:

- Diversos personagens tentando negociar simultaneamente.
- Personagens equipando e removendo itens durante movimentação.
- Personagens morrendo durante atividades de combate.
- Personagens entrando e saindo de áreas de guerra.
- Múltiplos personagens interagindo com o mesmo NPC.
- Personagens formando e desfazendo grupos.
- Personagens entrando e saindo de guildas.
- Personagens realizando operações simultâneas de inventário.
- Personagens utilizando skills em sequência.
- Personagens desconectando durante operações de persistência.

### 7.3. Identificação de problemas

Investigue:

- Race conditions.
- Deadlocks.
- Duplicação de itens.
- Perda de itens.
- Duplicação de recompensas.
- Inconsistência de inventário.
- Falhas de persistência.
- Problemas de concorrência no banco.
- Estados inválidos.
- Desincronização cliente-servidor.
- Falhas de quests.
- Problemas de respawn.
- Travamentos.
- Vazamentos de memória.
- Exploração de regras.

### 7.4. Reprodução de bugs

Todo bug identificado deve ser reproduzível.

Registre:

- Identificador do cenário.
- Perfil dos bots.
- Personagens envolvidos.
- Estado inicial.
- Sequência de ações.
- Pacotes envolvidos, quando relevantes.
- Horário.
- Logs.
- Resultado esperado.
- Resultado observado.
- Evidências.
- Possível causa raiz.

Permita executar novamente a mesma sequência com os mesmos parâmetros.

---

## 8. CENÁRIO D — SIMULAÇÃO DE COMPORTAMENTO SOCIAL

Implemente uma população com comportamentos sociais configuráveis.

O objetivo é aproximar a atividade do servidor de uma população de jogadores com diferentes interesses.

Considere:

- Conversas no chat.
- Formação de grupos.
- Formação de guildas.
- Convites.
- Comércio.
- Negociações.
- Participação em eventos.
- Interação com NPCs.
- Permanência em cidades.
- Deslocamento entre regiões.
- Participação em guerras.
- Busca por recursos.

Os bots devem ter perfis de comportamento e objetivos próprios.

A simulação social pode utilizar regras determinísticas e, opcionalmente, modelos de linguagem para gerar diálogos.

Os diálogos devem respeitar limites de frequência e conteúdo configuráveis, evitando inundar o servidor com mensagens artificiais.

---

## 9. ORQUESTRAÇÃO E EXECUÇÃO EM CONTAINERS

Desenvolva uma infraestrutura que permita executar os bots de maneira isolada e escalável.

### 9.1. Docker

Avalie a utilização de Docker para empacotar:

- Clientes headless.
- Workers de bots.
- Gerenciadores de cenários.
- Serviços de simulação.
- Ferramentas de análise.
- Serviços opcionais de IA.

### 9.2. Docker Compose

Crie uma estrutura de ambiente de desenvolvimento capaz de executar:

- Servidor WYD de testes.
- Banco de dados.
- Redis, quando necessário.
- Gerenciador de bots.
- Workers.
- Coletor de métricas.
- Prometheus.
- Grafana.
- Serviços auxiliares.

Os containers devem utilizar redes internas apropriadas.

A comunicação entre os componentes deve ser documentada.

### 9.3. Escalabilidade

Avalie estratégias para executar:

- Um bot por processo.
- Múltiplos bots por processo.
- Múltiplos workers.
- Múltiplos containers.
- Distribuição entre máquinas.
- Orquestração em Kubernetes, quando houver necessidade real.

Não crie um container individual para cada bot sem avaliar o custo operacional.

Priorize uma arquitetura que consiga executar uma população grande com baixo consumo de recursos.

### 9.4. Gerenciamento centralizado

Implemente um controlador central para:

- Criar cenários.
- Iniciar execuções.
- Interromper execuções.
- Configurar quantidade de bots.
- Distribuir perfis.
- Monitorar workers.
- Coletar resultados.
- Identificar falhas.
- Gerenciar execuções.

O controlador não deve depender de uma interface gráfica para funcionar.

Avalie disponibilizar CLI e API.

---

## 10. OBSERVABILIDADE E ANÁLISE DOS BOTS

Integre o ecossistema de bots à infraestrutura de observabilidade do projeto.

### 10.1. Métricas dos bots

Colete:

- Bots configurados.
- Bots ativos.
- Bots conectados.
- Bots desconectados.
- Bots em execução.
- Bots com erros.
- Estados atuais.
- Ações por segundo.
- Pacotes por segundo.
- Latência.
- Tempo de atividade.
- Consumo de recursos.
- Cenários concluídos.
- Cenários interrompidos.
- Bugs identificados.

### 10.2. Dashboard

Avalie a criação de um dashboard específico para o ambiente de simulação.

O dashboard deve permitir:

- Acompanhar a população.
- Visualizar estados dos bots.
- Monitorar conexões.
- Identificar falhas.
- Acompanhar testes de carga.
- Visualizar métricas de desempenho.
- Comparar execuções.
- Consultar resultados.
- Acompanhar indicadores econômicos.
- Investigar bugs.

Integre com Prometheus e Grafana quando adequado.

### 10.3. Logs e rastreabilidade

Cada execução deve possuir um identificador único.

Os eventos devem permitir correlacionar:

- Cenário.
- Worker.
- Bot.
- Personagem.
- Conexão.
- Ação.
- Pacote.
- Erro.
- Resultado.

Os logs devem ser estruturados e permitir reconstruir o comportamento de uma execução.

---

## 11. TESTES DETERMINÍSTICOS, REPRODUTIBILIDADE E QUALIDADE

A plataforma deve permitir repetir cenários de maneira consistente.

### 11.1. Seeds e aleatoriedade controlada

Quando os bots utilizarem comportamentos probabilísticos:

- Utilize seeds configuráveis.
- Registre as seeds utilizadas.
- Permita reproduzir a mesma sequência.
- Diferencie aleatoriedade controlada de comportamento não determinístico.

### 11.2. Cenários versionados

Armazene os cenários de teste em arquivos versionados.

Cada cenário deve definir:

- Objetivo.
- População.
- Perfis.
- Configurações.
- Duração.
- Ações.
- Condições.
- Métricas.
- Critérios de sucesso.
- Critérios de falha.

### 11.3. Testes de regressão

Quando um bug for identificado e corrigido, transforme o cenário correspondente em teste de regressão.

O objetivo é impedir que problemas já resolvidos reapareçam em alterações futuras.

### 11.4. Integração com CI/CD

Avalie a integração de diferentes níveis de teste nos pipelines.

Considere:

**Pull Requests**
- Testes unitários do cliente headless.
- Testes de protocolo.
- Testes funcionais rápidos.
- Cenários de regressão.
- Validação de configurações.

**Integração**
- Pequenas populações de bots.
- Testes de sistemas.
- Testes de persistência.
- Testes de concorrência controlados.

**Execuções periódicas**
- Testes de carga.
- Soak tests.
- Simulações econômicas.
- Testes de população ampliada.
- Bug Hunting automatizado.

Não execute testes de carga pesados em todos os Pull Requests sem avaliar custo, duração e estabilidade do ambiente.

---

## 12. SEGURANÇA, ISOLAMENTO E INTEGRIDADE DO AMBIENTE

A plataforma de bots deve funcionar exclusivamente em ambientes controlados e autorizados.

Implemente:

- Contas exclusivas para testes.
- Personagens identificáveis como bots.
- Separação de dados.
- Isolamento de rede.
- Limites de execução.
- Limites de conexões.
- Controle de recursos.
- Interrupção de emergência.
- Proteção contra execução acidental em produção.
- Auditoria de operações.
- Limpeza controlada de dados de teste.

Os bots não devem utilizar contas reais de jogadores nem modificar dados de produção.

O ambiente deve possuir mecanismos explícitos para impedir que um cenário de stress seja executado contra um destino não autorizado.

---

## 13. LABORATÓRIO DE EXPERIMENTOS E SIMULAÇÃO INTEGRADA

Evolua o ecossistema para permitir experimentos que combinem os resultados das demais fases do projeto.

Integre os bots aos conhecimentos produzidos pela:

- FASE 3 — Servidor.
- FASE 4 — Banco de dados.
- FASE 5 — DevOps e CI/CD.
- FASE 6 — Observabilidade e segurança.
- FASE 8 — Engenharia reversa do cliente.
- FASE 9 — IA e agentes.
- FASE 10 — Matemática e balanceamento do WYD.
- FASE 12 — Dashboard.

Exemplos de experimentos integrados:

- Comparar DPS de diferentes builds.
- Validar a curva de velocidade de ataque.
- Simular milhares de ataques.
- Comparar classes.
- Avaliar eficiência de skills.
- Simular progressão de personagens.
- Comparar taxas de drops.
- Investigar inflação.
- Testar alterações em fórmulas.
- Identificar regressões após mudanças no servidor.
- Comparar comportamento de diferentes versões.

Utilize os resultados matemáticos da FASE 10 para construir bots que executem cenários realistas de combate e validem o comportamento do servidor.

A plataforma deve permitir comparar o comportamento esperado, calculado pelo simulador matemático, com os resultados observados no servidor real.

---

## 14. ARQUITETURA RECOMENDADA E DECISÕES TÉCNICAS

Antes da implementação, realize uma análise comparativa das tecnologias e defina uma arquitetura adequada.

Avalie especialmente:

- Go para o núcleo de execução dos bots.
- Python para automação, análise e experimentação.
- Docker para isolamento.
- Docker Compose para ambientes locais.
- Kubernetes para escalabilidade futura, caso necessário.
- Prometheus para métricas.
- Grafana para visualização.
- Redis para coordenação ou filas, caso necessário.
- PostgreSQL ou o banco já utilizado pelo projeto para armazenamento de execuções e resultados.
- Ollama ou llama.cpp para LLMs opcionais.

Não adote todas as tecnologias automaticamente.

A escolha deve considerar o que já existe no projeto, a compatibilidade, a simplicidade operacional e o custo de manutenção.

A arquitetura deve separar pelo menos os seguintes componentes:

1. **Bot Engine:** execução dos comportamentos.
2. **Protocol Client:** comunicação com o servidor WYD.
3. **Bot Controller:** gerenciamento centralizado.
4. **Scenario Manager:** definição e execução dos cenários.
5. **Navigation Engine:** navegação e pathfinding.
6. **Behavior Engine:** FSM, Behavior Trees e heurísticas.
7. **Simulation Engine:** simulações econômicas e populacionais.
8. **Metrics Collector:** coleta de métricas.
9. **Results Store:** persistência dos resultados.
10. **Dashboard:** visualização e acompanhamento.
11. **AI Adapter:** integração opcional com LLMs.
12. **Test Integration:** integração com testes e CI/CD.

---

## 15. ROADMAP DE IMPLEMENTAÇÃO

Divida a implementação em etapas independentes e progressivas.

### ETAPA 1 — Pesquisa e arquitetura

- Auditar o protocolo.
- Identificar operações necessárias.
- Avaliar linguagens.
- Definir arquitetura.
- Criar ADRs.
- Mapear dependências.
- Criar Issues.

### ETAPA 2 — Cliente headless mínimo

- Implementar conexão.
- Implementar autenticação.
- Implementar seleção de personagem.
- Implementar entrada no mundo.
- Implementar leitura de mensagens.
- Implementar desconexão.
- Criar testes de protocolo.

### ETAPA 3 — Motor de bots

- Implementar FSM.
- Implementar estados.
- Implementar transições.
- Implementar perfis.
- Implementar execução de ações.
- Implementar gerenciamento de erros.

### ETAPA 4 — Navegação e combate

- Implementar navegação.
- Implementar waypoints.
- Implementar pathfinding.
- Implementar identificação de alvos.
- Implementar ataques.
- Implementar coleta.
- Implementar gerenciamento de recursos.

### ETAPA 5 — Orquestração

- Criar controlador.
- Criar workers.
- Implementar execução paralela.
- Criar containers.
- Criar Docker Compose.
- Implementar controle de cenários.

### ETAPA 6 — Stress testing

- Implementar carga progressiva.
- Implementar carga constante.
- Implementar pico de conexões.
- Implementar soak tests.
- Coletar métricas.
- Identificar limites.

### ETAPA 7 — Bug Hunting

- Implementar cenários funcionais.
- Implementar operações concorrentes.
- Reproduzir falhas.
- Registrar evidências.
- Criar testes de regressão.

### ETAPA 8 — Simulação econômica

- Implementar perfis econômicos.
- Implementar coleta.
- Implementar comércio.
- Implementar refinamento.
- Simular mercado.
- Analisar inflação e distribuição de riqueza.

### ETAPA 9 — Comportamento social

- Implementar grupos.
- Implementar guildas.
- Implementar chat.
- Implementar eventos.
- Implementar perfis sociais.
- Avaliar LLMs opcionais.

### ETAPA 10 — Observabilidade e CI/CD

- Integrar métricas.
- Criar dashboards.
- Implementar logs estruturados.
- Criar testes automatizados.
- Integrar cenários rápidos aos pipelines.
- Configurar execuções periódicas.

### ETAPA 11 — Simulação matemática integrada

- Integrar fórmulas da FASE 10.
- Comparar resultados esperados e observados.
- Testar builds.
- Testar velocidade.
- Testar dano.
- Validar regressões.
- Gerar relatórios de balanceamento.

---

## 16. CRITÉRIOS DE ACEITAÇÃO E CONCLUSÃO

Esta meta será considerada implementada quando existir uma plataforma funcional capaz de:

1. Conectar bots ao servidor sem abrir o cliente gráfico.
2. Executar múltiplos personagens simultaneamente.
3. Gerenciar comportamentos por FSM ou Behavior Trees.
4. Navegar por regiões do jogo.
5. Executar atividades básicas de combate.
6. Realizar operações de inventário.
7. Simular atividades econômicas.
8. Executar cenários de teste funcional.
9. Realizar testes de carga controlados.
10. Identificar e registrar falhas.
11. Reproduzir cenários determinísticos.
12. Coletar métricas de execução.
13. Exibir resultados em dashboards.
14. Executar em containers.
15. Integrar cenários selecionados ao CI/CD.
16. Operar sem depender de LLMs.
17. Permitir integração opcional com modelos locais.
18. Comparar resultados de combate com as fórmulas da FASE 10.
19. Preservar a segurança e o isolamento dos ambientes.
20. Possuir documentação técnica e operacional completa.

---

## 17. RESULTADO ESPERADO

O resultado final deve ser um ecossistema independente de simulação do WYD, capaz de criar uma população artificial de jogadores e executar atividades realistas em larga escala.

Essa infraestrutura deverá funcionar como:

- Laboratório de testes do servidor.
- Plataforma de load testing.
- Simulador de população.
- Simulador econômico.
- Ambiente de testes funcionais.
- Bug Hunter automatizado.
- Ferramenta de validação matemática.
- Ambiente de reprodução de falhas.
- Plataforma de regressão.
- Ferramenta de análise de capacidade.

**A visão de longo prazo é possuir uma população artificial controlável, reproduzível e escalável, capaz de exercitar continuamente os sistemas do WYD, revelar problemas antes dos jogadores, validar alterações e fornecer evidências concretas sobre a estabilidade, capacidade, economia e balanceamento do jogo.**

A implementação deve começar com um núcleo determinístico, leve e sem dependências de LLM. A inteligência generativa deve ser incorporada somente quando houver benefício demonstrável e sem comprometer o desempenho, a previsibilidade ou a capacidade de reprodução dos testes.