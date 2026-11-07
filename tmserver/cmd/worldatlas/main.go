// O comando worldatlas publica o layout do mundo que o servidor realmente usa: a
// tabela fixa de cidades e as rotas de teleporte (de tmserver/internal/world), as
// regiões nomeadas da tabela `region`, os objetos estáticos de world_item_seed, os
// geradores de spawn de npc_definition e um PNG reduzido do AttributeMap.dat.
//
// Mora em tmserver/cmd (não em webserver/cmd) porque lê
// tmserver/internal/content e tmserver/internal/world, que o Go só permite
// dentro da árvore do tmserver. Ele lê conteúdo e nunca escreve em tmserver/content/.
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

// bounds é um retângulo no espaço do mundo (0..4096 blocos).
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
	// Fit é a caixa justa em volta das células caminháveis dentro do retângulo do
	// Regions.txt. Nil quando a região não cobre nada por onde o AttributeMap deixa um
	// mob andar; a UI então cai no retângulo cru.
	Fit *bounds `json:"fit,omitempty"`
	// Color tinge a região em `region-map.png` e na amostra da tabela.
	Color  string `json:"color,omitempty"`
	Source string `json:"source"`
}

// regionRGB tinge uma região percorrendo o círculo de matiz com o ângulo áureo — 52
// regiões nomeadas precisam de 52 matizes distinguíveis, e uma paleta fixa de 12
// cores daria a mesma tonalidade a quatro regiões. Saturação/luminosidade ficam
// moderadas para o terreno continuar legível através da máscara.
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

// heat é uma grade grosseira de densidade dos geradores de spawn. O mapa lê "onde
// estão os mobs" dela em vez de desenhar 6k pontos, que em 1× é uma
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
	// Bounds é a caixa justa em volta de tudo que é caminhável: o mapa abre emoldurado
	// nela em vez de mostrar 4096² de vazio em volta de meia dúzia de ilhas.
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
	// Visão geral: o que a página carrega primeiro (pequeno, mundo inteiro).
	if err := renderTerrain(heightPath, mapPath, filepath.Join(*outDir, "world-map.png"), content.AttributeMapDim); err != nil {
		log.Fatalf("worldatlas: map overview: %v", err)
	}
	// Detalhe: 4096² nativo, pedido só quando o moderador dá zoom. Gravado
	// apenas se ficar num tamanho razoável para um asset local.
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

// loadRegionsFromDB lê as regiões nomeadas da tabela `region` — a que
// a issue 59 criou e o `importcontent` preenche a partir do Regions.txt. Ordenar
// por nome mantém o JSON e a paleta estáveis entre execuções; o parser de arquivo
// abaixo fica para o teste de paridade que prova que os dois lados concordam.
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

// regionsFromEntries mapeia linhas do banco para o payload do atlas. Puro para o
// teste de paridade exercitar o mapeamento sem conexão.
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

// loadRegionsFile interpreta o Regions.txt. O atlas não o lê mais em runtime;
// isto existe para o teste de paridade comparar o arquivo que o importador consumiu
// com as linhas que o atlas lê.
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

// renderTerrain desenha o mundo a partir das grades que o próprio servidor usa:
// HeightMap.dat (4096², um byte por bloco — a mesma grade que o route.Bake percorre)
// para o relevo, e AttributeMap.dat (1024²) para escurecer o que o servidor trata
// blocked.
//
// A arte de mapa do cliente (*.til / pacotes de textura) não faz parte deste
// repositório e não é redistribuível, então nada aqui deriva dela: a imagem vem de
// conteúdo que o servidor precisa carregar de qualquer forma. Isso também a mantém
// fiel — mexa no AttributeMap e os pixels sombreados andam junto com a colisão.
func renderTerrain(heightPath, attrPath, out string, size int) error {
	heights, err := content.LoadHeightMap(heightPath)
	if err != nil {
		return fmt.Errorf("height map: %w", err)
	}
	attrs, err := content.LoadGrid(attrPath, content.AttributeMapDim)
	if err != nil {
		return fmt.Errorf("attribute map: %w", err)
	}

	// size == AttributeMapDim dá a visão geral 1024² (4×4 blocos por pixel);
	// size == HeightMapDim dá o detalhe nativo 4096², um pixel por bloco —
	// é isso que a UI troca quando o moderador passa do zoom 2×.
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
			// Sombreado de relevo vindo do noroeste, a convenção dos mapas de relevo: a
			// inclinação decide o brilho, para cristas e vales lerem como volume
			// em vez de uma silhueta chapada.
			slope := float64(int(relief[max(y-1, 0)*size+max(x-1, 0)]) - int(relief[min(y+1, size-1)*size+min(x+1, size-1)]))
			light := 1 + slope/90
			if light < 0.72 {
				light = 0.72
			}
			if light > 1.32 {
				light = 1.32
			}
			if attrAt(x, y) >= 120 {
				// O bloqueado lê como estrutura, não como buraco: um tom de reboco claro e frio
				// que nunca se confunde com o chão caminhável mais escuro por baixo.
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

// terrainColor é a rampa hipsométrica: vazio, planície, campo, colina, montanha,
// pico — dessaturada de propósito para o overlay dourado continuar sendo o destaque da UI.
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

// renderRegions pinta cada região nomeada nas células que são caminháveis E
// têm relevo. O resultado é uma máscara raster, não os retângulos soltos do
// Regions.txt: o contorno segue o chão exatamente, o que a sobreposição de
// retângulos nunca conseguiria. Células fora de toda região ficam transparentes.
//
// O raster foi escolhido em vez de polilinhas traçadas de propósito: marching
// squares somaria algumas centenas de linhas de geometria chata para uma imagem
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

// spawnHeat agrupa os geradores numa grade cols×cols sobre o mundo. Cada
// célula de 64² blocos é um quadrado que a UI pinta com um retângulo translúcido.
func spawnHeat(spawns []spawn, cols int) heat {
	const world = 4096
	cell := world / cols
	data := make([]int, cols*cols)
	maxVal := 0
	for _, s := range spawns {
		if s.X < 0 || s.Y < 0 || s.X >= world || s.Y >= world {
			continue
		}
		index := (s.Y/cell)*cols + s.X/cell
		data[index]++
		if data[index] > maxVal {
			maxVal = data[index]
		}
	}
	return heat{Cols: cols, Rows: cols, Max: maxVal, Data: data}
}

// playableBounds devolve a caixa justa do mundo em volta do chão real,
// opcionalmente recortada por um retângulo. Uma célula só conta quando o
// AttributeMap deixa o mob passar E o HeightMap tem relevo ali (altura > 0).
//
// A checagem de altura não é cosmética: fora das ilhas o AttributeMap lê 0
// — "livre" — então uma caixa só por atributo abraça os 4096² inteiros e o mapa abre
// mostrando o mesmo vazio que deveria ter cortado.
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

// hashSources gera a impressão digital de toda entrada do atlas: os arquivos que
// continuam em disco mais o payload do banco já serializado.
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

// loadStaticItems lê os objetos estáticos do mundo (portões/portas) de
// world_item_seed — a tabela que substituiu o InitItem.csv (issue 58).
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

// loadSpawns lê os geradores de spawn de npc_definition — a tabela que
// substituiu o NPCGener.txt (issues 57/58). StartRange não é modelado na tabela
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
