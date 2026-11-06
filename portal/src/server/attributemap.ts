/**
 * BFF do `AttributeMapAdminService` (issue 27).
 *
 * Duas operações, o mesmo caminho de RPC:
 *
 * - `getAttributeMapInfo`: metadados + histograma do arquivo atual.
 * - `transformAttributeMap`: pura do lado do servidor — devolve os bytes novos.
 *   A pré-visualização descarta os bytes (não faz sentido mandar 1 MiB por RSC)
 *   e o download os devolve crus para o navegador.
 *
 * Nada aqui escreve `tmserver/content/TMsrv/run/AttributeMap.dat`; aplicar é substituir o
 * arquivo com backup e reiniciar o tmserver, como a tela instrui.
 */
import type {
  AttributeMapInfo,
  AttributeMapRect,
  AttributeMapFilter,
  TransformParams,
  TransformSummary,
} from "@/lib/attributemap";

import { adminCall } from "./admin";
import {
  attributeMapAdmin,
  type AttributeMapTransformRequest,
  type AttributeMapTransformResponse,
} from "./grpc";

export async function getAttributeMapInfo(): Promise<AttributeMapInfo> {
  const res = await adminCall<
    Record<string, never>,
    { result: number; info?: AttributeMapInfo }
  >(attributeMapAdmin.getAttributeMapInfo.bind(attributeMapAdmin), {}, { action: "map.info" });
  if (!res.info) throw new Error("web-api respondeu OK sem os metadados do AttributeMap");
  return {
    dim: res.info.dim,
    world_scale: res.info.world_scale,
    sha256: res.info.sha256 ?? "",
    histogram: res.info.histogram ?? [],
    meanings: res.info.meanings ?? [],
  };
}

function requestFor(params: TransformParams): AttributeMapTransformRequest {
  const rect: AttributeMapRect | undefined = params.rect ?? undefined;
  const filter: AttributeMapFilter | undefined = params.filter?.enabled
    ? {
        enabled: true,
        exact_value: params.filter.exact_value,
        mask: params.filter.mask,
        match_value: params.filter.match_value,
      }
    : undefined;
  return {
    operation: params.operation,
    operand: params.operand,
    ...(rect ? { rect } : {}),
    ...(filter ? { filter } : {}),
  };
}

async function transform(
  params: TransformParams,
  auditAction: string,
): Promise<AttributeMapTransformResponse> {
  return adminCall<AttributeMapTransformRequest, AttributeMapTransformResponse>(
    attributeMapAdmin.transformAttributeMap.bind(attributeMapAdmin),
    requestFor(params),
    { action: auditAction, target: `map:op${params.operation}` },
  );
}

/** Pré-visualização: mesmos parâmetros do download, sem carregar o payload. */
export async function previewAttributeMapTransform(
  params: TransformParams,
): Promise<TransformSummary> {
  const res = await transform(params, "map.preview");
  return {
    changed_count: res.changed_count ?? 0,
    before_histogram: res.before_histogram ?? [],
    after_histogram: res.after_histogram ?? [],
    original_sha256: res.original_sha256 ?? "",
    new_sha256: res.new_sha256 ?? "",
    filename: res.filename ?? "AttributeMap_New.dat",
  };
}

export interface TransformDownload {
  bytes: Buffer;
  filename: string;
  sha256: string;
  changedCount: number;
}

/** Executa a transformação e devolve os bytes — usado só pelo route de download. */
export async function transformAttributeMapBytes(
  params: TransformParams,
): Promise<TransformDownload> {
  const res = await transform(params, "map.download");
  const raw = res.data;
  if (!raw) throw new Error("web-api respondeu sem o payload do AttributeMap");

  const bytes = Buffer.isBuffer(raw)
    ? raw
    : typeof raw === "string"
      ? Buffer.from(raw, "base64")
      : Buffer.from(raw);

  return {
    bytes,
    filename: res.filename ?? "AttributeMap_New.dat",
    sha256: res.new_sha256 ?? "",
    changedCount: res.changed_count ?? 0,
  };
}
