# Silence Bot for Telegram supergroups

Silence Bot allows you to calm down all chat participants by muting them temporarily.

Just add the bot to your supergroup, give it message deletion and user restriction rights and you're all set.

>**Use /silence to enable silent mode.**
>
>All non-admin messages will be deleted in silent mode, any user who tries to send a message will get a temporary read-only restriction.

>**Use /silence again to disable silent mode.**
>
>All users will be unrestricted automatically and be able to chat.

**Please note that the bot requires running Redis instance to store data.**

I'm very new to Go, so I'll be happy if you make some pull requests.

## Building for all platforms
Make sure you have Go installed and just run
```
./build.sh
```

## Building manually
Install dependencies
```
go get ./...
```
And then build
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-s -w -extldflags "-static"' -o silencebot
```

## Running
Use `./silencebot -t <YOUR_TELEGRAM_TOKEN>` to start the bot.

By default it connects to localhost:6379 Redis instance without password and selects db 0.
You can customize this behavior, check `./silencebot -h` for all arguments.

## Running as a background service

There are two ways as for now: Docker compose and systemd service

**Don't forget to replace token!**

Docker compose is ready to use, but not recommended for stable environments as long as redis is running inside Docker.

[systemd service example](contrib/bot.service) (recommended)

## TODO

- [ ] Localization
- [ ] Embedded service autoinstall
- [ ] Pin "silence mod enabled" message and restore previous pin then
- [ ] Minimal hidden admin commands (`/stats`, `/health`, `/uptime` etc.)
