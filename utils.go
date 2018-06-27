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
