// Command worldatlas publishes the world layout the server actually uses: the
// fixed city table and teleport routes (from tmserver/internal/world), the named
// regions of the `region` table, the static objects of world_item_seed, the NPC
// spawn generators of npc_definition and a downsampled PNG of AttributeMap.dat.
//
// It lives under tmserver/cmd (not webserver/cmd) because it reads
// tmserver/internal/content and tmserver/internal/world, which Go only allows
// inside the tmserver tree. It reads content, never writes to tmserver/content/.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/world"
)

// bounds is a world-space rectangle (0..4096 blocks).
type bounds struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

type region struct {
	Name string `json:"name"`
	X1   int    `json:"x1"`
	Y1   int    `json:"y1"`
	X2   int    `json:"x2"`
	Y2   int    `json:"y2"`
	// Fit is the tight box around the walkable cells inside the Regions.txt
	// rectangle. Nil when the region covers nothing the AttributeMap lets a mob
	// walk on; the UI then falls back to the raw rectangle.
	Fit *bounds `json:"fit,omitempty"`
	// Color tints the region in `region-map.png` and in the table swatch.
	Color  string `json:"color,omitempty"`
	Source string `json:"source"`
}

// regionRGB tints a region by walking the hue wheel with the golden angle — 52
// named regions need 52 distinguishable hues, and a fixed 12-colour palette
// would give four regions the same tint. Saturation/lightness stay moderate so
// the terrain still reads through the mask.
func regionRGB(index int) (uint8, uint8, uint8) {
	hue := math.Mod(float64(index)*137.508, 360)
	return hslToRGB(hue, 0.55, 0.62)
}

func regionColor(index int) string {
	r, g, b := regionRGB(index)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

type staticItem struct {
	Order  int `json:"order"`
	Item   int `json:"item_index"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Rotate int `json:"rotate"`
}

type spawn struct {
	Generator int    `json:"generator"`
	Leader    string `json:"leader"`
	Follower  string `json:"follower"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Range     int    `json:"range"`
	MaxMob    int    `json:"max_mob"`
	RouteType int    `json:"route_type"`
}

// heat is a coarse density grid of the spawn generators. The map reads "where
// the mobs are" from it instead of drawing 6k points, which at 1× is an
// unreadable dust of dots.
type heat struct {
	Cols int   `json:"cols"`
	Rows int   `json:"rows"`
	Max  int   `json:"max"`
	Data []int `json:"data"` // row-major, Cols*Rows
}

type atlas struct {
	Version     string                `json:"version"`
	GeneratedAt string                `json:"generated_at"`
	SourceHash  string                `json:"source_hash"`
	Cities      []world.AtlasCity     `json:"cities"`
	Teleports   []world.AtlasTeleport `json:"teleports"`
	Regions     []region              `json:"regions"`
	StaticItems []staticItem          `json:"static_items"`
	Spawns      []spawn               `json:"spawns"`
	SpawnHeat   heat                  `json:"spawn_heat"`
	// Bounds is the tight box around everything walkable: the map opens framed on
	// it instead of showing 4096² of void around a handful of islands.
	Bounds *bounds        `json:"bounds,omitempty"`
	Counts map[string]int `json:"counts"`
}

var regionLine = regexp.MustCompile(`^\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)\s*=\s*(.+?)\s*$`)

func main() {
	contentDir := flag.String("content", "tmserver/content", "content tree (tmserver/content/): regiões e mapas")
	outDir := flag.String("out", "dist", "output directory")
	dsn := flag.String("dsn", os.Getenv("W2PP_DB_DSN"), "PostgreSQL DSN: itens estáticos e spawns vêm do banco")
	flag.Parse()

	cities := world.AtlasCities()
	teleports := world.AtlasTeleports()

	// Itens estáticos, spawns e regiões vêm do banco (issues 58 e 59): InitItem.csv,
	// NPCGener.txt e Regions.txt deixaram de ser lidos em runtime, e o atlas não
	// pode ficar preso a eles.
	if *dsn == "" {
		log.Fatalf("worldatlas: -dsn (or W2PP_DB_DSN) is required: static items and spawns come from the database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("worldatlas: connect: %v", err)
	}
	defer pool.Close()
	staticItems, err := loadStaticItems(ctx, pool)
	if err != nil {
		log.Fatalf("worldatlas: static items: %v", err)
	}
	spawns, err := loadSpawns(ctx, pool)
	if err != nil {
		log.Fatalf("worldatlas: spawns: %v", err)
	}
	regions, err := loadRegionsFromDB(ctx, pool)
	if err != nil {
		log.Fatalf("worldatlas: regions: %v", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("worldatlas: mkdir: %v", err)
	}

	mapPath := filepath.Join(*contentDir, "TMsrv", "run", "AttributeMap.dat")
	heightPath := filepath.Join(*contentDir, "TMsrv", "run", "HeightMap.dat")

	attrs, err := content.LoadGrid(mapPath, content.AttributeMapDim)
	if err != nil {
		log.Fatalf("worldatlas: attribute map: %v", err)
	}
	heights, err := content.LoadHeightMap(heightPath)
	if err != nil {
		log.Fatalf("worldatlas: height map: %v", err)
	}
	playable := playableBounds(heights, attrs, nil)
	for i := range regions {
		regions[i].Fit = playableBounds(heights, attrs, &bounds{X1: regions[i].X1, Y1: regions[i].Y1, X2: regions[i].X2, Y2: regions[i].Y2})
		regions[i].Color = regionColor(i)
	}
	if err := renderRegions(heights, attrs, regions, filepath.Join(*outDir, "region-map.png")); err != nil {
		log.Fatalf("worldatlas: region map: %v", err)
	}
	// Overview: what the page loads first (small, whole world).
	if err := renderTerrain(heightPath, mapPath, filepath.Join(*outDir, "world-map.png"), content.AttributeMapDim); err != nil {
		log.Fatalf("worldatlas: map overview: %v", err)
	}
	// Detail: native 4096², requested only when the moderator zooms in. Written
	// only if it stays under a sane size for a local asset.
	detailPath := filepath.Join(*outDir, "world-map-hd.png")
	if err := renderTerrain(heightPath, mapPath, detailPath, content.HeightMapDim); err != nil {
		log.Fatalf("worldatlas: map detail: %v", err)
	}
	if st, err := os.Stat(detailPath); err == nil {
		fmt.Printf("worldatlas: map overview=%d bytes detail=%d bytes\n", fileSize(filepath.Join(*outDir, "world-map.png")), st.Size())
	}

	payload := atlas{
		Version:     "1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Cities:      cities,
		Teleports:   teleports,
		Regions:     regions,
		StaticItems: staticItems,
		Spawns:      spawns,
		SpawnHeat:   spawnHeat(spawns, 64),
		Bounds:      playable,
		Counts: map[string]int{
			"cities":       len(cities),
			"teleports":    len(teleports),
			"regions":      len(regions),
			"static_items": len(staticItems),
			"spawns":       len(spawns),
		},
	}
	// O hash cobre arquivos E o payload do banco: mudar um item em
	// world_item_seed/npc_definition também invalida o cache do portal.
	dbPayload, err := json.Marshal(struct {
		Items   []staticItem `json:"items"`
		Spawns  []spawn      `json:"spawns"`
		Regions []region     `json:"regions"`
	}{staticItems, spawns, regions})
	if err != nil {
		log.Fatalf("worldatlas: marshal sources: %v", err)
	}
	payload.SourceHash = hashSources([]string{mapPath, heightPath}, dbPayload)

	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("worldatlas: marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "world-atlas.json"), raw, 0o644); err != nil {
		log.Fatalf("worldatlas: write: %v", err)
	}

	fmt.Printf("worldatlas: cities=%d teleports=%d regions=%d static_items=%d spawns=%d hash=%s\n",
		len(cities), len(teleports), len(regions), len(staticItems), len(spawns), payload.SourceHash)
}

// loadRegionsFromDB reads the named regions from the `region` table — the one
// issue 59 created and `importcontent` fills from Regions.txt. Ordering by name
// keeps the JSON and the palette stable between runs; the file parser below is
// kept for the parity test that proves both sides agree.
func loadRegionsFromDB(ctx context.Context, pool *pgxpool.Pool) ([]region, error) {
	rows, err := pool.Query(ctx, `SELECT name, x1, y1, x2, y2 FROM region ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.RegionEntry
	for rows.Next() {
		var e domain.RegionEntry
		if err := rows.Scan(&e.Name, &e.X1, &e.Y1, &e.X2, &e.Y2); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("region vazia: rode o importcontent (issue 59) antes de gerar o atlas")
	}
	return regionsFromEntries(entries), nil
}

// regionsFromEntries maps database rows to the atlas payload. Pure so the parity
// test can exercise the mapping without a connection.
func regionsFromEntries(entries []domain.RegionEntry) []region {
	out := make([]region, 0, len(entries))
	for _, e := range entries {
		out = append(out, region{
			Name: e.Name, X1: int(e.X1), Y1: int(e.Y1), X2: int(e.X2), Y2: int(e.Y2),
			Source: "region",
		})
	}
	return out
}

// loadRegionsFile parses Regions.txt. The atlas no longer reads it in runtime;
// this exists so the parity test can compare the file the importer consumed with
// the rows the atlas reads.
func loadRegionsFile(path string) ([]region, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []region
	for _, line := range strings.Split(string(raw), "\n") {
		text := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if text == "" || strings.HasPrefix(text, "//") {
			continue
		}
		m := regionLine.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		vals := make([]int, 4)
		for i := 0; i < 4; i++ {
			vals[i], _ = strconv.Atoi(m[i+1])
		}
		out = append(out, region{Name: m[5], X1: vals[0], Y1: vals[1], X2: vals[2], Y2: vals[3], Source: "Regions.txt"})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no region parsed")
	}
	return out, nil
}

// renderTerrain draws the world from the grids the server itself uses:
// HeightMap.dat (4096², one byte per block — the same grid route.Bake walks) for
// the relief, and AttributeMap.dat (1024²) to darken what the server treats as
// blocked.
//
// The client's map art (*.til / texture packs) is not part of this repository and
// is not redistributable, so nothing here derives from it: the image comes from
// content the server has to load anyway. That also keeps it truthful — change the
// AttributeMap and the shaded pixels move with collision.
func renderTerrain(heightPath, attrPath, out string, size int) error {
	heights, err := content.LoadHeightMap(heightPath)
	if err != nil {
		return fmt.Errorf("height map: %w", err)
	}
	attrs, err := content.LoadGrid(attrPath, content.AttributeMapDim)
	if err != nil {
		return fmt.Errorf("attribute map: %w", err)
	}

	// size == AttributeMapDim gives the 1024² overview (4×4 blocks per pixel);
	// size == HeightMapDim gives the native 4096² detail, one pixel per block —
	// that is what the UI swaps in when the moderator zooms past 2×.
	step := content.HeightMapDim / size
	if step < 1 {
		step = 1
	}
	attrAt := func(x, y int) int {
		ax := x * attrs.Dim / size
		ay := y * attrs.Dim / size
		if ax >= attrs.Dim {
			ax = attrs.Dim - 1
		}
		if ay >= attrs.Dim {
			ay = attrs.Dim - 1
		}
		return int(attrs.Data[ay*attrs.Dim+ax])
	}
	relief := make([]uint8, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sum := 0
			for dy := 0; dy < step; dy++ {
				for dx := 0; dx < step; dx++ {
					sum += int(heights.Data[(y*step+dy)*content.HeightMapDim+(x*step+dx)])
				}
			}
			relief[y*size+x] = uint8(sum / (step * step))
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r, g, b := terrainColor(int(relief[y*size+x]))
			// Hillshade from the north-west, the convention relief maps use: the
			// slope decides brightness, so ridges and valleys read as volume
			// instead of a flat silhouette.
			slope := float64(int(relief[max(y-1, 0)*size+max(x-1, 0)]) - int(relief[min(y+1, size-1)*size+min(x+1, size-1)]))
			light := 1 + slope/90
			if light < 0.72 {
				light = 0.72
			}
			if light > 1.32 {
				light = 1.32
			}
			if attrAt(x, y) >= 120 {
				// Blocked reads as structure, not as a hole: a cool, light plaster tone
				// that is never mistaken for the darker walkable ground underneath.
				r, g, b = r*0.45+46, g*0.45+50, b*0.45+58
				light = 1
			}
			img.Set(x, y, color.RGBA{
				R: clampByte(r * light),
				G: clampByte(g * light),
				B: clampByte(b * light),
				A: 255,
			})
		}
	}

	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

// terrainColor is the hypsometric ramp: void, lowland, field, hill, mountain,
// peak — desaturated on purpose so the gold overlay stays the UI accent.
func terrainColor(h int) (float64, float64, float64) {
	stops := [...]struct {
		h       int
		r, g, b float64
	}{
		{0, 12, 16, 21},
		{14, 24, 36, 36},
		{70, 38, 58, 47},
		{118, 52, 74, 55},
		{162, 88, 94, 65},
		{208, 120, 114, 89},
		{255, 160, 152, 130},
	}
	if h <= stops[0].h {
		return stops[0].r, stops[0].g, stops[0].b
	}
	for i := 1; i < len(stops); i++ {
		if h <= stops[i].h {
			prev, cur := stops[i-1], stops[i]
			t := float64(h-prev.h) / float64(cur.h-prev.h)
			return prev.r + (cur.r-prev.r)*t, prev.g + (cur.g-prev.g)*t, prev.b + (cur.b-prev.b)*t
		}
	}
	last := stops[len(stops)-1]
	return last.r, last.g, last.b
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// renderRegions paints every named region onto the cells that are walkable AND
// have relief. The result is a raster mask, not the loose rectangles of
// Regions.txt: the outline follows the ground exactly, which is what the
// rectangle overlay could never do. Cells outside every region stay transparent.
//
// A raster was chosen over traced polylines on purpose: marching squares would
// add a few hundred lines of fiddly geometry for a picture the browser can just
// composite.
func renderRegions(heights, attrs *content.Grid, regions []region, out string) error {
	size := attrs.Dim
	cell := heights.Dim / attrs.Dim
	if cell < 1 {
		cell = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for cy := 0; cy < size; cy++ {
		for cx := 0; cx < size; cx++ {
			if attrs.Data[cy*size+cx] >= 120 {
				continue
			}
			relief := 0
			for dy := 0; dy < cell; dy++ {
				row := (cy*cell + dy) * heights.Dim
				for dx := 0; dx < cell; dx++ {
					if h := int(heights.Data[row+cx*cell+dx]); h > relief {
						relief = h
					}
				}
			}
			if relief == 0 {
				continue
			}
			wx, wy := cx*cell, cy*cell
			for i := range regions {
				box := regions[i].Fit
				if box == nil {
					box = &bounds{X1: regions[i].X1, Y1: regions[i].Y1, X2: regions[i].X2, Y2: regions[i].Y2}
				}
				if wx < box.X1 || wy < box.Y1 || wx >= box.X2 || wy >= box.Y2 {
					continue
				}
				r, g, b := regionRGB(i)
				img.Set(cx, cy, color.RGBA{R: r, G: g, B: b, A: 96})
				break
			}
		}
	}

	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

// spawnHeat buckets the generators into a cols×cols grid over the world. Each
// cell of 64² blocks is a square the UI can paint with one translucent rect.
func spawnHeat(spawns []spawn, cols int) heat {
	const world = 4096
	cell := world / cols
	data := make([]int, cols*cols)
	max := 0
	for _, s := range spawns {
		if s.X < 0 || s.Y < 0 || s.X >= world || s.Y >= world {
			continue
		}
		index := (s.Y/cell)*cols + s.X/cell
		data[index]++
		if data[index] > max {
			max = data[index]
		}
	}
	return heat{Cols: cols, Rows: cols, Max: max, Data: data}
}

// playableBounds returns the tight world box around the world's real ground,
// optionally clipped to a rectangle. A cell counts only when the AttributeMap
// lets a mob through AND the HeightMap has relief there (height > 0).
//
// The height check is not cosmetic: outside the islands the AttributeMap reads 0
// — "free" — so an attribute-only box wraps the whole 4096² and the map opens
// showing the same void it was supposed to crop away.
func playableBounds(heights, attrs *content.Grid, clip *bounds) *bounds {
	cell := heights.Dim / attrs.Dim
	if cell < 1 {
		cell = 1
	}
	out := &bounds{X1: -1, Y1: -1, X2: -1, Y2: -1}
	for cy := 0; cy < attrs.Dim; cy++ {
		for cx := 0; cx < attrs.Dim; cx++ {
			if attrs.Data[cy*attrs.Dim+cx] >= 120 {
				continue
			}
			wx, wy := cx*cell, cy*cell
			if clip != nil && (wx < clip.X1 || wy < clip.Y1 || wx >= clip.X2 || wy >= clip.Y2) {
				continue
			}
			relief := 0
			for dy := 0; dy < cell; dy++ {
				row := (cy*cell + dy) * heights.Dim
				for dx := 0; dx < cell; dx++ {
					if h := int(heights.Data[row+cx*cell+dx]); h > relief {
						relief = h
					}
				}
			}
			if relief == 0 {
				continue
			}
			if out.X1 < 0 || wx < out.X1 {
				out.X1 = wx
			}
			if out.Y1 < 0 || wy < out.Y1 {
				out.Y1 = wy
			}
			if wx+cell > out.X2 {
				out.X2 = wx + cell
			}
			if wy+cell > out.Y2 {
				out.Y2 = wy + cell
			}
		}
	}
	if out.X1 < 0 {
		return nil
	}
	return out
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

// hashSources fingerprints every atlas input: the files that stay on disk plus
// the database payload already serialized.
func hashSources(paths []string, dbPayload []byte) string {
	h := sha256.New()
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		h.Write(raw)
	}
	h.Write(dbPayload)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// loadStaticItems reads the static world objects (gates/doors) from
// world_item_seed — the table that replaced InitItem.csv (issue 58).
func loadStaticItems(ctx context.Context, pool *pgxpool.Pool) ([]staticItem, error) {
	rows, err := pool.Query(ctx,
		`SELECT item_index, pos_x, pos_y, rotate FROM world_item_seed ORDER BY item_index, pos_x, pos_y`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []staticItem
	for rows.Next() {
		var item, x, y, rotate int16
		if err := rows.Scan(&item, &x, &y, &rotate); err != nil {
			return nil, err
		}
		out = append(out, staticItem{Order: len(out) + 1, Item: int(item), X: int(x), Y: int(y), Rotate: int(rotate)})
	}
	return out, rows.Err()
}

// loadSpawns reads the spawn generators from npc_definition — the table that
// replaced NPCGener.txt (issues 57/58). StartRange is not modelled in the table
// yet, so Range stays 0.
func loadSpawns(ctx context.Context, pool *pgxpool.Pool) ([]spawn, error) {
	rows, err := pool.Query(ctx, `
		SELECT coalesce(nullif(display_name, ''), template_name), coalesce(follower_template, ''),
		       pos_x, pos_y, max_num_mob, route_type, coalesce(generator_index, 0)
		FROM npc_definition
		WHERE origin = 'content'
		ORDER BY generator_index NULLS LAST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []spawn
	for rows.Next() {
		var leader, follower string
		var x, y, route int16
		var maxMob, generator int32
		if err := rows.Scan(&leader, &follower, &x, &y, &maxMob, &route, &generator); err != nil {
			return nil, err
		}
		out = append(out, spawn{
			Generator: int(generator), Leader: leader, Follower: follower,
			X: int(x), Y: int(y), MaxMob: int(maxMob), RouteType: int(route),
		})
	}
	return out, rows.Err()
}
