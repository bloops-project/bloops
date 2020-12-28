package main

import (
	"bloop/internal/bloopsbot"
	"bloop/internal/bloopsbot/resource"
	"bloop/internal/cache/cachelru"
	"bloop/internal/database"
	stateDb "bloop/internal/database/matchstate/database"
	statDb "bloop/internal/database/stat/database"
	userdb "bloop/internal/database/user/database"
	"bloop/internal/logging"
	"bloop/internal/server"
	"bloop/internal/shutdown"
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kelseyhightower/envconfig"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {
	_, _ = fmt.Fprint(os.Stdout, resource.Graffiti)
	_, _ = fmt.Fprintf(
		os.Stdout,
		resource.GreetingCLI,
		resource.ProjectName,
		resource.ProjectVersion,
		resource.TgBloopUrl,
		resource.GithubBloopUrl,
	)

	ctx, done := shutdown.New()
	defer done()
	logger := logging.FromContext(ctx)
	if err := realMain(ctx, done); err != nil {
		logger.Fatalf("main.realMain: %v", err)
	}
}

func realMain(ctx context.Context, done func()) error {
	logger := logging.FromContext(ctx)
	config := bloopsbot.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logger.Fatalf("processing the config: %v", err)
	}

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
