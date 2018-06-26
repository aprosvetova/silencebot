package main

import "time"
import (
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"strconv"
)

var b *tb.Bot
var db *redis.Client

func main() {
	redisHost := flag.String("rhost", "localhost:6379", "redis host and port")
	redisPwd := flag.String("rpwd", "", "redis password")
	redisDb := flag.Int("rdb", 0, "redis DB number")
	tgToken := flag.String("token", "", "telegram bot token")
	flag.Parse()

	if *tgToken == "" {
		log.Fatal("telegram bot token required")
	}

	db = redis.NewClient(&redis.Options{
		Addr:     *redisHost,
		Password: *redisPwd,
		DB:       *redisDb,
	})

	if db.DbSize().Err() != nil {
		log.Fatal("redis error")
	}

	poller := tb.NewMiddlewarePoller(&tb.LongPoller{Timeout: 10 * time.Second}, func(upd *tb.Update) bool {
		if upd.Message == nil || upd.Message.Chat.Type != tb.ChatSuperGroup {
			return false
		}
		return true
	})

	var err error
	b, err = tb.NewBot(tb.Settings{
		Token:  *tgToken,
		Poller: poller,
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/silence", silenceCommand)

	//uh really? why can't I just handle all Message updates?
	b.Handle(tb.OnText, checkMessage)
	b.Handle(tb.OnAudio, checkMessage)
	b.Handle(tb.OnContact, checkMessage)
	b.Handle(tb.OnDocument, checkMessage)
	b.Handle(tb.OnLocation, checkMessage)
	b.Handle(tb.OnPhoto, checkMessage)
	b.Handle(tb.OnSticker, checkMessage)
	b.Handle(tb.OnVenue, checkMessage)
	b.Handle(tb.OnVideo, checkMessage)
	b.Handle(tb.OnVideoNote, checkMessage)
	b.Handle(tb.OnVoice, checkMessage)

	b.Start()
}

func checkMessage(m *tb.Message) {
	if isSilent(m.Chat) {
		if isAdmin(m.Chat, m.Sender) {
			return
		}
		b.Delete(m)
		restrictUser(m.Chat, m.Sender)
	}
}

func silenceCommand(m *tb.Message) {
	if !isAdmin(m.Chat, m.Sender) {
		b.Delete(m)
		return
	}
	if !isAdmin(m.Chat, b.Me) {
		b.Reply(m, "Я не админ :(")
		db.Del(getAdminsKey(m.Chat))
		return
	}

	if isSilent(m.Chat) {
		setSilent(m.Chat, false)
		go unrestrictAll(m.Chat)
		b.Send(m.Chat, "*Режим тишины отключен. Можете общаться дальше.*", &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	} else {
		setSilent(m.Chat, true)
		b.Send(m.Chat, "*В чате активирован режим тишины!*", &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	}
}

func isAdmin(chat *tb.Chat, user *tb.User) bool {
	key := getAdminsKey(chat)
	if db.Exists(key).Val() == 0 {
		members, err := b.AdminsOf(chat)
		if err != nil {
			return false
		}
		var admins []interface{}
		found := false
		for _, member := range members {
			if member.CanDeleteMessages || member.Role == tb.Creator {
				admins = append(admins, member.User.ID)
				if member.User.ID == user.ID {
					found = true
				}
			}
		}
		db.SAdd(key, admins...)
		db.Expire(key, 10*time.Minute)
		return found
	} else {
		return db.SIsMember(key, user.ID).Val()
	}
}

func setSilent(chat *tb.Chat, silent bool) {
	key := getSilentKey(chat)
	if silent {
		db.Set(key, 1, 0)
	} else {
		db.Del(key)
	}
}

func isSilent(chat *tb.Chat) bool {
	return db.Exists(getSilentKey(chat)).Val() == 1
}

func restrictUser(chat *tb.Chat, user *tb.User) {
	b.Restrict(chat, &tb.ChatMember{User: user, RestrictedUntil: time.Now().Add(5 * time.Minute).Unix()})
	db.SAdd(getRestrictedKey(chat), user.ID)
}

func unrestrictAll(chat *tb.Chat) {
	key := getRestrictedKey(chat)
	users := db.SMembers(key).Val()
	db.Del(key)
	for _, user := range users {
		userID, err := strconv.Atoi(user)
		if err != nil {
			continue
		}
		member := &tb.ChatMember{User: &tb.User{ID: userID}, RestrictedUntil: tb.Forever(), Rights: tb.Rights{CanSendMessages: true}}
		b.Promote(chat, member)
		time.Sleep(100 * time.Millisecond)
	}
}

func getAdminsKey(chat *tb.Chat) string {
	return fmt.Sprintf("admins.%d", chat.ID)
}

func getSilentKey(chat *tb.Chat) string {
	return fmt.Sprintf("silent.%d", chat.ID)
}

func getRestrictedKey(chat *tb.Chat) string {
	return fmt.Sprintf("restricted.%d", chat.ID)
}
