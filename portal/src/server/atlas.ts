/**
 * Atlas do mundo gerado por `webserver/cmd/worldatlas`.
 *
 * O JSON sai das **mesmas** constantes que o servidor usa (cidades e rotas de
 * teleporte vêm de `tmserver/internal/world`) mais o conteúdo de `Release/`
 * (Regions.txt, InitItem.csv, NPCGener.txt) — o portal só lê o artefato. Se o
 * arquivo não existir, a tela explica como gerar; nunca cai em 500.
 */
import { readFile, stat } from "node:fs/promises";
import path from "node:path";


const DIST = process.env.WORLD_ATLAS_DIR ?? path.join(process.cwd(), "..", "dist");

export const ATLAS_FILE = process.env.WORLD_ATLAS_PATH ?? path.join(DIST, "world-atlas.json");
export const MAP_FILE = process.env.WORLD_MAP_PATH ?? path.join(DIST, "world-map.png");
/** Mesmo relevo em resolução nativa (4096², 1 px por bloco), carregado só no zoom. */
export const DETAIL_MAP_FILE =
  process.env.WORLD_MAP_DETAIL_PATH ?? path.join(DIST, "world-map-hd.png");
/** Máscara colorida das regiões nomeadas (o contorno segue o terreno). */
export const REGION_MAP_FILE = process.env.WORLD_REGION_MAP_PATH ?? path.join(DIST, "region-map.png");

export interface AtlasCity {
  index: number;
  name: string;
  spawn_x: number;
  spawn_y: number;
  limit_x1: number;
  limit_y1: number;
  limit_x2: number;
  limit_y2: number;
  war_x1: number;
  war_y1: number;
  war_x2: number;
  war_y2: number;
}

export interface AtlasTeleport {
  src_x: number;
  src_y: number;
  dst_x: number;
  dst_y: number;
  cost: number;
  requirement: string;
}

export interface AtlasBounds {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

export interface AtlasRegion {
  name: string;
  /** Cor da região em `#rrggbb`, usada na máscara e no swatch da tabela. */
  color?: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  /** Caixa justa sobre as células caminháveis da região; ausente = retângulo cru. */
  fit?: AtlasBounds;
  source: string;
}

export interface AtlasStaticItem {
  order: number;
  item_index: number;
  x: number;
  y: number;
  rotate: number;
}

export interface AtlasSpawnHeat {
  cols: number;
  rows: number;
  /** Maior contagem de geradores numa célula; 0 = atlas sem spawns. */
  max: number;
  /** Row-major, `rows * cols`, na mesma ordem de linha do mapa. */
  data: number[];
}

export interface AtlasSpawn {
  generator: number;
  leader: string;
  follower: string;
  x: number;
  y: number;
  range: number;
  max_mob: number;
  route_type: number;
}

export interface WorldAtlas {
  version: string;
  generated_at: string;
  source_hash: string;
  cities: AtlasCity[];
  teleports: AtlasTeleport[];
  regions: AtlasRegion[];
  static_items: AtlasStaticItem[];
  spawns: AtlasSpawn[];
  /** Densidade de geradores por célula de 64² blocos. */
  spawn_heat?: AtlasSpawnHeat;
  /** Caixa de tudo que é caminhável; é onde o mapa abre enquadrado. */
  bounds?: AtlasBounds;
  counts: Record<string, number>;
}

/**
 * Cache por `mtime`: o atlas é um artefato regerado pelo `worldatlas`, não um
 * recurso remoto.
 *
 * `unstable_cache` foi usado antes e tinha dois problemas: gravava também o
 * resultado `null` em disco (a tela continuava vazia depois de gerar o arquivo) e
 * não percebia que o JSON mudou — regenerar o atlas não aparecia na página até o
 * TTL vencer. Aqui nada é cacheado sem que o arquivo exista, e qualquer nova
 * escrita invalida.
 */
let cachedAtlas: { mtimeMs: number; atlas: WorldAtlas } | null = null;

export async function getWorldAtlas(): Promise<WorldAtlas | null> {
  try {
    const info = await stat(ATLAS_FILE);
    if (cachedAtlas && cachedAtlas.mtimeMs === info.mtimeMs) return cachedAtlas.atlas;
    const raw = await readFile(ATLAS_FILE, "utf8");
    const atlas = JSON.parse(raw) as WorldAtlas;
    cachedAtlas = { mtimeMs: info.mtimeMs, atlas };
    return atlas;
  } catch {
    return null;
  }
}

/** Visão geral do relevo; `null` quando o atlas ainda não foi gerado. */
export async function readWorldMapPng(): Promise<Uint8Array | null> {
  try {
    return await readFile(MAP_FILE);
  } catch {
    return null;
  }
}

/**
 * Detalhe em resolução nativa (4096²). É pesado de propósito: existe para o zoom
 * não ampliar blocos 4×4, e a página só o pede depois de passar de 2×.
 */
export async function readWorldMapHdPng(): Promise<Uint8Array | null> {
  try {
    return await readFile(DETAIL_MAP_FILE);
  } catch {
    return null;
  }
}

/** Máscara das regiões; `null` quando ainda não foi gerada. */
export async function readRegionMapPng(): Promise<Uint8Array | null> {
  try {
    return await readFile(REGION_MAP_FILE);
  } catch {
    return null;
  }
}

export const ATLAS_COMMAND =
  "go run ./tmserver/cmd/worldatlas -content Release -out dist";
