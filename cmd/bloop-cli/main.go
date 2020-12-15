package main

import (
	"bloop/internal/bloop"
	"bloop/internal/cache/hashlru"
	"bloop/internal/logging"
	"bloop/internal/shutdown"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	lru "github.com/hashicorp/golang-lru"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
)

func main() {
	_, _ = fmt.Fprint(os.Stdout, bloop.Graffiti)
	_, _ = fmt.Fprintf(os.Stdout, bloop.GreetingCLI, bloop.ProjectName, bloop.ProjectVersion, bloop.TgBloopUrl, bloop.GithubBloopUrl)

	ctx, done := shutdown.New()
	logger := logging.FromContext(ctx)
	defer done()
	config := bloop.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logger.Fatal(fmt.Errorf("processing the config: %v", err))
	}

	var token string
	fmt.Println("Enter your bot token:")
	for {
		_, err := fmt.Scanf("%s\n", &token)
		if err != nil {
			if err.Error() == "unexpected newline" {
				continue
			}
			logger.Fatal(fmt.Errorf("read token: %v", err))
		}
		break
	}
	_, _ = fmt.Fprint(os.Stdout, "token received: ", token, "\n")
	config.Token = token
	if config.Token == "" {
		logger.Fatalf(
			"Bot token not found, please visit %s to register your bot and get a token",
			bloop.BotFatherUrl,
		)
	}

	tg, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		logger.Fatal(fmt.Errorf("bot api: %v", err))
	}

	tg.Debug = config.Debug
	_, _ = fmt.Fprint(os.Stdout, "Authorization in telegram was successful: ", tg.Self.UserName, "\n")
	arc, err := lru.NewARC(config.CacheUsersNum)
	if err != nil {
		log.Fatal(fmt.Errorf("new arc cache: %v", err))
	}

	cache := hashlru.NewLRU(arc)
	manager := bloop.NewManager(tg, cache, &config)
	if err := manager.Run(ctx); err != nil {
		log.Fatal(fmt.Errorf("run: %v", err))
	}
}
