# Blooobot - telegram bot
```
___.    .__                                   ___.              __   
\_ |__  |  |    ____    ____    ____  ______  \_ |__    ____  _/  |_ 
 | __ \ |  |   /  _ \  /  _ \  /  _ \ \____ \  | __ \  /  _ \ \   __\
 | \_\ \|  |__(  <_> )(  <_> )(  <_> )|  |_> > | \_\ \(  <_> ) |  |  
 |___  /|____/ \____/  \____/  \____/ |   __/  |___  / \____/  |__|  
     \/                               |__|         \/                
                                                                     
```
## What is blooopbot?
What is blooopbot? This is a telegram bot created to organize offline games similar to tiktok quizzes. bloop has no localization and is only in Russian.

## Why?
It's just fun, it's a little activity in the family for the holidays

## Location
[bloop in Telegram](https://t.me/blooopbot)

## Development
Shitty code, I know, but it was done quickly for a family celebration

## Install
For CLI version make 
```
git clone https://github.com/robotomize/bloop.git
cd bloop
go build cmd/bloop-cli
./bloop-cli
```
or from docker
```
docker build -e BLOOP_TOKEN="BOT_TOKEN" -it . 
```