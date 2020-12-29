package resource

import "github.com/enescakir/emoji"

type Bloops struct {
	Name    string
	Image   string
	Points  int
	Task    string
	Seconds int
	Weight  int
}

type Category struct {
	Text   string
	Status bool
}

type Letter struct {
	Text   string
	Status bool
}

var (
	Letters = []Letter{
		{Text: "А", Status: true}, {Text: "Б", Status: true}, {Text: "В", Status: true}, {Text: "Г", Status: true},
		{Text: "Д", Status: true}, {Text: "Е", Status: true}, {Text: "Ж", Status: true}, {Text: "З", Status: true},
		{Text: "И", Status: true}, {Text: "К", Status: true}, {Text: "Л", Status: true}, {Text: "М", Status: true},
		{Text: "Н", Status: true}, {Text: "О", Status: true}, {Text: "П", Status: true}, {Text: "Р", Status: true},
		{Text: "С", Status: true}, {Text: "Т", Status: true}, {Text: "У", Status: true}, {Text: "Ф", Status: true},
		{Text: "Х", Status: true}, {Text: "Ц"}, {Text: "Ч"}, {Text: "Ш"}, {Text: "Э"}, {Text: "Ю"}, {Text: "Я"}}

	Categories = []Category{
		{Text: "Страна"}, {Text: "Город", Status: true}, {Text: "Овощ или фрукт", Status: true},
		{Text: "Имя", Status: true}, {Text: "Знаменитость"}, {Text: "Бренд", Status: true},
		{Text: "Животное", Status: true}, {Text: "Термин"},
		{Text: "Любое слово"},
	}

	RoundsNum  = []int{1, 2, 3, 4, 5}
	RoundTimes = []int{30, 45, 60}

	Bloopses = []Bloops{
		{Name: emoji.Cinema.String() + " Подпольный режиссер", Weight: 2, Seconds: +30, Points: +10, Task: "Ворвался подпольный режиссер и хочет, чтобы ты заменил категории на *кино и актеры*\nНазывай имена фильмов, актеров или режиссеров на выпавшую букву"},
		{Name: emoji.Flamingo.String() + "Фламинго", Weight: 2, Points: +10, Task: "Так получилось, что ты стал фламинго на время, когда называешь слова, ты должен стоять на одной ноге(можно держаться за что-нибудь)"},
		{Name: emoji.WomanSinger.String() + " На разогреве", Weight: 2, Points: +5, Task: "Каждое слово, которое ты называешь ты должен пропеть как Басков"},
		{Name: emoji.Hammer.String() + " Мастерство", Weight: 1, Points: +20, Seconds: +15, Task: "Время показать мастерство! Тебе нужно на каждую категорию назвать не одно слово, а два. За большее время"},
		{Name: emoji.ManLiftingWeights.String() + " Культурист", Weight: 2, Seconds: +5, Points: +10, Task: "Ты теперь мастер не только слова, но и тела, после каждого названного слова нужно присесть 1 раз"},
		{Name: emoji.PeopleWithBunnyEars.String() + " Командная работа", Weight: 1, Points: +8, Task: "Время работать в команде! Твой сосед справа называет слова вместе с тобой по очереди, ты начинаешь первым"},
		{Name: emoji.PersonRunning.String() + " Флэш", Weight: 2, Points: +10, Seconds: -5, Task: "Тебя называют быстрейший из живых, в этом раунде у тебя на 5 сек меньше времени, покажи скорость!"},
		//{Name: "Рекордсмен", Weight: 3, Points: 15, Task: "Тут всё не просто, поставь рекорд раунда! Если до тебя в раунде никто не сыграл, значит тебе повезло=)"},
		{Name: emoji.WaterWave.String() + " Волна удачи", Weight: 2, Points: +5, Task: "Тебе повезло, просто успей вовремя и получи +5 очков"},
		{Name: emoji.WomanGesturingNo.String() + " Неудача", Weight: 2, Points: -5, Task: "Карта не легла, ты получишь -5 очков в этом раунде"},
		{Name: emoji.LoudlyCryingFace.String() + " Депрессия", Weight: 1, Task: "Ты устал, в этом раунде за тебя играет человек напротив тебя, ты можешь дать ему смартфон"},
		{Name: emoji.IceHockey.String() + " Замена", Weight: 2, Task: "Ты можешь заменить одну из категорий на другую, соответственно тебе нужно будет назвать 2 слова на эту категорию"},
		{Name: emoji.Bowling.String() + " Страйк", Weight: 2, Points: +7, Task: "Ты выбил страйк в этом раунде, называй слова только на одну, любую категорию"},
		{Name: emoji.Bomb.String() + "Бомба", Weight: 1, Points: +10, Task: "Выпала бомба, выбери одну категорию на которую нужно назвать два слова, все остальные по 1му разу"},
		{Name: emoji.ManKneeling.String() + " Предложение", Weight: 2, Points: +5, Task: "У тебя важное событие, называй слова, встав на одно колено!"},
		{Name: emoji.Divide.String() + " Математик", Weight: 2, Seconds: +10, Points: +15, Task: "Ты вдруг стал счетоводом, каждый раз когда называешь слово, произнеси оставшееся количество секунд умноженное на 2. Например, если осталось 17 -> 34, если 23-> 46"},
		{Name: emoji.ClappingHands.String() + " Аплодисменты", Weight: 2, Points: +5, Task: "Задание для остальных игроков, когда игрок произносит слово на выбранную букву нужно хлопнуть в ладоши"},
		{Name: emoji.Ninja.String() + " Самурай", Weight: 2, Seconds: -10, Points: +20, Task: "Ты как самурай, готов ко всему, у тебя будет на 10 сек меньше времени, но в награду получишь +20 очков"},
		{Name: emoji.SeeNoEvilMonkey.String() + " Вслепую", Weight: 2, Points: +5, Task: "Кошмар! Надо называть слова, закрыв глаза, вслепую, ты справишься!"},
		{Name: emoji.Guitar.String() + " Музыкалити", Weight: 2, Points: +5, Task: "Ты идешь на звуки музыки, один из участников включает любую песню под которую вы играете раунд, конечно не на полную громкость"},
		{Name: emoji.MartialArtsUniform.String() + " Каратэ", Weight: 1, Points: +10, Seconds: +10, Task: "Ты долго тренировался и стал мастером боевых искуств, после каждого названного слова нужно изобразить удар каратэ с соответствующим звуком. У тебя будет много времени!"},
		{Name: emoji.FourLeafClover.String() + " Четырехлистный клевер", Weight: 3, Task: "Удача! Ты можешь заменить выпавшую букву на любую другую"},
		{Name: emoji.UmbrellaWithRainDrops.String() + " Ненастье", Weight: 2, Seconds: -5, Task: "Плохая погода, или настроение, вообщем у тебя сгорело 5 сек, нужно выкручиваться"},
		{Name: emoji.Rainbow.String() + " Радуга", Weight: 2, Task: "Радуга придумана, чтобы улыбаться, в этом раунде ты можешь исключить одну категорию на свой выбор"},
		{Name: emoji.Unicorn.String() + " Единорог", Weight: 1, Seconds: +7, Points: +7, Task: "Пришел единорог и требует поменять любую выпавшую букву на Е в этом раунде"},
		{Name: emoji.Snail.String() + " Улитка", Weight: 2, Seconds: +10, Points: -10, Task: "Ты как улитка в этом раунде, вроде времени много, но очков не прибавляется"},
		{Name: emoji.Mage.String() + " Маг", Weight: 2, Seconds: +15, Points: +10, Task: "Магический водоворот, маг говорит, чтобы выиграть нужно назвать дополнительно одно слово на магическую или фэнтэзи тематику на выпавшую букву"},
		{Name: emoji.RightFacingFist.String() + emoji.VictoryHand.String() + emoji.RaisedBackOfHand.String() + " Камень, ножницы, бумага", Weight: 2, Points: +10, Task: "С соседом слева играете в камень, ножницы, бумага, кто побеждает, тот играет раунд. Очки достаются тебе"},
		{Name: emoji.GemStone.String() + emoji.Owl.String() + " Что? Где? Когда?", Weight: 2, Seconds: -10, Points: +20, Task: "Вы команда что? где? когда? а действующий игрок ваш капитан, игрок называет слова, а остальные игроки должны подсказывать ему, чтобы быстрее завершить раунд"},
		{Name: emoji.Ship.String() + " В одной лодке", Weight: 2, Seconds: +15, Task: "Корабль тонет! Нужно работать в команде, все игроки должны по очереди называть слова. Начинает действующий игрок, за ним игрок слева и по часовой стрелке, вперед!"},
		{Name: emoji.DivingMask.String() + " Аквалангист", Weight: 2, Points: +10, Task: "Ты оказался под водой в акваланге! Нужно произносить слова зажав нос одной рукой!"},
		{Name: emoji.HighVoltage.String() + " Высокое напряжение", Weight: 1, Seconds: +15, Points: +15, Task: "Ты напряжен, ведь это супер игра, тебе нужно придумать слова, и когда на таймере останется 10сек назвать все слова разом!"},
		{Name: emoji.WomanRunning.String() + " Фитнес тренер", Weight: 2, Points: +5, Task: "У тебя появился персональный тренер, который может сменить программу, пусть это будет игрок справа, вместо выпавшей буквы он может загадать свою полегче!"},
		{Name: emoji.ManBouncingBall.String() + " КМС", Weight: 2, Points: +5, Task: "Ты кмс в интеллектульном клубе и решаешь проблемы приседаниями! Если не можешь придумать слово - приседаешь 1 раз, в следующий раз 2 и тд"},
		{Name: emoji.Brain.String() + " Вундеркинд", Weight: 1, Seconds: +30, Points: +10, Task: "Твой IQ зашкаливает и ты пришел всех поразить, в этом раунде тебе нужно назвать ОДНО слово, но не начинающееся, а заканчивающееся на выпавшую букву!"},
		{Name: emoji.CrystalBall.String() + " Гадалка", Weight: 2, Task: "Ты решил попробовать себя в астрологии, до начала раунда игрок напротив загадывает одну из категорий(используемых в игре), если ты угадал, то выигрываешь раунд автоматом(нажми стоп на таймере сразу после начала), если нет играешь как обычно"},
	}
)
