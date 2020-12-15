package game

import (
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	commandCreateText = "Создать"
	commandStartText  = emoji.Rocket.String() + " Начать"
	commandJoinText   = "Присоединиться"
	commandRatingText = emoji.Star.String() + " Таблица лидеров"
)

var (
	createMenuButton  = tgbotapi.NewKeyboardButton(commandCreateText)
	joinMenuButton    = tgbotapi.NewKeyboardButton(commandJoinText)
	commonMenuButtons = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			createMenuButton,
			joinMenuButton,
		),
	)
)

var (
	textThumbUp                            = emoji.ThumbsUp.String()
	textThumbDown                          = emoji.ThumbsDown.String()
	textGameClosedMsg                      = "Игровая комната закрыта"
	textLeaderboardHeader                  = "*Результаты игры*\n\n"
	textLeaderboardMedal                   = "%d. %s*%s*, %d очков, раундов: %d\n"
	textLeaderboardLine                    = "%d. *%s*, %d очков, раундов: %d\n"
	textRoundFavoriteMsg                   = "Раунд %d завершен"
	textClickStartBtnMsg                   = emoji.ChequeredFlag.String() + " Нажми кнопку, когда будешь готов"
	textStartBtnData                       = "Я готов!"
	textStopBtnData                        = "Стоп"
	textStartBtnDataAnswer                 = "Старт!"
	textStopBtnDataAnswer                  = "Стоп!"
	textTimerBtnData                       = "Таймер"
	textStartLetterMsg                     = "Слова на букву - "
	textNextPlayerMsg                      = "Следующий играет %s - *%s*"
	textPlayerLeftGameMsg                  = "Игрок %s покинул игру"
	textPlayerJoinedGameMsg                = "Игрок %s присоединился к игре"
	textStopPlayerRoundMsg                 = "Завершено! Ты набрал %d очков!"
	textStopPlayerRoundBroadcastMsg        = "%s набирает %d очков"
	textValidationRequiresMoreOnePlayerMsg = "Чтобы начать игру необходимо больше одного игрока"
	textVoteMsg                            = "Голосование, игрок всё правильно назвал?"
	textBroadcastCrashMsg                  = "Из-за ошибки в работе сервиса игра была аварийно завершена, поробуйте создать игру заново"
	textStopButton                         = "Нажми Стоп, когда закончишь"
)
