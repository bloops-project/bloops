package main

import (
	"context"
	"fmt"
	"net/http"
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
		resource.TgBloopURL,
		resource.GithubBloopURL,
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
	_, _ = fmt.Fprint(os.Stdout, resource.Graffiti)
	_, _ = fmt.Fprintf(
		os.Stdout,
		resource.GreetingCLI,
		resource.ProjectName,
		version,
		resource.TgBloopURL,
		resource.GithubBloopURL,
	)

	var token string
	fmt.Println("Enter your bot token:")
	for {
		_, err := fmt.Scanf("%s\n", &token)
		if err != nil {
			if err.Error() == "unexpected newline" {
				continue
			}
			logger.Fatalf("read token: %v", err)
		}
		if token == "" {
			_, _ = fmt.Fprintf(os.Stdout, "bot token not found, please visit %s to register your bot and get a token",
				resource.BotFatherURL)
			continue
		}

		break
	}

	_, _ = fmt.Fprint(os.Stdout, "Token received: ", token, "\n")
	config.BotToken = token

	var username string
	fmt.Println("Enter admin username:")
	for {
		_, err := fmt.Scanf("%s\n", &username)
		if err != nil {
			if err.Error() == "unexpected newline" {
				continue
			}
			logger.Fatalf("read token: %v", err)
		}

		if username == "" {
			_, _ = fmt.Fprintf(os.Stdout, "username is empty, enter valid username: \n")
			fmt.Println("Enter admin username:")
			continue
		}

		break
	}

	_, _ = fmt.Fprint(os.Stdout, "Username received: ", username, "\n")
	config.BotToken = token

	if username == "" {
		return fmt.Errorf(
			"bot token not found, please visit %s to register your bot and get a token",
			resource.BotFatherURL,
		)
	}

	tg, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		if err.Error() == "Not Found" {
			_, _ = fmt.Fprintf(os.Stdout, "Bot token not found\n")
			return fmt.Errorf("bot token not found: %w", err)
		}
		return fmt.Errorf("bot api: %w", err)
	}

	tg.Debug = config.Debug

	_, _ = fmt.Fprint(os.Stdout, "Authorization in telegram was successful: ", tg.Self.UserName, "\n")

	db, err := database.NewFromEnv(ctx, &config.DB)
	if err != nil {
		return fmt.Errorf("new database from env: %w", err)
	}

	defer db.Close(ctx)

	userCache, err := cachelru.NewLRU(config.CacheSize)
	if err != nil {
		return fmt.Errorf("can not create lru cache: %w", err)
	}

	statCache, err := cachelru.NewLRU(config.CacheSize)
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
