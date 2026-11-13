
## Nova análise técnica e refatoração do projeto para Rust
Vinhamos até agora refatorando o projeto antigo para GO, mas, fizemos uma g4rande análise técnica e descubrimos diversos problemas de padrões, comunicação, escolha da linguagem GO, onde rust seria muito mais eficiente
não aplicamos conceitos de dados frios e quentes, existem diversas inrregularidades de engenharia e arquitetura no projeto do servidor

Estavamos seguindo todo o roadmap para refatoração em go nos arquivos
D:\dolorestec\wyd2\OpenWYD\roadmap\ - Pasta com a lista de todas as issues
D:\dolorestec\wyd2\OpenWYD\pre-release-bots.md - definições da suite de bots
D:\dolorestec\wyd2\OpenWYD\pre-release.md - definições do projeto

Fizemos uma nova análise sobre a portabilidade para rust, usando padrões corretos de desenvolvimento, segue toda a análise:
Uma análise rigorosa e sem concessões do escopo e da arquitetura proposta para projetos modernos de reescrita como o WYD exige olhar além da boa intenção de "modernizar o código legado". Em ecossistemas de emulação de MMORPGs, a transição de bases arcaicas (C++ dos anos 2000) para paradigmas modernos frequentemente esbarra em armadilhas de engenharia corporativa que destroem a performance em tempo real.
Abaixo, a autópsia técnica e crítica sobre o desenho arquitetural esperado para esse tipo de iniciativa:

1. A Ilusão dos Microsserviços no Domínio de Jogos
O Erro Comum: Tentar aplicar uma arquitetura de microsserviços distribuídos (separando mapa, inventário, chat e party em processos distintos conectados por gRPC/HTTP ou sockets internos) é um erro fatal de engenharia de jogos.
O Impacto Real: O ciclo de jogo (game loop) de um MMORPG exige processamento determinístico e latência na casa de microssegundos/milissegundos. Introduzir saltos de rede entre serviços para validar um item ou calcular uma colisão gera jitter, contenção de I/O e degradação severa do P99 de latência.
Veredito Crítico: A divisão em múltiplos serviços só faz sentido estrito no Cold Path (autenticação, painel web, billing, matchmaking fora do mapa). O Hot Path (motor de jogo, física, entidades, colisão e gerenciamento de estado espacial) deve residir estritamente no mesmo espaço de memória.

2. O Calcanhar de Aquiles: Gerenciamento de Memória e Estruturas de Dados
A Abordagem ingênua (OOP Clássica): Manter hierarquias de classes orientadas a objetos (Player, Monster, Item como entidades espalhadas no heap com ponteiros brutos) anula qualquer ganho de uma nova linguagem. Isso gera uma enxurrada de cache misses (L1/L2/L3 da CPU), sufocando o desempenho do processador.
A Solução Obrigatória (DOD/ECS): Um projeto nova geração de verdade precisa abandonar a OOP tradicional e adotar Data-Oriented Design (DOD) ou um modelo Entity Component System (ECS). Dados contíguos em memória (Vec<Position>, Vec<Attribute>) são essenciais para varreduras vetorizadas via SIMD durante o processamento em lote de centenas de jogadores em uma mesma zona de combate (ex: guerras de guildas).

3. A Escolha da Linguagem: O Veredito Técnico
Go (Golang): Embora ágil para APIs e microsserviços periféricos, o coletor de lixo (Garbage Collector) do Go introduz pausas imprevisíveis (stop-the-world ou micro-stutters) inadmissíveis para a sincronização exata de pacotes de movimento e física de um MMORPG de ação competitiva.
Rust: É a escolha tecnicamente irrepreensível para o núcleo de um motor de jogo moderno. Oferece segurança de memória estrita em tempo de compilação, ausência total de GC overhead e concorrência sem data races garantida pelo compilador.

4. Concorrência e Pilha de Rede: Além dos Sockets Tradicionais
Modelo Thread-per-Core (Shared-Nothing): Em vez de travar o mundo inteiro com mutexes globais, o mapa deve ser particionado em zonas espaciais onde cada thread de CPU gerencia exclusivamente a sua região de forma isolada, comunicando-se apenas por lock-free ring buffers (como canais concorrentes de altíssimo rendimento).
I/O Assíncrono com io_uring: No subsistema de rede, wrappers legados baseados em select/poll ou epoll bloqueantes devem ser descartados em favor do io_uring (no Linux), permitindo operações de I/O assíncronas com zero-copy de buffers diretamente entre o kernel e o espaço de usuário.

5. O Que um Especialista Faria? (O Blueprint Definitivo para o WYD)
Core Monolítico Baseado em Rust (Hot Path): Um único processo altamente otimizado por nó de mundo, rodando um loop determinístico com Arena Allocators (ex: bumpalo) para zerar o custo de alocações dinâmicas de heap durante o gameplay.
Serialização Binária Zero-Copy: Eliminação de parsers manuais vulneráveis a buffer overflows ou formatos pesados como JSON/Protobuf, utilizando esquemas binários estritos e seguros contra estouro de buffer (como rkyv ou layouts de structs em Rust com validação estrita de pacotes).
Autoridade Absoluta do Servidor com Fixed-Point Math: Prevenção de exploits de velocidade e coordenadas através de matemática de ponto fixo, garantindo consistência determinística independentemente da arquitetura da CPU cliente.
Observabilidade Nativa: Métricas em tempo real de latência de tick, tamanho de filas de rede e alocações expostas via Prometheus/Grafana, abandonando os arcaicos printfs de console.

Temos toda a descrição da nova análise e refatoração do projeto em D:\dolorestec\wyd2\OpenWYD\pre-release-rust.md

Não devemos aproveitar nada do nosso antigo projeto em go, todos os arquivos atualis do projeto devem ser movidos para .deprecated-go, que deve ficar no gitignore, deve ficar na pasta do projeto apenas a pasta .deprecated-go e o README.md, devidamente atualziado apra toda a nova infra do GO
devemos limpar todo nosso abmviente de desenvolvimento windows, como caches, extensoões do vscode de go, arquivos do go do ambiente local, fazer uma limpeza de tudo relacionado a linguagem que não iremos mais usar, arquivos de usuário, desinstalções e vamos limpar completamente o ambiente para iniciar a reconfiguração do rust(essa deve ser a primeira issues a ser executada, depois do mapeamento completo
após a limpeza, vc deve iniciar a criação das pastas completas para toda a nova estrutura, organizando de forma coerente, criar com os arquivos gitkeep para manter as pastas vazias
apóis isso vamos fazer um levantamento global de tudo que iremos usar, tecnologias,frameworks, analise o projeto legado e veja tudo que vai precisar do rust para iniciar as instalações e configurações
o vscode deve ser completamente reconfigurado para as novas tecnologias
a branch develop deve ser a branch principal de desenvolvimento

analisar todo o conteudo dos arquivos .agents/skills/*, agents.md, devem ser criadas novas para uso das tecnnologicas adotadas
voce não tem autorização ainda praa gerar código/configs, sua função agora é planejamento, limpeza e organização do ambiente
após organizar todas as pastas, vc pode iniciar a criação da pasta roadmap/ e iniciar a escrita das issues, não escrever nenhum código até tudo está documentado em roadmap/*