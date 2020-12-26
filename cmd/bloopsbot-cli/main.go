package main

import (
	"bloop/internal/bloopsbot"
	"bloop/internal/bloopsbot/resource"
	"bloop/internal/cache/cachelru"
	"bloop/internal/database"
	statDb "bloop/internal/database/stat/database"
	userdb "bloop/internal/database/user/database"
	"bloop/internal/logging"
	"bloop/internal/shutdown"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kelseyhightower/envconfig"
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
	logger := logging.FromContext(ctx)
	defer done()
	config := bloopsbot.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logger.Fatalf("processing the config: %v", err)
	}

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
		break
	}

	_, _ = fmt.Fprint(os.Stdout, "token received: ", token, "\n")
	config.Token = token
	if config.Token == "" {
		logger.Fatalf(
			"Bot token not found, please visit %s to register your bot and get a token",
			resource.BotFatherUrl,
		)
	}

	tg, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		logger.Fatalf("bot api: %v", err)
	}

	tg.Debug = config.Debug
	_, _ = fmt.Fprint(os.Stdout, "Authorization in telegram was successful: ", tg.Self.UserName, "\n")
	db, err := database.NewFromEnv(ctx, &config.Db)
	if err != nil {
		logger.Fatalf("new database from env: %v", err)
	}

	defer db.Close(ctx)
	userCache, err := cachelru.NewLRU(config.CacheSize)
	if err != nil {
		logger.Fatalf("can not create lru cache: %v", err)
	}

	statCache, err := cachelru.NewLRU(config.CacheSize)
	if err != nil {
		logger.Fatalf("can not create lru cache: %v", err)
	}

	manager := bloopsbot.NewManager(tg, &config, userdb.New(db, userCache), statDb.New(db, statCache))
	if err := manager.Run(ctx); err != nil {
		logger.Fatalf("run: %v", err)
	}
}
