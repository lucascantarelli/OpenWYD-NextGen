// Command tmserver is the WYD game server (TMSrv): it speaks the legacy CPSock
// wire protocol to the unmodified WYD.exe 7662 client (tmserver/internal/protocol)
// and owns the in-memory world state through a single game-loop goroutine
// (tmserver/internal/world).
//
// This entrypoint only does wiring (guidelines §3): flags, logging, the gRPC
// client connections to dbServer/binServer, the listener and graceful shutdown.
// Without -dbserver the persistence falls back to a no-op (local bring-up).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/internal/ratetables"
	"openwyd-nextgen/internal/secure"
	"openwyd-nextgen/tmserver/internal/binclient"
	"openwyd-nextgen/tmserver/internal/combine"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/dbclient"
	"openwyd-nextgen/tmserver/internal/dropcfg"
	"openwyd-nextgen/tmserver/internal/handler"
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/loot"
	"openwyd-nextgen/tmserver/internal/mobcarry"
	"openwyd-nextgen/tmserver/internal/mobstat"
	"openwyd-nextgen/tmserver/internal/npccfg"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/route"
	"openwyd-nextgen/tmserver/internal/world"
	"openwyd-nextgen/tmserver/internal/worldcfg"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("tmserver stopped with error", "err", err)
		os.Exit(1)
	}
	logger.Info("tmserver stopped")
}

// envInt reads an integer flag default from the environment so container
// deployments (Railway) can set it as a variable like the other W2PP_* knobs;
// a missing or malformed value falls back to def.
func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// envBool reads a boolean flag default from the environment (Railway-style knob),
// accepting the usual truthy spellings; a missing or malformed value falls back to
// def.
func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// addrOrNone renders an empty flag value as "(none)" so the boot log reads
// clearly when an optional address (dbServer/binServer/content) is unset.
func addrOrNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

func run(logger *slog.Logger) error {
	addr := flag.String("addr", ":8281", "CPSock listen address for the client edge")
	dbAddr := flag.String("dbserver", os.Getenv("W2PP_DBSERVER"), "dbServer gRPC address (empty = no-op persistence)")
	binAddr := flag.String("binserver", os.Getenv("W2PP_BINSERVER"), "binServer gRPC address (empty = allow-all billing)")
	tlsCert := flag.String("tls-cert", os.Getenv("W2PP_TLS_CERT"), "client certificate (PEM) for internal mTLS")
	tlsKey := flag.String("tls-key", os.Getenv("W2PP_TLS_KEY"), "client private key (PEM)")
	tlsCA := flag.String("tls-ca", os.Getenv("W2PP_TLS_CA"), "CA (PEM) verifying dbServer/binServer")
	tlsServerName := flag.String("tls-server-name", os.Getenv("W2PP_TLS_SERVER_NAME"), "expected server name in internal certs")
	rejectChecksum := flag.Bool("reject-checksum", false, "drop connections on CPSock checksum mismatch (Fase 7; off by default)")
	maxMsgPerSec := flag.Float64("max-msg-per-sec", 200, "per-connection inbound message rate limit (0 = disabled)")
	msgBurst := flag.Int("msg-burst", 400, "per-connection message burst depth")
	// O conteúdo mora dentro da árvore do servidor (LEGACY-02): sem -content, o
	// boot lê tmserver/content/ a partir do diretório do repositório/da imagem.
	defContent := os.Getenv("W2PP_CONTENT")
	if defContent == "" {
		defContent = "tmserver/content"
	}
	contentDir := flag.String("content", defContent, "caminho da árvore de conteúdo (default tmserver/content; vazio = pula; valida taxas/mapas no boot)")
	teleportEditing := flag.Bool("teleport-editing", envBool("W2PP_TELEPORT_EDITING", false), "enable the moderator teleport-route overlay (issue 42); needs -dbserver. Applied at boot and re-read every 15s, so an edit reaches a running world without a restart. OFF by default: with it off the world resolves strictly from the compiled table")
	mobStatEditing := flag.Bool("mob-stat-editing", envBool("W2PP_MOB_STAT_EDITING", false), "enable the moderator mob/NPC template stat overlay (mob-template-editing-plan.md, the equivalent-tool successor to the legacy EDITAPPMOB); needs -dbserver and -content. Applied ONCE at boot, like every other content load — a moderator edit needs a tmServer restart to take effect (EDITAPPMOB itself required a server restart too), independent of the other overlays")
	itemEffectEditing := flag.Bool("item-effect-editing", envBool("W2PP_ITEM_EFFECT_EDITING", false), "enable the moderator item base-effect overlay (issue 44, the read-through side of the web editor); needs -dbserver (catalog and overlay source). Applied ONCE at boot: itemEffects is folded into the handler at construction, so a moderator edit needs a tmServer restart to reach a running world (same restart-to-apply as -mob-stat-editing). OFF by default: with it off the catalog is exactly item_catalog")
	ratesEditing := flag.Bool("rates-editing", envBool("W2PP_RATES_EDITING", false), "enable the moderator content-rate overlay (issue 36: SancRate/CompRate/QuestsRate); needs -dbserver (overlay source) and -content (rate files). Applied ONCE at boot, so an edit needs a tmServer restart. OFF by default: with it off the tables are exactly the files in tmserver/content/Common/Settings")
	dropConfigEditing := flag.Bool("drop-config-editing", envBool("W2PP_DROP_CONFIG_EDITING", false), "enable the moderator drop-rate overlay (issue 34: DropRate[64] divisor per Carry slot); needs -dbserver (overlay source). Applied ONCE at boot, so an edit needs a tmServer restart. OFF by default: with it off the table is exactly the code defaults in tmserver/internal/loot")
	mobCarryEditing := flag.Bool("mob-carry-editing", envBool("W2PP_MOB_CARRY_EDITING", false), "enable the moderator Carry[] overlay (issue 34: the drop table of each mob template); needs -dbserver (overlay source) and -content (mob templates). Applied ONCE at boot, so an edit needs a tmServer restart. OFF by default: with it off every template keeps exactly the Carry[] in tmserver/content/TMsrv/run/npc")
	defStatusAddr := os.Getenv("W2PP_STATUS_ADDR")
	if defStatusAddr == "" {
		defStatusAddr = ":80"
	}
	statusAddr := flag.String("status-addr", defStatusAddr, "HTTP channel-status listen address (serv00.htm); real WYD serves status on :80, separate from the game port. Empty disables")
	defMetricsAddr := os.Getenv("W2PP_METRICS_ADDR")
	if defMetricsAddr == "" {
		defMetricsAddr = ":9101"
	}
	metricsAddr := flag.String("metrics-addr", defMetricsAddr, "endpoint de métricas/health (Prometheus) em porta separada; vazio desliga")
	maxNightmare := flag.Int("max-nightmare", envInt("W2PP_MAX_NIGHTMARE", 3), "counted Nightmare admissions per map and cycle")
	kefraEntriesPerScroll := flag.Int("kefra-entries-per-scroll", envInt("W2PP_KEFRA_ENTRIES_PER_SCROLL", 100), "Hall of Kefra entries one sealed scroll grants at the Sobrevivente (legacy 100; servers commonly use 50 or 100)")
	clientVersion := flag.Int("client-version", envInt("W2PP_CLIENT_VERSION", 7640), "MSG_AccountLogin.ClientVersion the client must send (protocol-spec says 7640; this 7662 'Cavaleiros de Kersef' build sends 12000)")
	// Channel identity. The legacy indexes everything per "ServerIndex": the
	// citizen of a channel is ServerIndex+1 (_MSG_Quest.cpp:2436) and guild ids
	// carry it in their high bits (serverIndex*4096). One instance per channel, so
	// 0 is the default (channel 1) and each extra container gets its own value.
	serverIndex := flag.Int("server-index", envInt("W2PP_SERVER_INDEX", 0), "zero-based channel index this instance serves; citizenship is stamped as server-index+1")
	// Crossover key for the in-game channel switch (/srv): both channels and the
	// dbServer must share it (canais.md §8.1.2). Empty disables the command.
	crossoverKey := flag.String("crossover-key", os.Getenv("W2PP_CROSSOVER_KEY"), "chave HMAC compartilhada com o dbServer para o token de troca de canal (/srv); vazia desativa a troca in-game")
	doubleExp := flag.Bool("double-exp", envBool("W2PP_DOUBLE_EXP", false), "DOUBLEMODE: double PvE experience (gameconfig double)")
	newbieEvent := flag.Bool("newbie-event", envBool("W2PP_NEWBIE_EVENT", false), "NewbieEventServer: +15% exp and newbie under-100 bonus (gameconfig)")
	noviceChannel := flag.Int("novice-channel", envInt("W2PP_NOVICE_CHANNEL", 0), "número do canal verde (novato), 1-based e fixo como em produção: nele esta instância roda o evento newbie e recusa dano PvP. 0 = nenhum canal verde (o evento segue -newbie-event em todos os canais)")
	kefraLive := flag.Bool("kefra-live", envBool("W2PP_KEFRA_LIVE", false), "initial Kefra defeated state (true opens city); persisted cycle takes precedence; false halves PvE exp")
	// Issue #383: a fresh character must reach displayed level 35 in ~15 min on
	// the training grounds. The measured pace with the plain reward was ~1k exp/min
	// (×0.425 from Kefra/newbie included) against the ~11.7k needed; dropping
	// those reducers gives ~2.35×, and 6 closes the rest with headroom.
	trainingFieldExp := flag.Int("training-field-exp", envInt("W2PP_TRAINING_FIELD_EXP", 6), "EXP multiplier for training-grounds kills (AttributeMap 0x80 generators) by mortals below stored level 35; replaces the global event modifiers there. 0 = legacy reward")
	logSends := flag.Bool("log-sends", envBool("W2PP_LOG_SENDS", false), "log every S→C frame (conn/type/id/len) — client-freeze diagnostics (investigacao-freeze-cliente.md); high volume, enable only while reproducing an incident")
	// Cast-buff duration tuning (issue #229). The legacy formula is
	// (AffectTime+1)*(100+Special)/100 ticks of 8s, which puts an endgame character
	// at 30-100 min per buff — reported as far too long three times over (#92, #202,
	// #229). 100/0/0 restores the legacy durations exactly.
	affectScalePct := flag.Int("affect-scale-pct", envInt("W2PP_AFFECT_SCALE_PCT", 15), "percent of the legacy cast-affect duration to keep, applied only OUTSIDE the target band: a non-aggressive affect whose BASE (unmastered) duration is already under -affect-max-minutes keeps its legacy mastery curve (issue #236). 100 = legacy")
	affectMinSeconds := flag.Int("affect-min-seconds", envInt("W2PP_AFFECT_MIN_SECONDS", 60), "floor for NON-aggressive cast affects, in seconds; keeps the shortest buffs usable (0 = no floor)")
	affectMaxMinutes := flag.Int("affect-max-minutes", envInt("W2PP_AFFECT_MAX_MINUTES", 10), "cap for cast affects, in minutes; cuts the mastery tail (0 = no cap)")
	flag.Parse()

	// Echo the effective wiring at boot: the client-version and the resolved
	// dbServer/binServer addresses are the knobs most often misconfigured in a
	// container deploy (version-mismatch drops, or "produced zero addresses" when
	// an internal hostname is wrong), so surface them before anything connects.
	logger.Info("tmserver config",
		"client_version", *clientVersion,
		"dbserver", addrOrNone(*dbAddr),
		"binserver", addrOrNone(*binAddr),
		"content", addrOrNone(*contentDir),
		"mob_stat_editing", *mobStatEditing,
		"teleport_editing", *teleportEditing,
		"item_effect_editing", *itemEffectEditing,
		"rates_editing", *ratesEditing,
		"mob_carry_editing", *mobCarryEditing)

	// When -content is set, load and validate the content tree up front so a
	// missing/corrupt mount fails fast instead of surfacing mid-session. The
	// recipe→combine-family and AttributeMap-bit semantics remain UNVERIFIED
	// (PROGRESS Fase 5), so this validates and exposes the data; it does not
	// rewire gameplay on unverified mappings.
	var itemPrices map[int]int32
	var itemEffects map[int][]content.BaseEffect
	var itemReqs map[int]content.ItemReq
	var itemVolatiles, itemPos, itemUnique, itemGrades, itemExtra map[int]int
	var itemRanges map[int]int16
	var combineFamilies map[protocol.Type]handler.CombineFamily
	var odinCatalog combine.Catalog
	var spells *content.SkillData
	// Catálogos lidos do banco (issue 58, fase 2); nil quando o banco não tem
	// conteúdo importado e o boot cai nos arquivos de tmserver/content/.
	var catalogTemplates map[string][]byte
	var worldItems []content.InitItem
	var heights *content.Grid
	var attrs *content.Grid
	// Dust-refine ceilings (QA 05/11): moved out of the code into
	// Common/Settings/RefineRate.txt so a moderator decides how far each dust goes.
	// A missing/broken file is not fatal — the file is a tuning surface, and the
	// world keeps the legacy 6/9/10/11 instead of refusing to boot.
	refinePolicy := content.DefaultRefinePolicy()
	if *contentDir != "" {
		if p, perr := content.LoadRefinePolicy(*contentDir + "/Common/Settings/RefineRate.txt"); perr != nil {
			logger.Warn("refine policy: legacy defaults", "err", perr)
		} else {
			refinePolicy = p
		}
	}
	logger.Info("refine policy", "po_max", refinePolicy.OriMax, "pl_max", refinePolicy.PLMax,
		"gem_level", refinePolicy.GemLevel, "hard_cap", refinePolicy.HardCap,
		"server_index", *serverIndex)

	var sancRate *content.SancRate
	var compRate *content.CompRate
	var questRates *content.QuestRates
	// Taxas, mapas e a página de status continuam em arquivo (issues 59/60).
	// item_catalog, skill_catalog, npc_template e world_item_seed vêm do banco:
	// os CSV e o npc/ saíram do conteúdo na fase 5 da issue 58.
	if *contentDir != "" {
		c, err := loadContent(*contentDir, logger)
		if err != nil {
			return err
		}
		heights = c.heights
		attrs = c.attrs
		sancRate = c.sanc
		compRate = c.comp
		questRates = c.quests
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	metricsReg := metrics.New("tmserver")
	go metrics.Serve(ctx, *metricsAddr, metricsReg, logger)

	clientCreds, err := secure.ClientCreds(secure.Config{
		CertFile: *tlsCert, KeyFile: *tlsKey, CAFile: *tlsCA, ServerName: *tlsServerName,
	})
	if err != nil {
		return err
	}

	// Persistence: real dbServer adapter when -dbserver is set, else no-op. The
	// connection is retained (dbConn) so the NPC-config overlay can share it.
	var persist world.Persistence = world.NopPersistence{}
	var dbConn *grpc.ClientConn
	var worldEvents worldcfg.Source
	if *dbAddr != "" {
		// O interceptor cliente injeta o trace_id do contexto (sessão/login) no
		// metadata de toda RPC, então o dbServer registra o mesmo id (OBS-02).
		conn, err := grpc.NewClient(*dbAddr,
			grpc.WithTransportCredentials(clientCreds),
			grpc.WithChainUnaryInterceptor(metrics.UnaryClientInterceptor()))
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		dbConn = conn
		persist = dbclient.New(conn)
		worldEvents = dbclient.NewWorldEventConfig(conn)
		logger.Info("dbServer wired", "addr", *dbAddr)
	} else {
		logger.Warn("no -dbserver: using no-op persistence (logins report no account; Kefra cycle is not preserved across restarts)")
	}

	// Catálogos de conteúdo (issue 58): o banco é a ÚNICA fonte. ItemList.csv,
	// SkillData.csv, npc/* e InitItem.csv saíram do repositório na fase 5, então
	// não há mais fallback: banco sem catálogo recusa o boot. Um mundo sem itens
	// sobe "saudável" e só falha no primeiro combate — pior que um boot negado.
	if dbConn != nil {
		fetchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		cat, cerr := dbclient.NewContentCatalog(dbConn).Fetch(fetchCtx)
		cancel()
		if cerr != nil {
			return fmt.Errorf("load content catalogs from the database: %w", cerr)
		}
		if len(cat.Items) == 0 {
			return fmt.Errorf("item_catalog is empty: seed the content catalogue (tmserver/cmd/importcontent or seedgen)")
		}
		items := content.NewItemList(cat.Items)
		itemPrices, itemEffects, itemReqs = items.Prices(), items.BaseEffects(), items.Requirements()
		itemVolatiles, itemPos, itemUnique = items.Volatiles(), items.Positions(), items.Uniques()
		itemGrades, itemExtra, itemRanges = items.Grades(), items.Extras(), items.Ranges()
		// A combinação (Odin) depende das taxas de combine, que continuam em
		// arquivo: só é reconstruída quando o Content/ está montado.
		if compRate != nil {
			odinCatalog = handler.NewCombineCatalog(items, compRate)
			combineFamilies = handler.DefaultCombineFamilies(odinCatalog)
		}
		spells = content.NewSkillData(cat.Skills)
		catalogTemplates = cat.Templates
		worldItems = cat.WorldItems
		logger.Info("content catalogs loaded from the database",
			"items", items.Len(), "skills", spells.Len(),
			"templates", len(catalogTemplates), "world_items", len(worldItems))
	}

	// The client fetches a channel-status page over HTTP before the CPSock
	// connect; serve it from the content tree when available.
	var statusFile string
	var baseMobs map[int][]byte
	var summonMobs [][]byte
	var vineMob []byte
	var castleQuests []content.CastleQuest
	if *contentDir != "" {
		statusFile = filepath.Join(*contentDir, "Common", "serv00.htm")
		if bm, err := content.LoadBaseMobs(*contentDir); err != nil {
			logger.Warn("base mob templates not loaded", "err", err)
		} else {
			baseMobs = bm
			logger.Info("base mob templates loaded", "classes", len(baseMobs))
		}
		if sm, missing, err := content.LoadBaseSummons(*contentDir); err != nil {
			logger.Warn("base summon templates not loaded (BM evocations disabled)", "err", err)
		} else {
			summonMobs = sm
			if len(missing) > 0 {
				logger.Warn("optional summon templates missing", "names", missing)
			}
			logger.Info("base summon templates loaded", "count", len(summonMobs)-len(missing))
		}
		// Muro de Espinhos is CreateMob("Vinha", ..., "Boss", 3) in the C++,
		// but the shipped Linux content tree carries the wall template as npc/Vine.
		if vm, ok := catalogTemplates["Vine"]; ok {
			vineMob = vm
			logger.Info("vine template loaded", "source", "database")
		} else {
			logger.Warn("vine template not loaded (skill 98 cannot create Muro de Espinhos)",
				"reason", "absent from npc_template")
		}
		if cq, err := content.LoadCastleQuests(filepath.Join(*contentDir, "Common", "Settings", "CastleQuest.txt")); err != nil {
			logger.Warn("castle quests not loaded (Castle/Zakum disabled)", "err", err)
		} else {
			castleQuests = cq
			logger.Info("castle quests loaded", "count", len(castleQuests))
		}
	}

	// Moderator mob/NPC template stat overlay (mob-template-editing-plan.md, the
	// equivalent-tool successor to the legacy EDITAPPMOB): the single switch is
	// -mob-stat-editing (W2PP_MOB_STAT_EDITING), off by default. Unlike
	// The carry overlay applies to ANY npc/<template_name> file — monsters
	// included, the tool's primary rebalancing use case — not just the
	// DB-managed merchant subset, so it is deliberately independent of
	// the NPC catalogue (its own source, its own dependency check). Fetched ONCE here,
	// synchronously, before any template gets loaded below: there is no
	// hot-reload for this feature, matching EDITAPPMOB's own restart-to-apply
	// behavior (it wrote the file; the server only read it at boot too).
	var mobStatOverrides map[string]mobstat.Override
	if *mobStatEditing {
		if dbConn == nil {
			return fmt.Errorf("-mob-stat-editing requires -dbserver (config source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		overrides, ferr := dbclient.NewMobStatSource(dbConn).Fetch(fetchCtx)
		cancel()
		if ferr != nil {
			return fmt.Errorf("fetch mob template stat overrides: %w", ferr)
		}
		mobStatOverrides = overrides
		logger.Info("mob template stat overlay enabled (moderator editing)", "overrides", len(mobStatOverrides))
	}

	// Moderator Carry[] overlay (issue 34, fatia 3): the drop table of each mob
	// template. Same shape as the stat overlay — fetched once, before any template
	// is loaded, because the table is baked into the bytes the world spawns from
	// and there is no hot-reload. -mob-carry-editing (W2PP_MOB_CARRY_EDITING), off
	// by default so an unseeded DB never changes a single drop.
	var mobCarryOverrides map[string]mobcarry.Override
	if *mobCarryEditing {
		if dbConn == nil {
			return fmt.Errorf("-mob-carry-editing requires -dbserver (overlay source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		overrides, ferr := dbclient.NewMobCarrySource(dbConn).Fetch(fetchCtx)
		cancel()
		if ferr != nil {
			return fmt.Errorf("fetch mob carry overrides: %w", ferr)
		}
		mobCarryOverrides = overrides
		logger.Info("mob carry overlay enabled (moderator editing)", "templates", len(mobCarryOverrides))
	}

	// NPC spawn catalogue (issue 57): npc_definition in Postgres is the SINGLE
	// source of spawn blocks AND of the templates (npc_template), so -dbserver is
	// the only dependency — NPCGener.txt e npc/ saíram do conteúdo na fase 5.
	// The tmServer registers one generator slot per content block and
	// materializes the entities. Without -dbserver the world boots with no NPCs,
	// which the dev topology relies on — hence a loud warning, not a fatal error.
	var npcConfig npccfg.Source
	if dbConn == nil {
		logger.Warn("no NPC catalogue: -dbserver is required, the world will boot with no monsters",
			"dbserver", addrOrNone(*dbAddr))
	} else {
		npcConfig = dbclient.NewNpcConfig(dbConn, func(name string) ([]byte, error) {
			// Só o banco: o blob de 816 B vem de npc_template, garantido pelo
			// CHECK (length(blob) = 816) e normalizado no import.
			b, ok := catalogTemplates[name]
			if !ok {
				return nil, fmt.Errorf("npc template %q not in npc_template", name)
			}
			return mobcarry.ApplyOverride(mobstat.ApplyOverride(b, name, mobStatOverrides), name, mobCarryOverrides), nil
		}, logger)
	}

	// Moderator teleport overlay (issue 42): the single switch is
	// -teleport-editing (W2PP_TELEPORT_EDITING). Unlike the NPC overlay this one
	// does not spawn or materialize anything — route resolution already happens in
	// world.teleportDest, so the overlay is just a read-only snapshot swapped
	// atomically. A 15s poll keeps a running world in sync with the web.
	if *teleportEditing {
		if dbConn == nil {
			return fmt.Errorf("-teleport-editing requires -dbserver (config source)")
		}
		telCfg := dbclient.NewTeleportConfig(dbConn)
		applyTeleportOverlay := func(actx context.Context) error {
			routes, ferr := telCfg.Routes(actx)
			if ferr != nil {
				return ferr
			}
			rows := make([]world.OverlayRoute, 0, len(routes))
			disabled := 0
			for _, r := range routes {
				if !r.Enabled {
					disabled++
				}
				rows = append(rows, world.OverlayRoute{
					SrcX: r.SrcX, SrcY: r.SrcY, DstX: r.DstX, DstY: r.DstY,
					Cost: r.Cost, Requirement: r.Requirement, Enabled: r.Enabled,
				})
			}
			applied := world.SetTeleportOverlay(rows)
			logger.Info("teleport overlay applied", "rows", applied, "disabled", disabled)
			return nil
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := applyTeleportOverlay(fetchCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("fetch teleport overlay: %w", err)
		}
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
					if perr := applyTeleportOverlay(pctx); perr != nil {
						logger.Warn("teleport overlay poll failed", "err", perr)
					}
					pcancel()
				}
			}
		}()
		logger.Info("teleport config overlay enabled (moderator editing)", "poll", "15s")
	}

	// Seed the world-event RNG from the wall clock so the weather sequence differs
	// between boots (handler.worldEventRNGSeed explains why the fixed seed is only
	// for tests). Zero means "use the fixed seed", so keep it out of range.
	eventSeed := uint32(time.Now().UnixNano())
	if eventSeed == 0 {
		eventSeed = 1
	}
	// Moderator content-rate overlay (issue 36): the read-through side of
	// /config. Same shape as the item-effect overlay — one flag, applied once at
	// boot, because SancRate/CompRate/QuestRates are read-only snapshots from the
	// moment the world starts (an edit takes effect on the next restart).
	if *ratesEditing {
		if dbConn == nil || *contentDir == "" {
			return fmt.Errorf("-rates-editing requires both -dbserver (overlay source) and -content (rate files)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rows, ferr := dbclient.NewContentRateConfig(dbConn).Overrides(fetchCtx)
		cancel()
		if ferr != nil {
			return fmt.Errorf("fetch content rate overlay: %w", ferr)
		}
		overrides := make([]ratetables.Override, 0, len(rows))
		for _, r := range rows {
			overrides = append(overrides, ratetables.Override{
				File: r.File, Family: r.Family, Name: r.Name, Value: r.Value,
			})
		}
		applied, ignored := content.ApplyRateOverrides(sancRate, compRate, questRates, overrides)
		var refineApplied, refineIgnored int
		refinePolicy, refineApplied, refineIgnored = content.ApplyRefineOverrides(refinePolicy, overrides)
		logger.Info("refine policy overlay", "applied", refineApplied, "ignored", refineIgnored,
			"po_max", refinePolicy.OriMax, "pl_max", refinePolicy.PLMax,
			"gem_level", refinePolicy.GemLevel, "hard_cap", refinePolicy.HardCap)
		logger.Info("content rate overlay applied",
			"rate_overrides_applied", applied, "rate_overrides_ignored", ignored, "rows", len(rows))
		// "content loaded" runs before this block, so the effective Ori row is
		// logged here: the operator needs to see the number the world will use,
		// not only how many rows were fetched.
		if sancRate != nil {
			logger.Info("content rate overlay effective", "sancrate_ori", sancRow(sancRate, 0))
		}
	}

	// Moderator drop-rate overlay (issue 34): /drops/editar writes
	// drop_rate_override; the world reads DropRate[64] from the moment it starts,
	// so this is applied ONCE here. Values only — no rand() call is added, so RNG
	// parity is untouched.
	if *dropConfigEditing {
		if dbConn == nil {
			return fmt.Errorf("-drop-config-editing requires -dbserver (overlay source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rows, ferr := dbclient.NewDropConfig(dbConn).Overrides(fetchCtx)
		cancel()
		if ferr != nil {
			return fmt.Errorf("fetch drop rate overlay: %w", ferr)
		}
		overrides := make([]dropcfg.Override, 0, len(rows))
		for _, r := range rows {
			overrides = append(overrides, dropcfg.Override{Slot: r.Slot, Value: r.Value})
		}
		applied, ignored := dropcfg.Apply(overrides)
		logger.Info("drop rate overlay applied",
			"drop_overrides_applied", applied, "drop_overrides_ignored", ignored, "rows", len(rows),
			"slot_12", loot.DropRate[12])
	}

	// Moderator item base-effect overlay (issue 44): the read-through side of the
	// web editor at /itens/[index]. The single switch is -item-effect-editing
	// (W2PP_ITEM_EFFECT_EDITING), off by default, and — like -mob-stat-editing —
	// it is applied ONCE here, right before the handler is built: ItemEffects is an
	// immutable snapshot folded into the score, so a running world keeps what it
	// booted with. Precedence is unchanged: an item instance's own stEffect/stValue
	// still beats the catalog (here, content + overlay) at score time.
	if *itemEffectEditing {
		if dbConn == nil {
			return fmt.Errorf("-item-effect-editing requires -dbserver (catalog and overlay source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rows, ferr := dbclient.NewItemEffectConfig(dbConn).Effects(fetchCtx)
		cancel()
		if ferr != nil {
			return fmt.Errorf("fetch item effect overlay: %w", ferr)
		}
		overrides := make([]content.ItemEffectOverride, 0, len(rows))
		for _, r := range rows {
			overrides = append(overrides, content.ItemEffectOverride{
				ItemIndex: r.ItemIndex, Effect: r.Effect, Value: r.Value,
			})
		}
		overlaid, stats := content.ApplyItemEffectOverrides(itemEffects, overrides)
		itemEffects = overlaid
		logger.Info("item effect overlay applied",
			"rows", len(rows), "replaced", stats.Replaced, "added", stats.Added, "ignored", stats.Ignored)
	}

	// Process flags snapshot (issue 36, aba "Flags"): the portal shows what this
	// process booted with, read-only. These flags live in the compose command and
	// the W2PP_* environment; the web never edits them, so a restart is the only
	// way to change one. A failure here must not stop the server — the snapshot is
	// observability, not gameplay.
	if dbConn != nil {
		snapshot := []dbclient.InstanceFlag{
			{Name: "addr", Value: *addr, RestartRequired: true, Hint: "porta CPSock do cliente (docker-compose command)"},
			{Name: "status-addr", Value: *statusAddr, RestartRequired: true, Hint: "HTTP de status (serv00.htm) que o cliente sonda"},
			{Name: "client-version", Value: strconv.Itoa(*clientVersion), RestartRequired: true, Hint: "versão aceita no handshake; divergente = 0x102"},
			{Name: "dbserver", Value: *dbAddr, RestartRequired: true, Hint: "persistência; vazio = no-op"},
			{Name: "binserver", Value: *binAddr, RestartRequired: true, Hint: "billing; vazio = allow-all"},
			{Name: "content", Value: *contentDir, RestartRequired: true, Hint: "árvore de conteúdo (taxas, mapas, BaseMob/BaseSummon)"},
			{Name: "double-exp", Value: strconv.FormatBool(*doubleExp), RestartRequired: true, Hint: "DOUBLEMODE: dobra EXP PvE"},
			{Name: "newbie-event", Value: strconv.FormatBool(*newbieEvent), RestartRequired: true, Hint: "NewbieEventServer: +15% EXP e bônus abaixo do nível 100"},
			{Name: "kefra-live", Value: strconv.FormatBool(*kefraLive), RestartRequired: true, Hint: "evento do Kefra ativo"},
			{Name: "training-field-exp", Value: strconv.FormatInt(int64(*trainingFieldExp), 10), RestartRequired: true, Hint: "multiplicador de EXP do Campo de Treino"},
			{Name: "novice-channel", Value: strconv.Itoa(*noviceChannel), RestartRequired: true, Hint: "canal verde (novato) fixo, 1-based; 0 = nenhum"},
			{Name: "max-nightmare", Value: strconv.Itoa(*maxNightmare), RestartRequired: true, Hint: "admissões contadas de Nightmare por mapa/ciclo"},
			{Name: "mob-stat-editing", Value: strconv.FormatBool(*mobStatEditing), RestartRequired: true, Hint: "overlay de stats de mob"},
			{Name: "mob-carry-editing", Value: strconv.FormatBool(*mobCarryEditing), RestartRequired: true, Hint: "overlay do Carry[] (drop) dos templates"},
			{Name: "teleport-editing", Value: strconv.FormatBool(*teleportEditing), RestartRequired: true, Hint: "overlay de rotas de teleporte (relido a cada 15 s)"},
			{Name: "item-effect-editing", Value: strconv.FormatBool(*itemEffectEditing), RestartRequired: true, Hint: "overlay de efeitos base de item"},
			{Name: "rates-editing", Value: strconv.FormatBool(*ratesEditing), RestartRequired: true, Hint: "overlay de taxas (refino/combine/quests)"},
			{Name: "drop-config-editing", Value: strconv.FormatBool(*dropConfigEditing), RestartRequired: true, Hint: "overlay de taxa de drop por slot"},
		}
		flagCtx, cancelFlags := context.WithTimeout(ctx, 10*time.Second)
		err := dbclient.NewInstanceFlags(dbConn).Record(flagCtx, "tmserver", snapshot)
		cancelFlags()
		if err != nil {
			logger.Warn("record instance flags", "err", err)
		}
	}

	dispatch := handler.New(handler.Config{
		MaxNightmare:          *maxNightmare,
		KefraEntriesPerScroll: *kefraEntriesPerScroll,
		Log:                   logger, ClientVersion: int32(*clientVersion), BaseMobs: baseMobs, SummonMobs: summonMobs, VineMob: vineMob, ItemPrices: itemPrices, ItemEffects: itemEffects, ItemReqs: itemReqs,
		ItemVolatiles: itemVolatiles, ItemPos: itemPos, ItemUnique: itemUnique, ItemGrades: itemGrades, ItemExtra: itemExtra, Spells: spells, Heights: heights, Attrs: attrs,
		SancRate:         sancRate,
		RefinePolicy:     refinePolicy,
		ServerIndex:      *serverIndex,
		NoviceChannel:    *noviceChannel,
		CrossoverKey:     *crossoverKey,
		ExpEvents:        level.ExpEvents{DoubleMode: *doubleExp, NewbieEvent: *newbieEvent, KefraLive: *kefraLive},
		TrainingFieldExp: int64(*trainingFieldExp),
		CombineFamilies:  combineFamilies,
		OdinCatalog:      odinCatalog,
		CombineCatalog:   odinCatalog,
		CompRate:         compRate,
		QuestRates:       questRates,
		NpcConfig:        npcConfig,
		WorldEvents:      worldEvents,
		CastleQuests:     castleQuests,
		EventRNGSeed:     eventSeed,
		AffectDuration: world.AffectDuration{
			ScalePct: *affectScalePct,
			MinTicks: handler.AffectTicksFromSeconds(*affectMinSeconds),
			MaxTicks: handler.AffectTicksFromMinutes(*affectMaxMinutes),
		},
	})
	w := world.New(world.Config{
		RejectChecksum: *rejectChecksum,
		MaxMsgPerSec:   *maxMsgPerSec,
		MsgBurst:       *msgBurst,
		StatusFile:     statusFile,
		ItemRanges:     itemRanges,
		LogSends:       *logSends,
		Metrics:        metricsReg,
	}, logger, persist, dispatch.Handle)
	// Mob-AI pulse: monsters acquire/chase/melee nearby players each tick (mobai.go).
	w.SetTickHandler(world.DefaultMobTick, dispatch.Tick)
	// Party teardown on disconnect: unlink the party bond and reap the summons
	// before the slot is freed (party.go SessionEnd).
	w.SetSessionEndHandler(dispatch.SessionEnd)
	// The newbie flag has two owners: the dispatcher's ExpEvents (the EXP bonus)
	// and the world (the sub-120 spawn HP handicap). Set here, BEFORE the boot
	// population below, so the spawns are handicapped too — the portal config may
	// override it later via ApplyWorldEventConfigBoot.
	w.SetNewbieEvent(*newbieEvent)

	// Billing gate: real binServer adapter when -binserver is set, else allow-all.
	if *binAddr != "" {
		conn, err := grpc.NewClient(*binAddr,
			grpc.WithTransportCredentials(clientCreds),
			grpc.WithChainUnaryInterceptor(metrics.UnaryClientInterceptor()))
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		w.SetBilling(binclient.New(conn))
		logger.Info("binServer wired", "addr", *binAddr)
	}

	// Channel-status HTTP server on its own port (real WYD serves serv00.htm on
	// :80, separate from the game's :8281 — general-config.h of the snalmir
	// reference). The client probes status here, then opens the CPSock game
	// connection to the game port; keeping them apart avoids the client seeing an
	// HTTP server on the game port.
	if *statusAddr != "" {
		go serveStatusHTTP(ctx, *statusAddr, statusFile, logger)
	}

	// Static world objects (gates/doors) before the loop starts, so id assignment
	// is single-threaded and follows file order.
	if len(worldItems) > 0 {
		seedWorldItems(w, worldItems, logger)
	}
	// Populate the world from the NPC spawn catalogue (before Serve starts the
	// loop, so spawning is single-threaded): register the generator slots from the
	// snapshot, then materialize the entities from that same snapshot.
	if npcConfig != nil {
		// Retries while the dbServer is still coming up; ctx aborts the wait on
		// SIGTERM instead of holding the process for the whole retry budget.
		snap, err := dispatch.FetchNPCConfig(ctx)
		if err != nil {
			return err
		}
		auditMobTemplates(snap.Defs, logger)
		registerNPCGenerators(w, snap.Defs, attrs, logger)
		if err := dispatch.ApplyNPCConfigBoot(w, snap); err != nil {
			return err
		}
	}
	if worldEvents != nil {
		dispatch.ApplyWorldEventConfigBoot(w)
	}
	dispatch.ApplyGuildStateBoot(w)

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	logger.Info("tmserver listening", "addr", *addr, "mtls", *tlsCert != "")

	return w.Serve(ctx, ln)
}

// seedWorldItems spawns the static world objects (gates/doors) from
// world_item_seed before Serve starts the loop, so id assignment is
// single-threaded and follows import order (matching the legacy CreateItem
// sequence the client expects). An empty catalogue seeds nothing.
func seedWorldItems(w *world.World, items []content.InitItem, logger *slog.Logger) {
	seeded := 0
	for _, it := range items {
		// Gates seed open (parity with CreateItem); locking arrives with the deferred
		// event/timer systems. SeedWorldItem returns -1 only when the id table is full.
		if id := w.SeedWorldItem(world.Item{Index: it.Index}, it.PosX, it.PosY, world.StateOpen); id >= 0 {
			seeded++
		}
	}
	logger.Info("world items seeded", "count", seeded)
}

// registerNPCGenerators builds the world's generator slots from the spawn
// catalogue (issue 57) instead of parsing NPCGener.txt at boot. Slot i is content
// block i, exactly the file order the legacy server used, so the index ranges that
// classify event blocks (IsWaterGenerator, NightmareGenerator, rune quest) keep
// working. A block the importer skipped (missing/undecodable template) leaves its
// slot nil, and event-owned blocks are registered but left to their event code.
func registerNPCGenerators(w *world.World, defs []npccfg.Definition, attrs *content.Grid, logger *slog.Logger) {
	slots := 0
	for _, def := range defs {
		if def.Origin == "content" && def.GeneratorIndex >= 0 && def.GeneratorIndex+1 > slots {
			slots = def.GeneratorIndex + 1
		}
	}
	if slots == 0 {
		logger.Warn("npc catalogue has no content blocks: the world boots with no monsters")
		return
	}
	gens := make([]*world.Generator, slots)
	registered, eventOwned, training := 0, 0, 0
	for _, def := range defs {
		if def.Origin != "content" || def.GeneratorIndex < 0 {
			continue
		}
		g := &world.Generator{
			MinuteGenerate: def.MinuteGenerate, MinGroup: def.MinGroup, MaxGroup: def.MaxGroup, MaxNumMob: def.MaxNumMob,
			RouteType: def.RouteType, Formation: def.Formation,
			SegX: def.SegX, SegY: def.SegY, SegRange: def.SegRange, SegWait: def.SegWait,
			LeaderTmpl: def.Template, FollowerTmpl: def.FollowerTemplate,
			FightAction: def.FightAction, DieAction: def.DieAction,
			TrainingField: handler.TrainingFieldAt(attrs, def.SegX[0], def.SegY[0]),
		}
		gens[def.GeneratorIndex] = g
		registered++
		if world.EventOwnedGenerator(def.GeneratorIndex) {
			eventOwned++
		}
		if g.TrainingField {
			training++
		}
	}
	w.RegisterGenerators(gens)
	if attrs == nil {
		logger.Warn("training-grounds generators not marked: AttributeMap.dat not loaded, field EXP multiplier disabled")
	}
	logger.Info("npc generators registered", "generators", registered, "slots", slots,
		"event_owned", eventOwned, "training_grounds", training)
}

// auditMobTemplates warns once per template whose kill reward is zero or beyond the
// legacy award gate (10M, MobKilled.cpp:1284): the content tree must be restamped
// with cmd/exptool or players gain nothing from those kills (issue #43). The check
// came from the NPCGener.txt boot loader, which no longer exists (issue 57).
func auditMobTemplates(defs []npccfg.Definition, logger *slog.Logger) {
	const maxSaneMobExp = 10_000_000
	seen := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if len(def.Template) < 16 {
			continue
		}
		name := strings.TrimRight(string(def.Template[0:16]), "\x00 ")
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		if mb := protocol.ParseMobBasics(def.Template); mb.Merchant == 0 && mb.Level >= 1 &&
			(mb.Exp <= 0 || mb.Exp > maxSaneMobExp) {
			logger.Warn("monster template has unbalanced Exp (run cmd/exptool)",
				"npc", name, "level", mb.Level, "exp", mb.Exp)
		}
	}
}

// serveStatusHTTP runs the channel-status web server (serv00.htm). It answers any
// path with the status page so the client's GET succeeds regardless of the exact
// file it asks for. The body is read per request so it can be edited live.
func serveStatusHTTP(ctx context.Context, addr, statusFile string, logger *slog.Logger) {
	defaultBody := []byte("4\r\n-1\r\n-1\r\n-1\r\n-1\r\n-1\r\n-1\r\n-1\r\n-1\r\n-1\r\n")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := defaultBody
		if statusFile != "" {
			if b, err := os.ReadFile(statusFile); err == nil {
				body = b
			}
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(body)
		logger.Info("served status (http)", "ip", r.RemoteAddr, "req", r.Method+" "+r.URL.Path)
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() { <-ctx.Done(); _ = srv.Close() }()
	logger.Info("status server listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("status server stopped", "err", err)
	}
}

// loadContent loads and validates the tmserver/content/ content tree (Fase 5 loaders).
// The rates and catalogs are required (a broken mount is a hard error); the maps
// are large and optional (a warning when absent). It logs what was loaded so the
// operator can confirm the mount is correct.
func loadContent(dir string, logger *slog.Logger) (*loadedContent, error) {
	comp, err := content.LoadCompRate(filepath.Join(dir, "Common", "Settings", "CompRate.txt"))
	if err != nil {
		return nil, err
	}
	sanc, err := content.LoadSancRate(filepath.Join(dir, "Common", "Settings", "SancRate.txt"))
	if err != nil {
		return nil, err
	}
	quests, err := content.LoadQuestRates(filepath.Join(dir, "Common", "Settings", "QuestsRate.txt"))
	if err != nil {
		return nil, err
	}
	// The Ori/Lac rates are worth logging outright: SancRate.txt only overrides the
	// indices it lists, so a broken mount silently leaves the compiled defaults and
	// the operator would otherwise not notice.
	logger.Info("content loaded",
		"comprate_families", comp.Families(),
		"sancrate_ori", sancRow(sanc, 0), "sancrate_lac", sancRow(sanc, 1))

	// Maps are optional: 17 MiB HeightMap + 1 MiB AttributeMap aren't required to
	// accept logins; warn rather than fail when they aren't mounted. When both
	// load, bake the attribute blocks into the height grid once (the boot-time
	// BASE_ApplyAttribute) — the result drives mob pathfinding (route.Next).
	var heights *content.Grid
	attr, err := content.LoadGrid(filepath.Join(dir, "TMsrv", "run", "AttributeMap.dat"), content.AttributeMapDim)
	if err != nil {
		logger.Warn("attribute map not loaded", "err", err)
	}
	hm, err := content.LoadHeightMap(filepath.Join(dir, "TMsrv", "run", "HeightMap.dat"))
	if err != nil {
		logger.Warn("height map not loaded", "err", err)
	}
	if hm != nil && attr != nil {
		route.Bake(hm, attr)
		heights = hm
		logger.Info("walkability grid baked", "dim", hm.Dim)
	} else if hm != nil || attr != nil {
		logger.Warn("mob pathfinding disabled: need BOTH HeightMap.dat and AttributeMap.dat")
	}
	return &loadedContent{comp: comp, sanc: sanc, quests: quests, heights: heights, attrs: attr}, nil
}

// loadedContent is what a mounted tmserver/content/ tree yields. It is a struct rather
// than a return list only because the list had grown past readability.
type loadedContent struct {
	comp    *content.CompRate
	sanc    *content.SancRate
	quests  *content.QuestRates
	heights *content.Grid
	attrs   *content.Grid // raw AttributeMap (training-grounds bit); nil when not mounted
}

// sancRow renders one anvil's rate row for the boot log.
func sancRow(s *content.SancRate, anvil int) []int {
	row := make([]int, 0, 12)
	for i := 0; i < 12; i++ {
		row = append(row, s.Rate(anvil, i))
	}
	return row
}
