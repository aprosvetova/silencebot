package main

import (
	"os"
	"strconv"
	"time"

	alog "github.com/apex/log"
	"github.com/apex/log/handlers/cli"

	"github.com/Unknwon/i18n"
	"github.com/go-redis/redis"
	"github.com/spf13/pflag"
	tb "gopkg.in/tucnak/telebot.v2"
	"strings"
)

var b *tb.Bot
var db *redis.Client

var log = &alog.Logger{
	Level:   alog.DebugLevel,
	Handler: cli.New(os.Stderr),
}

func main() {
	i18n.SetMessage("en-US", "locale/en-US.ini")
	i18n.SetMessage("ru-RU", "locale/ru-RU.ini")

	redisHost := pflag.StringP("rhost", "h", "localhost:6379", "redis host and port")
	redisPwd := pflag.StringP("rpwd", "p", "", "redis password")
	redisDb := pflag.IntP("rdb", "n", 0, "redis DB number (default 0)")
	tgToken := pflag.StringP("token", "t", "", "telegram bot token (required)")
	pflag.Parse()

	if *tgToken == "" {
		log.Fatal("telegram bot token required")
	}

	db = redis.NewClient(&redis.Options{
		Addr:     *redisHost,
		Password: *redisPwd,
		DB:       *redisDb,
	})

	if err := db.DbSize().Err(); err != nil {
		log.WithError(err).Fatal("redis error")
	}

	poller := tb.NewMiddlewarePoller(&tb.LongPoller{Timeout: 10 * time.Second}, func(upd *tb.Update) bool {
		if upd.Message != nil && upd.Message.Chat.Type != tb.ChatSuperGroup {
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
		log.WithError(err).Fatal("can't init bot instance")
	}

	b.Handle("/silence", silenceCommand)
	b.Handle("/switchlang", switchLangCommand)

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

	b.Handle(tb.OnPinned, savePinnedMessage)

	b.Handle(tb.OnAddedToGroup, showWelcomeMessage)

	b.Start()
}

func savePinnedMessage(m *tb.Message) {
	db.Set(getPinnedMessageKey(m.PinnedMessage.Chat), m.PinnedMessage.ID, 0)
}

func restorePinnedMessage(c *tb.Chat) {
	msgID, err := db.Get(getPinnedMessageKey(c)).Int64()
	if err != nil || msgID == 0 {
		b.Unpin(c)
		return
	}
	b.Pin(&tb.Message{
		Chat: c,
		ID:   int(msgID),
	}, tb.Silent)
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

func showWelcomeMessage(m *tb.Message) {
	b.Send(m.Chat, strings.Replace(i18n.Tr(getLang(m.Chat), "welcome_message"), "\\n", "\n", -1), tb.ModeMarkdown)
}

func switchLangCommand(m *tb.Message) {
	if !isAdmin(m.Chat, m.Sender) {
		b.Delete(m)
		return
	}
	currentLang := getLang(m.Chat)
	newLang := "ru-RU"
	if currentLang == "ru-RU" {
		newLang = "en-US"
	}
	setLang(m.Chat, newLang)
	b.Reply(m, i18n.Tr(newLang, "language_switched"))
}

func silenceCommand(m *tb.Message) {
	if !isAdmin(m.Chat, m.Sender) {
		b.Delete(m)
		return
	}
	if !isAdmin(m.Chat, b.Me) {
		log.WithField("chatID", m.Chat.ID).Debug("bot has no admin rights")
		b.Reply(m, i18n.Tr(getLang(m.Chat), "bot_not_admin"))
		db.Del(getAdminsKey(m.Chat))
		return
	}

	if isSilent(m.Chat) {
		setSilent(m.Chat, false)
		log.WithField("chatID", m.Chat.ID).Debug("disabling silent mode")
		go unrestrictAll(m.Chat)
		b.Send(m.Chat, i18n.Tr(getLang(m.Chat), "silence_disabled"), tb.ModeMarkdown)
		restorePinnedMessage(m.Chat)
	} else {
		setSilent(m.Chat, true)
		log.WithField("chatID", m.Chat.ID).Debug("enabled silent mode")
		msg, err := b.Send(m.Chat, i18n.Tr(getLang(m.Chat), "silence_enabled"), tb.ModeMarkdown)
		if err == nil {
			b.Pin(msg, tb.Silent)
		}
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
