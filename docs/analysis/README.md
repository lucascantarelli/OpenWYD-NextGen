# Dossiê de análise — OpenWYD / Dolorestec WYD2

Análise feita em **2026-11-04** sobre o repositório (código Go, portal, conteúdo, cliente e
documentação) e sobre o que a comunidade publica a respeito do WYD. É o material de referência para o
plano de execução em `roadmap/PLANO-2026-11.md`.

Regra de leitura: **número sem fonte é opinião**. Cada afirmação sobre o código aponta caminho ou
comando que a produz; cada afirmação sobre o jogo/cliente que veio da web aponta a URL e, quando a
página não pôde ser lida, está marcada como *não verificado*.

## Documentos

| Documento | Responde |
|-----------|----------|
| [estado-atual.md](estado-atual.md) | O que é o projeto, inventário do repositório, respostas 1–10 e as 3 perguntas adicionais, lacunas da documentação |
| [migracao-legado.md](migracao-legado.md) | O que do C++ (`Source/`) já foi portado, o que falta, dependências e limites da migração completa |
| [persistencia-conteudo.md](persistencia-conteudo.md) | Quais arquivos de `Release/` o servidor lê em runtime, o que já vive no Postgres, o que pode (e o que não deve) sair do arquivo |
| [cliente-launcher.md](cliente-launcher.md) | Cliente 7.64/12000: engine, assets, patches, o que dá para modernizar e como fazer launcher + auto-update |
| [qa-ingame.md](qa-ingame.md) | Triagem das issues que só fecham com o cliente real, ordem de teste e o sistema de feedback/bugs |
| [qualidade-e-processo.md](qualidade-e-processo.md) | Plano de engenharia: arquitetura, testes, CI/CD, observabilidade, segurança, documentação e arquivos de IA |

## Resumo executivo (5 linhas)

1. O servidor Go já cobre **login, personagem, mundo, combate, loja, teleporte, guildas, eventos e
   persistência**; 31 de 48 issues fecharam. O que falta fechar **não é código**: é evidência no
   cliente real (11 issues 🟡) e features de conteúdo/admin (17 issues ⏳).
2. O legado C++ não é mais dependência de runtime — é **referência de comportamento**. Das 12
   soluções em `Source/Code`, só `ClientPatch_v7662` continua indispensável (e é do cliente).
3. `Release/` **pode** migrar para o banco quase inteiro (catálogos, NPCs, drops, rates), com exceção
   dos mapas binários (`HeightMap/AttributeMap.dat`) e dos assets do cliente.
4. O cliente é **fechado, DirectX 9, 32-bit**, com modding viável por assets (`.wys` = DDS + 1 byte) e
   por hooks da `ClientPatch`; resolução/UI sem fonte exigem patch de binário.
5. A documentação é boa em engenharia reversa (21 arquivos de handler, 104 documentos) e **fraca em
   produto**: não há visão de conjunto, índice de sistemas de jogo, nem ADRs. É o que este dossiê
   começa a corrigir.
