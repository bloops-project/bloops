# Bloopsbot - telegram bot for offline quizzes.
```
___.   .__                                 ___.           __   
\_ |__ |  |   ____   ____ ______  ______   \_ |__   _____/  |_ 
 | __ \|  |  /  _ \ /  _ \\____ \/  ___/    | __ \ /  _ \   __\
 | \_\ \  |_(  <_> |  <_> )  |_> >___ \     | \_\ (  <_> )  |  
 |___  /____/\____/ \____/|   __/____  >____|___  /\____/|__|  
     \/                   |__|       \/_____/   \/                                                                          
```

## What is bloopsbot?
What is bloopsbot? 🤖 This is a telegram bot created to organize offline games similar to tiktok quizzes. bloop has no localization and is only in Russian.

## Location
You can use it here -> [bloops in Telegram](https://t.me/bloops_bot)

## Why?
🎄🎄🎄 The project was created for playing with the family during the holidays. This is just fun.

## Features
* 🕹️ Offline format, well suited for activities with friends
* 🎲 Quiz games - in 30 seconds, you need to name one word from several categories starting with a certain letter
* 💎 Bloops - you can add additional tasks that diversify the process
* 👽 Players have profiles, you can see your statistics
* 👨‍💻 You can use a CLI or deploy from a container

## Language
Only in Russian 😔

## Development
This is shitty code, I know there is no testing in it, but this is my little hackathon to get it done quickly for the holidays.

## 🚀 Quick start
For CLI version make 
1. *Clone repo*
```
$ git clone https://github.com/robotomize/bloop.git
```
2. *Build CLI application*
```
$ cd bloop
$ go build cmd/bloop-cli
```
3. *Register your bot token* [bot father](https://t.me/BotFather)
```
$ ./bloop-cli
```

To build a docker image run the following commands
```
$ docker build -e BLOOP_TOKEN="BOT_TOKEN" -it . 
```
Or you can build the service by adding the bot token to the environment variables
```
$ go build cmd/bloop-srv
```
## Contact
tg: [@robotomize](https://t.me/robotomize)