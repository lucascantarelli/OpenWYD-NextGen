// Command webserver is the web-api: the gRPC edge the Next.js BFF calls
// server-side (web-platform-plan.md). It owns the web platform's account flows
// (sign-up, credential check) over the same `account` table and argon2id hashing
// as dbServer, but is a SEPARATE service from dbServer's legacy AccountService.
//
// Usage:
//
//	webserver [-addr :7600] -dsn <postgres-url> [-tls-cert … -tls-key … -tls-ca …] [-content <tmserver/content/>]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/internal/playcode"
	"openwyd-nextgen/internal/secure"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/webserver/internal/account"
	"openwyd-nextgen/webserver/internal/accountadmin"
	"openwyd-nextgen/webserver/internal/attributemap"
	"openwyd-nextgen/webserver/internal/characteradmin"
	"openwyd-nextgen/webserver/internal/characters"
	"openwyd-nextgen/webserver/internal/dailyreward"
	"openwyd-nextgen/webserver/internal/donaterevenue"
	"openwyd-nextgen/webserver/internal/donateshop"
	"openwyd-nextgen/webserver/internal/donatetopup"
	"openwyd-nextgen/webserver/internal/dropadmin"
	"openwyd-nextgen/webserver/internal/droptool"
	"openwyd-nextgen/webserver/internal/grpcsrv"
	"openwyd-nextgen/webserver/internal/itemcatalog"
	"openwyd-nextgen/webserver/internal/itemeffectadmin"
	"openwyd-nextgen/webserver/internal/itemicons"
	"openwyd-nextgen/webserver/internal/mobtemplateadmin"
	"openwyd-nextgen/webserver/internal/mobtemplates"
	"openwyd-nextgen/webserver/internal/npcadmin"
	"openwyd-nextgen/webserver/internal/npctemplates"
	"openwyd-nextgen/webserver/internal/ranking"
	"openwyd-nextgen/webserver/internal/ratesadmin"
	"openwyd-nextgen/webserver/internal/skillcatalog"
	"openwyd-nextgen/webserver/internal/teleportadmin"
	"openwyd-nextgen/webserver/internal/worldevent"
	"openwyd-nextgen/webserver/internal/worldstateadmin"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("webserver failed", "err", err)
		os.Exit(1)
	}
}

// run applies migrations and serves the gRPC AccountWebService until the process
// receives SIGINT/SIGTERM, then stops gracefully. It shares store.Migrate with
// dbServer, so booting either service brings the schema up to date.
func run(logger *slog.Logger) error {
	addr := flag.String("addr", ":7600", "gRPC listen address")
	dsn := flag.String("dsn", envOr("W2PP_DB_DSN", ""), "PostgreSQL DSN (or W2PP_DB_DSN)")
	tlsCert := flag.String("tls-cert", os.Getenv("W2PP_TLS_CERT"), "server certificate (PEM)")
	tlsKey := flag.String("tls-key", os.Getenv("W2PP_TLS_KEY"), "server private key (PEM)")
	tlsCA := flag.String("tls-ca", os.Getenv("W2PP_TLS_CA"), "client CA (PEM) for mTLS")
	contentDir := flag.String("content", envOr("W2PP_CONTENT", "tmserver/content"), "caminho da árvore de conteúdo para taxas e mapas (catálogos vêm do banco; default tmserver/content)")
	iconManifestPath := flag.String("item-icons-manifest", os.Getenv("W2PP_ITEM_ICONS_MANIFEST"), "generated item-icon manifest (empty = fallback-only)")
	metricsAddr := flag.String("metrics-addr", envOr("W2PP_METRICS_ADDR", ":9104"), "endpoint de métricas/health (Prometheus) em porta separada; vazio desliga")
	flag.Parse()

	if *dsn == "" {
		return fmt.Errorf("-dsn (or W2PP_DB_DSN) is required")
	}
	// The web client's one-time login codes (ADR 017 there) stay off without
	// a secret shared with the portal. Environment only: a flag would show up
	// in the process list.
	playCodeSecret := os.Getenv("W2PP_PLAY_CODE_SECRET")
	playCodes := playcode.NewVerifier(playCodeSecret)
	if playCodeSecret != "" && playCodes == nil {
		return fmt.Errorf("W2PP_PLAY_CODE_SECRET must have at least %d characters", playcode.MinSecretLength)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()
	if err := store.MigrateWithEnv(ctx, pool); err != nil {
		return err
	}

	metricsReg := metrics.New("webserver")
	go metrics.Serve(ctx, *metricsAddr, metricsReg, logger)
	metrics.RegisterPGXPool(metricsReg, "webserver", pool)
	metrics.RegisterTableRows(metricsReg, "webserver", "delivery_queue_rows", "Linhas na delivery_queue.", "SELECT count(*) FROM delivery_queue", pool)

	creds, err := secure.ServerCreds(secure.Config{CertFile: *tlsCert, KeyFile: *tlsKey, CAFile: *tlsCA})
	if err != nil {
		return err
	}
	srv := grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(metrics.UnaryServerInterceptor("webserver", metricsReg, logger)))
	st := store.New(pool)
	npcAdmin := npcadmin.New(st)
	npcAdmin.SetLogger(logger)
	mobTemplateAdmin := mobtemplateadmin.New(st)
	donate := donateshop.New(st)
	dailyRwd := dailyreward.New(st)
	topup := donatetopup.New(st)
	revenue := donaterevenue.New(st)
	attrMap := attributemap.New(st, *contentDir)
	worldEvents := worldevent.New(st)
	teleportAdmin := teleportadmin.New(st)
	characterAdmin := characteradmin.New(st)
	itemEffectAdmin := itemeffectadmin.New(st)
	worldStateAdmin := worldstateadmin.New(st)
	ratesAdmin := ratesadmin.New(st, *contentDir)
	dropAdmin := dropadmin.New(st)
	// Stays zero-valued until the catalog loads: ItemCatalogService then serves
	// an empty list, which the boot log calls out loudly.
	var itemCatalog itemcatalog.Catalog
	var iconManifest itemicons.Manifest
	if *iconManifestPath != "" {
		iconManifest, err = itemicons.Load(*iconManifestPath)
		if err != nil {
			return err
		}
		logger.Info("loaded item icon manifest", "version", iconManifest.PackVersion,
			"mapped", iconManifest.MappedItems, "icons", iconManifest.DistinctIcons)
	} else {
		logger.Warn("item icon manifest not configured; catalog is fallback-only",
			"configuration", "W2PP_ITEM_ICONS_MANIFEST")
	}

	// CatÃ¡logos vÃªm do Postgres (issue 58): ItemList.csv, SkillData.csv e npc/
	// saÃ­ram de tmserver/content/ na fase 5, entÃ£o escanear a Ã¡rvore sÃ³ produzia WARN de
	// catÃ¡logo vazio a cada boot (issue 62). -content fica sÃ³ com rates e mapa.
	itemRows, err := st.ListItemCatalog(ctx)
	if err != nil {
		return fmt.Errorf("load item catalog: %w", err)
	}
	catalog := itemcatalog.FromEntries(itemRows)
	if *iconManifestPath != "" {
		itemcatalog.ApplyIcons(&catalog, iconManifest)
	}
	withIconKey, withIconURL := catalog.IconCoverage()
	if len(catalog.Items) == 0 {
		logger.Warn("item catalog is empty in the database; item picker will be empty",
			"table", "item_catalog", "seed", "tmserver/cmd/importcontent")
	}
	logger.Info("loaded item catalog from the database", "count", len(catalog.Items), "version", catalog.Version,
		"icon_pack_version", catalog.IconPackVersion, "items_with_icon_key", withIconKey,
		"items_with_icon_url", withIconURL)
	npcAdmin.SetItemCatalog(catalog)
	// The character screen shows item names/icons instead of raw indexes.
	characterAdmin.SetItemCatalog(catalog)
	// Efeitos base: o overlay da web mescla com o que o banco declara.
	itemEffectAdmin.SetCatalog(catalog)
	// O editor de Carry[] valida o Ã­ndice e mostra nome, nunca o nÃºmero.
	dropAdmin.SetItemCatalog(catalog)
	itemCatalog = catalog

	skillRows, err := st.ListSkillCatalog(ctx)
	if err != nil {
		return fmt.Errorf("load skill catalog: %w", err)
	}
	skillCatalog := skillcatalog.FromEntries(skillRows)
	if len(skillCatalog.Skills) == 0 {
		logger.Warn("skill catalog is empty in the database; skill names stay as raw indexes",
			"table", "skill_catalog")
	}
	logger.Info("loaded skill catalog from the database", "count", len(skillCatalog.Skills), "version", skillCatalog.Version)
	characterAdmin.SetSkillCatalog(skillCatalog)

	// npc_template guarda o STRUCT_MOB canÃ´nico de cada template: Ã© a mesma
	// fonte do picker de NPC, do picker de mob e do DropTool (issue 62).
	templateRows, err := st.ListNPCTemplates(ctx)
	if err != nil {
		return fmt.Errorf("load npc templates: %w", err)
	}
	blobs := make(map[string][]byte, len(templateRows))
	for _, t := range templateRows {
		blobs[t.Name] = t.Blob
	}
	if len(blobs) == 0 {
		logger.Warn("npc_template is empty in the database; template pickers will be empty", "table", "npc_template")
	}

	merchantTemplates, npcStats := npctemplates.FromBlobs(blobs)
	logger.Info("loaded merchant npc templates from the database", "count", len(merchantTemplates), "layouts", npcStats)
	npcAdmin.SetTemplates(merchantTemplates)

	mobTemplates, mobStats := mobtemplates.FromBlobs(blobs)
	logger.Info("loaded mob templates from the database", "count", len(mobTemplates), "layouts", mobStats)
	mobTemplateAdmin.SetTemplates(mobTemplates)

	readMobTemplate := func(name string) ([]byte, error) {
		if blob, ok := blobs[name]; ok {
			return blob, nil
		}
		return nil, fmt.Errorf("template %q not found in npc_template", name)
	}
	mobTemplateAdmin.SetTemplateReader(readMobTemplate)
	// Carry[] por template (issue 34): decodifica o blob real e recusa um
	// template que nÃ£o existe.
	dropAdmin.SetTemplateReader(readMobTemplate)

	dropsOpts := droptool.Options{}
	if *contentDir != "" {
		exclusions, err := droptool.LoadContentExclusions(*contentDir)
		if err != nil {
			logger.Warn("drop exclusion scan failed; continuing without exclusions", "content", *contentDir, "err", err)
		}
		dropsOpts.Exclusions = exclusions
	}
	drops, dropStats := droptool.FromBlobs(catalog.Items, blobs, dropsOpts)
	logger.Info("loaded drop catalog from the database", "items", len(drops.Items), "mobs", len(drops.Mobs),
		"drops", len(drops.Drops), "layouts", dropStats)
	npcAdmin.SetDropCatalog(drops)
	webv1.RegisterAccountWebServiceServer(srv, grpcsrv.New(account.New(st)).WithPlayCodes(playCodes))
	// Contas, papéis e banimento (issue 32): leitura para staff, escrita auditada
	// só para admin (bloqueio também para moderador, alvo jogador).
	webv1.RegisterAccountAdminServiceServer(srv, grpcsrv.NewAccountAdmin(accountadmin.New(st)))
	webv1.RegisterRankingWebServiceServer(srv, grpcsrv.NewRanking(ranking.New(st)))
	webv1.RegisterCharacterWebServiceServer(srv, grpcsrv.NewCharacters(characters.New(st)))
	webv1.RegisterItemCatalogServiceServer(srv, grpcsrv.NewItemCatalog(itemCatalog))
	webv1.RegisterNpcAdminServiceServer(srv, grpcsrv.NewNpcAdmin(npcAdmin))
	webv1.RegisterMobTemplateAdminServiceServer(srv, grpcsrv.NewMobTemplateAdmin(mobTemplateAdmin))
	webv1.RegisterAttributeMapAdminServiceServer(srv, grpcsrv.NewAttributeMapAdmin(attrMap))
	webv1.RegisterDonateAdminServiceServer(srv, grpcsrv.NewDonateAdmin(donate))
	webv1.RegisterDonateShopServiceServer(srv, grpcsrv.NewDonateShop(donate))
	webv1.RegisterDailyRewardAdminServiceServer(srv, grpcsrv.NewDailyRewardAdmin(dailyRwd))
	webv1.RegisterDailyRewardServiceServer(srv, grpcsrv.NewDailyReward(dailyRwd))
	webv1.RegisterDonateTopupServiceServer(srv, grpcsrv.NewDonateTopup(topup))
	webv1.RegisterDonateRevenueAdminServiceServer(srv, grpcsrv.NewDonateRevenue(revenue))
	webv1.RegisterWorldEventAdminServiceServer(srv, grpcsrv.NewWorldEventAdmin(worldEvents))
	webv1.RegisterTeleportAdminServiceServer(srv, grpcsrv.NewTeleportAdmin(teleportAdmin))
	webv1.RegisterCharacterAdminServiceServer(srv, grpcsrv.NewCharacterAdmin(characterAdmin))
	webv1.RegisterItemEffectAdminServiceServer(srv, grpcsrv.NewItemEffectAdmin(itemEffectAdmin))
	webv1.RegisterWorldStateAdminServiceServer(srv, grpcsrv.NewWorldStateAdmin(worldStateAdmin))
	webv1.RegisterContentRatesServiceServer(srv, grpcsrv.NewContentRates(ratesAdmin))
	webv1.RegisterDropAdminServiceServer(srv, grpcsrv.NewDropAdmin(dropAdmin))

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", *addr, err)
	}
	logger.Info("webserver serving", "addr", *addr, "mtls", *tlsCert != "", "play_codes", playCodes != nil)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

// envOr returns the environment value for key, or def when unset.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
