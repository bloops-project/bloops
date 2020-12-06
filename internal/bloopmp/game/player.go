package game

func newPlayer(chatId, userId int64, firstName string) *player {
	return &player{userId: userId, chatId: chatId, firstName: firstName, rates: []*rate{}}
}

type player struct {
	firstName string
	userId    int64
	chatId    int64
	rates     []*rate
}
