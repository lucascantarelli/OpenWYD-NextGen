"use client";

import { useRouter } from "next/navigation";
import { useCallback } from "react";

import { Combobox, type ComboOption } from "@/components/Combobox";
import type { MobTemplateFile } from "@/lib/mob";

const LIMIT = 30;

/**
 * Seletor do template cujo `Carry[]` será editado.
 *
 * A lista de templates já vem do BFF (o mesmo scan de `/mobs`), então a busca é
 * local e não dispara request por tecla. A URL carrega o **nome do arquivo**,
 * que é a chave estável que o tmserver usa — nada de índice posicional.
 */
export function CarryTemplatePicker({
  templates,
  current,
}: {
  templates: readonly MobTemplateFile[];
  current?: string;
}) {
  const router = useRouter();

  const load = useCallback(
    async (query: string): Promise<ComboOption[]> => {
      const term = query.trim().toLowerCase();
      const matches = templates.filter((template) => {
        if (!term) return true;
        return (
          template.template_name.toLowerCase().includes(term) ||
          template.display_name.toLowerCase().includes(term)
        );
      });
      return matches.slice(0, LIMIT).map((template) => ({
        value: template.template_name,
        label: template.display_name || template.template_name,
        hint: template.merchant > 0 ? `${template.template_name} · mercador` : template.template_name,
      }));
    },
    [templates],
  );

  return (
    <div style={{ maxWidth: 460 }}>
      <Combobox
        label="Template do mob"
        placeholder="nome do mob ou do arquivo…"
        load={load}
        onSelect={(option) => router.push(`/drops/carry?template=${encodeURIComponent(option.value)}`)}
      />
      {current && (
        <p className="ui-field__hint" style={{ margin: "var(--space-1) 0 0" }}>
          editando <code>{current}</code> — escolha outro acima para trocar
        </p>
      )}
    </div>
  );
}
