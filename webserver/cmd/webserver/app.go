// Wiring do webserver (web-api gRPC), separado do `main` para ser testável sem
// `os.Args`, sem abrir porta e sem banco real (REFACTOR-01).
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
	"openwyd-nextgen/internal/domain"
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

// config é a configuração resolvida de uma execução do webserver.
type config struct {
	addr             string
	dsn              string
	tlsCert          string
	tlsKey           string
	tlsCA            string
	contentDir       string
	iconManifestPath string
	metricsAddr      string
}

// parseConfig lê as flags (defaults vindos do ambiente) para o `config`, sem
// encerrar o processo quando uma flag é inválida.
func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("webserver", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.addr, "addr", ":7600", "gRPC listen address")
	fs.StringVar(&cfg.dsn, "dsn", envOr("W2PP_DB_DSN", ""), "PostgreSQL DSN (or W2PP_DB_DSN)")
	fs.StringVar(&cfg.tlsCert, "tls-cert", os.Getenv("W2PP_TLS_CERT"), "server certificate (PEM)")
	fs.StringVar(&cfg.tlsKey, "tls-key", os.Getenv("W2PP_TLS_KEY"), "server private key (PEM)")
	fs.StringVar(&cfg.tlsCA, "tls-ca", os.Getenv("W2PP_TLS_CA"), "client CA (PEM) for mTLS")
	fs.StringVar(&cfg.contentDir, "content", envOr("W2PP_CONTENT", "tmserver/content"), "caminho da árvore de conteúdo para taxas e mapas (catálogos vêm do banco; default tmserver/content)")
	fs.StringVar(&cfg.iconManifestPath, "item-icons-manifest", os.Getenv("W2PP_ITEM_ICONS_MANIFEST"), "generated item-icon manifest (empty = fallback-only)")
	fs.StringVar(&cfg.metricsAddr, "metrics-addr", envOr("W2PP_METRICS_ADDR", ":9104"), "endpoint de métricas/health (Prometheus) em porta separada; vazio desliga")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	return cfg, nil
}

// deps é tudo o que o wiring consome e que veio de fora do processo (banco e
// ambiente). Existe para que `newApp` possa ser exercitado num teste sem DSN.
type deps struct {
	pool      *pgxpool.Pool
	st        *store.Store
	items     []domain.ItemCatalogEntry
	skills    []domain.SkillCatalogEntry
	templates []domain.NPCTemplateEntry
	icons     itemicons.Manifest
	playCodes *playcode.Verifier
}

// app é o servidor montado e pronto para servir; o listener só existe em `run`.
type app struct {
	srv       *grpc.Server
	pool      *pgxpool.Pool
	addr      string
	mtls      bool
	playCodes bool
}

// newApp valida a configuração, abre o banco, carrega os catálogos e monta o
// gRPC. Não abre porta e não bloqueia.
func newApp(ctx context.Context, cfg config, logger *slog.Logger) (*app, error) {
	d, err := openDeps(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	a, err := newAppWithDeps(cfg, *d, logger)
	if err != nil {
		if d.pool != nil {
			d.pool.Close()
		}
		return nil, err
	}
	return a, nil
}

// openDeps faz o I/O de boot: pool, migrations, manifest de ícones e os três
// catálogos de conteúdo.
func openDeps(ctx context.Context, cfg config, logger *slog.Logger) (*deps, error) {
	if cfg.dsn == "" {
		return nil, fmt.Errorf("-dsn (or W2PP_DB_DSN) is required")
	}
	// Os códigos de login de uso único do cliente web ficam desligados sem um
	// segredo compartilhado com o portal. Só por ambiente: uma flag apareceria
	// na lista de processos.
	playCodeSecret := os.Getenv("W2PP_PLAY_CODE_SECRET")
	playCodes := playcode.NewVerifier(playCodeSecret)
	if playCodeSecret != "" && playCodes == nil {
		return nil, fmt.Errorf("W2PP_PLAY_CODE_SECRET must have at least %d characters", playcode.MinSecretLength)
	}

	pool, err := pgxpool.New(ctx, cfg.dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := store.MigrateWithEnv(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	d := &deps{pool: pool, st: store.New(pool), playCodes: playCodes}

	metricsReg := metrics.New("webserver")
	go metrics.Serve(ctx, cfg.metricsAddr, metricsReg, logger)
	metrics.RegisterPGXPool(metricsReg, "webserver", pool)
	metrics.RegisterTableRows(metricsReg, "webserver", "delivery_queue_rows", "Linhas na delivery_queue.", "SELECT count(*) FROM delivery_queue", pool)

	if cfg.iconManifestPath != "" {
		d.icons, err = itemicons.Load(cfg.iconManifestPath)
		if err != nil {
			pool.Close()
			return nil, err
		}
		logger.Info("loaded item icon manifest", "version", d.icons.PackVersion,
			"mapped", d.icons.MappedItems, "icons", d.icons.DistinctIcons)
	} else {
		logger.Warn("item icon manifest not configured; catalog is fallback-only",
			"configuration", "W2PP_ITEM_ICONS_MANIFEST")
	}

	// Catálogos vêm do Postgres: ItemList.csv, SkillData.csv e npc/ saíram da
	// árvore de conteúdo, então escanear o diretório só produzia WARN de catálogo
	// vazio a cada boot. -content fica só com taxas e mapa.
	if d.items, err = d.st.ListItemCatalog(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("load item catalog: %w", err)
	}
	if d.skills, err = d.st.ListSkillCatalog(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("load skill catalog: %w", err)
	}
	if d.templates, err = d.st.ListNPCTemplates(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("load npc templates: %w", err)
	}
	return d, nil
}

// newAppWithDeps monta o servidor gRPC a partir de dependências já resolvidas.
// É o mesmo caminho de `newApp` sem o I/O, e é o que o teste exercita.
func newAppWithDeps(cfg config, d deps, logger *slog.Logger) (*app, error) {
	metricsReg := metrics.New("webserver")
	creds, err := secure.ServerCreds(secure.Config{CertFile: cfg.tlsCert, KeyFile: cfg.tlsKey, CAFile: cfg.tlsCA})
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(metrics.UnaryServerInterceptor("webserver", metricsReg, logger)))
	st := d.st
	if st == nil {
		st = store.New(d.pool)
	}

	npcAdmin := npcadmin.New(st)
	npcAdmin.SetLogger(logger)
	mobTemplateAdmin := mobtemplateadmin.New(st)
	donate := donateshop.New(st)
	dailyRwd := dailyreward.New(st)
	topup := donatetopup.New(st)
	revenue := donaterevenue.New(st)
	attrMap := attributemap.New(st, cfg.contentDir)
	worldEvents := worldevent.New(st)
	teleportAdmin := teleportadmin.New(st)
	characterAdmin := characteradmin.New(st)
	itemEffectAdmin := itemeffectadmin.New(st)
	worldStateAdmin := worldstateadmin.New(st)
	ratesAdmin := ratesadmin.New(st, cfg.contentDir)
	dropAdmin := dropadmin.New(st)

	catalog := itemcatalog.FromEntries(d.items)
	if len(catalog.Items) == 0 {
		// Fail closed: um picker vazio não é "sem conteúdo", é seed faltando, e o
		// boot é o único lugar onde isso aparece antes de um jogador tropeçar nele.
		return nil, fmt.Errorf("item_catalog is empty: seed the content catalogue (tmserver/cmd/importcontent or seedgen)")
	}
	if cfg.iconManifestPath != "" {
		itemcatalog.ApplyIcons(&catalog, d.icons)
	}
	withIconKey, withIconURL := catalog.IconCoverage()
	logger.Info("loaded item catalog from the database", "count", len(catalog.Items), "version", catalog.Version,
		"icon_pack_version", catalog.IconPackVersion, "items_with_icon_key", withIconKey,
		"items_with_icon_url", withIconURL)
	npcAdmin.SetItemCatalog(catalog)
	// A tela de personagem mostra nome/ícone do item, não o índice cru.
	characterAdmin.SetItemCatalog(catalog)
	// Efeitos base: o overlay da web mescla com o que o banco declara.
	itemEffectAdmin.SetCatalog(catalog)
	// O editor de Carry[] valida o índice e mostra nome, nunca o número.
	dropAdmin.SetItemCatalog(catalog)

	skillCatalog := skillcatalog.FromEntries(d.skills)
	if len(skillCatalog.Skills) == 0 {
		logger.Warn("skill catalog is empty in the database; skill names stay as raw indexes",
			"table", "skill_catalog")
	}
	logger.Info("loaded skill catalog from the database", "count", len(skillCatalog.Skills), "version", skillCatalog.Version)
	characterAdmin.SetSkillCatalog(skillCatalog)

	// npc_template guarda o STRUCT_MOB canônico de cada template: é a mesma
	// fonte do picker de NPC, do picker de mob e do DropTool.
	blobs := make(map[string][]byte, len(d.templates))
	for _, t := range d.templates {
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
	// Carry[] por template: decodifica o blob real e recusa um template que não
	// existe.
	dropAdmin.SetTemplateReader(readMobTemplate)

	dropsOpts := droptool.Options{}
	if cfg.contentDir != "" {
		exclusions, err := droptool.LoadContentExclusions(cfg.contentDir)
		if err != nil {
			logger.Warn("drop exclusion scan failed; continuing without exclusions", "content", cfg.contentDir, "err", err)
		}
		dropsOpts.Exclusions = exclusions
	}
	drops, dropStats := droptool.FromBlobs(catalog.Items, blobs, dropsOpts)
	logger.Info("loaded drop catalog from the database", "items", len(drops.Items), "mobs", len(drops.Mobs),
		"drops", len(drops.Drops), "layouts", dropStats)
	npcAdmin.SetDropCatalog(drops)

	webv1.RegisterAccountWebServiceServer(srv, grpcsrv.New(account.New(st)).WithPlayCodes(d.playCodes))
	// Contas, permissões e banimento: leitura para a equipe, escrita auditada só
	// para administrador (bloqueio também para moderador, alvo jogador).
	webv1.RegisterAccountAdminServiceServer(srv, grpcsrv.NewAccountAdmin(accountadmin.New(st)))
	webv1.RegisterRankingWebServiceServer(srv, grpcsrv.NewRanking(ranking.New(st)))
	webv1.RegisterCharacterWebServiceServer(srv, grpcsrv.NewCharacters(characters.New(st)))
	webv1.RegisterItemCatalogServiceServer(srv, grpcsrv.NewItemCatalog(catalog))
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

	return &app{srv: srv, pool: d.pool, addr: cfg.addr, mtls: cfg.tlsCert != "", playCodes: d.playCodes != nil}, nil
}

// run abre o listener e serve até o ctx ser cancelado (SIGINT/SIGTERM).
func (a *app) run(ctx context.Context, logger *slog.Logger) error {
	if a.pool != nil {
		defer a.pool.Close()
	}
	ln, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", a.addr, err)
	}
	logger.Info("webserver serving", "addr", a.addr, "mtls", a.mtls, "play_codes", a.playCodes)

	errCh := make(chan error, 1)
	go func() { errCh <- a.srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		a.srv.GracefulStop()
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

// runFromArgs liga o `main`: resolve a configuração, instala o handler de
// SIGINT/SIGTERM e roda o app até o encerramento.
func runFromArgs(args []string, logger *slog.Logger) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	a, err := newApp(ctx, cfg, logger)
	if err != nil {
		return err
	}
	return a.run(ctx, logger)
}

// envOr devolve o valor de ambiente de key, ou def quando não está definida.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
