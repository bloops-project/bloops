package buildinfo

const (
	Graffiti       = "___.   .__                                 ___.           __   \n\\_ |__ |  |   ____   ____ ______  ______   \\_ |__   _____/  |_ \n | __ \\|  |  /  _ \\ /  _ \\\\____ \\/  ___/    | __ \\ /  _ \\   __\\\n | \\_\\ \\  |_(  <_> |  <_> )  |_> >___ \\     | \\_\\ (  <_> )  |  \n |___  /____/\\____/ \\____/|   __/____  >____|___  /\\____/|__|  \n     \\/                   |__|       \\/_____/   \\/             \n\n"
	GreetingCLI    = "%s %s \nbuild time: %s\ntg: %s\ngithub: %s\n"
	GithubBloopURL = "https://github.com/robotomize/bloopsbot.git"
	TgBloopURL     = "https://t.me/@bloops_bot"
	BotFatherURL   = "https://t.me/BotFather"
)

var (
	BuildTag string = "v0.0.0"
	Name     string = "bloopsbot"
	Time     string = ""
)

type buildinfo struct{}

func (buildinfo) Tag() string {
	return BuildTag
}

func (buildinfo) Name() string {
	return Name
}

func (buildinfo) Time() string {
	return Time
}

var Info buildinfo
