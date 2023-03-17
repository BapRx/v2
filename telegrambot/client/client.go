// Copyright 2023 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package client // import "miniflux.app/telegram/client"

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"miniflux.app/logger"
)

var bots = make(map[string]tgbotapi.BotAPI)

// Create a new Telegram bot API client.
func New(botToken, chatID string) error {
	var bot *tgbotapi.BotAPI
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return err
	}
	logger.Info("[Telegram] Bot configured for the chat ID %s", chatID)
	bots[chatID] = *bot
	return nil
}

// Returns a Telegram bot API client from a Chat ID.
func Get(chatID string) (tgbotapi.BotAPI, error) {
	bot, ok := bots[chatID]
	if !ok {
		return tgbotapi.BotAPI{}, fmt.Errorf("There is no running Telegram bot for the provided chat ID (%s)", chatID)
	}
	return bot, nil
}

func SendMessage(chatID string, msg tgbotapi.MessageConfig) error {
	bot, err := Get(chatID)
	if err != nil {
		return err
	}
	if _, err := bot.Send(msg); err != nil {
		if err.Error() == "Too Many Requests" {
			logger.Debug("telegram: rate limited while sending message, sleeping for 5 seconds")
			time.Sleep(5 * time.Second)
			if _, err := bot.Send(msg); err != nil {
				logger.Error("telegram: sending message failed: %w", err)
			}
		} else {
			logger.Error("telegram: sending message failed: %w", err)
		}
	}
	return nil
}
