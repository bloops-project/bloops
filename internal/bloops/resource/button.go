package resource

import (
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	// common menu button text
	CreateButtonText  = "Создать игру"
	LeaveButtonText   = emoji.ChequeredFlag.String() + " Выйти"
	StartButtonText   = emoji.Rocket.String() + " Начать"
	JoinButtonText    = "Присоединиться к игре"
	RatingButtonText  = emoji.Star.String() + " Таблица лидеров"
	RuleButtonText    = "Правила"
	ProfileButtonText = emoji.Alien.String() + " Профиль"

	// builder inline button text
	InlineNextText = "Далее"
	InlinePrevText = "Назад"
	InlineDoneText = emoji.ChequeredFlag.String() + " Завершить"
)
var (
	// keyboard buttons
	CreateButton  = tgbotapi.NewKeyboardButton(CreateButtonText)
	JoinButton    = tgbotapi.NewKeyboardButton(JoinButtonText)
	LeaveButton   = tgbotapi.NewKeyboardButton(LeaveButtonText)
	StartButton   = tgbotapi.NewKeyboardButton(StartButtonText)
	RatingButton  = tgbotapi.NewKeyboardButton(RatingButtonText)
	RulesButton   = tgbotapi.NewKeyboardButton(RuleButtonText)
	ProfileButton = tgbotapi.NewKeyboardButton(ProfileButtonText)

	CommonButtons = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(CreateButton),
		tgbotapi.NewKeyboardButtonRow(JoinButton),
		tgbotapi.NewKeyboardButtonRow(RulesButton, ProfileButton),
	)
	LeaveMenuButton = tgbotapi.NewKeyboardButton(LeaveButtonText)
)
