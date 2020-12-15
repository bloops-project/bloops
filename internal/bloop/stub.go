package bloop

import (
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	createButtonLabel = "Создать игру"
	leaveButtonLabel  = emoji.ChequeredFlag.String() + " Выйти"
	startButtonLabel  = emoji.Rocket.String() + " Начать"
	joinButtonLabel   = "Присоединиться к игре"
	ratingLabel       = emoji.Star.String() + " Таблица лидеров"
	rulesLabel        = "Правила"
)

var (
	cmdStart     = "/start"
	cmdRules     = "/rules"
	cmdAddPlayer = "/add"
)

var (
	createMenuButton  = tgbotapi.NewKeyboardButton(createButtonLabel)
	joinMenuButton    = tgbotapi.NewKeyboardButton(joinButtonLabel)
	leaveMenuButton   = tgbotapi.NewKeyboardButton(leaveButtonLabel)
	startGameButton   = tgbotapi.NewKeyboardButton(startButtonLabel)
	ratingButton      = tgbotapi.NewKeyboardButton(ratingLabel)
	rulesButton       = tgbotapi.NewKeyboardButton(rulesLabel)
	commonMenuButtons = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(createMenuButton),
		tgbotapi.NewKeyboardButtonRow(joinMenuButton),
		tgbotapi.NewKeyboardButtonRow(rulesButton),
	)
)

var (
	textAuthorGreetingMsg = "\n\nТак ты являешься ведущим, тебе нужно нажать на кнопку " +
		startButtonLabel + ", когда все игроки присоединятся"
	textJoinedGameMsg                      = "Ты присоединился к игре! "
	textGameRoomNotFoundMsg                = "Игровая комната не найдена"
	textSendJoinedCodeMsg                  = "Отправь код подключения к игре"
	textLeavingSessionsMsg                 = "Ты покинул все игровые сеансы"
	textSendOfflinePlayerUsernameMsg       = "Отправь имя оффлайн пользователя"
	textGameRoomNotFound                   = "Тебе нужно присоединиться к игре, чтобы добавлять оффлайн игроков"
	textOfflinePlayerAdded                 = "Оффлайн игрок добавлен. Все сообщения будут приходить тебе"
	textCreationGameCompletedSuccessfulMsg = emoji.Unicorn.String() + " Игровая комната создана.\n\nДля входа нужно " +
		"нажать кнопку *Присоединится к игре* и ввести этот код.\n\n" +
		"Отправь его своим друзьям, чтобы они смогли присоединиться " + emoji.VideoGame.String()
	textSettingsMsg = "Настраиваем параметры игры"

	textGreetingMsg = emoji.ChristmasTree.String() + "Привет, %s\n\n" +
		"Это bloop " + emoji.Robot.String() + " - бот, для игры в небольшие викторины, где игроки должны за " + emoji.Stopwatch.String() +
		" 30 сек назвать по одному слову, начинающемуся на заданную букву из предложенных категорий.\n\n" +
		emoji.Robot.String() + " Бот сделан для ведения игр небольшой компанией, собравшейся в оффлайне." +
		"Он подсчитывает очки, генерирует буквы, создает лидерборды, и задает правила, а вы играете!" + emoji.Unicorn.String() + "\n\n" +
		"Правила: " + cmdRules + "\n" +
		"Обратная связь: @robotomize\n" +
		"Проект на github: [bloop](https://robotomize.me)"

	textRulesMsg = emoji.Bookmark.String() + " *Правила игры* - игроки должны за " + emoji.Stopwatch.String() + " 30 сек" +
		" назвать по одному слову, начинающемуся на заданную букву из предложенных категорий. Чем быстрее называют," +
		" тем большее количество очков получают.По итогам нескольких раундов и количеству очков определяется победители\n\n" +
		emoji.CrossMark.String() + " *Ограничения* - от 2х человек, бот " + emoji.Robot.String() +
		" сделан для ведения игр небольшой компанией, собравшейся в оффлайне.\n\n" +
		emoji.Joystick.String() + " *Что делать?* - для начала ведущий игрок создаёт игру, для этого ему нужно нажать " +
		"*Создать игру* и выполнить все действия. Ему будет выслан код, который он сообщает игрокам. Затем игроки " +
		"присоединяются к игре и ведуший нажимает кнопку *Начать*" + emoji.Rocket.String() + "\n\n" +
		emoji.Loudspeaker.String() + " В игру можно добавить голосование, если оно необходимо. " +
		"После завершенного раунда игроки голосуют справился ли игрок. Если большинство считает, что да, " +
		"то игрок оставляется свои очки, а если нет - теряет их. Если 50 на 50, то игрок получит половину очков\n\n" +
		"Обратная связь: @robotomize\n" +
		"Проект на github: [bloop](https://robotomize.me)"
)
