"use client";

import { useState } from "react";

import { itemInitials } from "@/lib/item";
import type { CatalogItem } from "@/server/grpc";

const SIZE = 32;

/**
 * Ícone do item com fallback estável.
 *
 * Regra do guia (`docs/integrations/item-icons-nextjs.md §URL e renderização`):
 * começa no fallback, sobrepõe a imagem só quando ela carrega e **nunca** tenta
 * de novo depois de um erro — `onError` marca `failed` e a `<img>` sai da árvore.
 * `icon_key` é opaco: só decidimos *se* há ícone, nunca remontamos a URL.
 */
export function ItemIcon({
  item,
  iconPackVersion,
}: {
  item: CatalogItem;
  iconPackVersion: string;
}) {
  const [failed, setFailed] = useState(false);
  const src = item.icon_url && item.icon_key && iconPackVersion ? item.icon_url : "";
  const showImage = Boolean(src) && !failed;

  return (
    <span
      className="ui-item-icon"
      data-slot={item.slots?.[0] ?? "none"}
      data-grade={item.grade}
      aria-hidden="true"
    >
      <span>{itemInitials(item)}</span>
      {showImage && (
        // eslint-disable-next-line @next/next/no-img-element -- URL absoluta do storage-manager, não do Next.
        <img
          src={src}
          alt=""
          width={SIZE}
          height={SIZE}
          loading="lazy"
          decoding="async"
          onError={() => setFailed(true)}
        />
      )}
    </span>
  );
}
