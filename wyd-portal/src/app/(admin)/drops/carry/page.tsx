import { CarryEditor } from "@/components/drop/carry-editor";
import { CarryTemplatePicker } from "@/components/drop/carry-template-picker";
import { DropsNav } from "@/components/drop/drops-nav";
import { RetryButton } from "@/components/RetryButton";
import type { MobCarry } from "@/lib/mobcarry";
import type { MobTemplateFile } from "@/lib/mob";
import { AdminError } from "@/server/errors";
import { listMobCarry } from "@/server/carry";
import { listMobTemplates } from "@/server/mobs";

export const dynamic = "force-dynamic";

export const metadata = { title: "Itens que caem · WYD2" };

function message(err: unknown, fallback: string): string {
  return err instanceof AdminError ? err.message : fallback;
}

/**
 * `Carry[]` por template (issue 34, fatia 3).
 *
 * O template é escolhido na URL (`?template=`), então o link é compartilhável e
 * o botão voltar funciona. Sem template, a página só mostra o seletor — não
 * carrega tabela nenhuma.
 */
export default async function CarryPage({
  searchParams,
}: {
  searchParams: Promise<{ template?: string }>;
}) {
  const { template } = await searchParams;

  let templates: MobTemplateFile[] = [];
  let listError: string | null = null;
  try {
    templates = await listMobTemplates();
  } catch (err) {
    listError = message(err, "não foi possível listar os templates de mob do web-api");
  }

  let carry: MobCarry | null = null;
  let carryError: string | null = null;
  if (template) {
    try {
      carry = await listMobCarry(template);
    } catch (err) {
      carryError = message(err, "não foi possível ler o Carry[] deste template");
    }
  }

  return (
    <>
      <DropsNav active="carry" />

      <div className="ui-filters" style={{ marginTop: "var(--space-3)" }}>
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Itens que caem
          </h1>
          <p className="ui-page-sub" style={{ margin: "var(--space-1) 0 0" }}>
            A tabela <code>Carry[]</code> de cada template: qual item cai e em qual faixa de
            raridade. A posição é a raridade; o item é o que aparece no chão.
          </p>
        </div>
      </div>

      <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <strong>Aplicação</strong>
        <p style={{ margin: "var(--space-1) 0 0" }}>
          O tmserver lê este overlay <strong>uma vez no boot</strong> com{" "}
          <code>W2PP_MOB_CARRY_EDITING=true</code>. Sem restart, o drop continua o do último boot —
          e com a flag desligada os arquivos mandam, sem overlay nenhum.
        </p>
      </div>

      {listError ? (
        <div className="ui-card ui-empty">
          <strong>Não deu para listar os templates</strong>
          <p>{listError}</p>
          <RetryButton />
        </div>
      ) : (
        <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
          <CarryTemplatePicker templates={templates} current={template} />
          {templates.length === 0 && (
            <p className="ui-field__hint" style={{ margin: "var(--space-2) 0 0" }}>
              o web-api subiu sem <code>-content</code>: o seletor fica vazio até a árvore de
              conteúdo ser montada.
            </p>
          )}
        </div>
      )}

      {!template && !listError && (
        <div className="ui-card ui-empty">
          <strong>Escolha um mob para ver a tabela de drop</strong>
          <p>
            Cada linha é uma posição do <code>Carry[]</code>, agrupada pela raridade que a posição
            vale: “Sempre cai”, “Ultra raro”, “Comum”…
          </p>
        </div>
      )}

      {template && carryError && (
        <div className="ui-card ui-empty">
          <strong>Não deu para ler o Carry deste template</strong>
          <p>{carryError}</p>
          <RetryButton />
        </div>
      )}

      {carry && <CarryEditor carry={carry} />}
    </>
  );
}
