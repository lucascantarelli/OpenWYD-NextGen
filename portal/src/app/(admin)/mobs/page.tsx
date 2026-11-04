import { RetryButton } from "@/components/RetryButton";
import type { MobTemplateFile } from "@/lib/mob";
import { AdminError } from "@/server/errors";
import { listMobTemplates } from "@/server/mobs";

import { MobTable } from "./mob-table";

export const dynamic = "force-dynamic";

export const metadata = { title: "Mobs e stats · WYD2" };

/**
 * Lista de templates de mob (issue 25).
 *
 * Diferente de `/npcs` (que edita posição/loja dos mercadores definidos no
 * banco), aqui o alvo é o **arquivo** de template: qualquer um dos 620, com os
 * stats de combate. O override vive em Postgres; o arquivo nunca é reescrito.
 */
export default async function MobsPage() {
  let templates: MobTemplateFile[] = [];
  let error: string | null = null;

  try {
    templates = await listMobTemplates();
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível listar os templates de mob do web-api";
  }

  const merchants = templates.filter((template) => template.merchant > 0).length;
  const overrides = templates.filter((template) => template.overridden).length;

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Mobs e stats
          </h1>
          <p className="ui-page-sub" style={{ margin: "var(--space-1) 0 0" }}>
            Nível, HP/MP, atributos, EXP, skills e <code>Equip[]</code> de qualquer template. O
            editor abre com os valores do arquivo (read-through) e o que você salvar vira um
            override no banco.
          </p>
        </div>
      </div>

      <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <strong>Aplicação</strong>
        <p style={{ margin: "var(--space-1) 0 0" }}>
          O tmserver lê os overrides <strong>uma vez no boot</strong> com{" "}
          <code>W2PP_MOB_STAT_EDITING=true</code>. Sem restart, nada muda no mundo.
        </p>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: "var(--space-3)" }}>
          <strong>Não deu para listar os templates</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && templates.length === 0 && (
        <div className="ui-card ui-empty">
          <strong>Nenhum template encontrado</strong>
          <p>
            O web-api varre <code>Release/TMsrv/run/npc/</code> no boot — se a lista está vazia, ele
            subiu sem <code>-content</code> ou a árvore de conteúdo está faltando.
          </p>
        </div>
      )}

      {!error && templates.length > 0 && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: "var(--space-3)" }}>
            {templates.length} templates · {merchants} com merchant · {overrides} com override · o nome
            do arquivo é a chave estável usada pelo tmserver
          </p>
          <MobTable templates={templates} />
        </>
      )}
    </>
  );
}
