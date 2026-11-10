# MISSÃO: WYD2 — PESQUISA PROFUNDA, ENGENHARIA DE CONHECIMENTO, AUDITORIA FUNCIONAL E MAPEAMENTO COMPLETO DE IMPLEMENTAÇÃO

## 1. CONTEXTO E OBJETIVO PRINCIPAL

Você é o agente de IA responsável pela pesquisa, análise, documentação, engenharia de conhecimento e auditoria funcional completa do projeto WYD2.

Sua missão é adquirir conhecimento técnico aprofundado sobre o funcionamento do WYD2, abrangendo todos os sistemas, mecânicas, regras de negócio, fórmulas matemáticas, atributos, progressão, itens, equipamentos, mapas, NPCs, quests, eventos, economia, comunicação, funcionalidades do cliente e comportamentos do servidor.

Você deve atuar como um especialista em:

- WYD / With Your Destiny.
- WYD2 e suas diferentes versões.
- Arquitetura de servidores MMORPG.
- Engenharia reversa documental e análise de sistemas legados.
- Análise de clientes e protocolos de jogos.
- Matemática e balanceamento de MMORPGs.
- Engenharia de software e arquitetura de sistemas.
- Segurança de servidores e prevenção de exploits.
- Documentação técnica e gerenciamento de projetos.
- Planejamento de implementação e gestão de issues.

**O objetivo não é apenas conhecer superficialmente as funcionalidades. Você precisa entender como cada sistema funciona, como se relaciona com os demais, quais são suas regras, fórmulas, condições, limitações, dependências e comportamentos esperados.**

Queremos construir uma base de conhecimento técnica suficientemente completa para que possamos implementar, reproduzir, testar, corrigir e evoluir todos os sistemas do jogo no nosso servidor WYD2.

Não presuma que os sistemas listados abaixo representam a totalidade do jogo. Eles são o ponto de partida obrigatório da investigação.

Você deve identificar funcionalidades adicionais, sistemas esquecidos, comportamentos implícitos, mecânicas secundárias e dependências que não foram mencionadas.

---

# 2. REFERÊNCIAS OBRIGATÓRIAS DO PROJETO

Antes de iniciar qualquer levantamento, estude integralmente as referências disponíveis no projeto.

## 2.1. Projeto legado — OpenWYD

Repositório de referência:

- OpenWYD — GitHub.
- Branch `main`.

Utilize o código legado como fonte de investigação do comportamento original do jogo.

Analise, quando disponíveis:

- Estruturas de dados.
- Entidades e personagens.
- Atributos e fórmulas.
- Regras de negócio.
- Sistemas de combate.
- Sistemas de itens.
- Tabelas e configurações.
- Estruturas de mapas.
- NPCs e monstros.
- Quests e eventos.
- Protocolos de comunicação.
- Processamento de pacotes.
- Persistência.
- Regras de progressão.
- Configurações do servidor.
- Comportamentos do cliente.
- Implementações incompletas ou legadas.
- Bugs conhecidos e correções existentes.

Não se limite à leitura de nomes de funções ou arquivos. Investigue o fluxo de execução e as regras efetivamente implementadas.

## 2.2. Documentação e instruções locais

Consulte obrigatoriamente:

`D:\dolorestec\wyd2\OpenWYD\pre-release-rust.md`

`D:\dolorestec\wyd2\OpenWYD\start-rust.md`

`D:\dolorestec\wyd2\OpenWYD\docs`

Utilize todas as referências e skills disponíveis no projeto.

Antes de criar documentação ou issues, identifique:

- Padrões documentais existentes.
- Templates oficiais.
- Convenções de nomenclatura.
- Estrutura de diretórios.
- Padrões de implementação.
- Organização do roadmap.
- Critérios de conclusão.
- Padrões de issues.
- Checklists existentes.
- Decisões arquiteturais já documentadas.

Não crie uma estrutura documental paralela quando já existir um padrão oficial.

## 2.3. Cliente do projeto

Diretório:

`D:\dolorestec\wyd2\OpenWYD\client`

Este é o cliente que será utilizado pelo nosso servidor.

Analise seu funcionamento e utilize-o como referência principal para compreender a integração cliente/servidor que estamos desenvolvendo.

Investigue:

- Funcionalidades existentes.
- Interface e telas.
- Fluxos de navegação.
- Personagem e inventário.
- Equipamentos.
- Skills.
- Barra de habilidades.
- Mapas.
- NPCs.
- Comunicação.
- Login e seleção de personagens.
- Mudança de canais.
- Loja.
- Trocas.
- Grupos.
- PVP.
- Metereologia do jogo e seus efeitos.
- Mensagens.
- Efeitos visuais.
- Sons.
- Recursos gráficos.
- Funcionalidades implementadas e ausentes.

## 2.4. Cliente original do WYD Global — referência externa

Diretório:

`C:\Users\Canta\AppData\Local\wyd_launcher`

Este diretório contém o cliente real do WYD Global, relacionado à versão RaidHunt.

É uma referência de terceiros, destinada exclusivamente à investigação e comparação de comportamentos reais.

Utilize-o para conferir funcionalidades, interfaces, mecânicas, configurações, efeitos, comportamentos e informações que possam não estar suficientemente documentadas no OpenWYD.

Não confunda funcionalidades específicas dessa versão com comportamentos universais de todas as versões do WYD.

Registre diferenças entre versões sempre que forem relevantes.

---

# 3. METODOLOGIA DE PESQUISA

A pesquisa deve ser profunda, sistemática e baseada em múltiplas fontes.

## 3.1. Fontes de investigação

Pesquise e confronte informações provenientes de:

1. Código-fonte do OpenWYD.
2. Documentação oficial e histórica disponível.
3. Fóruns antigos e comunidades do WYD.
4. Fóruns brasileiros e internacionais.
5. Servidores privados.
6. Comunidades e discussões de jogadores.
7. Wikis e bancos de dados de itens.
8. Documentações técnicas disponíveis.
9. Vídeos demonstrativos e tutoriais.
10. Materiais históricos de versões anteriores.
11. Cliente original do WYD Global.
12. Configurações e arquivos de recursos do cliente.
13. Issues, commits e histórico do projeto.
14. Skills e documentação interna do nosso projeto.

Pesquise também informações em diferentes idiomas, especialmente português, inglês e espanhol.

Não confie cegamente em uma única fonte.

Quando encontrar informações divergentes:

- Registre as versões envolvidas.
- Identifique a origem de cada informação.
- Compare as evidências.
- Determine qual comportamento é efetivamente utilizado como referência.
- Documente as incertezas.

## 3.2. Classificação das informações

Todo conhecimento coletado deve receber uma classificação:

- **CONFIRMADO:** comprovado por implementação, documentação confiável ou testes reproduzíveis.
- **VALIDADO:** confirmado por múltiplas fontes independentes.
- **PROVÁVEL:** sustentado por evidências, mas sem confirmação definitiva.
- **HIPÓTESE:** comportamento ainda não comprovado.
- **DIVERGENTE:** existem diferenças entre versões ou fontes.
- **DESCONHECIDO:** informação que ainda precisa ser investigada.

Sempre identifique a versão do jogo à qual a informação pertence, quando conhecida.

Não transforme hipóteses em regras oficiais.

## 3.3. Registro das fontes

Para cada sistema pesquisado, registre:

- Fonte.
- URL ou caminho do arquivo.
- Versão do jogo.
- Trecho de código, quando aplicável.
- Arquivo e função relacionados.
- Evidência encontrada.
- Conclusão.
- Grau de confiança.
- Informações ainda pendentes.

Crie rastreabilidade entre conhecimento, implementação existente, gaps e issues.

---

# 4. MAPEAMENTO COMPLETO DOS SISTEMAS DO JOGO

Todos os tópicos abaixo são obrigatórios.

Para cada sistema, investigue o funcionamento completo, suas regras, fórmulas, configurações, dependências, comportamentos no cliente e servidor, interações com outros sistemas e critérios de validação.

## 4.1. Personagem, atributos e distribuição de pontos

Investigue completamente:

- Distribuição de pontos do personagem.
- Atributos básicos.
- Atributos derivados.
- Atributos ocultos ou internos.
- Matemática de atributos.
- O que cada atributo aumenta.
- Influência de cada ponto na performance.
- Limites mínimos e máximos.
- Fórmulas de cálculo.
- Arredondamentos.
- Multiplicadores.
- Penalidades.
- Bônus temporários e permanentes.
- Atributos provenientes de equipamentos.
- Atributos provenientes de skills.
- Atributos provenientes de montarias e crias.
- Atributos provenientes de buffs.
- Atributos provenientes de eventos.
- Diferenças entre atributos exibidos e valores efetivamente utilizados pelo servidor.

Investigue também todos os atributos adicionais que não foram mencionados.

## 4.2. Combate, velocidade e sobrevivência

Mapeie integralmente:

**Velocidade**
- Velocidade de ataque.
- Velocidade de movimento.
- Influência dos atributos.
- Limites e caps.
- Intervalos entre ataques.
- Influência de equipamentos e skills.
- Diferenças entre velocidade exibida e efetiva.

**Crítico**
- Sistema de ataques críticos.
- Probabilidade de crítico.
- Quantidade de ataques críticos.
- Relação entre velocidade e frequência de críticos.
- Fórmulas e multiplicadores.
- Limites.
- Influência de equipamentos, buffs e atributos.

**Evasão**
- Cálculo de evasão.
- Probabilidade de esquiva.
- Influência dos atributos.
- Relação entre precisão e evasão.
- Limites e condições.

**Imunidade**
- Tipos de imunidade.
- Resistências.
- Imunidade a efeitos.
- Imunidade a ataques.
- Condições de ativação.
- Interação com skills e equipamentos.

**Defesa**
- Defesa física.
- Defesa mágica, se existente.
- Redução de dano.
- Absorção.
- Resistências.
- Cálculo de dano recebido.
- Relação entre ataque, defesa e nível.
- Fórmulas e caps.

**HP e Mana**
- Fórmulas de HP.
- Fórmulas de MP.
- Recuperação.
- Regeneração.
- Consumo.
- Limites.
- Influência de atributos.
- Influência de equipamentos e buffs.
- Comportamento em combate.

Investigue também dano mínimo e máximo, precisão, resistência elemental, redução de dano, penetração, bloqueio, reflect, roubo de vida, roubo de mana, efeitos de status, controle de multidões e demais mecânicas de combate existentes.

## 4.3. Experiência, evolução e progressão

Investigue:

- XP recebido ao matar monstros.
- XP por nível e faixa de nível.
- XP de quests.
- XP de itens de quest.
- Itens de bônus.
- Bônus temporários.
- Bônus permanentes.
- Bônus de grupo.
- Bônus de canal.
- Bônus de eventos.
- Multiplicadores.
- Penalidades.
- Regras de distribuição.
- Influência de nível e diferença de nível.
- Fórmulas de evolução.
- Limites de XP.
- Regras de progressão.

Compare separadamente:

- Mortal.
- Arch.
- Celestial.
- Subcelestial.

Investigue as diferenças de nivelamento, requisitos, progressão, XP, quests, equipamentos, atributos e funcionalidades disponíveis em cada estágio.

## 4.4. Sistemas de Mortal, Arch, Celestial e Subcelestial

Documente cada estágio como uma progressão independente e também como parte da evolução geral do personagem.

**Mortal**
- Todas as funcionalidades disponíveis.
- Progressão completa.
- Quests obrigatórias e opcionais.
- Requisitos.
- Itens necessários.
- NPCs.
- Mapas.
- Equipamentos.
- Skills.
- Limitações.
- Sistemas desbloqueados.
- Recompensas.
- Transição para Arch.

**Arch**
- Criação e requisitos.
- Quests necessárias.
- Progressão.
- Atributos.
- Skills.
- Equipamentos.
- Sistemas disponíveis.
- Restrições.
- Recompensas.
- Transição para Celestial.

**Celestial**
- Requisitos.
- Quests.
- Progressão.
- Atributos.
- Equipamentos.
- Skills.
- Sistemas exclusivos.
- Limitações.
- Recompensas.
- Integração com os demais estágios.

**Subcelestial**
- Requisitos.
- Criação.
- Progressão.
- Quests.
- Atributos.
- Equipamentos.
- Skills.
- Sistemas exclusivos.
- Limitações.
- Recompensas.

Identifique todas as diferenças entre os quatro estágios, inclusive aquelas que não são visíveis diretamente na interface.

## 4.5. Cidadania, Kibita e reinos

Investigue:

- Sistema de cidadania.
- Registro de cidadania.
- Mudança de cidadania.
- Restrições.
- Requisitos.
- NPCs relacionados.
- Funcionamento da Kibita.
- Relação entre cidadania e reinos.
- Benefícios e penalidades.
- Regras de permanência.
- Integração com guerras e guilds.

**Reinos:**

Mapeie completamente os reinos, suas funcionalidades, hierarquias, sistemas, NPCs, quests, permissões, territórios, benefícios, impostos, guerras e interações entre jogadores.

Investigue as funções administrativas e econômicas dos reinos e suas relações com as cidades.

## 4.6. Mapas, zonas, teletransportes e localização

Faça um levantamento completo de todos os mapas e zonas conhecidos do WYD.

Não se limite aos mapas citados abaixo.

Investigue:

- Cidades.
- Áreas externas.
- Zonas de caça.
- Mapas de quests.
- Mapas de eventos.
- Mapas de chefes.
- Mapas especiais.
- Zonas de acesso restrito.
- Zonas PVP.
- Zonas seguras.
- Zonas de guild.
- Mapas de progressão.
- Áreas de treinamento.

Investigue especificamente:

- Pergaminhos.
- Pesadelos.
- Zona da LAN.
- Kefra.
- Terra de Fogo.
- Mapas de gelo.
- Água.
- Coliseu.
- Demais zonas especiais.

Para cada mapa, documente:

- Nome e identificação.
- Coordenadas e dimensões.
- Entradas e saídas.
- NPCs.
- Monstros.
- Bosses.
- Localizações.
- Teletransportes.
- Pergaminhos relacionados.
- Requisitos de acesso.
- Restrições.
- Quests associadas.
- Eventos associados.
- Horários.
- Respawn.
- Mecânicas especiais.
- Interações entre jogadores.

Crie um mapeamento de localização de todos os monstros, NPCs e bosses, incluindo posições, áreas de spawn, intervalos de respawn e condições especiais.

## 4.7. Sistema de monstros e bosses

Investigue:

- Tipos de monstros.
- Classes e categorias.
- Níveis.
- Atributos.
- HP.
- Ataque.
- Defesa.
- Velocidade.
- Resistências.
- Habilidades.
- IA e comportamento.
- Agressividade.
- Perseguição.
- Alcance.
- Distância de ataque.
- Respawn.
- Drops.
- XP.
- Bosses.
- Bosses especiais.
- Eventos de spawn.
- Condições de aparecimento.
- Recompensas.
- Interação com jogadores e grupos.

Documente os comportamentos de IA e as diferenças entre monstros comuns, elites e chefes, quando aplicável.

## 4.8. Itens, equipamentos e catálogo completo

Precisamos construir um catálogo extremamente completo de itens.

Investigue o sistema de itens do WYD em sua totalidade.

**Tipos de equipamentos:**
- Armas.
- Armaduras.
- Escudos.
- Elmos.
- Luvas.
- Botas.
- Acessórios.
- Anéis.
- Colares.
- Equipamentos específicos de cada classe.
- Equipamentos Mortais.
- Equipamentos Arch.
- Equipamentos Celestiais.
- Equipamentos Subcelestiais.
- Equipamentos especiais.

Documente diferenças, restrições, limites, requisitos, utilização e progressão de equipamentos.

**Sistema de atributos dos itens:**
- Tipos de atributos.
- Adicionais.
- Bônus.
- Combinações comuns.
- Combinações raras.
- Atributos aleatórios.
- Atributos fixos.
- Limites.
- Compatibilidade.
- Efeitos no personagem.
- Regras de composição.

**Classes de itens:**
- Classe A.
- Classe B.
- Classe C.
- Classe D.
- Classe E.
- Classe F.
- Outras classes existentes.

Investigue a finalidade de cada classe, seus requisitos, restrições, relações com refinação e composição.

**Raridade:**

Pesquise as raridades dos itens, suas classificações, diferenças, probabilidades, origem, obtenção, atributos e influência na economia.

**Catálogo:**

Aprimore o catálogo existente do projeto com o máximo de informações relevantes que puder ser comprovado.

O objetivo é termos controle e conhecimento real sobre o sistema de itens, incluindo suas propriedades, comportamentos, restrições, combinações, dependências e formas de obtenção.

## 4.9. Drop e obtenção de itens

Investigue:

- Sistema de drop.
- Probabilidades.
- Tabelas.
- Drop por monstro.
- Drop por boss.
- Drop por mapa.
- Drop por evento.
- Drop por quest.
- Drop especial.
- Drop de equipamentos.
- Drop de consumíveis.
- Drop de moedas.
- Limites.
- Bônus de drop.
- Bônus por canal.
- Multiplicadores.
- Influência de eventos.
- Distribuição de recompensas.
- Regras de propriedade dos itens.
- Proteção contra duplicação.
- Descarte.
- Desaparecimento de itens no chão.
- Tempo de permanência.

Investigue as regras de drop individual, compartilhado e em grupo, quando existentes.

## 4.10. Refinação e composição

Mapeie integralmente todos os sistemas de aprimoramento.

**Refinação com Poeira de Ori e Poeira de Lac:**
- Funcionamento.
- Requisitos.
- Probabilidades.
- Custos.
- Materiais.
- Limites.
- Falhas.
- Penalidades.
- Resultados.

**Composição Ancient (Anct):**
- Requisitos.
- Materiais.
- Combinações.
- Probabilidades.
- Resultados.
- Atributos.
- Limitações.
- Diferenças entre equipamentos.

**Sistema de gemas:**
- Tipos de gemas.
- Funções.
- Aplicação.
- Alteração de adicionais.
- Bônus.
- Combinações.
- Restrições.
- Probabilidades.

**Refinação +10:**
- Composição.
- Itens necessários.
- Requisitos.
- Materiais.
- NPCs.
- Probabilidades.
- Resultados.
- Falhas.

**Refinação +11, +12, +13, +14 e +15:**

Investigue individualmente cada estágio, sem presumir que utilizam as mesmas regras do +10.

Documente:

- Fórmulas.
- Materiais.
- Custos.
- Requisitos.
- Probabilidades.
- NPCs.
- Itens auxiliares.
- Proteções contra falhas.
- Penalidades.
- Resultados.
- Restrições por tipo de equipamento.

## 4.11. Montarias, crias e amagos

Investigue completamente:

- Montarias.
- Crias.
- Amagos.
- Tipos.
- Classificações.
- Atributos.
- Evolução.
- Alimentação.
- Reprodução ou composição, quando existente.
- Equipamentos.
- Bônus.
- Skills.
- Requisitos.
- Limites.
- Progressão.
- Efeitos no personagem.
- Interações com outros sistemas.

## 4.12. Capas e pinturas

Investigue:

- Sistema de capas.
- Tipos de capas.
- Quests de obtenção.
- Requisitos.
- Refinação.
- Composição.
- Atributos.
- Benefícios.
- Classes.
- Limites.
- Mudanças de aparência.
- Pinturas.
- Tipos de pintura.
- Aplicação.
- Remoção.
- Restrições.
- Efeitos visuais.
- NPCs relacionados.

## 4.13. Skills e habilidades

Este é um dos sistemas que exige maior profundidade de investigação.

Não basta mapear nomes e descrições. É necessário entender o funcionamento real de cada habilidade.

**Skills Sephira:**

Investigue todas as habilidades Sephira, incluindo, mas não se limitando a:

- Ressurreição.
- Vinhas / Muro de Espinhos.
- Spectral.
- Concentração.
- Canhão Conjurador.
- Demais habilidades existentes.

Para cada skill, documente:

- Descrição.
- Requisitos.
- Nível.
- Custo de mana.
- Cooldown.
- Alcance.
- Área de efeito.
- Duração.
- Fórmula.
- Dano.
- Cura.
- Efeitos.
- Buffs.
- Debuffs.
- Animações.
- Efeitos visuais.
- Interações com atributos.
- Interações com equipamentos.
- Restrições.
- Condições de utilização.

**Skill Sephira Soul:**

Mapeie completamente o sistema Sephira Soul e todas as suas configurações.

Investigue:

- Funcionamento.
- Configurações.
- Runas.
- Combinações.
- Bônus.
- Efeitos.
- Requisitos.
- Restrições.
- Progressão.
- Influência nos atributos.
- Interação com outras habilidades.
- Efeitos visuais.

**Skills de evocação:**

Investigue todas as habilidades de evocação, seus requisitos, funcionamento, efeitos, entidades invocadas, duração, atributos, comportamento e interações.

**Skills de transformação:**

Investigue todas as habilidades de transformação, seus requisitos, efeitos, atributos, modificações no personagem, duração, restrições e comportamento.

**Sistema geral de skills:**

Investigue também:

- Aprendizado.
- Evolução.
- Distribuição de pontos.
- Requisitos.
- Barra de habilidades.
- Atalhos.
- Cooldowns.
- Persistência.
- Uso simultâneo.
- Interrupções.
- Efeitos visuais.
- Efeitos sonoros.
- Buffs e debuffs.
- Skills passivas.
- Skills ativas.
- Skills de grupo.
- Skills de classe.
- Skills exclusivas.
- Cancelamento.
- Validação pelo servidor.
- Comportamento em PVP e PVE.

## 4.14. Quests e progressão narrativa

Mapeie todas as quests conhecidas do jogo.

Não se limite às quests de progressão principal.

Investigue:

- Quests iniciais.
- Quests de classe.
- Quests de Mortal.
- Quests de Arch.
- Quests de Celestial.
- Quests de Subcelestial.
- Quests de cidadania.
- Quests de reinos.
- Quests de capas.
- Quests de equipamentos.
- Quests de evolução.
- Quests de eventos.
- Quests especiais.
- Quests diárias, se existentes.
- Quests repetíveis.
- Quests temporárias.
- Quests secretas ou ocultas.

Para cada quest, documente:

- Identificador.
- Nome.
- História.
- Descrição.
- NPC inicial.
- NPC final.
- Pré-requisitos.
- Nível necessário.
- Itens necessários.
- Monstros envolvidos.
- Mapas.
- Objetivos.
- Etapas.
- Condições de conclusão.
- Recompensas.
- XP.
- Itens.
- Moedas.
- Desbloqueios.
- Limites de repetição.
- Tempo.
- Dependências de outras quests.

Documente os scripts e as condições de execução das quests, quando identificáveis.

Crie mapas de dependência e progressão das quests.

## 4.15. NPCs e interações

Investigue todos os tipos de NPCs.

Inclua:

- NPCs das cidades.
- NPCs de quests.
- NPCs de criação.
- NPCs de refinação.
- NPCs de composição.
- NPCs de comércio.
- NPCs de eventos.
- NPCs de teletransporte.
- NPCs de reinos.
- NPCs de guilds.
- NPCs especiais.
- NPCs relacionados a monstros e sistemas específicos.

Para cada NPC, documente:

- Identificação.
- Localização.
- Função.
- Serviços.
- Diálogos.
- Condições de interação.
- Requisitos.
- Custos.
- Itens envolvidos.
- Quests associadas.
- Sistemas relacionados.

## 4.16. Consumíveis e itens especiais

Investigue completamente:

- Poções de HP.
- Poções de Mana.
- Poções de efeitos.
- Poções de bônus.
- Perzen.
- Esferas.
- Comidas.
- Baús.
- Consumíveis.
- Chaves.
- Itens de abertura de portões.
- Itens de quest.
- Itens de teletransporte.
- Pergaminhos.
- Itens especiais.
- Itens temporários.
- Itens de evento.
- Itens de proteção.
- Itens de bônus.

Para cada item, documente uso, requisitos, duração, efeitos, cooldown, restrições, consumo e interação com outros sistemas.

## 4.17. Guilds, Fame e guerras

Investigue:

**Fame:**
- Obtenção.
- Perda.
- Cálculo.
- Benefícios.
- Limites.
- Rankings.
- Relações com outros sistemas.

**Guilds:**
- Criação.
- Requisitos.
- Hierarquia.
- Membros.
- Convites.
- Remoção.
- Cargos.
- Permissões.
- Liderança.
- Alianças.
- Rivalidades.
- Territórios.
- Benefícios.
- Evolução.
- Integração com cidades e reinos.

**Guerras:**
- Tipos.
- Regras.
- Participação.
- Horários.
- Mapas.
- Objetivos.
- Pontuação.
- Recompensas.
- Condições de vitória.
- Penalidades.
- Mecânicas.
- Relação com guilds e reinos.

## 4.18. Eventos

Faça um levantamento completo dos eventos conhecidos do WYD.

Investigue eventos antigos, atuais e específicos de diferentes versões.

Para cada evento, documente:

- Nome.
- Tipo.
- Funcionamento.
- Regras.
- Mecânicas.
- Horários.
- Duração.
- Mapas.
- NPCs.
- Monstros.
- Bosses.
- Moedas.
- Itens específicos.
- Recompensas.
- Requisitos.
- Participação.
- Limites.
- Pontuação.
- Classificação.
- Condições de vitória.
- Condições de encerramento.

Mapeie eventos automáticos, agendados, administrativos, competitivos, cooperativos e sazonais, quando existentes.

## 4.19. Kefra e sistemas especiais

Investigue Kefra como um sistema completo, não apenas como um mapa ou boss.

Mapeie:

- Funcionamento.
- Requisitos.
- Horários.
- Spawn.
- Bosses.
- Quests.
- NPCs.
- Mapas.
- Recompensas.
- Itens.
- Mecânicas.
- Participação.
- Interações com canais.
- Relações com reinos, guilds e eventos.

Investigue também Pesadelos, Zona da LAN, Água, Coliseu e demais sistemas especiais.

## 4.20. Cidades, propriedades e impostos

Mapeie completamente o sistema de cidades.

Investigue:

- Cidades.
- Proprietários.
- Guilds proprietárias.
- Áreas próprias.
- Territórios.
- Impostos.
- Arrecadação.
- Receitas.
- Distribuição de valores.
- Permissões.
- Benefícios.
- NPCs.
- Funcionalidades exclusivas.
- Controle administrativo.
- Integração com guerras.
- Relação entre cidades e reinos.

As cidades podem ter proprietários que arrecadam impostos. Entenda as regras efetivas desse sistema e como os valores são calculados e distribuídos.

## 4.21. Economia do jogo

Pesquise profundamente a economia do WYD.

Investigue:

- Gold.
- Moedas.
- Moedas especiais.
- Impostos.
- Receitas.
- Despesas.
- Comércio.
- Drop.
- NPCs comerciantes.
- Lojas.
- Loja pessoal.
- Taxas.
- Refinação.
- Composição.
- Eventos.
- Recompensas.
- Transações.
- Transferências.
- Criação e destruição de moeda.
- Circulação de recursos.
- Inflação.
- Abuso econômico.
- Duplicação.
- Exploits.
- Equilíbrio entre oferta e demanda.

Identifique todos os mecanismos que introduzem ou retiram moedas e itens da economia.

Crie um mapa do fluxo econômico do servidor.

## 4.22. Trocas, grupos e interação entre jogadores

Investigue integralmente:

**Troca:**
- Solicitação.
- Aceitação.
- Recusa.
- Inserção de itens.
- Inserção de moedas.
- Confirmação.
- Cancelamento.
- Validação.
- Persistência.
- Proteção contra duplicação.
- Concorrência.
- Desconexão durante a troca.

**Loja pessoal:**
- Abertura.
- Fechamento.
- Cadastro de produtos.
- Preços.
- Compra.
- Limites.
- Persistência.
- Taxas.
- Concorrência.
- Proteções contra fraudes.

**Grupo:**
- Criação.
- Convites.
- Aceitação.
- Recusa.
- Remoção.
- Liderança.
- Distribuição de XP.
- Distribuição de drops.
- Benefícios.
- Limites.
- Comunicação.
- Desconexão.
- Reentrada.

Investigue outras formas de interação entre jogadores.

## 4.23. Sistema completo de PVP

Investigue:

- PVP individual.
- PVP em grupo.
- PVP entre guilds.
- PVP entre reinos.
- Áreas de PVP.
- Regras de combate.
- Restrições.
- Penalidades.
- Recompensas.
- Rankings.
- Pontuação.
- Fame.
- Morte.
- Ressurreição.
- Proteção.
- Tempo de combate.
- Buffs e debuffs.
- Interações entre skills.
- Diferenças entre PVP e PVE.
- Regras de segurança.
- Anti-cheat.

Investigue também Coliseu, arenas e demais sistemas competitivos.

## 4.24. Comunicação e canais

Mapeie todos os sistemas de comunicação.

**Canais de chat:**
- Chat local.
- Chat global.
- Chat de grupo.
- Chat de guild.
- Chat privado.
- Chat de reino.
- Outros canais existentes.

**Trombeta:**

Investigue o item Trombeta, que permite comunicação entre todos os canais ativos do servidor.

O jogador utiliza o item para enviar uma mensagem que pode ser visualizada em todos os canais ativos.

Documente:

- Consumo.
- Funcionamento.
- Custos.
- Restrições.
- Distribuição.
- Identificação do remetente.
- Tratamento de mensagens.
- Integração entre canais.

**Mensagens do jogo:**
- Avisos.
- Mensagens de sistema.
- Mensagens de eventos.
- Eros.
- Comunicados administrativos.
- Mensagens globais.
- Notificações.
- Alertas.
- Mensagens automáticas.

Investigue formatos, prioridades, persistência e comportamento do cliente.

## 4.25. Cash, Premium Mail e itens especiais

Investigue:

- Sistema de Cash.
- Moedas de Cash.
- Créditos.
- Premium Mail.
- Entrega de itens.
- Identificação de itens.
- Origem dos itens.
- Histórico.
- Resgate.
- Expiração.
- Restrições.
- Integração com contas.
- Integração com personagens.
- Segurança.
- Prevenção contra duplicação.

**Identificação de itens:**

Todos os itens provenientes de sistemas especiais, incluindo Cash e Premium Mail, devem ser investigados quanto à possibilidade de identificação de origem e rastreabilidade.

Mapeie a identificação em todos os itens e como ela deve ser preservada durante trocas, armazenamento, movimentações e demais operações.

## 4.26. Canais e troca de canal

Investigue detalhadamente o funcionamento original da troca entre canais.

O comportamento esperado é permitir que o jogador transite entre canais do mesmo servidor mantendo seus dados e seu estado de jogo.

A hipótese inicial é:

1. O jogador solicita a troca de canal.
2. O servidor valida a solicitação.
3. O estado da sessão é preparado para transferência.
4. O personagem é sincronizado e persistido conforme necessário.
5. A conexão com o canal anterior é encerrada de forma controlada.
6. O cliente estabelece conexão com o novo canal.
7. A sessão é recuperada ou reconstruída no novo canal.
8. O personagem retorna ao jogo com seus dados preservados.

Essa é uma hipótese de arquitetura, não uma implementação definitiva.

Pesquise o comportamento original do jogo e determine a solução tecnicamente correta para nossa arquitetura.

Investigue:

- Identificação do jogador.
- Identificação da sessão.
- Autenticação.
- Estado do personagem.
- Estado de combate.
- Inventário.
- Equipamentos.
- Skills.
- Grupos.
- Guilds.
- Buffs.
- Localização.
- Persistência.
- Sincronização.
- Reconexão.
- Timeout.
- Concorrência.
- Desconexões inesperadas.
- Transferências simultâneas.
- Prevenção contra duplicação de sessões.
- Recuperação em caso de falha.

O jogador não deve perder itens, atributos, progresso ou informações durante a troca.

## 4.27. Login, logout e gerenciamento de sessões

Implemente o mapeamento completo dos fluxos de autenticação e sessão.

Investigue:

- Login.
- Autenticação.
- Seleção de personagem.
- Entrada no mundo.
- Inicialização do personagem.
- Recuperação de estado.
- Desconexão.
- Logout.
- Reconexão.
- Expiração de sessão.
- Timeout.
- Persistência.
- Concorrência.
- Sessões duplicadas.
- Desconexões inesperadas.

**Requisito obrigatório:**

Login e Logoff completos, sem depender exclusivamente de timeout para liberar ou finalizar sessões.

O encerramento deve ser explícito, controlado, seguro e consistente.

Investigue os fluxos de cliente e servidor, incluindo as situações em que o cliente fecha inesperadamente.

## 4.28. Tela de personagem e interface

Mapeie a tela de personagem completa e todas as funcionalidades disponíveis.

Investigue:

- Informações gerais.
- Nome.
- Classe.
- Nível.
- Estágio.
- Atributos.
- Distribuição de pontos.
- HP.
- Mana.
- Experiência.
- Equipamentos.
- Inventário.
- Skills.
- Status.
- Bônus.
- Buffs.
- Aparência.
- Montarias.
- Crias.
- Capas.
- Informações adicionais.

Identifique todas as ações disponíveis, estados visuais e atualizações em tempo real.

## 4.29. Barra de skills e persistência

Investigue o funcionamento completo da barra de habilidades.

Requisito:

A barra de skills deve manter sua configuração após:

- Logout.
- Login.
- Troca de canal.
- Reinicialização do cliente.
- Reinicialização do servidor.

Investigue:

- Slots.
- Atalhos.
- Associação de habilidades.
- Itens consumíveis.
- Persistência.
- Sincronização.
- Validação.
- Alterações em tempo real.
- Compatibilidade com diferentes personagens.

## 4.30. Efeitos visuais e aparência

Investigue os efeitos corretos dos itens e habilidades no personagem.

Inclua:

- Aparência de equipamentos.
- Armas.
- Armaduras.
- Capas.
- Montarias.
- Transformações.
- Skills.
- Buffs.
- Debuffs.
- Efeitos de refinação.
- Efeitos de composição.
- Efeitos especiais.
- Animações.
- Partículas.
- Sons.
- Alterações visuais no personagem.

O cliente deve apresentar corretamente os efeitos correspondentes ao estado real do personagem no servidor.

## 4.31. Itens iniciais e experiência inicial

Investigue o processo de criação de personagens e a experiência inicial do jogador.

Mapeie:

- Itens iniciais.
- Equipamentos iniciais.
- Consumíveis.
- Poções de HP e Mana.
- Quantidades.
- Distribuição de atributos.
- Skills iniciais.
- NPCs iniciais.
- Mapas iniciais.
- Quests introdutórias.
- Orientações.
- Progressão inicial.
- Restrições.
- Evolução até os estágios seguintes.

## 4.32. Buffs, scripts e temporizadores

Investigue todos os sistemas de buffs e efeitos temporários.

Mapeie:

- Buffs.
- Debuffs.
- Duração.
- Cooldown.
- Acúmulo.
- Substituição.
- Cancelamento.
- Persistência.
- Expiração.
- Renovação.
- Interação entre efeitos.
- Scripts de quests.
- Scripts de eventos.
- Temporizadores.
- Agendamentos.
- Efeitos automáticos.
- Efeitos administrativos.

Investigue como os temporizadores funcionam e como devem se comportar em desconexões, reinicializações e trocas de canal.

## 4.33. Tempos, resets e ciclos do servidor

Investigue todos os temporizadores e ciclos do jogo.

Inclua:

- Horários de eventos.
- Respawn de monstros.
- Respawn de bosses.
- Cooldowns.
- Quests temporizadas.
- Buffs.
- Debuffs.
- Limites diários.
- Limites semanais.
- Resets.
- Reinicializações.
- Ciclos de eventos.
- Ciclos de economia.
- Tempos de mapas especiais.
- Tempos de guerras.
- Tempos de Kefra.
- Tempos de sistemas especiais.

Documente também o comportamento durante reinicializações e recuperação do servidor.

---

# 5. CLIENTE, SERVIDOR E FUNCIONALIDADES BÁSICAS NÃO MAPEADAS

Além dos sistemas de gameplay, pesquise todas as funcionalidades necessárias para permitir que um personagem acesse e utilize integralmente o mundo do jogo.

Não presuma que os itens acima representam todas as funcionalidades básicas do cliente/servidor.

Investigue e documente:

- Criação de conta.
- Autenticação.
- Seleção de servidor.
- Seleção de canal.
- Seleção de personagem.
- Criação de personagem.
- Exclusão de personagem.
- Entrada no mundo.
- Carregamento de mapas.
- Carregamento de entidades.
- Sincronização de jogadores.
- Movimentação.
- Direção.
- Animações.
- Ataques.
- Interações.
- Inventário.
- Equipamentos.
- Banco ou armazenamento, se existente.
- Comércio.
- NPCs.
- Chat.
- Grupo.
- Guild.
- Quests.
- Skills.
- Eventos.
- Morte.
- Ressurreição.
- Teletransporte.
- Troca de canal.
- Desconexão.
- Reconexão.
- Logout.
- Atualização de estado.
- Persistência.
- Notificações.
- Tratamento de erros.
- Sincronização temporal.

Investigue ainda protocolo de comunicação, estrutura de pacotes, validações, identificação de entidades, estados de conexão e mecanismos de sincronização.

Produza um mapa completo das funcionalidades necessárias para considerar um personagem funcional dentro do servidor.

---

# 6. BUGS, HACKS E VULNERABILIDADES CONHECIDAS

Realize uma pesquisa massiva sobre bugs, falhas e exploits documentados em versões antigas do WYD e em servidores privados.

O objetivo é estudar os problemas históricos para evitar que sejam reproduzidos no nosso projeto.

Investigue:

- Duplicação de itens.
- Duplicação de moedas.
- Exploits de troca.
- Exploits de inventário.
- Exploits de refinação.
- Exploits de composição.
- Exploits de quests.
- Exploits de XP.
- Exploits de drop.
- Exploits de PVP.
- Exploits de skills.
- Exploits de movimentação.
- Exploits de teletransporte.
- Exploits de canais.
- Abuso de sessões.
- Manipulação de pacotes.
- Modificação de cliente.
- Speed hack.
- Attack speed hack.
- Bots.
- Automação indevida.
- Manipulação de valores.
- Replay de requisições.
- Race conditions.
- Abuso de temporizadores.
- Falhas de persistência.
- Falhas de autenticação.
- Falhas de autorização.
- Problemas de concorrência.

Para cada problema encontrado:

1. Documente o comportamento.
2. Identifique a versão afetada.
3. Identifique a causa provável ou confirmada.
4. Registre as evidências.
5. Explique o impacto.
6. Identifique o sistema afetado.
7. Descreva as condições necessárias para ocorrência.
8. Proponha mecanismos preventivos.
9. Crie critérios de validação.
10. Avalie se o projeto atual possui proteção.

Não execute exploits contra servidores de terceiros nem utilize ambientes sem autorização.

A investigação técnica deve priorizar prevenção, reprodução controlada em ambiente próprio e testes defensivos.

Crie um catálogo de ameaças e vulnerabilidades conhecidas do WYD.

---

# 7. ANÁLISE DA EXPERIÊNCIA DOS JOGADORES

Pesquise também o que os jogadores falam sobre o WYD.

Utilize fóruns, comunidades, discussões, avaliações e servidores privados como fontes de levantamento.

Identifique:

**Pontos positivos:**
- Sistemas mais apreciados.
- Mecânicas tradicionais.
- Funcionalidades que os jogadores consideram indispensáveis.
- Aspectos que tornam o jogo interessante.
- Sistemas que mantêm a comunidade ativa.
- Características que não devem ser perdidas.

**Pontos negativos:**
- Sistemas considerados ultrapassados.
- Mecânicas mal balanceadas.
- Problemas de progressão.
- Problemas econômicos.
- Problemas de interface.
- Problemas de desempenho.
- Problemas de segurança.
- Funcionalidades inconvenientes.
- Falhas recorrentes.
- Sistemas que geram frustração.

Nosso objetivo é preservar a identidade do WYD, mantendo seus sistemas característicos, mas melhorar o que for tecnicamente possível.

Para cada melhoria proposta, registre:

- Problema identificado.
- Evidências e opiniões encontradas.
- Quantidade ou recorrência de relatos, quando possível.
- Impacto.
- Possível solução.
- Riscos de alteração.
- Compatibilidade com a experiência original.
- Prioridade.

Não altere silenciosamente regras históricas do jogo. Separe reprodução fiel do comportamento original e propostas de melhoria.

---

# 8. MATEMÁTICA COMPLETA E BALANCEAMENTO

A pesquisa matemática não deve se limitar às fórmulas citadas neste documento.

Faça um levantamento de todas as matemáticas existentes no WYD.

Investigue:

- Atributos.
- Combate.
- Dano.
- Defesa.
- Crítico.
- Evasão.
- Imunidade.
- Precisão.
- Velocidade.
- HP.
- Mana.
- XP.
- Drop.
- Refinação.
- Composição.
- Probabilidades.
- Economia.
- Impostos.
- Progressão.
- Skills.
- Buffs.
- Debuffs.
- Grupos.
- PVP.
- PVE.
- Monstros.
- Bosses.
- Eventos.
- Recompensas.
- Multiplicadores.
- Penalidades.
- Limites.
- Arredondamentos.

Para cada fórmula:

- Apresente a expressão matemática.
- Explique cada variável.
- Identifique os valores de entrada.
- Identifique os multiplicadores.
- Explique a ordem das operações.
- Registre limites e arredondamentos.
- Informe as fontes.
- Demonstre exemplos numéricos.
- Apresente casos extremos.
- Identifique possíveis inconsistências.

Quando a fórmula não puder ser confirmada, apresente as hipóteses separadamente.

Crie uma referência matemática centralizada para o projeto.

---

# 9. SISTEMA DE BOTS PARA TESTES E SIMULAÇÃO DE POPULAÇÃO

Precisamos investigar e planejar um sistema de bots integrado ao servidor.

O objetivo é criar uma população simulada de jogadores reais, capaz de interagir com o mundo do jogo.

Não queremos apenas bots que enviem pacotes ou executem movimentos repetitivos.

Queremos agentes que possam executar fluxos completos de gameplay.

## 9.1. Comportamento dos bots

Os bots devem ser capazes de:

- Aparecer no mapa como personagens normais.
- Movimentar-se.
- Interagir com NPCs.
- Executar quests.
- Evoluir.
- Distribuir atributos.
- Aprender skills.
- Utilizar habilidades.
- Equipar itens.
- Refinações e composições.
- Caçar monstros.
- Enfrentar bosses.
- Participar de eventos.
- Participar de PVP.
- Criar grupos.
- Entrar em grupos.
- Interagir com guilds.
- Realizar trocas.
- Comprar e vender itens.
- Utilizar lojas pessoais.
- Utilizar o chat.
- Utilizar consumíveis.
- Utilizar teletransportes.
- Trocar de canal.
- Morrer e ressuscitar.
- Executar todas as atividades disponíveis no jogo.

Os bots devem compreender os sistemas de progressão de Mortal, Arch, Celestial e Subcelestial.

## 9.2. Simulação de população real

Investigue como criar uma população artificial com comportamentos variados.

Considere:

- Diferentes perfis de jogadores.
- Diferentes níveis.
- Diferentes classes.
- Diferentes objetivos.
- Diferentes estilos de jogo.
- Horários de atividade.
- Tempos de inatividade.
- Interações sociais.
- Formação de grupos.
- Comércio.
- Participação em eventos.
- Evolução individual.
- Progressão econômica.
- Comportamentos cooperativos e competitivos.

Os bots devem existir efetivamente no mundo, com estado e comportamento coerentes.

## 9.3. Integração com LLM

Investigue a possibilidade de integração com modelos de linguagem.

Considere:

- Decisão de objetivos.
- Planejamento.
- Interpretação de contexto.
- Memória de experiências.
- Comportamento social.
- Conversação.
- Tomada de decisão.
- Aprendizado baseado em resultados.
- Perfis de personalidade.

Avalie uma arquitetura híbrida na qual a lógica determinística executa ações e regras de gameplay, enquanto LLMs são utilizados para decisões de alto nível e comportamentos que realmente se beneficiem deles.

Não dependa de LLM para operações que exigem precisão matemática, validação autoritativa ou alto desempenho.

## 9.4. Stress tests e bug hunting automatizado

Os bots também devem funcionar como ferramentas de engenharia e testes.

Investigue:

- Testes de carga.
- Testes de capacidade.
- Testes de concorrência.
- Testes de persistência.
- Testes de economia.
- Testes de combate.
- Testes de grupos.
- Testes de eventos.
- Testes de troca.
- Testes de canais.
- Testes de reconexão.
- Testes de limites.
- Testes de integração.
- Testes de regressão.
- Detecção de comportamentos anormais.

Planeje cenários de testes automatizados capazes de identificar bugs e comportamentos inconsistentes.

Os bots devem ser capazes de executar cenários reproduzíveis, registrar resultados e gerar evidências para investigação.

---

# 10. AUDITORIA COMPLETA DO ROADMAP-RUST

Esta é uma das etapas mais importantes da missão.

Analise integralmente o roadmap do projeto Rust e todas as issues existentes.

Precisamos saber se tudo o que foi solicitado neste documento já está devidamente mapeado e planejado.

Não basta procurar palavras semelhantes. É necessário verificar se cada sistema está realmente documentado com profundidade suficiente.

## 10.1. Auditoria de cobertura

Para cada requisito:

- Verifique se existe documentação.
- Verifique se existe issue.
- Verifique se a issue possui escopo adequado.
- Verifique se os critérios de aceite estão completos.
- Verifique se existem dependências.
- Verifique se há subtarefas.
- Verifique se existem testes planejados.
- Verifique se o comportamento original está documentado.
- Verifique se há informações matemáticas.
- Verifique se existem evidências.
- Verifique se a implementação atual contempla o requisito.

Classifique cada item como:

- Implementado e validado.
- Implementado parcialmente.
- Documentado, mas não implementado.
- Mapeado em issue, mas insuficientemente especificado.
- Não documentado.
- Não investigado.
- Dependente de confirmação externa.

## 10.2. Identificação de gaps

Procure:

- Sistemas completamente ausentes.
- Funcionalidades esquecidas.
- Issues genéricas demais.
- Issues sem critérios de aceite.
- Documentação superficial.
- Dependências não mapeadas.
- Contradições.
- Implementações incompletas.
- Falta de testes.
- Ausência de regras matemáticas.
- Ausência de validações.
- Ausência de segurança.
- Falta de integração com o cliente.
- Falta de persistência.
- Falta de observabilidade.

Não considere um sistema coberto apenas porque existe uma issue com seu nome.

A cobertura deve ser funcional e técnica.

## 10.3. Atualização do checklist

Atualize o checklist oficial de implementação do projeto.

Inclua todos os requisitos deste documento que ainda não estejam contemplados.

Preserve o histórico e os itens já existentes.

Não remova funcionalidades ou informações do checklist atual.

Organize os itens por domínio, dependências e prioridade.

---

# 11. DOCUMENTAÇÃO TÉCNICA E BASE DE CONHECIMENTO

Toda a pesquisa deve resultar em uma base de conhecimento sólida dentro da pasta `docs` do projeto.

Utilize os padrões documentais já existentes.

Não crie documentação desorganizada, duplicada ou desconectada das issues.

## 11.1. Estrutura de conhecimento

Organize a documentação por domínios, conforme a estrutura existente do projeto.

Considere, quando compatível com os padrões atuais:

- Personagens e atributos.
- Matemática e balanceamento.
- Combate.
- Progressão.
- Mortal, Arch, Celestial e Subcelestial.
- Itens e equipamentos.
- Refinação e composição.
- Skills.
- Quests.
- Mapas e zonas.
- Monstros e bosses.
- NPCs.
- Eventos.
- Economia.
- Guilds e guerras.
- Cidades e reinos.
- Comunicação.
- Cliente e servidor.
- Sessões e canais.
- Segurança e exploits.
- Bots e testes.
- Bugs históricos.
- Experiência dos jogadores.
- Regras de negócio.
- Catálogo de itens.

## 11.2. Documentação por sistema

Cada sistema deve conter, quando aplicável:

1. Visão geral.
2. Objetivo.
3. Funcionamento original.
4. Regras de negócio.
5. Fórmulas matemáticas.
6. Estruturas de dados.
7. Entidades envolvidas.
8. NPCs e mapas.
9. Requisitos.
10. Dependências.
11. Fluxos de execução.
12. Comportamento do cliente.
13. Comportamento do servidor.
14. Persistência.
15. Protocolos.
16. Estados e transições.
17. Restrições.
18. Casos especiais.
19. Bugs conhecidos.
20. Vulnerabilidades.
21. Testes necessários.
22. Implementação atual no Rust.
23. Gaps identificados.
24. Issues relacionadas.
25. Fontes e evidências.
26. Pendências de investigação.

Não preencha campos com informações inventadas apenas para completar o template.

---

# 12. MAPEAMENTO E CRIAÇÃO DE ISSUES

Todas as funcionalidades ausentes ou insuficientemente especificadas devem ser transformadas em issues seguindo o template oficial do projeto.

Não crie issues genéricas como "Implementar sistema de itens" quando o sistema exige dezenas de funcionalidades independentes.

Divida o trabalho em unidades de implementação tecnicamente coerentes.

## 12.1. Cada issue deve conter

- Título claro.
- Identificador ou categoria.
- Descrição.
- Contexto.
- Objetivo.
- Comportamento esperado.
- Funcionamento original do WYD.
- Regras de negócio.
- Requisitos funcionais.
- Requisitos não funcionais.
- Fórmulas, quando aplicáveis.
- Estruturas envolvidas.
- Dependências.
- Integrações.
- Critérios de aceite.
- Cenários de teste.
- Casos extremos.
- Requisitos de segurança.
- Referências documentais.
- Fontes de pesquisa.
- Prioridade.
- Complexidade estimada.
- Riscos.
- Definition of Done.

## 12.2. Organização

Organize as issues por:

- Domínio.
- Subsistema.
- Épico.
- Dependência.
- Prioridade.
- Fase de implementação.

Identifique claramente o que precisa ser implementado primeiro.

Não duplique issues existentes.

Quando uma issue existente estiver incompleta, avalie sua atualização em vez de criar outra.

Mantenha rastreabilidade entre:

**Requisito → Documentação → Issue → Implementação → Teste → Validação.**

---

# 13. REGRAS DE EXECUÇÃO E COMPORTAMENTO DO AGENTE

Esta missão exige investigação real, não respostas superficiais.

## 13.1. Não interrompa a pesquisa prematuramente

Nossa sessão inicial de trabalho é de aproximadamente uma hora.

Você pode utilizar esse período tranquilamente para pesquisar, analisar, aprender, confrontar informações e documentar.

Não tenha pressa para responder.

Priorize profundidade e qualidade.

Caso a pesquisa não seja concluída em uma hora, registre o progresso e continue nas próximas sessões, sem perder o contexto.

Não declare uma investigação concluída quando ainda existirem gaps relevantes.

## 13.2. Explore o projeto antes de propor alterações

Antes de documentar qualquer implementação:

1. Examine a estrutura do projeto.
2. Consulte as skills disponíveis.
3. Leia as instruções.
4. Leia a documentação.
5. Examine o roadmap.
6. Analise as issues.
7. Pesquise o comportamento original.
8. Verifique o cliente.
9. Compare com o código legado.
10. Identifique o que já existe no Rust.

Não proponha implementar novamente funcionalidades que já estejam prontas e validadas.

## 13.3. Não remova informações

Preserve integralmente:

- Documentação existente.
- Issues existentes.
- Checklists.
- Decisões arquiteturais.
- Regras de negócio.
- Histórico.
- Informações técnicas.

A missão é enriquecer, complementar, corrigir e organizar.

Não elimine conteúdo existente sem autorização explícita.

## 13.4. Não invente comportamentos

Quando não souber algo:

- Pesquise.
- Compare fontes.
- Examine o código.
- Registre a incerteza.
- Identifique o que precisa ser confirmado.

Não crie fórmulas, probabilidades, requisitos ou regras de negócio sem evidências.

## 13.5. Investigue além da lista

A lista apresentada é um ponto de partida.

Você deve identificar outros sistemas, funcionalidades e mecânicas que não foram mencionados.

Sempre que encontrar algo relevante:

1. Investigue.
2. Entenda.
3. Documente.
4. Relacione com os demais sistemas.
5. Verifique se já existe no roadmap.
6. Crie ou atualize a issue correspondente.

## 13.6. Faça perguntas quando necessário

Não interrompa a pesquisa para perguntar algo que possa ser descoberto por meio da documentação, código, cliente ou fontes externas.

Primeiro investigue.

Faça perguntas apenas quando existir uma decisão que realmente dependa de informação ou autorização do responsável pelo projeto.

Agrupe as dúvidas em uma lista objetiva, apresentando o contexto e as alternativas encontradas.

## 13.7. Transparência sobre o trabalho

Não declare que pesquisou fontes que não acessou.

Não declare que analisou arquivos que não abriu.

Não declare que testou comportamentos que não reproduziu.

Não declare que atualizou issues ou documentos sem ter realizado a operação.

Diferencie claramente pesquisa, análise, documentação, planejamento e implementação efetivamente executada.

---

# 14. RESULTADOS ESPERADOS

Ao final da investigação, precisamos ter:

**1. Base de conhecimento WYD2**

Documentação abrangente de todos os sistemas, mecânicas, regras, fórmulas e funcionalidades.

**2. Catálogo completo de itens**

Com propriedades, atributos, raridades, classes, combinações, requisitos, efeitos, obtenção e demais informações relevantes.

**3. Catálogo de mapas, monstros, NPCs e bosses**

Com identificadores, posições, funcionalidades, requisitos, respawns, drops e relações com quests e eventos.

**4. Documentação matemática**

Com fórmulas, variáveis, exemplos, limites, multiplicadores e evidências.

**5. Mapa de progressão**

Mortal, Arch, Celestial e Subcelestial, com todas as quests, requisitos, desbloqueios e funcionalidades.

**6. Catálogo de quests e eventos**

Com funcionamento, dependências, recompensas, requisitos e fluxos completos.

**7. Catálogo de bugs e vulnerabilidades**

Com evidências, impactos, causas e propostas de prevenção.

**8. Análise da experiência dos jogadores**

Com pontos positivos, negativos e oportunidades de melhoria.

**9. Arquitetura de bots**

Com planejamento de população simulada, gameplay autônomo, integração opcional com LLM e testes de capacidade.

**10. Auditoria do roadmap-rust**

Com análise de cobertura, funcionalidades ausentes, documentação incompleta e issues insuficientes.

**11. Issues completas**

Todas as funcionalidades ainda necessárias devem estar mapeadas seguindo o template oficial.

**12. Checklist atualizado**

Todos os requisitos identificados devem estar contemplados no checklist oficial, preservando o conteúdo existente.

**13. Matriz de rastreabilidade**

Deve permitir identificar exatamente onde cada funcionalidade está documentada, planejada, implementada e testada.

---

# 15. PRIMEIRA ETAPA OBRIGATÓRIA — DIAGNÓSTICO E IMERSÃO

Não comece criando centenas de issues.

Primeiro, faça uma imersão no projeto.

Execute esta sequência:

### Fase 1 — Conhecimento do projeto
- Leia todas as instruções e skills relevantes.
- Analise a documentação.
- Entenda a arquitetura e os objetivos do projeto Rust.
- Examine o roadmap.
- Examine os checklists.
- Identifique os templates de issues.

### Fase 2 — Conhecimento do WYD
- Estude o OpenWYD.
- Investigue o funcionamento original.
- Pesquise as fontes externas.
- Analise o cliente do projeto.
- Compare com o cliente original do WYD Global.
- Identifique diferenças entre versões.

### Fase 3 — Mapeamento de sistemas
- Organize todos os sistemas por domínio.
- Investigue fórmulas.
- Identifique dependências.
- Mapeie funcionalidades esquecidas.
- Registre evidências.
- Construa a base de conhecimento.

### Fase 4 — Auditoria do Rust
- Compare cada sistema com o que já existe.
- Avalie a implementação atual.
- Identifique gaps.
- Identifique bugs e inconsistências.
- Avalie a cobertura documental e de testes.

### Fase 5 — Documentação e planejamento
- Enriqueça a documentação.
- Atualize os documentos existentes.
- Atualize o checklist.
- Crie ou complemente as issues.
- Organize dependências e prioridades.
- Construa a matriz de rastreabilidade.

### Fase 6 — Relatório final
Apresente:

- O que foi efetivamente investigado.
- Quais fontes foram consultadas.
- Quais sistemas foram compreendidos.
- Quais informações foram confirmadas.
- Quais informações continuam incertas.
- Quais documentos foram criados ou atualizados.
- Quais issues foram criadas ou atualizadas.
- Quais gaps foram identificados.
- Quais funcionalidades ainda precisam de pesquisa.
- Quais decisões dependem de aprovação.

---

# 16. DIRETRIZ FINAL

**Quero que você realmente se torne um especialista no funcionamento do WYD2 antes de considerar o planejamento completo.**

Não interprete esta solicitação como uma simples tarefa de revisão de documentação.

Estamos construindo um servidor MMORPG que precisa reproduzir e compreender profundamente os sistemas do WYD, com possibilidade de corrigir problemas históricos e melhorar a experiência dos jogadores.

Precisamos conhecer tanto os sistemas evidentes quanto as regras internas que normalmente passam despercebidas.

Entenda como cada sistema funciona isoladamente e como todos se conectam.

Um atributo pode alterar o dano, a velocidade, o crítico e o comportamento de uma skill. Uma quest pode depender de um NPC, um item, um mapa, um monstro, uma condição de progressão e um temporizador. Um evento pode interferir na economia, no drop, nos canais e na progressão dos personagens.

Essas relações precisam ser documentadas.

**Não queremos apenas uma lista de funcionalidades. Queremos um modelo de conhecimento do jogo.**

A qualidade da implementação futura depende diretamente da qualidade dessa investigação.

Comece pela imersão completa nas referências do projeto e pela auditoria do roadmap-rust. Depois, avance sistematicamente pelos sistemas, registrando conhecimento, evidências, dependências, gaps e issues.

Se encontrar algo que não foi solicitado, mas que faça parte do funcionamento do WYD, investigue e inclua.

Se encontrar algo que já esteja implementado, valide antes de propor qualquer alteração.

Se encontrar algo incorreto, documente a evidência e proponha a correção.

Se encontrar algo desconhecido, não invente: investigue e registre a pendência.

**A missão é alcançar a maior cobertura técnica e funcional possível do WYD2, construindo uma documentação que permita implementar, testar, validar, manter e evoluir o servidor sem depender de conhecimento informal ou suposições.**

Comece agora pela Fase 1. Não apresente apenas um plano do que pretende fazer: execute a investigação, produza os artefatos e mantenha o progresso documentado.