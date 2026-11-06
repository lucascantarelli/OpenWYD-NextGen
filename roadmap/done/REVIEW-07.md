# REVIEW-07 — Legado C++ (`Source/`) e ClientPatch_v7662: o que é, o que compila, o que fica

> **Análise profunda 06/11/2026:** números medidos, correções das afirmações envelhecidas e
> referências externas em [`REVISAO-PROFUNDA-2026-11-06.md`](../REVISAO-PROFUNDA-2026-11-06.md) §4.

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** 🟢 concluída (06/11/2026), resta o spike opcional da `LEGACY-03`

## Estado verificado (06/11/2026)

- **Inventário e classificação: feitos.** `docs/analysis/migracao-legado.md` classifica as soluções de
  `Source/Code/`; 7 ferramentas substituídas foram para `Source/Code/_legacy/` com README de
  substituição (468 KB), fora do build e da imagem.
- **ClientPatch: documentado.** `docs/analysis/legado-clientpatch.md` mapeia os hooks; o projeto segue
  em `Source/Code/ClientPatch_v7662` como referência, porque é dependência do **cliente**, não do
  servidor. Nenhum código Go lê `Release/` nem carrega a DLL.
- **Decisão registrada:** `docs/adr/0006-arquivamento-legado-cpp.md` (aceita) — `Source/` fica como
  referência de wire; nada é apagado.
- **Arquivos ainda lidos pelo servidor:** taxas, `serv00.htm`, os dois mapas e `BaseMob`/`BaseSummon`.
  O ADR 0004 registra a exceção e o porquê, o que responde ao "por que precisamos de
  `tmserver/content/`".
- **Resta opcional:** `LEGACY-03` (hook por assinatura em vez de endereço fixo). Não bloqueia nada em Go.

## Objetivo

Fechar a pergunta "isso ainda serve?" com evidência: classificar cada projeto de `Source/` em
**referência**, **ferramenta substituída** ou **dependência real do cliente**, e decidir o destino de
`ClientPatch_v7662` com uma alternativa moderna viável.

## Contexto/evidência (05/11/2026)

- `Source/` = 99 `.cpp` + 44 `.h` + 1 solução MSVC (`Cavaleiros de Kersef.sln`). Nada disso entra no
  build Go nem na imagem Docker (o `Dockerfile` compila só Go).
- `Source/Code/` tem `TMSrv`, `DBSrv`, `BISrv`, `CPSock`, `NPTool`, `DropTool`, `EDITAPPMOB`,
  `EDITAPPSHOP`, `ExpTool`, `SearchPass`, `AttributeMap_Editor`, `ZerarSkill` e `ClientPatch_v7662`.
- **ClientPatch_v7662** é um projeto C++ (`main.cpp`, `Hook.cpp`, `Functions.cpp`, `PE_Hook.h`,
  `Main.h`, `config_DX.ini`, `.vcxproj` v142) que gera uma **DLL injetada no `WYD.exe`**. Em
  `DllMain` → `DLL_PROCESS_ATTACH` ele torna a seção `.text` gravável (`VirtualProtect`), aplica
  patches e restaura a proteção (`docs/migration/client-patch-hooks.md` §0, com `arquivo:linha`).
- Duas técnicas: **patch de byte** (ex.: `0x53AC6A` `jz`→`jmp`, desliga checagem de checksum) e
  **trampoline hook** via `PE_Hook.h` (`JMP_NEAR`/`CALL_NEAR` → funções `NKD_*`).
- É dependência **do cliente**, não do servidor: o tmServer apenas assume o comportamento resultante
  (ex.: checksum não-rejeitante por padrão).
- Guard anti-execução avulsa: só roda se o host for `WYD.EXE` (`main.cpp:49-55`).

## Analisar tudo
- Mapear o que pode ser removido, migrado, atualizado, otimizado, refatorado, melhorado, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analizar arquivos que podem ser reorganizador, renomeados, atualizados, migrados, otimizados, refatorados, melhorados, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Remover arquivos que não estão sendo usados, duplicados, obsoletos, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analisar a pasta scripts, deixar apenas o que realmente será usado, remover o que já foi migrado, está sem uso.


## Respostas diretas

| Pergunta | Resposta |
|---|---|
| O que é? | DLL de modding que reescreve bytes do cliente em memória; não recompila o jogo. |
| Com quem fala? | Com o próprio processo do cliente (hooks internos); indiretamente com o servidor, mudando o que o cliente envia/aceita. |
| Está sendo compilado? | Só quando alguém abre a solução no MSVC v142; o repositório não tem build CI para isso. |
| Quem usa? | O `WYD.exe` lançado pelo launcher; o servidor depende do comportamento que ela habilita. |
| Dá para portar para Go? | **Não.** Roda dentro de um processo x86 de 32 bits no Windows, escrevendo opcodes na memória; Go não hospeda esse modelo. |
| Modernização viável? | Manter C++ e melhorar o mecanismo (loader assinado, hooks com assinatura de padrão em vez de endereço fixo, log de patch aplicado); mover para o **servidor** tudo que não precisa ser visual (ex.: SkillDelay, textos). |

## Passos

1. **Inventário** de `Source/Code/*`: para cada solução, uma linha com "o que faz / portado para Go? /
   ferramenta substituída por (seedgen, importcontent, portal) / ainda necessário".
2. **Provar o que é referência**: para cada item marcado como portado, apontar o pacote Go
   correspondente (ex.: `CPSock` → `tmserver/internal/protocol`).
3. **ClientPatch**: documentar o mapa completo de hooks (já iniciado em `client-patch-hooks.md`),
   listar o que dá para mover ao servidor e o que é intrinsecamente cliente.
4. **Decidir destino**: manter `Source/` no repositório como referência (custo baixo, valor de
   auditoria) **ou** mover para um repositório/arquivo separado; registrar a decisão em ADR
   (REVIEW-06).
5. **Se modernizar**: provar viabilidade com um spike de **um** hook usando assinatura de bytes em vez
   de endereço fixo, para sobreviver a uma nova build do cliente.

## Arquivos

- `docs/analysis/migracao-legado.md` (tabela final por solução)
- `docs/migration/client-patch-hooks.md` (mapa completo + o que migra para o servidor)
- `docs/adr/000X-source-legado.md`
- (se aprovado) `Source/` reorganizado ou externo

## Testes/verificação

- Build do cliente com a DLL aplicada (manual, Windows) e login no servidor local.
- Log de patch: cada hook aplicado registra endereço + bytes originais.

## Critérios de aceite

- [ ] Toda solução de `Source/Code` classificada com evidência (referência / substituída / necessária).
- [ ] ClientPatch descrito hook a hook, com o que pode virar servidor.
- [ ] Decisão sobre manter `Source/` no repo registrada em ADR.
- [ ] Nenhum documento afirmando que o servidor "depende" da DLL sem apontar o comportamento exato.

## Riscos/rollback

Risco: tratar como "referência" algo que ainda é necessário e perder a capacidade de diagnosticar o
cliente. Mitigação: a classificação exige apontar o substituto em Go no mesmo commit. Rollback: nada é
apagado nesta issue — só classificado e documentado.

## Issues que este review abre

- `LEGACY-01` inventário por solução C++ — ✅ concluída
- `LEGACY-02` conteúdo de runtime para dentro do servidor Go — ✅ concluída
- `LEGACY-03` spike de hook por assinatura — aberta e opcional (só se a DLL for modernizada)

## O que faltou analisar

- por uqe ainda precisamos ler arquivos xt/csv já quie portamos tudo para o banco ?
- os arquivos de mapa do cliente/servidor está alinhados, são identicos ?
- o que ainda não está sendo persistido no banco de dados e precisa ser migrado para lá ?
- quais arquicos ainda são lidos por algum serviço ? precisa mesmo ler ? já está migrado e devidamente cadastrado no banco de dados ? 
- o código já está sem dependencia dos arquivos de itens,skills, npcs, todos os dados estão no banco, corretamente cadastrados ?
- quais arquivos ainda são lidos por algum serviço ? precisa mesmo ler ? já está migrado e devidamente cadastrado no banco de dados ?
- existe algum real motivo para o tmsrv ainda ler esses arquivos e não só o banco de dados ?
- o banco de dados deve ser a unica fonte real de dado, não devemos mais depender de nenhum arquivo.
- migrations e arquivos de seed devem ser analisados,atualizados, normalizados, otimizados, refatorados, melhorados, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- a inicialização do banco de dados deve ser feita apenas com migrations e arquivos de seed, não devemos mais depender de nenhum arquivo externo.
- o banco deve ser capaz de subir do zero, completamenete, com o conteudo normalizado, otimizado, refatorado, melhorado, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- por que precisamos do conteudo de tmrsv/content ?, creio que tudo aquilo pode virar tabela e dado no banco, sendo consultado pelo servidor via banco de dados, removendo a dependencia de arquivos externos, deixando o banco de dados como unica fonte real de dado.