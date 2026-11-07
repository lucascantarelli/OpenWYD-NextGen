// Wiring do tmserver: configuração, montagem do mundo e ciclo de vida do
// processo. O `main` só resolve o contexto e chama `newApp`/`run`; o corpo do
// `newApp` é o antigo `run`, agora recebendo um `config` explícito em vez de ler
// flags (REFACTOR-01).
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
	"path/filepath"
	"strconv"
	"strings"
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

// config é a configuração resolvida de uma execução do tmserver. Nada aqui lê
// `flag` ou o ambiente: `parseConfig` faz isso e entrega valores prontos.
type config struct {
	addr                  string
	dbAddr                string
	binAddr               string
	tlsCert               string
	tlsKey                string
	tlsCA                 string
	tlsServerName         string
	rejectChecksum        bool
	maxMsgPerSec          float64
	msgBurst              int
	contentDir            string
	teleportEditing       bool
	mobStatEditing        bool
	itemEffectEditing     bool
	ratesEditing          bool
	dropConfigEditing     bool
	mobCarryEditing       bool
	statusAddr            string
	metricsAddr           string
	maxNightmare          int
	kefraEntriesPerScroll int
	clientVersion         int
	serverIndex           int
	crossoverKey          string
	doubleExp             bool
	newbieEvent           bool
	noviceChannel         int
	kefraLive             bool
	trainingFieldExp      int
	logSends              bool
	affectScalePct        int
	affectMinSeconds      int
	affectMaxMinutes      int
}

// parseConfig lê as flags e os defaults do ambiente para o `config`. Flag
// inválida devolve erro; quem decide o código de saída é o `main`.
func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("tmserver", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.addr, "addr", ":8281", "CPSock listen address for the client edge")
	fs.StringVar(&cfg.dbAddr, "dbserver", os.Getenv("W2PP_DBSERVER"), "dbServer gRPC address (empty = no-op persistence)")
	fs.StringVar(&cfg.binAddr, "binserver", os.Getenv("W2PP_BINSERVER"), "binServer gRPC address (empty = allow-all billing)")
	fs.StringVar(&cfg.tlsCert, "tls-cert", os.Getenv("W2PP_TLS_CERT"), "client certificate (PEM) for internal mTLS")
	fs.StringVar(&cfg.tlsKey, "tls-key", os.Getenv("W2PP_TLS_KEY"), "client private key (PEM)")
	fs.StringVar(&cfg.tlsCA, "tls-ca", os.Getenv("W2PP_TLS_CA"), "CA (PEM) verifying dbServer/binServer")
	fs.StringVar(&cfg.tlsServerName, "tls-server-name", os.Getenv("W2PP_TLS_SERVER_NAME"), "expected server name in internal certs")
	fs.BoolVar(&cfg.rejectChecksum, "reject-checksum", false, "drop connections on CPSock checksum mismatch (Fase 7; off by default)")
	fs.Float64Var(&cfg.maxMsgPerSec, "max-msg-per-sec", 200, "per-connection inbound message rate limit (0 = disabled)")
	fs.IntVar(&cfg.msgBurst, "msg-burst", 400, "per-connection message burst depth")
	// O conteúdo mora dentro da árvore do servidor (LEGACY-02): sem -content, o
	// boot lê tmserver/content/ a partir do diretório do repositório/da imagem.
	defContent := os.Getenv("W2PP_CONTENT")
	if defContent == "" {
		defContent = "tmserver/content"
	}
	fs.StringVar(&cfg.contentDir, "content", defContent, "caminho da árvore de conteúdo (default tmserver/content; vazio = pula; valida taxas/mapas no boot)")
	fs.BoolVar(&cfg.teleportEditing, "teleport-editing", envBool("W2PP_TELEPORT_EDITING", false), "enable the moderator teleport-route overlay (issue 42); needs -dbserver. Applied at boot and re-read every 15s, so an edit reaches a running world without a restart. OFF by default: with it off the world resolves strictly from the compiled table")
	fs.BoolVar(&cfg.mobStatEditing, "mob-stat-editing", envBool("W2PP_MOB_STAT_EDITING", false), "enable the moderator mob/NPC template stat overlay (mob-template-editing-plan.md, the equivalent-tool successor to the legacy EDITAPPMOB); needs -dbserver and -content. Applied ONCE at boot, like every other content load — a moderator edit needs a tmServer restart to take effect (EDITAPPMOB itself required a server restart too), independent of the other overlays")
	fs.BoolVar(&cfg.itemEffectEditing, "item-effect-editing", envBool("W2PP_ITEM_EFFECT_EDITING", false), "enable the moderator item base-effect overlay (issue 44, the read-through side of the web editor); needs -dbserver (catalog and overlay source). Applied ONCE at boot: itemEffects is folded into the handler at construction, so a moderator edit needs a tmServer restart to reach a running world (same restart-to-apply as -mob-stat-editing). OFF by default: with it off the catalog is exactly item_catalog")
	fs.BoolVar(&cfg.ratesEditing, "rates-editing", envBool("W2PP_RATES_EDITING", false), "enable the moderator content-rate overlay (issue 36: SancRate/CompRate/QuestsRate); needs -dbserver (overlay source) and -content (rate files). Applied ONCE at boot, so an edit needs a tmServer restart. OFF by default: with it off the tables are exactly the files in tmserver/content/Common/Settings")
	fs.BoolVar(&cfg.dropConfigEditing, "drop-config-editing", envBool("W2PP_DROP_CONFIG_EDITING", false), "enable the moderator drop-rate overlay (issue 34: DropRate[64] divisor per Carry slot); needs -dbserver (overlay source). Applied ONCE at boot, so an edit needs a tmServer restart. OFF by default: with it off the table is exactly the code defaults in tmserver/internal/loot")
	fs.BoolVar(&cfg.mobCarryEditing, "mob-carry-editing", envBool("W2PP_MOB_CARRY_EDITING", false), "enable the moderator Carry[] overlay (issue 34: the drop table of each mob template); needs -dbserver (overlay source) and -content (mob templates). Applied ONCE at boot, so an edit needs a tmServer restart. OFF by default: with it off every template keeps exactly the Carry[] in tmserver/content/TMsrv/run/npc")
	defStatusAddr := os.Getenv("W2PP_STATUS_ADDR")
	if defStatusAddr == "" {
		defStatusAddr = ":80"
	}
	fs.StringVar(&cfg.statusAddr, "status-addr", defStatusAddr, "HTTP channel-status listen address (serv00.htm); real WYD serves status on :80, separate from the game port. Empty disables")
	defMetricsAddr := os.Getenv("W2PP_METRICS_ADDR")
	if defMetricsAddr == "" {
		defMetricsAddr = ":9101"
	}
	fs.StringVar(&cfg.metricsAddr, "metrics-addr", defMetricsAddr, "endpoint de métricas/health (Prometheus) em porta separada; vazio desliga")
	fs.IntVar(&cfg.maxNightmare, "max-nightmare", envInt("W2PP_MAX_NIGHTMARE", 3), "counted Nightmare admissions per map and cycle")
	fs.IntVar(&cfg.kefraEntriesPerScroll, "kefra-entries-per-scroll", envInt("W2PP_KEFRA_ENTRIES_PER_SCROLL", 100), "Hall of Kefra entries one sealed scroll grants at the Sobrevivente (legacy 100; servers commonly use 50 or 100)")
	fs.IntVar(&cfg.clientVersion, "client-version", envInt("W2PP_CLIENT_VERSION", 7640), "MSG_AccountLogin.ClientVersion the client must send (protocol-spec says 7640; this 7662 'Cavaleiros de Kersef' build sends 12000)")
	// Identidade de canal. O legado indexa tudo por "ServerIndex": o
	// cidadão de um canal é ServerIndex+1 (_MSG_Quest.cpp:2436) e os ids de guilda
	// o carregam nos bits altos (serverIndex*4096). Uma instância por canal, então
	// 0 é o default (canal 1) e cada container extra pega o próprio valor.
	fs.IntVar(&cfg.serverIndex, "server-index", envInt("W2PP_SERVER_INDEX", 0), "zero-based channel index this instance serves; citizenship is stamped as server-index+1")
	// Chave de crossover para a troca de canal in-game (/srv): os dois canais e o
	// dbServer precisam compartilhá-la (canais.md §8.1.2). Vazia desliga o comando.
	fs.StringVar(&cfg.crossoverKey, "crossover-key", os.Getenv("W2PP_CROSSOVER_KEY"), "chave HMAC compartilhada com o dbServer para o token de troca de canal (/srv); vazia desativa a troca in-game")
	fs.BoolVar(&cfg.doubleExp, "double-exp", envBool("W2PP_DOUBLE_EXP", false), "DOUBLEMODE: double PvE experience (gameconfig double)")
	fs.BoolVar(&cfg.newbieEvent, "newbie-event", envBool("W2PP_NEWBIE_EVENT", false), "NewbieEventServer: +15% exp and newbie under-100 bonus (gameconfig)")
	fs.IntVar(&cfg.noviceChannel, "novice-channel", envInt("W2PP_NOVICE_CHANNEL", 0), "número do canal verde (novato), 1-based e fixo como em produção: nele esta instância roda o evento newbie e recusa dano PvP. 0 = nenhum canal verde (o evento segue -newbie-event em todos os canais)")
	fs.BoolVar(&cfg.kefraLive, "kefra-live", envBool("W2PP_KEFRA_LIVE", false), "initial Kefra defeated state (true opens city); persisted cycle takes precedence; false halves PvE exp")
	// Issue #383: um personagem novo precisa chegar ao nível exibido 35 em ~15 min no
	// campo de treino. O ritmo medido com a recompensa pura era ~1k exp/min
	// (×0,425 de Kefra/newbie incluído) contra os ~11,7k necessários; retirar
	// esses redutores dá ~2,35×, e 6 fecha o resto com folga.
	fs.IntVar(&cfg.trainingFieldExp, "training-field-exp", envInt("W2PP_TRAINING_FIELD_EXP", 6), "EXP multiplier for training-grounds kills (AttributeMap 0x80 generators) by mortals below stored level 35; replaces the global event modifiers there. 0 = legacy reward")
	fs.BoolVar(&cfg.logSends, "log-sends", envBool("W2PP_LOG_SENDS", false), "log every S→C frame (conn/type/id/len) — client-freeze diagnostics (investigacao-freeze-cliente.md); high volume, enable only while reproducing an incident")
	// Ajuste de duração de buff de conjuração (issue #229). A fórmula legada é
	// (AffectTime+1)*(100+Special)/100 ticks de 8s, o que põe um personagem de
	// endgame em 30-100 min por buff — relatado como longo demais três vezes
	// (#92, #202, #229). 100/0/0 restaura as durações legadas exatamente.
	fs.IntVar(&cfg.affectScalePct, "affect-scale-pct", envInt("W2PP_AFFECT_SCALE_PCT", 15), "percent of the legacy cast-affect duration to keep, applied only OUTSIDE the target band: a non-aggressive affect whose BASE (unmastered) duration is already under -affect-max-minutes keeps its legacy mastery curve (issue #236). 100 = legacy")
	fs.IntVar(&cfg.affectMinSeconds, "affect-min-seconds", envInt("W2PP_AFFECT_MIN_SECONDS", 60), "floor for NON-aggressive cast affects, in seconds; keeps the shortest buffs usable (0 = no floor)")
	fs.IntVar(&cfg.affectMaxMinutes, "affect-max-minutes", envInt("W2PP_AFFECT_MAX_MINUTES", 10), "cap for cast affects, in minutes; cuts the mastery tail (0 = no cap)")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	return cfg, nil
}

// app é o mundo montado e pronto para servir. O listener e o servidor de status
// só existem em `run`, então montar o app num teste não abre porta nenhuma.
type app struct {
	w          *world.World
	addr       string
	statusAddr string
	statusFile string
	mtls       bool
	closers    []func()
}

// envInt lê um default de flag inteira do ambiente para que deploys em
// container (Railway) possam defini-la como variável, como os outros botões W2PP_*;
// um valor ausente ou malformado cai para def.
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

// envBool lê um default de flag booleana do ambiente (botão estilo Railway),
// aceitando as grafias verdadeiras usuais; um valor ausente ou malformado cai
// para def.
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

// addrOrNone renderiza um valor de flag vazio como "(none)" para o log de boot
// ficar claro quando um endereço opcional (dbServer/binServer/content) não está definido.
func addrOrNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

// newApp monta o mundo a partir do `config`: conteúdo, clientes gRPC, overlays,
// handler e população inicial. Não abre porta e não bloqueia no `Serve`.
func newApp(ctx context.Context, cfg config, logger *slog.Logger) (a *app, err error) {
	var closers []func()
	defer func() {
		if err != nil {
			closeAll(closers)
		}
	}()

	// Ecoa a ligação efetiva no boot: a versão de cliente e os endereços
	// resolvidos de dbServer/binServer são os botões mais frequentemente mal
	// configurados num deploy em container (drops por version-mismatch, ou
	// "produced zero addresses" quando um hostname interno está errado), então
	logger.Info("tmserver config",
		"client_version", cfg.clientVersion,
		"dbserver", addrOrNone(cfg.dbAddr),
		"binserver", addrOrNone(cfg.binAddr),
		"content", addrOrNone(cfg.contentDir),
		"mob_stat_editing", cfg.mobStatEditing,
		"teleport_editing", cfg.teleportEditing,
		"item_effect_editing", cfg.itemEffectEditing,
		"rates_editing", cfg.ratesEditing,
		"mob_carry_editing", cfg.mobCarryEditing)

	// Com -content definido, carregue e valide a árvore de conteúdo de antemão para
	// que um mount ausente/corrompido falhe rápido em vez de aparecer no meio da
	// sessão. As semânticas receita→família de combine e dos bits do AttributeMap
	// seguem UNVERIFIED (PROGRESS Fase 5), então isto valida e expõe os dados; não
	// religa gameplay sobre mapeamentos não verificados.
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
	// Tetos do refino com pó (QA 05/11): saíram do código para
	// Common/Settings/RefineRate.txt, para um moderador decidir até onde cada pó vai.
	// Um arquivo ausente/quebrado não é fatal — o arquivo é superfície de ajuste e o
	// mundo mantém o legado 6/9/10/11 em vez de recusar o boot.
	refinePolicy := content.DefaultRefinePolicy()
	if cfg.contentDir != "" {
		if p, perr := content.LoadRefinePolicy(cfg.contentDir + "/Common/Settings/RefineRate.txt"); perr != nil {
			logger.Warn("refine policy: legacy defaults", "err", perr)
		} else {
			refinePolicy = p
		}
	}
	logger.Info("refine policy", "po_max", refinePolicy.OriMax, "pl_max", refinePolicy.PLMax,
		"gem_level", refinePolicy.GemLevel, "hard_cap", refinePolicy.HardCap,
		"server_index", cfg.serverIndex)

	var sancRate *content.SancRate
	var compRate *content.CompRate
	var questRates *content.QuestRates
	// Taxas, mapas e a página de status continuam em arquivo (issues 59/60).
	// item_catalog, skill_catalog, npc_template e world_item_seed vêm do banco:
	// os CSV e o npc/ saíram do conteúdo na fase 5 da issue 58.
	if cfg.contentDir != "" {
		c, err := loadContent(cfg.contentDir, logger)
		if err != nil {
			return nil, err
		}
		heights = c.heights
		attrs = c.attrs
		sancRate = c.sanc
		compRate = c.comp
		questRates = c.quests
	}

	metricsReg := metrics.New("tmserver")
	go metrics.Serve(ctx, cfg.metricsAddr, metricsReg, logger)

	clientCreds, err := secure.ClientCreds(secure.Config{
		CertFile: cfg.tlsCert, KeyFile: cfg.tlsKey, CAFile: cfg.tlsCA, ServerName: cfg.tlsServerName,
	})
	if err != nil {
		return nil, err
	}

	// Persistência: adaptador real de dbServer quando -dbserver está definido, senão no-op.
	// A conexão é mantida (dbConn) para a sobreposição de config de NPC compartilhá-la.
	var persist world.Persistence = world.NopPersistence{}
	var dbConn *grpc.ClientConn
	var worldEvents worldcfg.Source
	if cfg.dbAddr != "" {
		// O interceptor cliente injeta o trace_id do contexto (sessão/login) no
		// metadata de toda RPC, então o dbServer registra o mesmo id (OBS-02).
		conn, err := grpc.NewClient(cfg.dbAddr,
			grpc.WithTransportCredentials(clientCreds),
			grpc.WithChainUnaryInterceptor(metrics.UnaryClientInterceptor()))
		if err != nil {
			return nil, err
		}
		closers = append(closers, func() { _ = conn.Close() })
		dbConn = conn
		persist = dbclient.New(conn)
		worldEvents = dbclient.NewWorldEventConfig(conn)
		logger.Info("dbServer wired", "addr", cfg.dbAddr)
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
			return nil, fmt.Errorf("load content catalogs from the database: %w", cerr)
		}
		if len(cat.Items) == 0 {
			return nil, fmt.Errorf("item_catalog is empty: seed the content catalogue (tmserver/cmd/importcontent or seedgen)")
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

	// O cliente busca uma página de status de canal por HTTP antes do
	// connect CPSock; sirva-a da árvore de conteúdo quando disponível.
	var statusFile string
	var baseMobs map[int][]byte
	var summonMobs [][]byte
	var vineMob []byte
	var castleQuests []content.CastleQuest
	if cfg.contentDir != "" {
		statusFile = filepath.Join(cfg.contentDir, "Common", "serv00.htm")
		if bm, err := content.LoadBaseMobs(cfg.contentDir); err != nil {
			logger.Warn("base mob templates not loaded", "err", err)
		} else {
			baseMobs = bm
			logger.Info("base mob templates loaded", "classes", len(baseMobs))
		}
		if sm, missing, err := content.LoadBaseSummons(cfg.contentDir); err != nil {
			logger.Warn("base summon templates not loaded (BM evocations disabled)", "err", err)
		} else {
			summonMobs = sm
			if len(missing) > 0 {
				logger.Warn("optional summon templates missing", "names", missing)
			}
			logger.Info("base summon templates loaded", "count", len(summonMobs)-len(missing))
		}
		// Muro de Espinhos é CreateMob("Vinha", ..., "Boss", 3) no C++,
		// mas a árvore de conteúdo Linux distribuída traz o template do muro como npc/Vine.
		if vm, ok := catalogTemplates["Vine"]; ok {
			vineMob = vm
			logger.Info("vine template loaded", "source", "database")
		} else {
			logger.Warn("vine template not loaded (skill 98 cannot create Muro de Espinhos)",
				"reason", "absent from npc_template")
		}
		if cq, err := content.LoadCastleQuests(filepath.Join(cfg.contentDir, "Common", "Settings", "CastleQuest.txt")); err != nil {
			logger.Warn("castle quests not loaded (Castle/Zakum disabled)", "err", err)
		} else {
			castleQuests = cq
			logger.Info("castle quests loaded", "count", len(castleQuests))
		}
	}

	// Sobreposição de stats de template de mob/NPC do moderador (mob-template-editing-plan.md, o
	// sucessor equivalente da ferramenta legada EDITAPPMOB): o único interruptor é
	// -mob-stat-editing (W2PP_MOB_STAT_EDITING), desligado por default. A
	// sobreposição de stats se aplica a QUALQUER arquivo npc/<template_name> — monstros
	// incluídos, o caso de uso principal de rebalanceamento da ferramenta — não só ao
	// subconjunto de mercadores gerenciado pelo banco, então é de propósito independente do
	// catálogo de NPCs (fonte própria, checagem de dependência própria). Buscada UMA vez
	// aqui, de forma síncrona, antes de qualquer template ser carregado abaixo: não há
	// hot-reload para este recurso, igual ao comportamento de reiniciar-para-aplicar
	// do EDITAPPMOB (ele escrevia o arquivo; o servidor também só o lia no boot).
	var mobStatOverrides map[string]mobstat.Override
	if cfg.mobStatEditing {
		if dbConn == nil {
			return nil, fmt.Errorf("-mob-stat-editing requires -dbserver (config source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		overrides, ferr := dbclient.NewMobStatSource(dbConn).Fetch(fetchCtx)
		cancel()
		if ferr != nil {
			return nil, fmt.Errorf("fetch mob template stat overrides: %w", ferr)
		}
		mobStatOverrides = overrides
		logger.Info("mob template stat overlay enabled (moderator editing)", "overrides", len(mobStatOverrides))
	}

	// Sobreposição de Carry[] do moderador (issue 34, fatia 3): a tabela de drop de cada
	// template de mob. Mesma forma da sobreposição de stats — buscada uma vez, antes de
	// qualquer template ser carregado, porque a tabela está gravada nos bytes de onde o
	// mundo gera as entidades e não há hot-reload. -mob-carry-editing
	// (W2PP_MOB_CARRY_EDITING), desligado por default para que um banco não semeado nunca mude um drop.
	var mobCarryOverrides map[string]mobcarry.Override
	if cfg.mobCarryEditing {
		if dbConn == nil {
			return nil, fmt.Errorf("-mob-carry-editing requires -dbserver (overlay source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		overrides, ferr := dbclient.NewMobCarrySource(dbConn).Fetch(fetchCtx)
		cancel()
		if ferr != nil {
			return nil, fmt.Errorf("fetch mob carry overrides: %w", ferr)
		}
		mobCarryOverrides = overrides
		logger.Info("mob carry overlay enabled (moderator editing)", "templates", len(mobCarryOverrides))
	}

	// Catálogo de spawn de NPCs (issue 57): npc_definition no Postgres é a ÚNICA
	// fonte de blocos de spawn E dos templates (npc_template), então -dbserver é
	// a única dependência — NPCGener.txt e npc/ saíram do conteúdo na fase 5.
	// O tmServer registra um slot de gerador por bloco de conteúdo e
	// materializa as entidades. Sem -dbserver o mundo sobe sem NPCs,
	// o que a topologia de dev pressupõe — daí um aviso alto, não um erro fatal.
	var npcConfig npccfg.Source
	if dbConn == nil {
		logger.Warn("no NPC catalogue: -dbserver is required, the world will boot with no monsters",
			"dbserver", addrOrNone(cfg.dbAddr))
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

	// Sobreposição de teleporte do moderador (issue 42): o único interruptor é
	// -teleport-editing (W2PP_TELEPORT_EDITING). Diferente da sobreposição de NPC, esta
	// não gera nem materializa nada — a resolução de rota já acontece em
	// world.teleportDest, então a sobreposição é só um snapshot somente leitura trocado
	// atomicamente. Um poll de 15s mantém um mundo rodando em sincronia com a web.
	if cfg.teleportEditing {
		if dbConn == nil {
			return nil, fmt.Errorf("-teleport-editing requires -dbserver (config source)")
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
			return nil, fmt.Errorf("fetch teleport overlay: %w", err)
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

	// Semeia o RNG de eventos do mundo a partir do relógio de parede para a sequência
	// de clima diferir entre boots (handler.worldEventRNGSeed explica por que a semente
	// fixa é só para testes). Zero significa "usa a semente fixa", então mantenha fora da faixa.
	eventSeed := uint32(time.Now().UnixNano())
	if eventSeed == 0 {
		eventSeed = 1
	}
	// Sobreposição de taxas de conteúdo do moderador (issue 36): o lado de leitura
	// de /config. Mesma forma da sobreposição de efeito de item — uma flag, aplicada uma
	// vez no boot, porque SancRate/CompRate/QuestRates são snapshots somente leitura
	// do momento em que o mundo começa (uma edição vale no próximo restart).
	if cfg.ratesEditing {
		if dbConn == nil || cfg.contentDir == "" {
			return nil, fmt.Errorf("-rates-editing requires both -dbserver (overlay source) and -content (rate files)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rows, ferr := dbclient.NewContentRateConfig(dbConn).Overrides(fetchCtx)
		cancel()
		if ferr != nil {
			return nil, fmt.Errorf("fetch content rate overlay: %w", ferr)
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
		// "content loaded" roda antes deste bloco, então a linha Ori efetiva é
		// registrada aqui: o operador precisa ver o número que o mundo vai usar,
		// não só quantas linhas foram buscadas.
		if sancRate != nil {
			logger.Info("content rate overlay effective", "sancrate_ori", sancRow(sancRate, 0))
		}
	}

	// Sobreposição de taxa de drop do moderador (issue 34): /drops/editar escreve
	// drop_rate_override; o mundo lê DropRate[64] do momento em que começa,
	// então isto é aplicado UMA vez aqui. Só valores — nenhuma chamada de rand() é
	// adicionada, então a paridade de RNG fica intocada.
	if cfg.dropConfigEditing {
		if dbConn == nil {
			return nil, fmt.Errorf("-drop-config-editing requires -dbserver (overlay source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rows, ferr := dbclient.NewDropConfig(dbConn).Overrides(fetchCtx)
		cancel()
		if ferr != nil {
			return nil, fmt.Errorf("fetch drop rate overlay: %w", ferr)
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

	// Sobreposição de efeito base de item do moderador (issue 44): o lado de leitura
	// do editor web em /itens/[index]. O único interruptor é -item-effect-editing
	// (W2PP_ITEM_EFFECT_EDITING), desligado por default e — como -mob-stat-editing —
	// aplicado UMA vez aqui, logo antes do handler ser construído: ItemEffects é um
	// snapshot imutável dobrado no score, então um mundo rodando mantém o que
	// carregou no boot. A precedência não muda: o stEffect/stValue da instância do
	// item ainda vence o catálogo (aqui, conteúdo + sobreposição) na hora do score.
	if cfg.itemEffectEditing {
		if dbConn == nil {
			return nil, fmt.Errorf("-item-effect-editing requires -dbserver (catalog and overlay source)")
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		rows, ferr := dbclient.NewItemEffectConfig(dbConn).Effects(fetchCtx)
		cancel()
		if ferr != nil {
			return nil, fmt.Errorf("fetch item effect overlay: %w", ferr)
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

	// Snapshot das flags de processo (issue 36, aba "Flags"): o portal mostra com o que
	// este processo subiu, somente leitura. Essas flags vivem no comando do compose e
	// no ambiente W2PP_*; a web nunca as edita, então um restart é o único
	// jeito de mudar uma. Uma falha aqui não pode parar o servidor — o snapshot é
	// observabilidade, não gameplay.
	if dbConn != nil {
		snapshot := []dbclient.InstanceFlag{
			{Name: "addr", Value: cfg.addr, RestartRequired: true, Hint: "porta CPSock do cliente (docker-compose command)"},
			{Name: "status-addr", Value: cfg.statusAddr, RestartRequired: true, Hint: "HTTP de status (serv00.htm) que o cliente sonda"},
			{Name: "client-version", Value: strconv.Itoa(cfg.clientVersion), RestartRequired: true, Hint: "versão aceita no handshake; divergente = 0x102"},
			{Name: "dbserver", Value: cfg.dbAddr, RestartRequired: true, Hint: "persistência; vazio = no-op"},
			{Name: "binserver", Value: cfg.binAddr, RestartRequired: true, Hint: "billing; vazio = allow-all"},
			{Name: "content", Value: cfg.contentDir, RestartRequired: true, Hint: "árvore de conteúdo (taxas, mapas, BaseMob/BaseSummon)"},
			{Name: "double-exp", Value: strconv.FormatBool(cfg.doubleExp), RestartRequired: true, Hint: "DOUBLEMODE: dobra EXP PvE"},
			{Name: "newbie-event", Value: strconv.FormatBool(cfg.newbieEvent), RestartRequired: true, Hint: "NewbieEventServer: +15% EXP e bônus abaixo do nível 100"},
			{Name: "kefra-live", Value: strconv.FormatBool(cfg.kefraLive), RestartRequired: true, Hint: "evento do Kefra ativo"},
			{Name: "training-field-exp", Value: strconv.FormatInt(int64(cfg.trainingFieldExp), 10), RestartRequired: true, Hint: "multiplicador de EXP do Campo de Treino"},
			{Name: "novice-channel", Value: strconv.Itoa(cfg.noviceChannel), RestartRequired: true, Hint: "canal verde (novato) fixo, 1-based; 0 = nenhum"},
			{Name: "max-nightmare", Value: strconv.Itoa(cfg.maxNightmare), RestartRequired: true, Hint: "admissões contadas de Nightmare por mapa/ciclo"},
			{Name: "mob-stat-editing", Value: strconv.FormatBool(cfg.mobStatEditing), RestartRequired: true, Hint: "overlay de stats de mob"},
			{Name: "mob-carry-editing", Value: strconv.FormatBool(cfg.mobCarryEditing), RestartRequired: true, Hint: "overlay do Carry[] (drop) dos templates"},
			{Name: "teleport-editing", Value: strconv.FormatBool(cfg.teleportEditing), RestartRequired: true, Hint: "overlay de rotas de teleporte (relido a cada 15 s)"},
			{Name: "item-effect-editing", Value: strconv.FormatBool(cfg.itemEffectEditing), RestartRequired: true, Hint: "overlay de efeitos base de item"},
			{Name: "rates-editing", Value: strconv.FormatBool(cfg.ratesEditing), RestartRequired: true, Hint: "overlay de taxas (refino/combine/quests)"},
			{Name: "drop-config-editing", Value: strconv.FormatBool(cfg.dropConfigEditing), RestartRequired: true, Hint: "overlay de taxa de drop por slot"},
		}
		flagCtx, cancelFlags := context.WithTimeout(ctx, 10*time.Second)
		err := dbclient.NewInstanceFlags(dbConn).Record(flagCtx, "tmserver", snapshot)
		cancelFlags()
		if err != nil {
			logger.Warn("record instance flags", "err", err)
		}
	}

	dispatch := handler.New(handler.Config{
		MaxNightmare:          cfg.maxNightmare,
		KefraEntriesPerScroll: cfg.kefraEntriesPerScroll,
		Log:                   logger, ClientVersion: int32(cfg.clientVersion), BaseMobs: baseMobs, SummonMobs: summonMobs, VineMob: vineMob, ItemPrices: itemPrices, ItemEffects: itemEffects, ItemReqs: itemReqs,
		ItemVolatiles: itemVolatiles, ItemPos: itemPos, ItemUnique: itemUnique, ItemGrades: itemGrades, ItemExtra: itemExtra, Spells: spells, Heights: heights, Attrs: attrs,
		SancRate:         sancRate,
		RefinePolicy:     refinePolicy,
		ServerIndex:      cfg.serverIndex,
		NoviceChannel:    cfg.noviceChannel,
		CrossoverKey:     cfg.crossoverKey,
		ExpEvents:        level.ExpEvents{DoubleMode: cfg.doubleExp, NewbieEvent: cfg.newbieEvent, KefraLive: cfg.kefraLive},
		TrainingFieldExp: int64(cfg.trainingFieldExp),
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
			ScalePct: cfg.affectScalePct,
			MinTicks: handler.AffectTicksFromSeconds(cfg.affectMinSeconds),
			MaxTicks: handler.AffectTicksFromMinutes(cfg.affectMaxMinutes),
		},
	})
	w := world.New(world.Config{
		RejectChecksum: cfg.rejectChecksum,
		MaxMsgPerSec:   cfg.maxMsgPerSec,
		MsgBurst:       cfg.msgBurst,
		StatusFile:     statusFile,
		ItemRanges:     itemRanges,
		LogSends:       cfg.logSends,
		Metrics:        metricsReg,
	}, logger, persist, dispatch.Handle)
	// Pulso da IA de mobs: monstros adquirem/perseguem/atacam corpo a corpo jogadores próximos a cada tick (mobai.go).
	w.SetTickHandler(world.DefaultMobTick, dispatch.Tick)
	// Desmontagem de party ao desconectar: desvincula o laço de party e remove as
	// invocações antes do slot ser liberado (party.go SessionEnd).
	w.SetSessionEndHandler(dispatch.SessionEnd)
	// A flag de newbie tem dois donos: os ExpEvents do dispatcher (o bônus de EXP)
	// e o world (o handicap de HP dos spawns abaixo de 120). Definido aqui, ANTES da
	// população de boot abaixo, para que os spawns também sofram o handicap — a config
	// do portal pode sobrescrevê-la depois via ApplyWorldEventConfigBoot.
	w.SetNewbieEvent(cfg.newbieEvent)

	// Gate de billing: adaptador real de binServer quando -binserver está definido, senão allow-all.
	if cfg.binAddr != "" {
		conn, err := grpc.NewClient(cfg.binAddr,
			grpc.WithTransportCredentials(clientCreds),
			grpc.WithChainUnaryInterceptor(metrics.UnaryClientInterceptor()))
		if err != nil {
			return nil, err
		}
		closers = append(closers, func() { _ = conn.Close() })
		w.SetBilling(binclient.New(conn))
		logger.Info("binServer wired", "addr", cfg.binAddr)
	}
	if len(worldItems) > 0 {
		seedWorldItems(w, worldItems, logger)
	}
	if npcConfig != nil {
		// Repete enquanto o dbServer ainda está subindo; ctx aborta a espera em
		// SIGTERM em vez de segurar o processo por todo o orçamento de retentativas.
		snap, err := dispatch.FetchNPCConfig(ctx)
		if err != nil {
			return nil, err
		}
		auditMobTemplates(snap.Defs, logger)
		registerNPCGenerators(w, snap.Defs, attrs, logger)
		if err := dispatch.ApplyNPCConfigBoot(w, snap); err != nil {
			return nil, err
		}
	}
	if worldEvents != nil {
		dispatch.ApplyWorldEventConfigBoot(w)
	}
	dispatch.ApplyGuildStateBoot(w)

	return &app{
		w:          w,
		addr:       cfg.addr,
		statusAddr: cfg.statusAddr,
		statusFile: statusFile,
		mtls:       cfg.tlsCert != "",
		closers:    closers,
	}, nil
}

// run abre o servidor de status e o listener CPSock e serve até o ctx ser
// cancelado. As conexões gRPC abertas no `newApp` são fechadas na saída.
func (a *app) run(ctx context.Context, logger *slog.Logger) error {
	// Servidor HTTP de status de canal em porta própria (o WYD real serve serv00.htm
	// na :80, separado do :8281 do jogo — general-config.h da referência
	// snalmir). O cliente sonda o status aqui e então abre a conexão
	// CPSock do jogo na porta do jogo; mantê-las separadas evita o cliente ver um
	// servidor HTTP na porta do jogo. Fica em `run` para que montar o app não escute.
	if a.statusAddr != "" {
		go serveStatusHTTP(ctx, a.statusAddr, a.statusFile, logger)
	}
	ln, err := net.Listen("tcp", a.addr)
	if err != nil {
		closeAll(a.closers)
		return err
	}
	logger.Info("tmserver listening", "addr", a.addr, "mtls", a.mtls)

	err = a.w.Serve(ctx, ln)
	closeAll(a.closers)
	return err
}

// closeAll fecha as conexões gRPC na ordem inversa da abertura.
func closeAll(closers []func()) {
	for i := len(closers) - 1; i >= 0; i-- {
		closers[i]()
	}
}

// seedWorldItems gera os objetos estáticos do mundo (portões/portas) a partir de
// world_item_seed antes de Serve iniciar o loop, para a atribuição de id ser
// single-threaded e seguir a ordem de importação (casando com a sequência
// CreateItem legada que o cliente espera). Um catálogo vazio não semeia nada.
func seedWorldItems(w *world.World, items []content.InitItem, logger *slog.Logger) {
	seeded := 0
	for _, it := range items {
		// Portões nascem abertos (paridade com CreateItem); o trancamento chega com os
		// sistemas de evento/timer adiados. SeedWorldItem devolve -1 só quando a tabela de ids está cheia.
		if id := w.SeedWorldItem(world.Item{Index: it.Index}, it.PosX, it.PosY, world.StateOpen); id >= 0 {
			seeded++
		}
	}
	logger.Info("world items seeded", "count", seeded)
}

// registerNPCGenerators monta os slots de gerador do mundo a partir do catálogo
// de spawn (issue 57) em vez de ler NPCGener.txt no boot. O slot i é o bloco de
// conteúdo i, exatamente a ordem de arquivo que o servidor legado usava, para as faixas de índice que o cliente já conhece continuarem
// classify event blocks (IsWaterGenerator, NightmareGenerator, rune quest) keep
// funcionando. Um bloco que o importador pulou (template ausente/indecodificável) deixa seu
// slot nil, e blocos de propriedade de evento são registrados mas ficam para o código do evento.
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

// auditMobTemplates avisa uma vez por template cuja recompensa de kill é zero ou além
// do gate legado de premiação (10M, MobKilled.cpp:1284): a árvore de conteúdo precisa ser
// restampada com cmd/exptool ou os jogadores não ganham nada nessas kills (issue #43).
// A checagem veio do loader de boot do NPCGener.txt, que não existe mais (issue #57).
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

// serveStatusHTTP roda o servidor web de status de canal (serv00.htm). Ele responde
// qualquer caminho com a página de status para o GET do cliente passar independente do
// arquivo exato pedido. O corpo é lido por requisição para poder ser editado ao vivo.
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

// loadContent carrega e valida a árvore de conteúdo tmserver/content/ (loaders da Fase 5).
// As taxas e catálogos são obrigatórios (um mount quebrado é erro fatal); os mapas
// são grandes e opcionais (aviso quando ausentes). Registra o que foi carregado para o
// operador confirmar que o mount está correto.
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
	// As taxas Ori/Lac merecem ser registradas explicitamente: o SancRate.txt só
	// sobrescreve os índices que lista, então um mount quebrado deixa os defaults
	// compilados em silêncio e o operador não notaria.
	logger.Info("content loaded",
		"comprate_families", comp.Families(),
		"sancrate_ori", sancRow(sanc, 0), "sancrate_lac", sancRow(sanc, 1))

	// Mapas são opcionais: 17 MiB de HeightMap + 1 MiB de AttributeMap não são
	// necessários para aceitar logins; avise em vez de falhar quando não estiverem
	// montados. Quando ambos carregam, grave os blocos de atributo no grid de altura
	// uma vez (o BASE_ApplyAttribute de boot) — o resultado dirige o pathfinding dos
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

// loadedContent é o que uma árvore tmserver/content/ montada rende. É uma struct em
// vez de uma lista de retorno só porque a lista tinha crescido além da legibilidade.
type loadedContent struct {
	comp    *content.CompRate
	sanc    *content.SancRate
	quests  *content.QuestRates
	heights *content.Grid
	attrs   *content.Grid // raw AttributeMap (training-grounds bit); nil when not mounted
}

// sancRow renderiza a linha de taxa de uma bigorna para o log de boot.
func sancRow(s *content.SancRate, anvil int) []int {
	row := make([]int, 0, 12)
	for i := 0; i < 12; i++ {
		row = append(row, s.Rate(anvil, i))
	}
	return row
}
