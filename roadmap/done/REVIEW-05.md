# REVIEW-05 — Banco de dados: roles, retenção, naming e recursos do Postgres

> **Análise profunda 06/11/2026:** números medidos, correções das afirmações envelhecidas e
> referências externas em [`REVISAO-PROFUNDA-2026-11-06.md`](../REVISAO-PROFUNDA-2026-11-06.md) §4.

**Fase:** revisão · **Depende de:** 58 (concluída) · **Esforço:** M · **Estado:** ✅ concluída (06/11/2026) — DB-01 e DB-02 fechadas

## Estado verificado (06/11/2026)

Reconciliado contra o HEAD, achado por achado:

- **Erros de `relation`/`column` no log do Postgres: encerrados.** Nenhum código referencia
  `instance_flag` nem `audit_log`; a tabela é `server_instance_flags` e as colunas `version` existem em
  `sapphire_balance`, `npc_config_meta` e `world_event_meta`. Eram consultas de uma versão anterior,
  não migration faltando.
- **Roles/DSN: feito (DB-01).** `docker-compose.yaml` não tem mais DSN de superusuário nos serviços:
  `W2PP_DB_DSN` usa `openwyd_app` (DML) e `W2PP_DB_MIGRATE_DSN` usa `openwyd_migrator` (DDL), criadas
  pelo serviço one-shot `db-roles` a partir de `scripts/db-roles.sql`. Provado no banco: `openwyd_app`
  lê `characters`/`npc_shop_item` e `CREATE TABLE` é negado.
- **Restore: provado.** `scripts/db-restore.sh` restaura o dump num banco descartável e compara a
  contagem de todas as tabelas: **50 tabelas idênticas à origem** (06/11), banco descartável removido.
- **Stack local no HEAD:** reconstruído e reiniciado; o e2e `TestE2ESmokeLogin` passa
  (`LOGIN OK — CNFAccountLogin`) e o `dbserver` lista `characters` como `openwyd_app` sem erro de
  relação. `make seedgen` rodou duas vezes com hash idêntico (18229 linhas, 12 tabelas).
- **`character` → `characters`: feito.** A migration `0003_rename_character_to_characters` renomeia a
  tabela e os três índices, e o `store` consulta `characters` nas 27 referências — sem alias nem view
  de compatibilidade.
- **Skills da FM: corrigido.** `npc_shop_item` vendia `Garnet` (2444) no slot da 8ª magia branca e
  trazia a magia negra (5032–5039) e parte da especial trocadas por itens nos NPCs 152 e 263; o 37
  (Cap.Cavaleiros) tinha 14 skills espalhadas entre poções e equipamento. A
  `0004_fix_skill_shop_layout` repõe o layout canônico de 24 skills (3 famílias de 8) copiando o NPC
  de referência, e o seed foi regerado por `seedgen`. Um teste em `internal/migrations` (roda em
  `go test ./...`) falha quando um slot de skill aponta para item e prova que detecta o caso antigo.
- **`docs/audits/`: removido.** A auditoria vive em `docs/analysis/banco-de-dados.md`.

## Objetivo

Fechar o que o baseline deixou em aberto: papéis de banco separados, backup com restauração testada,
teste de `down`, naming consistente e uso consciente de views/índices — sem reintroduzir duas fontes
de verdade.

## Contexto/evidência

- Baseline squasheado e adotado: `0001_schema.up.sql` + 12 `seed_*.up.sql`, 49 tabelas de domínio +
  `schema_migrations`, 18.232 linhas de referência validadas em banco vazio (`docs/analysis/banco-de-dados.md` §11).
- Pendências já registradas no §10 do mesmo documento: **roles `openwyd_app`/`openwyd_ro` e DSN sem
  superusuário** (P1), **retenção de backup + teste de restauração** (P1), **`character` →
  `characters`** (P2, quebra código), **views de relatório** (P2), **revisão dos 19 índices de
  auditoria sem leitura** (P2), além de **teste de `down`** e de migração em base vazia no CI (REVIEW-04).
- Todos os serviços usam o mesmo superusuário `postgres` (DSN do compose deriva de
  `POSTGRES_PASSWORD`); nada impede o portal de escrever em tabela de personagem vivo por engano de
  DSN.
- `scripts/db-audit.sql`, `db-backup.sh` e `db-baseline.sh` existem; falta rotina de restore
  verificada.

## Analisar tudo - Possivel bug no cadastro de skills de FM
Magia branca: oitava skill está cadastrada como garnet, um item

Magia Negra: todas as skills estão erradas, aparecem itens
Magia especial: primeira, segunda e oitava skill erradas

creio que existam mais coisas que devem cadastradas nesse NPC, use os outros npcs de skill das outras classes como referencia para ajustar o npc da FM
o que gerou esse problema ?
fizemos a migração dos arquivos de itens e skills para o banco, isso foi algum problema de cadastro ? 
existem alguns problemas de itens dentro do banco ? 
vc consegue montar análises para encontrar determinados tipos de incoerencias que geram esses bugs de cadastro/falta de dados/ referencias erradas ?
Conseguimos aplicar uma melhoria na modelagem do banco de dados ? aplicar boas praticas ?
pesquisar:
 - nomes desnormalizados
 - despadronizados
 - dados incoerentes
 - nomes com padrões incoerentes
fazer realmente uma organização, normalização nos cadastros do banco, organizar itens repetidos, identificar o que são, qual utilidade, 
reorganizar cadastros para melhorar os registros, diferencia corretamente o que é cada item/skill, corrigir possiveis problemas,
incluir descrição em itens, incluir informações no banco para que serve o item, onde ele é usado, montar pequenas descrições, para sabermos onde o item pode ser usado, para que pode ser usado, se está relacionado a algum sistema de evolução tipo arch,celestial, destravar level, e outros sistemas do jogo, para poder saber se o item é uma cópia de outro com função especifica, precisamos saber essas informações para termos uma melhor gestão dos itens, do uso dos itens, das skills, melhor descrição das skills, cadastrar no banco o real funcionamento ca skill, o calculo, para futuras análises, entre outras informações
propor melhorias, organização, faça pesquisa de itens do wyd, funcções de itens, propor uma melhoria completa nos nossos dados, 
deixar nosso catalogo completamente limpo, organizado, com dados reais, coerentes devidamente preenchidos, identificados, 
analisar se podemos fazer algum tipo de remodelagem para melhorar os cadastros,aplicar boas praticas de modelagem de dados para organizar nosso banco
tudo isso é apenas uma análise e propostas, verificar viabilidade de execução, propor melhorias, resolver problemas de relacionamento,analisar se precisamos criar views/materialized views para ganho de desempaenho, criar melhores relações, ou não e preciso, traria problemas ou complexidade, sem ganhos reais.
Alguns erros encontrados no log
db | 2026-11-05 21:01:59.887 UTC [16881] ERROR:  relation "instance_flag" does not exist at character 35
db | 2026-11-05 21:03:40.116 UTC [17054] ERROR:  column "version" does not exist at character 8
db | 2026-11-05 21:04:09.883 UTC [17124] ERROR:  column "name" does not exist at character 350
db | 2026-11-05 21:06:13.895 UTC [17313] ERROR:  relation "audit_log" does not exist at character 54
db | 2026-11-05 21:06:51.192 UTC [17399] ERROR:  column "version" does not exist at character 12
Faça uma pesquisa extensa na internet sobre todos os sistemas, itens, como é executado o sistema, prerequisitos, tudo, documente em docs/sistemas/
quero que vc entenda como tudo funciona, evolução, guerras, eventos, uso dos itens, para que cada coisa serve e documente tudo, para podermos implementar corretamente todos os comportamentos.
pesquise referencias em sites de servidores privates/oficiais para referencias de eventos, uso de itens, para que cada item serve, bugs, skills, novos sistemas diferentes dos originais/padrões
crie uma tabela de eventos, predefinidos, baseado em pesquisas de eventos de outros servidores, para criar templates de eventos, com toda a configuração dos eventos, itens usados,recompensas,etc.
analisar as nossas migrations, o que pode ser melhorado após a análise completa, o que deve ser atualizar, reaorganizado,e demais ajustes.
proponha melhorias, modernizações, avanções.
foco principal é implementar todos os comportamentos padrões/oficiais, depois iremos partir para curtomizações.

## Passos

1. **Roles**: `openwyd_app` (DML no que o boot precisa), `openwyd_ro` (portal de leitura) e
   `openwyd_migrator` (DDL); DSNs por serviço no compose; `ALTER DEFAULT PRIVILEGES` no schema.
2. **Teste de restauração**: `scripts/db-restore.sh` que restaura o último dump em um banco
   descartável e compara contagem por tabela; rodar mensal e registrar data em `docs/analysis/banco-de-dados.md`.
3. **Teste de `down`**: em base efêmera, aplicar `0001_schema.up.sql` e depois `down`, validando que
   o schema some sem erro (o `down` é `DROP SCHEMA … CASCADE`, então o teste é de fumaça, não de
   reversibilidade de dados).
4. **Naming**: decidir `character` → `characters` com uma entrega só (migration + código + portal +
   testes) — não deixar meio-caminho.
5. **Índices de auditoria**: para cada um dos 19, medir `pg_stat_user_indexes.idx_scan` após uma
   semana de uso; remover os que nunca são lidos e o dado não justifique.
6. **Views**: criar só quando houver consumo real (portal/relatório), com `security_invoker` para não
   furar RLS/papéis.

## Arquivos

- `internal/migrations/` (nova migration de roles — nunca reutilizando número)
- `docker-compose.yaml`, `.env.example` (DSNs por serviço)
- `scripts/db-restore.sh` (novo)
- `docs/analysis/banco-de-dados.md` (evidência)

## Testes/verificação

```bash
bash scripts/db-baseline.sh                     # dry-run: validação em base vazia
bash scripts/db-restore.sh --check              # restaura em base descartável e compara contagens
docker exec openwyd-db-1 psql -U postgres -c "\du"   # papéis e atributos
```

## Critérios de aceite

- [ ] Nenhum serviço conecta como superusuário; portal só lê o que precisa ler.
- [ ] Restore do último backup validado, com data e contagem registradas.
- [ ] `down` do baseline testado em base efêmera.
- [ ] Decisão sobre `characters` tomada e executada (ou issue própria fechada).
- [ ] Índices de auditoria revisados com `idx_scan` como evidência.

## Riscos/rollback

Risco: trocar DSN por papel sem GRANT completo quebra o boot em produção (foi o padrão dos bugs do
gate da fase 4). Mitigação: aplicar em base efêmera primeiro, com `store.Migrate` + boot completo, e
só então no dev. Rollback: o compose volta ao DSN antigo por variável de ambiente.

## Issues que este review abre

- `DB-01` roles e DSN por serviço + restore testado — ✅ concluída 06/11
- `DB-02` teste de `down` + revisão dos índices de auditoria — ✅ concluída 06/11 (down provado em base descartável; 17 índices removidos, 2 mantidos com `idx_scan`)

## O que faltou analisar

- por que ainda precisamos ler arquivos txt/csv já qu tudo deve  ser portadopara o banco ?
- o que ainda não está sendo persistido no banco de dados e precisa ser migrado para lá ?
- existe algum real motivo para o tmsrv ainda ler esses arquivos e não só o banco de dados ?
- o banco de dados deve ser a unica fonte real de dado, não devemos mais depender de nenhum arquivo.
- migrations e arquivos de seed devem ser analisados,atualizados, normalizados, otimizados, refatorados, melhorados, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- a inicialização do banco de dados deve ser feita apenas com migrations e arquivos de seed, não devemos mais depender de nenhum arquivo externo.
- o banco deve ser capaz de subir do zero, completamenete, com o conteudo normalizado, otimizado, refatorado, melhorado, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- todo o migration/seed deve ser atualizado para não depender de nenhum arquivo externo, tudo deve ser persistido no banco de dados, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- todo o banco deve ter seu conteudo normalizado, otimizado, refatorado, melhorado, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- após todo o mogration ser ajustado e garantirmos que não irá entrar/conter mais nenhum arquivo externo, não precisaremos mais importar nenhum dados, usando nenhuma ferramente, tivermos os dados completos garantidos em seeds/migrations, devemos remover todas as ferramentas que migravam dados de csv elas ficam obsoletas, remover testes de normalização desnecessarios.
- garantir o banco de dados com dados limpos,normalizados, sem duplicações, sem lacunas de dados, padronizado, otimizado, refatorado, melhorado, usando todos os recursos que tragam ganhos reais, nosso portgres é o 17, pesquisar melhores abordagens, praticas e otimizações.

### Respostas (06/11/2026)

- **Por que ainda há arquivo?** Só restam exceções do ADR 0004: taxas em `Common/Settings/*.txt`,
  `serv00.htm`, os dois mapas binários e os templates `BaseMob`/`BaseSummon`. Catálogos de item, skill,
  NPC e spawn já vêm do banco (issue 58).
- **O banco sobe do zero?** Sim: `0001_schema` + `0002_normalize_text` + 12 `seed_*`, validados em base
  vazia por `scripts/db-baseline.sh`.
- **Quando remover as ferramentas de import?** `tmserver/cmd/importcontent` e `importteleports` só saem
  depois de fechar as exceções acima; hoje servem para reprocessar árvore externa/backup.