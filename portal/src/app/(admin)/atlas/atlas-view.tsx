"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@/components/Badge";
import { DataTable, type Column } from "@/components/DataTable";
import { useToast } from "@/components/Toast";
import { fmtCoin } from "@/lib/format";
import type {
  AtlasCity,
  AtlasRegion,
  AtlasSpawn,
  AtlasStaticItem,
  AtlasTeleport,
  WorldAtlas,
} from "@/server/atlas";

const WORLD = 4096;
const SPAWN_LIMIT = 2000;
const ZOOM_MIN = 1;
// Com o detalhe nativo (4096 px) o palco de ~940 px chega a 1:1 por volta de 4×;
// 6× ainda mostra ~1,5 px de tela por bloco em vez de esticar blocos.
const ZOOM_MAX = 6;

interface View {
  scale: number;
  x: number;
  y: number;
}

/**
 * Mantém a imagem cobrindo o palco: sem isso o zoom descola e sobra área vazia
 * nas bordas (o clássico “arrastei e o mapa sumiu”).
 */
function clampView(view: View, size: number): View {
  const scale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, view.scale));
  const span = (scale - 1) * size;
  return {
    scale,
    x: Math.min(0, Math.max(-span, view.x)),
    y: Math.min(0, Math.max(-span, view.y)),
  };
}

/**
 * Nome do lugar de uma coordenada: cidade próxima tem precedência (é onde os NPCs
 * de teleporte ficam), senão a região nomeada que contém o ponto, senão a cidade
 * mais próxima com a distância — em vez de só repetir os números.
 */
function placeLabel(atlas: WorldAtlas, x: number, y: number): string {
  let nearestCity: AtlasCity | null = null;
  let nearestDistance = Number.POSITIVE_INFINITY;
  for (const city of atlas.cities) {
    const distance = Math.hypot(city.spawn_x - x, city.spawn_y - y);
    if (distance < nearestDistance) {
      nearestCity = city;
      nearestDistance = distance;
    }
  }
  if (nearestCity && nearestDistance <= 400) return nearestCity.name;
  const region = atlas.regions.find((candidate) => {
    const box = candidate.fit ?? candidate;
    return x >= box.x1 && x <= box.x2 && y >= box.y1 && y <= box.y2;
  });
  if (region) return region.name;
  if (nearestCity) return `${nearestCity.name} (${Math.round(nearestDistance)})`;
  return "área sem nome";
}

/** Converte um ponto da tela em coordenada do mundo (0..4096). */
function worldPoint(rect: DOMRect, view: View, clientX: number, clientY: number) {
  const size = rect.width || 1;
  const x = Math.round((((clientX - rect.left - view.x) / view.scale) / size) * WORLD);
  const y = Math.round((((clientY - rect.top - view.y) / view.scale) / size) * WORLD);
  if (x < 0 || y < 0 || x > WORLD || y > WORLD) return null;
  return { x, y };
}

type Layer = "cities" | "regions" | "teleports" | "heat" | "spawns" | "items";

/**
 * Mapa + camadas do atlas. As coordenadas do jogo são 0..4096 (um bloco por
 * célula do `AttributeMap` 1024²), então o SVG usa `viewBox="0 0 4096 4096"`
 * e cobre exatamente o PNG de 512².
 */
export function AtlasView({
  atlas,
  itemNames,
}: {
  atlas: WorldAtlas;
  /** índice do item → nome do catálogo; vazio quando o catálogo não carregou. */
  itemNames: Record<string, string>;
}) {
  const toast = useToast();
  const [layers, setLayers] = useState<Record<Layer, boolean>>({
    cities: true,
    regions: true,
    teleports: true,
    heat: false,
    spawns: false,
    items: false,
  });
  const [selected, setSelected] = useState<string | null>(null);
  const stageRef = useRef<HTMLDivElement | null>(null);
  const drag = useRef<{ id: number; startX: number; startY: number; x: number; y: number } | null>(null);
  const pointers = useRef(new Map<number, { x: number; y: number }>());
  const pinch = useRef<{ distance: number; scale: number } | null>(null);
  const [fullscreen, setFullscreen] = useState(false);

  function pointerDistance(): number {
    const [a, b] = [...pointers.current.values()];
    return a && b ? Math.hypot(a.x - b.x, a.y - b.y) : 0;
  }
  const [view, setView] = useState<View>({ scale: 1, x: 0, y: 0 });
  const [hover, setHover] = useState<{ x: number; y: number } | null>(null);
  const [picked, setPicked] = useState<{ x: number; y: number } | null>(null);
  const [gotoX, setGotoX] = useState("");
  const [gotoY, setGotoY] = useState("");

  /** “ir para”: atende chamado de jogador por coordenada sem caçar no mapa. */
  function goToCoordinate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const x = Number(gotoX);
    const y = Number(gotoY);
    if (!Number.isFinite(x) || !Number.isFinite(y) || x < 0 || y < 0 || x > WORLD || y > WORLD) {
      toast("coordenada fora do mundo (0–4096)", "error");
      return;
    }
    focusOn(x, y, 4, true);
  }

  /** Zoom ancorado no cursor: o ponto sob o ponteiro não se mexe. */
  const zoomBy = useCallback((factor: number, px?: number, py?: number) => {
    setView((current) => {
      const size = stageRef.current?.clientWidth ?? 0;
      const anchorX = px ?? size / 2;
      const anchorY = py ?? size / 2;
      const scale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, current.scale * factor));
      const ratio = scale / current.scale;
      return clampView(
        {
          scale,
          x: anchorX - (anchorX - current.x) * ratio,
          y: anchorY - (anchorY - current.y) * ratio,
        },
        size,
      );
    });
  }, []);

  /** Centraliza o mapa num ponto (usado pelo “ver no mapa” das cidades). */
  const focusOn = useCallback((x: number, y: number, scale = 4, mark = true) => {
    const size = stageRef.current?.clientWidth ?? 0;
    const target = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, scale));
    setView(
      clampView(
        {
          scale: target,
          x: size / 2 - (x / WORLD) * size * target,
          y: size / 2 - (y / WORLD) * size * target,
        },
        size,
      ),
    );
    if (mark) setPicked({ x: Math.round(x), y: Math.round(y) });
  }, []);

  /** Enquadra o mapa na caixa de tudo que é caminhável (issue 48). */
  const frameOnPlayable = useCallback(() => {
    const stage = stageRef.current;
    const box = atlas.bounds;
    if (!stage || !box) {
      setView({ scale: 1, x: 0, y: 0 });
      return;
    }
    const span = Math.max(box.x2 - box.x1, box.y2 - box.y1, 1);
    const scale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, (WORLD / span) * 0.94));
    focusOn((box.x1 + box.x2) / 2, (box.y1 + box.y2) / 2, scale, false);
  }, [atlas.bounds, focusOn]);

  // Esc sai da tela cheia (o palco vira uma camada fixa sobre o portal).
  useEffect(() => {
    if (!fullscreen) return;
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") setFullscreen(false);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [fullscreen]);

  // Abre em cima da primeira cidade (Armia), não no mundo inteiro.
  //
  // Enquadrar no `bounds` não resolveu porque as áreas jogáveis ocupam o 4096²
  // todo (regiões 128→4076, spawns de 321 a 3474): o vazio está ENTRE as ilhas,
  // não em volta delas. Abrir a 3× na cidade dá uma tela útil de imediato, e
  // “ajustar” (ou a tecla 0) volta ao mundo inteiro.
  useEffect(() => {
    const id = window.requestAnimationFrame(() => {
      const city = atlas.cities[0];
      if (city) focusOn(city.spawn_x, city.spawn_y, 3, false);
      else frameOnPlayable();
    });
    return () => window.cancelAnimationFrame(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // `wheel` precisa de listener não-passivo: o React registra wheel como
  // passivo e o preventDefault seria ignorado (a página rolaria junto).
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    function onWheel(event: WheelEvent) {
      event.preventDefault();
      const rect = stage!.getBoundingClientRect();
      zoomBy(event.deltaY < 0 ? 1.25 : 1 / 1.25, event.clientX - rect.left, event.clientY - rect.top);
    }
    stage.addEventListener("wheel", onWheel, { passive: false });
    return () => stage.removeEventListener("wheel", onWheel);
  }, [zoomBy]);

  function onPointerDown(event: React.PointerEvent<HTMLDivElement>) {
    if (event.button !== 0) return;
    pointers.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
    // Dois dedos = pinch; o arrasto simples sai de cena para os gestos não brigarem.
    if (pointers.current.size >= 2) {
      drag.current = null;
      pinch.current = { distance: pointerDistance(), scale: view.scale };
      return;
    }
    drag.current = { id: event.pointerId, startX: event.clientX, startY: event.clientY, x: event.clientX, y: event.clientY };
    try {
      event.currentTarget.setPointerCapture(event.pointerId);
    } catch {
      // ponteiro sintético (teste/automação) não pode ser capturado
    }
  }

  function onPointerMove(event: React.PointerEvent<HTMLDivElement>) {
    const stage = stageRef.current;
    if (!stage) return;
    const rect = stage.getBoundingClientRect();
    if (pointers.current.has(event.pointerId)) {
      pointers.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
    }

    if (pointers.current.size >= 2) {
      const [a, b] = [...pointers.current.values()];
      const start = pinch.current;
      if (start && start.distance > 0 && a && b) {
        const distance = Math.hypot(a.x - b.x, a.y - b.y);
        const midX = (a.x + b.x) / 2 - rect.left;
        const midY = (a.y + b.y) / 2 - rect.top;
        setView((value) => {
          const scale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, start.scale * (distance / start.distance)));
          const ratio = scale / value.scale;
          return clampView(
            { scale, x: midX - (midX - value.x) * ratio, y: midY - (midY - value.y) * ratio },
            rect.width,
          );
        });
      }
      return;
    }

    const current = drag.current;
    if (current && current.id === event.pointerId) {
      const dx = event.clientX - current.x;
      const dy = event.clientY - current.y;
      current.x = event.clientX;
      current.y = event.clientY;
      setView((value) => clampView({ ...value, x: value.x + dx, y: value.y + dy }, rect.width));
    }
    setHover(worldPoint(rect, view, event.clientX, event.clientY));
  }

  function onPointerUp(event: React.PointerEvent<HTMLDivElement>) {
    pointers.current.delete(event.pointerId);
    if (pointers.current.size < 2) pinch.current = null;
    const current = drag.current;
    drag.current = null;
    const stage = stageRef.current;
    if (!stage) return;
    // Arrastar não é clicar: só lê o ponto quando o ponteiro quase não andou.
    if (current && Math.hypot(event.clientX - current.startX, event.clientY - current.startY) > 4) return;
    setPicked(worldPoint(stage.getBoundingClientRect(), view, event.clientX, event.clientY));
  }

  function toggle(layer: Layer) {
    setLayers((current) => ({ ...current, [layer]: !current[layer] }));
  }

  async function copy(x: number, y: number) {
    try {
      await navigator.clipboard.writeText(`${x},${y}`);
      toast(`copiado: ${x},${y}`, "ok");
    } catch {
      toast(`${x},${y}`, "info");
    }
  }

  /** O que existe no ponto clicado: região, cidade e spawn mais próximos. */
  /**
   * Camada de calor: uma célula de 64² blocos vira um retângulo translúcido.
   * Memoizada porque o arrasto re-renderiza a view a cada movimento e 4k nós
   * reconciliados por frame travariam o pan.
   */
  const heatLayer = useMemo(() => {
    const grid = atlas.spawn_heat;
    if (!grid || grid.max <= 0) return null;
    const cell = WORLD / grid.cols;
    const cells = grid.data.map((value, index) => {
      if (value === 0) return null;
      const cx = index % grid.cols;
      const cy = Math.floor(index / grid.cols);
      // Raiz quadrada: uma célula tem 813 geradores e as demais dezenas, então a
      // escala linear deixaria o mapa quase todo uniforme.
      const intensity = Math.sqrt(value / grid.max);
      return (
        <rect
          key={index}
          x={cx * cell}
          y={cy * cell}
          width={cell}
          height={cell}
          fill={`rgba(224,106,106,${(0.1 + intensity * 0.55).toFixed(2)})`}
        />
      );
    });
    return <g aria-hidden>{cells}</g>;
  }, [atlas.spawn_heat]);

  const pickedInfo = useMemo(() => {
    if (!picked) return null;
    let city: AtlasCity | null = null;
    let cityDistance = Number.POSITIVE_INFINITY;
    for (const candidate of atlas.cities) {
      const distance = Math.hypot(candidate.spawn_x - picked.x, candidate.spawn_y - picked.y);
      if (distance < cityDistance) {
        city = candidate;
        cityDistance = distance;
      }
    }
    let spawn: AtlasSpawn | null = null;
    let spawnDistance = Number.POSITIVE_INFINITY;
    for (const candidate of atlas.spawns) {
      const distance = Math.hypot(candidate.x - picked.x, candidate.y - picked.y);
      if (distance < spawnDistance) {
        spawn = candidate;
        spawnDistance = distance;
      }
    }
    const regions = atlas.regions.filter(
      (region) =>
        picked.x >= region.x1 && picked.x <= region.x2 && picked.y >= region.y1 && picked.y <= region.y2,
    );
    return { city, cityDistance, spawn, spawnDistance, regions };
  }, [picked, atlas]);

  const regionColumns = useMemo<Column<AtlasRegion>[]>(
    () => [
      {
        key: "color",
        header: "",
        render: (r) => (
          <span
            aria-hidden
            style={{
              display: "inline-block",
              width: 12,
              height: 12,
              borderRadius: 3,
              background: r.color ?? "transparent",
            }}
          />
        ),
      },
      { key: "name", header: "Região", sortValue: (r) => r.name },
      {
        key: "rect",
        header: "Retângulo",
        sortValue: (r) => r.x1,
        render: (r) => (
          <span className="ui-mono">
            {r.x1},{r.y1} → {r.x2},{r.y2}
          </span>
        ),
      },
      {
        key: "action",
        header: "",
        render: (r) => (
          <span style={{ display: "inline-flex", gap: 6 }}>
            <button type="button" className="ui-btn ui-btn--ghost" onClick={() => copy(r.x1, r.y1)}>
              copiar
            </button>
            <button type="button" className="ui-btn ui-btn--ghost" onClick={() => setSelected(r.name)}>
              destacar
            </button>
          </span>
        ),
      },
    ],
    // `copy`/`setSelected` são estáveis o suficiente para a tabela.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const teleportColumns = useMemo<Column<AtlasTeleport>[]>(
    () => [
      {
        key: "src",
        header: "Origem",
        sortValue: (t) => t.src_x,
        render: (t) => (
          <span>
            {placeLabel(atlas, t.src_x, t.src_y)}{" "}
            <span className="ui-mono" title="coordenada">
              {t.src_x},{t.src_y}
            </span>
          </span>
        ),
      },
      {
        key: "dst",
        header: "Destino",
        sortValue: (t) => t.dst_x,
        render: (t) => (
          <span>
            {placeLabel(atlas, t.dst_x, t.dst_y)}{" "}
            <span className="ui-mono" title="coordenada">
              {t.dst_x},{t.dst_y}
            </span>
          </span>
        ),
      },
      { key: "cost", header: "Custo", align: "right", sortValue: (t) => t.cost, render: (t) => fmtCoin(t.cost) },
      {
        key: "req",
        header: "Requisito",
        render: (t) =>
          t.requirement === "none" ? (
            <span className="ui-field__hint">—</span>
          ) : (
            <Badge tone={t.requirement === "fairy" ? "accent" : "warn"}>{t.requirement}</Badge>
          ),
      },
    ],
    [],
  );

  const spawnColumns = useMemo<Column<AtlasSpawn>[]>(
    () => [
      { key: "gen", header: "#", align: "right", sortValue: (s) => s.generator },
      { key: "leader", header: "Líder", sortValue: (s) => s.leader, render: (s) => <span className="ui-mono">{s.leader}</span> },
      { key: "follower", header: "Seguidor", render: (s) => <span className="ui-mono">{s.follower || "—"}</span> },
      { key: "pos", header: "Posição", render: (s) => <span className="ui-mono">{s.x},{s.y}</span> },
      { key: "range", header: "Raio", align: "right", sortValue: (s) => s.range },
      { key: "max", header: "Máx. mobs", align: "right", sortValue: (s) => s.max_mob },
      {
        key: "action",
        header: "",
        render: (s) => (
          <button type="button" className="ui-btn ui-btn--ghost" onClick={() => copy(s.x, s.y)}>
            copiar
          </button>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const cityColumns = useMemo<Column<AtlasCity>[]>(
    () => [
      { key: "name", header: "Cidade", sortValue: (c) => c.name },
      { key: "spawn", header: "Spawn", render: (c) => <span className="ui-mono">{c.spawn_x},{c.spawn_y}</span> },
      {
        key: "limit",
        header: "Limite",
        render: (c) => (
          <span className="ui-mono">
            {c.limit_x1},{c.limit_y1} → {c.limit_x2},{c.limit_y2}
          </span>
        ),
      },
      {
        key: "war",
        header: "Arena de guerra",
        render: (c) => (
          <span className="ui-mono">
            {c.war_x1},{c.war_y1} → {c.war_x2},{c.war_y2}
          </span>
        ),
      },
      {
        key: "action",
        header: "",
        render: (c) => (
          <button type="button" className="ui-btn ui-btn--ghost" onClick={() => focusOn(c.spawn_x, c.spawn_y)}>
            ver no mapa
          </button>
        ),
      },
    ],
    [],
  );

  const itemColumns = useMemo<Column<AtlasStaticItem>[]>(
    () => [
      { key: "order", header: "#", align: "right", sortValue: (i) => i.order },
      {
        key: "item",
        header: "Item",
        sortValue: (i) => itemNames[String(i.item_index)] ?? `#${i.item_index}`,
        render: (i) => (
          <span title={`índice ${i.item_index}`}>
            {itemNames[String(i.item_index)] ?? `item #${i.item_index}`}
          </span>
        ),
      },
      { key: "pos", header: "Posição", render: (i) => <span className="ui-mono">{i.x},{i.y}</span> },
      { key: "rotate", header: "Rotação", align: "right" },
    ],
    [itemNames],
  );

  const spawnsShown = atlas.spawns.slice(0, SPAWN_LIMIT);

  return (
    <>
      <div className="ui-filters">
        {(
          [
            ["cities", "cidades"],
            ["regions", "regiões"],
            ["teleports", "teleportes"],
            ["heat", "calor de spawn"],
            ["spawns", "spawns"],
            ["items", "itens fixos"],
          ] as [Layer, string][]
        ).map(([layer, label]) => (
          <button
            key={layer}
            type="button"
            className="ui-chip"
            aria-pressed={layers[layer]}
            onClick={() => toggle(layer)}
          >
            {label}
          </button>
        ))}
        <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
          atlas {atlas.source_hash} · gerado {atlas.generated_at.slice(0, 16).replace("T", " ")}Z
        </span>
      </div>

      <div className="ui-atlas">
        <div
          ref={stageRef}
          className={`ui-atlas__stage${fullscreen ? " ui-atlas__stage--full" : ""}`}
          role="application"
          tabIndex={0}
          aria-label="Mapa do mundo: arraste para mover, roda ou +/- para o zoom, setas para deslocar"
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerLeave={() => {
            drag.current = null;
            setHover(null);
          }}
          onDoubleClick={(event) => {
            const stage = stageRef.current;
            if (!stage) return;
            const rect = stage.getBoundingClientRect();
            zoomBy(1.6, event.clientX - rect.left, event.clientY - rect.top);
          }}
          onKeyDown={(event) => {
            const size = stageRef.current?.clientWidth ?? 0;
            const step = 60;
            if (event.key === "ArrowLeft" || event.key === "ArrowRight" || event.key === "ArrowUp" || event.key === "ArrowDown") {
              event.preventDefault();
              setView((value) =>
                clampView(
                  {
                    ...value,
                    x: value.x + (event.key === "ArrowLeft" ? step : event.key === "ArrowRight" ? -step : 0),
                    y: value.y + (event.key === "ArrowUp" ? step : event.key === "ArrowDown" ? -step : 0),
                  },
                  size,
                ),
              );
            } else if (event.key === "+" || event.key === "=") {
              event.preventDefault();
              zoomBy(1.4);
            } else if (event.key === "-" || event.key === "_") {
              event.preventDefault();
              zoomBy(1 / 1.4);
            } else if (event.key === "0") {
              event.preventDefault();
              frameOnPlayable();
            }
          }}
        >
          <div
            className="ui-atlas__canvas"
            style={{ transform: `translate(${view.x}px, ${view.y}px) scale(${view.scale})` }}
          >
            {/* eslint-disable-next-line @next/next/no-img-element -- PNG estático servido pela rota do atlas. */}
            <img
              className="ui-atlas__map"
              src={`/api/atlas/map?v=${atlas.source_hash}`}
              alt="Mapa do mundo: relevo do HeightMap com a colisão do AttributeMap"
              width={1024}
              height={1024}
              draggable={false}
            />
            {view.scale > 1.5 && (
              // Empilhado sobre a visão geral: se o detalhe ainda está baixando,
              // o mapa aparenta normal em vez de piscar em branco. O limiar é 1,5
              // porque o primeiro degrau dos botões já dá 1,4×.
              // eslint-disable-next-line @next/next/no-img-element
              <img
                className="ui-atlas__map ui-atlas__map--detail"
                src={`/api/atlas/map?detail=1&v=${atlas.source_hash}`}
                alt=""
                aria-hidden
                draggable={false}
                decoding="async"
              />
            )}
            {layers.regions && (
              // Raster das regiões: o contorno segue as células caminháveis, em vez
              // dos retângulos frouxos do Regions.txt.
              // eslint-disable-next-line @next/next/no-img-element
              <img
                className="ui-atlas__map ui-atlas__map--regions"
                src={`/api/atlas/map?layer=regions&v=${atlas.source_hash}`}
                alt=""
                aria-hidden
                draggable={false}
              />
            )}
            <svg className="ui-atlas__overlay" viewBox={`0 0 ${WORLD} ${WORLD}`} role="img" aria-label="Camadas do atlas">
              {layers.heat && heatLayer}
          {layers.cities &&
            atlas.cities.map((city) => (
              <g key={city.index}>
                <rect
                  x={city.limit_x1}
                  y={city.limit_y1}
                  width={Math.max(city.limit_x2 - city.limit_x1, 2)}
                  height={Math.max(city.limit_y2 - city.limit_y1, 2)}
                  fill="rgba(200,162,74,0.18)"
                  stroke="rgba(200,162,74,0.9)"
                  strokeWidth={3}
                />
                <circle cx={city.spawn_x} cy={city.spawn_y} r={10} fill="#c8a24a" />
              </g>
            ))}
          {layers.regions &&
            atlas.regions
              // O preenchimento agora é o raster; o SVG só destaca a escolhida.
              .filter((region) => region.name === selected)
              .map((region) => {
                const box = region.fit ?? region;
                return (
                  <rect
                    key={region.name}
                    x={box.x1}
                    y={box.y1}
                    width={Math.max(box.x2 - box.x1, 2)}
                    height={Math.max(box.y2 - box.y1, 2)}
                    fill="none"
                    stroke={region.color ?? "#55b97c"}
                    strokeWidth={5}
                  />
                );
              })}
          {atlas.bounds && (
            <rect
              x={atlas.bounds.x1}
              y={atlas.bounds.y1}
              width={atlas.bounds.x2 - atlas.bounds.x1}
              height={atlas.bounds.y2 - atlas.bounds.y1}
              fill="none"
              stroke="rgba(233,237,244,0.28)"
              strokeDasharray="28 22"
              strokeWidth={4}
            />
          )}
          {layers.teleports &&
            atlas.teleports.map((route) => (
              <g key={`${route.src_x}-${route.src_y}`}>
                <line
                  x1={route.src_x}
                  y1={route.src_y}
                  x2={route.dst_x}
                  y2={route.dst_y}
                  stroke="rgba(200,162,74,0.55)"
                  strokeWidth={2}
                  strokeDasharray="12 10"
                />
                <circle cx={route.src_x} cy={route.src_y} r={8} fill="#c8a24a" />
                <circle cx={route.dst_x} cy={route.dst_y} r={5} fill="#e9edf4" />
              </g>
            ))}
          {layers.spawns &&
            spawnsShown.map((spawn) => (
              <circle key={spawn.generator} cx={spawn.x} cy={spawn.y} r={4} fill="rgba(224,106,106,0.75)" />
            ))}
          {layers.items &&
            atlas.static_items.map((item) => (
              <rect key={item.order} x={item.x - 6} y={item.y - 6} width={12} height={12} fill="#55b97c" />
            ))}
              {picked && (
                <g>
                  <circle cx={picked.x} cy={picked.y} r={16} fill="none" stroke="#e9edf4" strokeWidth={4} />
                  <line x1={picked.x - 26} y1={picked.y} x2={picked.x + 26} y2={picked.y} stroke="#e9edf4" strokeWidth={3} />
                  <line x1={picked.x} y1={picked.y - 26} x2={picked.x} y2={picked.y + 26} stroke="#e9edf4" strokeWidth={3} />
                </g>
              )}
            </svg>
          </div>
        </div>

        <div className="ui-atlas__bar">
          <button type="button" className="ui-btn ui-btn--ghost" aria-label="Diminuir zoom" onClick={() => zoomBy(1 / 1.4)}>
            −
          </button>
          <button type="button" className="ui-btn ui-btn--ghost" aria-label="Aumentar zoom" onClick={() => zoomBy(1.4)}>
            +
          </button>
          <button
            type="button"
            className="ui-btn ui-btn--ghost"
            onClick={() => {
              setPicked(null);
              frameOnPlayable();
            }}
          >
            ajustar
          </button>
          <button
            type="button"
            className="ui-btn ui-btn--ghost"
            aria-pressed={fullscreen}
            onClick={() => setFullscreen((value) => !value)}
          >
            {fullscreen ? "sair" : "tela cheia"}
          </button>
          <span className="ui-mono">{view.scale.toFixed(1)}×</span>
          <span className="ui-mono" title="coordenada sob o ponteiro">
            {hover ? `${hover.x},${hover.y}` : "—,—"}
          </span>
          <span>
            {picked ? (
              <>
                ponto <span className="ui-mono">{picked.x},{picked.y}</span>
                {pickedInfo && pickedInfo.regions.length > 0
                  ? ` · ${pickedInfo.regions.map((r) => r.name).join(", ")}`
                  : " · fora de região nomeada"}
                {pickedInfo?.city
                  ? ` · cidade mais próxima: ${pickedInfo.city.name} (${Math.round(pickedInfo.cityDistance)} blocos)`
                  : ""}
                {pickedInfo?.spawn
                  ? ` · spawn ${pickedInfo.spawn.leader} a ${Math.round(pickedInfo.spawnDistance)} blocos`
                  : ""}
              </>
            ) : (
              "clique no mapa para ler o ponto"
            )}
          </span>
          <form className="ui-atlas__goto" onSubmit={goToCoordinate}>
            <input
              className="ui-input ui-atlas__goto-input"
              inputMode="numeric"
              aria-label="coordenada X (0 a 4096)"
              placeholder="x"
              value={gotoX}
              onChange={(event) => setGotoX(event.target.value)}
            />
            <input
              className="ui-input ui-atlas__goto-input"
              inputMode="numeric"
              aria-label="coordenada Y (0 a 4096)"
              placeholder="y"
              value={gotoY}
              onChange={(event) => setGotoY(event.target.value)}
            />
            <button type="submit" className="ui-btn ui-btn--ghost">
              ir
            </button>
          </form>
          {picked && (
            <button type="button" className="ui-btn ui-btn--ghost" onClick={() => copy(picked.x, picked.y)}>
              copiar
            </button>
          )}
        </div>
      </div>

      <p className="ui-field__hint" style={{ margin: "8px 0 16px" }}>
        {layers.spawns
          ? `spawns: mostrando ${Math.min(spawnsShown.length, SPAWN_LIMIT)} de ${atlas.spawns.length} (os demais estão na tabela)`
          : "ligue “spawns” para os pontos crus, ou “calor” para a densidade"}
        {atlas.spawn_heat
          ? ` · calor: pico de ${atlas.spawn_heat.max} geradores numa célula de ${Math.round(WORLD / atlas.spawn_heat.cols)}² blocos`
          : ""}
        {" · "}
        exportar: <a href="/api/atlas/export?layer=regions&format=csv">regiões.csv</a>{" · "}
        <a href="/api/atlas/export?layer=teleports&format=csv">teleportes.csv</a>{" · "}
        <a href="/api/atlas/export?layer=spawns&format=csv">spawns.csv</a>{" · "}
        <a href="/api/atlas/export?layer=all&format=json">tudo.json</a>
      </p>

      <h2 className="ui-page-title" style={{ fontSize: 15 }}>
        Cidades ({atlas.cities.length})
      </h2>
      <DataTable rows={atlas.cities} columns={cityColumns} getRowKey={(c) => c.index} pageSizes={[10]} initialPageSize={10} />

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Regiões nomeadas ({atlas.regions.length})
      </h2>
      <DataTable
        rows={atlas.regions}
        columns={regionColumns}
        getRowKey={(r) => `${r.name}-${r.x1}-${r.y1}`}
        searchText={(r) => r.name}
        searchPlaceholder="Buscar região…"
        pageSizes={[25, 50, 100]}
        initialPageSize={25}
      />

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Rotas de teleporte ({atlas.teleports.length})
      </h2>
      <DataTable rows={atlas.teleports} columns={teleportColumns} getRowKey={(t) => `${t.src_x}-${t.src_y}`} pageSizes={[25, 50]} initialPageSize={25} />

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Geradores de spawn ({atlas.spawns.length})
      </h2>
      <DataTable
        rows={atlas.spawns}
        columns={spawnColumns}
        getRowKey={(s) => s.generator}
        searchText={(s) => `${s.generator} ${s.leader} ${s.follower}`}
        searchPlaceholder="Buscar por template ou gerador…"
        pageSizes={[50, 100, 200]}
        initialPageSize={50}
      />

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Itens fixos / portões ({atlas.static_items.length})
      </h2>
      <DataTable
        rows={atlas.static_items}
        columns={itemColumns}
        getRowKey={(i) => i.order}
        searchText={(i) => `${i.order} ${i.item_index} ${itemNames[String(i.item_index)] ?? ""}`}
        searchPlaceholder="Buscar por item…"
        pageSizes={[50]}
        initialPageSize={50}
      />
    </>
  );
}
