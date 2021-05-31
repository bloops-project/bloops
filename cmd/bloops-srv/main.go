package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/bloops-games/bloops/internal/buildinfo"

	"github.com/bloops-games/bloops/internal/cache"

	"github.com/bloops-games/bloops/internal/bloopsbot"
	"github.com/bloops-games/bloops/internal/database"
	stateDb "github.com/bloops-games/bloops/internal/database/matchstate/database"
	statDb "github.com/bloops-games/bloops/internal/database/stat/database"
	userdb "github.com/bloops-games/bloops/internal/database/user/database"
	"github.com/bloops-games/bloops/internal/logging"
	"github.com/bloops-games/bloops/internal/server"
	"github.com/bloops-games/bloops/internal/shutdown"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kelseyhightower/envconfig"
)

var version string

func main() {
	_, _ = fmt.Fprint(os.Stdout, buildinfo.Graffiti)
	_, _ = fmt.Fprintf(
		os.Stdout,
		buildinfo.GreetingCLI,
		buildinfo.ProjectName,
		version,
		buildinfo.TgBloopURL,
		buildinfo.GithubBloopURL,
	)

	ctx, done := shutdown.New()
	defer done()
	config := bloopsbot.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logging.DefaultLogger().Fatalf("processing the config: %w", err)
	}

	logger := logging.NewLogger(config.Debug)

	if err := realMain(ctx, config, done); err != nil {
		logger.Fatalf("main.realMain: %v", err)
	}
}

func realMain(ctx context.Context, config bloopsbot.Config, done func()) error {
	logger := logging.FromContext(ctx).Named("main.realMain")
	if config.BotToken == "" {
		return fmt.Errorf(
			"bot token not found, please visit %s to register your bot and get a token",
			buildinfo.BotFatherURL,
		)
	}

	tg, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return fmt.Errorf("bot api: %w", err)
	}

	tg.Debug = config.Debug

	_, _ = fmt.Fprint(os.Stdout, "Authorization in telegram was successful: ", tg.Self.UserName, "\n")

	db, err := database.NewFromEnv(ctx, &config.DB)
	if err != nil {
		return fmt.Errorf("new database from env: %w", err)
	}

	defer db.Close(ctx)

	userCache, err := cache.NewLRU(config.CacheSize)
	if err != nil {
		return fmt.Errorf("can not create lru cache: %w", err)
	}

	statCache, err := cache.NewLRU(config.CacheSize)
	if err != nil {
		return fmt.Errorf("can not create lru cache: %w", err)
	}

	srv, err := server.New(config.Port)
	if err != nil {
		return fmt.Errorf("server.New: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", server.HandleHealth(ctx))

	go func() {
		if err := srv.ServeHTTP(ctx, &http.Server{Handler: mux}); err != nil {
			logger.Fatalf("srv.ServeHTTP: %v", err)
			done()
		}
	}()

	go func() {
		if err := http.ListenAndServe(":"+config.ProfPort, nil); err != nil {
			logger.Fatalf("pprof default sever: %v", err)
			done()
		}
	}()

	manager := bloopsbot.NewManager(tg, &config, userdb.New(db, userCache), statDb.New(db, statCache), stateDb.New(db))
	if err := manager.Run(ctx); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
