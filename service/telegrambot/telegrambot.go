// Copyright 2023 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package telegrambot // import "miniflux.app/service/telegrambot"

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"miniflux.app/logger"
	"miniflux.app/model"
	"miniflux.app/storage"
	"miniflux.app/telegrambot/client"
	"miniflux.app/worker"
)

// Serve get updates from the Telegram API
func Serve(store *storage.Storage, pool *worker.Pool, botToken string, allowedChatIDs []string) error {
	bot := client.New(botToken)
	go getUpdates(bot, store, pool, allowedChatIDs)

	return nil
}

func getUpdates(bot *tgbotapi.BotAPI, store *storage.Storage, pool *worker.Pool, allowedChatIDs []string) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil {
			continue
		}
		if !update.Message.IsCommand() {
			continue
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "readAll":
			msg.Text = "readAll"
		case "read_all":
			var err error
			var user *model.User
			if user, err = store.UserByTelegramChatID(update.Message.Chat.ID); err != nil {
				logger.Error("[Telegram Bot] UserByTelegramChatID failed: %v", err)
			}
			if err = store.MarkAllAsRead(user.ID); err != nil {
				logger.Error("[Telegram Bot] MarkAllAsRead failed: %v", err)
			} else {
				msg.Text = "Successfully marked everything as read!"
			}
		default:
			msg.Text = "Available commands: /read_all"
		}
		if len(msg.Text) == 0 {
			return
		}
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
