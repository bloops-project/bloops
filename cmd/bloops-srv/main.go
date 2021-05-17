package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/bloops-games/bloops/internal/bloopsbot"
	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	"github.com/bloops-games/bloops/internal/cache/cachelru"
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
	_, _ = fmt.Fprint(os.Stdout, resource.Graffiti)
	_, _ = fmt.Fprintf(
		os.Stdout,
		resource.GreetingCLI,
		resource.ProjectName,
		version,
		resource.TgBloopUrl,
		resource.GithubBloopUrl,
	)

	ctx, done := shutdown.New()
	defer done()
	config := bloopsbot.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logging.DefaultLogger().Fatalf("processing the config: %v", err)
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
			resource.BotFatherUrl,
		)
	}

	tg, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return fmt.Errorf("bot api: %v", err)
	}

	tg.Debug = config.Debug

	_, _ = fmt.Fprint(os.Stdout, "Authorization in telegram was successful: ", tg.Self.UserName, "\n")

	db, err := database.NewFromEnv(ctx, &config.Db)
	if err != nil {
		return fmt.Errorf("new database from env: %v", err)
	}

	defer db.Close(ctx)

	userCache, err := cachelru.NewLRU(config.CacheSize)
	if err != nil {
		return fmt.Errorf("can not create lru cache: %v", err)
	}

	statCache, err := cachelru.NewLRU(config.CacheSize)
	if err != nil {
		return fmt.Errorf("can not create lru cache: %v", err)
	}

	srv, err := server.New(config.Port)
	if err != nil {
		return fmt.Errorf("server.New: %v", err)
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
		return fmt.Errorf("run: %v", err)
	}

	return nil
}
