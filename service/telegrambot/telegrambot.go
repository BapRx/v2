// Copyright 2023 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package telegrambot // import "miniflux.app/service/telegrambot"

import (
	"fmt"
	"log"
	"strings"

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
		if update.Message != nil {
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
			if len(msg.Text) > 0 {
				if _, err := bot.Send(msg); err != nil {
					log.Panic(err)
				}
			}
		} else if update.CallbackQuery != nil {
			logger.Info("update.CallbackQuery.Data: %s", update.CallbackQuery.Data)
			data := strings.Split(update.CallbackQuery.Data, "/")
			action := data[0]
			logger.Info("action: %s", action)
			entryHash := data[1]
			logger.Info("entryHash: %s", entryHash)
			var user *model.User
			var err error
			if user, err = store.UserByTelegramChatID(update.CallbackQuery.From.ID); err != nil {
				logger.Error("[Telegram Bot] UserByTelegramChatID failed: %v", err)
			}
			var entry *model.Entry
			if entry, err = store.EntryByHash(entryHash); err != nil {
				logger.Error("[Telegram Bot] UserByTelegramChatID failed: %v", err)
			}
			entryIDSlice := []int64{}
			entryIDSlice = append(entryIDSlice, entry.ID)
			var newCallbackAction string
			switch action {
			case "read":
				logger.Info("Marking entry %d as unread.", entryIDSlice)
				if err = store.SetEntriesStatus(user.ID, entryIDSlice, "read"); err != nil {
					logger.Error("[Telegram Bot] SetEntriesStatus failed: %v", err)
				}
				newCallbackAction = "unread"
			case "unread":
				logger.Info("Marking entry %d as read.", entryIDSlice)
				if err = store.SetEntriesStatus(user.ID, entryIDSlice, "unread"); err != nil {
					logger.Error("[Telegram Bot] SetEntriesStatus failed: %v", err)
				}
				newCallbackAction = "read"
			}
			callbackAction := fmt.Sprintf("%s/%s", newCallbackAction, entryHash)
			if len(callbackAction) > 64 {
				callbackAction = callbackAction[:64]
			}
			buttonRow := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("Open", entry.URL),
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Mark as %s", newCallbackAction), callbackAction),
			)
			if entry.CommentsURL != "" {
				commentButton := tgbotapi.NewInlineKeyboardButtonURL("Comments", entry.CommentsURL)
				buttonRow = append(buttonRow, commentButton)
			}
			msg := tgbotapi.NewEditMessageReplyMarkup(
				update.CallbackQuery.Message.Chat.ID,
				update.CallbackQuery.Message.MessageID,
				tgbotapi.NewInlineKeyboardMarkup(buttonRow),
			)
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
		}
	}
}
