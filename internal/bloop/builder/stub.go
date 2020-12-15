package builder

import "github.com/enescakir/emoji"

var (
	commandNextText = "Далее"
	commandPrevText = "Назад"
	commandDoneText = emoji.ChequeredFlag.String() + " Завершить"
)

var (
	textChooseCategories            = "Выбери категории, в которых игроки называют слова или напиши свой вариант"
	textChooseRoundsNum             = "Выбери количество раундов в игре. Это количество раз, которое каждый игрок сможет сыграть"
	textDeleteComplexLetters        = "Убери сложные буквы"
	textVoteAllowed                 = emoji.Loudspeaker.String() + " Добавлять голосование?"
	textConfigurationDone           = "Завершить процесс создания игры?"
	textAddLeastCategoryToComplete  = "Добавьте хотя бы одну категорию для завершения"
	textAddLeastOneLetterToComplete = "Добавьте хотя бы одну букву для завершения"
	textAddedCategory               = "Добавлена категория %s"
	textDeletedCategory             = "Удалена категория %s"
	textRoundNum                    = "Количество раундов - %d"
	textAddedLetter                 = "Добавлена буква %s"
	textDeletedLetter               = "Удалена буква %s"
	textVoteYes                     = emoji.ThumbsUp.String() + " Да"
	textVoteNo                      = emoji.ThumbsDown.String() + " Нет"
)
