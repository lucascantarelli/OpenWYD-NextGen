# História

!!! info "Como ler esta página"
    O servidor não carrega uma "bíblia" de lore: ele implementa **mecânicas**. Esta página registra o
    que está de fato implementado e onde, e marca como **UNVERIFIED** o que depende de fonte externa
    (cliente, textos de NPC, material do servidor original). Nada aqui é narrativa inventada.

## O que o servidor sabe sobre a origem do mundo

O cliente 7.66 carrega os textos de história e de missão nos próprios assets, não no servidor. O que o
servidor guarda é estado: personagem, guilda, castelo, evento. A consequência prática é que **reescrever
lore não exige mudar o servidor** — exige mudar o conteúdo do cliente.

## Marcos medidos no repositório

| Marco | Onde está | Estado |
|---|---|---|
| Servidor original em C++ (TMSrv/DBSrv/BISrv) | `Source/Code/` (arquivado como referência) | evidência de wire e formato |
| Reescrita em Go do servidor | `tmserver/`, `dbserver/`, `binserver/`, `webserver/` | operacional |
| Catálogo de itens, skills e NPCs saiu dos CSV para o banco | `internal/migrations/0001_schema.up.sql` | concluído |
| Portal de administração | `portal/` | em expansão |
| Dois canais simultâneos | `docker-compose.yaml`, `docs/migration/canais.md` | operacional |
| Troca de canal em jogo | `internal/crossover/` | implementada; falta teste no cliente |

## Linha do tempo do projeto

O histórico real do projeto está no git e no roadmap (`roadmap/index.md`), não aqui: esta página é
sobre o **mundo**, o roadmap é sobre o **trabalho**. Quando os dois se misturam, o leitor não sabe se
está lendo ficção ou changelog.

## O que falta documentar

- Nomes de reinos, cidades e NPCs canônicos — estão no conteúdo do cliente (`Language.txt` e afins);
  ainda não foram extraídos para uma tabela consultável.
- A origem narrativa das guerras de castelo: o servidor implementa a **mecânica**
  (`tmserver/content/Common/Settings/CastleQuest.txt`), não a história.
