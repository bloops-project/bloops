package main

import (
	"bloop/internal/bloop"
	"bloop/internal/cache/cachelru"
	"bloop/internal/logging"
	"bloop/internal/shutdown"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	lru "github.com/hashicorp/golang-lru"
	"github.com/kelseyhightower/envconfig"
	"os"
)

func main() {
	_, _ = fmt.Fprint(os.Stdout, bloop.Graffiti)
	_, _ = fmt.Fprintf(
		os.Stdout,
		bloop.GreetingCLI,
		bloop.ProjectName,
		bloop.ProjectVersion,
		bloop.TgBloopUrl,
		bloop.GithubBloopUrl,
	)

	ctx, done := shutdown.New()
	logger := logging.FromContext(ctx)
	defer done()
	config := bloop.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logger.Fatalf("processing the config: %v", err)
	}

	if config.Token == "" {
		logger.Fatalf(
			"Bot token not found, please visit %s to register your bot and get a token",
			bloop.BotFatherUrl,
		)
	}

	tg, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		logger.Fatalf("bot api: %v", err)
	}

	tg.Debug = config.Debug
	_, _ = fmt.Fprint(os.Stdout, "Authorization in telegram was successful: ", tg.Self.UserName, "\n")
	arc, err := lru.NewARC(config.CacheUsersNum)
	if err != nil {
		logger.Fatalf("new arc cache: %v", err)
	}

	cache := cachelru.NewLRU(arc)
	manager := bloop.NewManager(tg, cache, &config)
	if err := manager.Run(ctx); err != nil {
		logger.Fatalf("run: %v", err)
	}
}
