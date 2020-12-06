package main

import (
	"bloop/internal/bloopmp"
	"bloop/internal/cache/hashlru"
	"bloop/internal/logging"
	"bloop/internal/shutdown"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	lru "github.com/hashicorp/golang-lru"
	"github.com/kelseyhightower/envconfig"
	"log"
)

func main() {
	ctx, done := shutdown.New()
	logger := logging.FromContext(ctx)
	defer done()
	config := bloopmp.Config{}
	if err := envconfig.Process("", &config); err != nil {
		logger.Fatal(err)
	}

	tg, err := tgbotapi.NewBotAPI("1497320412:AAE8dIr1RzaJfgIhmHJIq9mSg-QmnZCtFJA")
	if err != nil {
		logger.Fatal(err)
	}
	tg.Debug = true
	logger.Info("Authorized on account ", tg.Self.UserName)
	arc, err := lru.NewARC(256)
	if err != nil {
		log.Fatal(err)
	}

	cache := hashlru.NewLRU(arc)
	manager := bloopmp.NewManager(tg, cache, &config)
	if err := manager.Run(ctx); err != nil {
		log.Println(err)
	}
}
