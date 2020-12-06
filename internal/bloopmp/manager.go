package bloopmp

import (
	"bloop/internal/bloopmp/builder"
	"bloop/internal/bloopmp/game"
	"bloop/internal/cache"
	"context"
	"fmt"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"hash/fnv"
	"strconv"
	"sync"
	"time"
)

func NewManager(tg *tgbotapi.BotAPI, userCache cache.Cache, config *Config) *Manager {
	return &Manager{
		tg:                  tg,
		users:               userCache,
		config:              config,
		userBuildSessions:   map[int64]*builder.Session{},
		userPlayingSessions: map[int64]*game.Session{},
		playingSessions:     map[int64]*game.Session{},
		cmdCb:               map[int64]func(msg string) error{},
	}
}

type Manager struct {
	mtx sync.RWMutex

	ctx                 context.Context
	tg                  *tgbotapi.BotAPI // tg api instance
	config              *Config
	users               cache.Cache
	userBuildSessions   map[int64]*builder.Session
	userPlayingSessions map[int64]*game.Session
	playingSessions     map[int64]*game.Session
	cmdCb               map[int64]func(msg string) error
}

func (m *Manager) Run(ctx context.Context) error {
	m.ctx = ctx
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60
	updates, err := m.tg.GetUpdatesChan(upd)
	if err != nil {
		return err
	}

	go m.cleaning(ctx)

	for {
		select {
		case update := <-updates:
			user := m.recvUser(update)
			if update.Message != nil {
				if err := m.handleCommand(user, update); err != nil {
					fmt.Println(err)
				}
			}

			if update.CallbackQuery != nil {
				if err := m.handleCallbackQuery(user, update); err != nil {
					fmt.Println(err)
				}
			}
		case <-ctx.Done():
			// shutdown
			return nil
		}
	}
}

func (m *Manager) cleaning(ctx context.Context) {
	timer := time.NewTicker(1 * time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for _, session := range m.userBuildSessions {
				if time.Since(session.CreatedAt) > m.config.BuildingTime {
					session.Stop()
				}
			}
		}
	}
}

// map[callback_data] ->

func (m *Manager) handleCommand(user *User, upd tgbotapi.Update) error {
	switch upd.Message.Text {
	case "/start":
		if err := m.handleStartButton(user, upd.Message.Chat.ID); err != nil {
			return err
		}
	case createButtonLabel:
		if err := m.handleCreateButton(user, upd.Message.Chat.ID); err != nil {
			return err
		}
	case joinButtonLabel:
		if err := m.handleJoinButton(user, upd.Message.Chat.ID); err != nil {
			return err
		}
	case breakButtonLabel:
		if err := m.handleBreakButton(user, upd.Message.Chat.ID); err != nil {
			return err
		}
	default:
		if cb, ok := m.cmdCb[user.Id]; ok {
			if err := cb(upd.Message.Text); err != nil {
				return err
			}
			delete(m.cmdCb, user.Id)
			return nil
		}

		if session, ok := m.userBuildSessions[user.Id]; ok {
			if err := session.Execute(upd); err != nil {
				return err
			}

			return nil
		}

		if session, ok := m.userPlayingSessions[user.Id]; ok {
			if err := session.Execute(user.Id, upd); err != nil {
				return err
			}

			return nil
		}
	}
	return nil
}

func (m *Manager) handleCallbackQuery(user *User, upd tgbotapi.Update) error {
	if session, ok := m.userBuildSessions[user.Id]; ok {
		if err := session.Execute(upd); err != nil {
			return err
		}
	}

	if session, ok := m.userPlayingSessions[user.Id]; ok {
		if err := session.Execute(user.Id, upd); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) roomHash() (int64, error) {
	h := fnv.New32a()
	bytes, err := time.Now().MarshalBinary()
	if err != nil {
		return 0, fmt.Errorf("hash binary encode error: %w", err)
	}

	_, err = h.Write(bytes)
	if err != nil {
		return 0, fmt.Errorf("hash write error: %w", err)
	}

	return int64(h.Sum32() >> 20), nil
}

func (m *Manager) handleStartButton(user *User, chatId int64) error {
	msg := tgbotapi.NewMessage(
		chatId,
		fmt.Sprintf(
			"Привет, %s это экспериментальный игровой telegram проект @bloop. %s удачи!",
			user.Username,
			emoji.Unicorn.String(),
		))
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return err
	}

	return nil
}

func (m *Manager) buildGameConfig(session *builder.Session) game.Config {
	config := game.Config{
		AuthorId:   session.AuthorId,
		RoundsNum:  session.RoundsNum,
		Categories: []string{},
		Letters:    []string{},
	}

	for _, category := range session.Categories {
		if category.Status {
			config.Categories = append(config.Categories, category.Text)
		}
	}

	for _, letter := range session.Letters {
		if letter.Status {
			config.Letters = append(config.Letters, letter.Text)
		}
	}

	return config
}

func (m *Manager) handleCreateButton(user *User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, "Настраиваем игру!")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(breakMenuButton))
	if _, err := m.tg.Send(msg); err != nil {
		return err
	}

	session, err := builder.NewSession(m.tg, chatId, user.Id, func(session *builder.Session) error {
		defer func() {
			session.Stop()
			delete(m.userBuildSessions, user.Id)
		}()

		hash, err := m.roomHash()
		if err != nil {
			return err
		}

		for {
			if _, ok := m.playingSessions[hash]; !ok {
				session := game.NewGame(m.buildGameConfig(session), m.tg, user.Id)
				session.Run(m.ctx)
				m.playingSessions[hash] = session
				break
			}
		}

		msg := tgbotapi.NewMessage(chatId, `Поздравляем, ты создал игру! Для входа нужно нажать кнопку *присоединится* и ввести этот код. 
Отправь код своим друзьям, чтобы они тоже присоединялись! Удачи`)
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := m.tg.Send(msg); err != nil {
			return err
		}

		msg = tgbotapi.NewMessage(chatId, strconv.Itoa(int(hash)))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = commonMenuButtons
		if _, err := m.tg.Send(msg); err != nil {
			return err
		}

		return nil
	}, func(session *builder.Session) error {
		defer func() {
			session.Stop()
			delete(m.userBuildSessions, user.Id)
		}()
		msg := tgbotapi.NewMessage(chatId, `Прогресс создания игры был удалён`)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = commonMenuButtons
		if _, err := m.tg.Send(msg); err != nil {
			return err
		}

		return nil
	}, m.config.BuildingTime)

	if err != nil {
		return err
	}

	m.userBuildSessions[user.Id] = session
	session.Run(m.ctx)

	return nil
}

func (m *Manager) handleBreakButton(user *User, chatId int64) error {
	delete(m.userBuildSessions, user.Id)
	if session, ok := m.userPlayingSessions[user.Id]; ok {
		session.Leave(user.Id)
	}

	delete(m.userPlayingSessions, user.Id)

	msg := tgbotapi.NewMessage(chatId, "Ты покинул все игровые сеансы")
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return err
	}

	return nil
}

func (m *Manager) handleJoinButton(usr *User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, "Отправь код подключения к игре")
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return err
	}
	m.cmdCb[usr.Id] = func(msg string) error {
		n, err := strconv.Atoi(msg)
		if err != nil {
			return err
		}

		if session, ok := m.playingSessions[int64(n)]; ok {
			session.RegisterPlayer(usr.Id, chatId, usr.FirstName)
			text := "Ты присоединился к игре! "
			m.userPlayingSessions[usr.Id] = session
			if session.AuthorId == usr.Id {
				text = text + `Так как ты являешься автором, у тебя есть кнопка запуска. Когда все игроки присоединятся нажми *Начать* для запуска игры!`
			}

			msg := tgbotapi.NewMessage(chatId, text)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					breakMenuButton,
					runGameButton,
				),
			)

			if _, err := m.tg.Send(msg); err != nil {
				return err
			}
		}

		return nil
	}
	return nil
}

func (m *Manager) handlePlayingSession(user *User, upd tgbotapi.Update) error {
	if upd.CallbackQuery != nil {

	}

	if upd.Message != nil {

	}

	return nil
}

func (m *Manager) fetchTgUser(tgUser *tgbotapi.User) *User {
	u, ok := m.users.Get(int64(tgUser.ID))
	if !ok {
		newUser := NewUser(tgUser)
		m.users.Add(newUser.Id, newUser)
		u = newUser
	}

	return u.(*User)
}

func (m *Manager) recvUser(upd tgbotapi.Update) *User {
	var (
		user   *User
		tgUser *tgbotapi.User
	)

	if upd.CallbackQuery != nil {
		tgUser = upd.CallbackQuery.From
	}

	if upd.Message != nil {
		tgUser = upd.Message.From
	}

	if tgUser != nil {
		return m.fetchTgUser(tgUser)
	}

	return user
}
