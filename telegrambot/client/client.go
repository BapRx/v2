// Copyright 2023 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package client // import "miniflux.app/telegram/client"

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"miniflux.app/logger"
)

var bot *tgbotapi.BotAPI

// Serve get updates from the Telegram API
func New(botToken string) *tgbotapi.BotAPI {
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logger.Fatal("Telegram bot failed to start: %w", err)
	}

	return bot
}

func SendMessage(msg tgbotapi.MessageConfig) {
	if _, err := bot.Send(msg); err != nil {
		if err.Error() == "Too Many Requests" {
			logger.Debug("telegram: rate limited while sending message, sleeping for 5 seconds")
			time.Sleep(5)
			if _, err := bot.Send(msg); err != nil {
				logger.Error("telegram: sending message failed: %w", err)
			}
		} else {
			logger.Error("telegram: sending message failed: %w", err)
		}
	}
}
