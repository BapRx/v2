// Copyright 2023 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package telegrambot // import "miniflux.app/service/telegrambot"

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"miniflux.app/logger"
	"miniflux.app/model"
	"miniflux.app/storage"
	"miniflux.app/telegrambot/client"
)

// Serve get updates from the Telegram API
func Serve(store *storage.Storage) error {
	telegramIntegrations, err := store.GetConfiguredTelegramIntegrations()
	if err != nil {
		return fmt.Errorf("[Telegram Bot] Unable to fetch the configured Telegram integrations: %v", err)
	} else {
		s := reflect.ValueOf(telegramIntegrations)
		for i := 0; i < s.Len(); i++ {
			integration := s.Index(i).Interface().(*model.Integration)
			if err := client.New(integration.TelegramBotToken, integration.TelegramBotChatID); err != nil {
				return fmt.Errorf("[Telegram Bot] Unable to start the Telegram bot: %v", err)
			}
			go getUpdates(store, integration.TelegramBotChatID)
		}
	}

	return nil
}

func getUpdates(store *storage.Storage, chatIDStr string) {
	bot, err := client.Get(chatIDStr)
	if err != nil {
		logger.Error("[Telegram Bot] Unable to grab the Telegram bot: %v", err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	allowedChatID, _ := strconv.ParseInt(chatIDStr, 10, 64)
	for update := range updates {
		if update.FromChat().ID != allowedChatID {
			logger.Error("[Telegram Bot] Chat ID unauthorized! %s", update.FromChat().ID)
			continue
		}
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "read_all":
				confirmationButton := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Yes!", "read_all/"),
					tgbotapi.NewInlineKeyboardButtonData("No.", "cancel/"),
				)
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(confirmationButton)
				msg.Text = "Are you sure?"
			default:
				msg.Text = "Available commands: /read_all"
			}
			if len(msg.Text) > 0 {
				if _, err := bot.Send(msg); err != nil {
					logger.Error("[Telegram Bot] Failed to send the message: %v", err)
				}
			}
		} else if update.CallbackQuery != nil {
			data := strings.Split(update.CallbackQuery.Data, "/")
			action := data[0]
			entryHash := data[1]
			var entry *model.Entry
			entryIDSlice := []int64{}
			var err error
			if len(entryHash) > 0 {
				if entry, err = store.EntryByHash(entryHash); err != nil {
					logger.Error("[Telegram Bot] UserByTelegramChatID failed: %v", err)
				}
				entryIDSlice = append(entryIDSlice, entry.ID)
			}
			var user *model.User
			if user, err = store.UserByTelegramChatID(update.CallbackQuery.From.ID); err != nil {
				logger.Error("[Telegram Bot] UserByTelegramChatID failed: %v", err)
			}
			var newCallbackAction string
			var message string
			switch action {
			case "read":
				logger.Debug("[Telegram Bot] Marking entry %d as read.", entryIDSlice)
				if err = store.SetEntriesStatus(user.ID, entryIDSlice, "read"); err != nil {
					logger.Error("[Telegram Bot] SetEntriesStatus failed: %v", err)
				}
				newCallbackAction = "unread"
			case "unread":
				logger.Debug("[Telegram Bot] Marking entry %d as unread.", entryIDSlice)
				if err = store.SetEntriesStatus(user.ID, entryIDSlice, "unread"); err != nil {
					logger.Error("[Telegram Bot] SetEntriesStatus failed: %v", err)
				}
				newCallbackAction = "read"
			case "read_all":
				logger.Debug("[Telegram Bot] Marking all entries as read.")
				var err error
				var user *model.User
				if user, err = store.UserByTelegramChatID(update.CallbackQuery.Message.Chat.ID); err != nil {
					logger.Error("[Telegram Bot] UserByTelegramChatID failed: %v", err)
				}
				if err = store.MarkAllAsRead(user.ID); err != nil {
					logger.Error("[Telegram Bot] MarkAllAsRead failed: %v", err)
				} else {
					message = "Successfully marked everything as read!"
				}
			case "cancel":
				logger.Debug("[Telegram Bot] Cancelling action.")
				msg := tgbotapi.NewEditMessageTextAndMarkup(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
					"Action canceled.",
					tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
					},
				)
				if _, err := bot.Send(msg); err != nil {
					logger.Error("[Telegram Bot] Failed to send the message: %v", err)
				}
			}
			if len(newCallbackAction) > 0 {
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
					logger.Error("[Telegram Bot] Failed to send the message: %v", err)
				}
			} else if len(message) > 0 {
				msg := tgbotapi.NewEditMessageTextAndMarkup(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
					message,
					tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
					},
				)
				if _, err := bot.Send(msg); err != nil {
					logger.Error("[Telegram Bot] Failed to send the message: %v", err)
				}
			}
		}
	}
}
