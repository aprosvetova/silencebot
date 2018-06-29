package main

import (
	"fmt"

	tb "gopkg.in/tucnak/telebot.v2"
)

func getAdminsKey(chat *tb.Chat) string {
	return fmt.Sprintf("admins.%d", chat.ID)
}

func getSilentKey(chat *tb.Chat) string {
	return fmt.Sprintf("silent.%d", chat.ID)
}

func getRestrictedKey(chat *tb.Chat) string {
	return fmt.Sprintf("restricted.%d", chat.ID)
}

func getPinnedMessageKey(chat *tb.Chat) string {
	return fmt.Sprintf("pinned.%d", chat.ID)
}

func getLangKey(chat *tb.Chat) string {
	return fmt.Sprintf("lang.%d", chat.ID)
}

func getLang(chat *tb.Chat) string {
	lang := db.Get(getLangKey(chat)).Val()
	if lang == "" {
		return "en-US"
	}
	return lang
}

func setLang(chat *tb.Chat, lang string) {
	db.Set(getLangKey(chat), lang, 0)
}

func detectLang(chat *tb.Chat, locale string) {

}
