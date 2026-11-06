# Issue 46 — Catálogo de skills (leitura + overlay)

**Fase:** 6 · **Depende de:** 22, 25 · **Bloqueia:** — · **Esforço:** M · **Estado:** 🟡 (nomes já resolvidos na UI; edição/overlay pendente)

## Objetivo

Ver e ajustar as **151 skills** de `Release/Common/SkillData.csv` pela web — nome, classe, nível mínimo,
mana, dano/base, alvo e alcance — com overlay no Postgres, para balancear skill sem editar CSV nem
rebuildar o tmserver.

## Contexto / evidência

- `SkillData.csv` (151 entradas) é lido no boot pelo tmserver; **nenhuma RPC** expõe skill
  (`ItemCatalogService` só cobre itens, `MobTemplateAdminService` cobre `Equip`/skills do mob).
- O contrato do arquivo está em `docs/migration/data-formats.md` (colunas de skill) — a UI não
  pode inventar campo: o parser é a fonte da verdade.
- Skills afetam combate direto: mudar dano/alcance muda PvP e PvE de todos, então precisa de preview e
  diff antes de salvar (mesmo padrão da 33/44).

## Passos

1. `webserver/internal/skillcatalog`: leitor do `SkillData.csv` com o mesmo parser do tmserver (tipos,
   colunas, encoding), devolvendo `SkillEntry { index, name, class, level, mana, damage, range, target,
   max_level, flags[] }`.
2. `web.proto` + `skilladmin`: `SkillCatalogService.ListSkills` (leitura, como `ItemCatalogService`) e
   `SkillAdminService.UpsertSkill`/`DeleteSkillOverride` (overlay `skill_overlay`, com `base_hash` para
   detectar mudança no CSV).
3. Validação: faixa por campo (mana ≥ 0, nível 1..400, alcance ≥ 1, classe na tabela de classes), e
   recusa de `index` inexistente (diferente do `item_price`, aqui o índice é validado — a lição da 24).
4. tmserver: `W2PP_SKILL_EDITING=true` aplica o overlay no boot e no poll; precedência overlay > CSV.
5. UI `/skills`: `DataTable` com ícone/nome/classe/nível/mana/dano, filtros por classe e faixa de nível,
   detalhe com edição e **diff** antes de salvar; link "quem usa" (templates de mob da 25 e classes).
6. Export CSV espelhando o formato do `SkillData.csv` para revisão externa.

## Arquivos

- Criados: `webserver/internal/skillcatalog/*.go`, `webserver/internal/skilladmin/*.go`,
  `internal/migrations/0015_skill_overlay.*.sql`,
  `portal/src/app/(admin)/skills/{page.tsx,skill-table.tsx}`,
  `portal/src/server/skills.ts`, `portal/src/server/actions/skills.ts`, `portal/src/lib/skill.ts`.
- Alterados: `api/web/v1/web.proto`, `tmserver/internal/content` (read-through),
  `portal/src/components/{nav.ts,Icon.tsx}`, `portal/src/middleware.ts`.
- Proibido: editar `Release/Common/SkillData.csv`; reconstruir colunas no TypeScript sem espelhar o
  parser Go.

## Testes / verificação

```bash
go test ./webserver/internal/skillcatalog/... -v
cd portal && npx tsc --noEmit && npm run build
```

Evidência: listar 151 skills; mudar a mana de uma skill de nível 1 em -10 e ver o custo no cliente após
restart; índice inexistente → 422 com mensagem.

## Critérios de aceite

- [ ] 151 skills listadas com contagem igual ao log `skills=151` do tmserver
- [ ] Edição de campo com diff e validação de faixa
- [ ] Overlay aplicado no boot e no poll (`W2PP_SKILL_EDITING=true`)
- [ ] Índice inexistente → 422 (não aceitar como o `item_price` aceita)
- [ ] Export CSV no formato do `SkillData.csv`
- [ ] Conta `player` recebe 403; negação auditada

## Riscos / rollback

- **Risco:** divergir do parser Go e gerar skill inválida no boot. **Mitigação:** validação no web-api
  usando o **mesmo** pacote de leitura; `base_hash` avisa quando o CSV mudou por baixo.
- **Risco:** balanceamento global acidental. **Mitigação:** diff obrigatório + export CSV antes de
  aplicar.
- **Rollback:** deletar as linhas de `skill_overlay` volta ao CSV.
