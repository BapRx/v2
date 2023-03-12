// Copyright 2021 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package telegrambot // import "miniflux.app/integration/telegrambot"

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"miniflux.app/model"
	"miniflux.app/telegrambot/client"
)

// PushEntry pushes entry to telegram chat using integration settings provided
func PushEntry(entry *model.Entry, botToken, chatID string, sendContent bool) error {
	tplStr := "<b>{{ .Title }}</b>"
	if sendContent {
		tplStr += "\n\n{{ .Content }}"
	}
	tpl, err := template.New("message").Parse(tplStr)
	if err != nil {
		return fmt.Errorf("telegrambot: template parsing failed: %w", err)
	}

	var result bytes.Buffer
	if err := tpl.Execute(&result, entry); err != nil {
		return fmt.Errorf("telegrambot: template execution failed: %w", err)
	}

	chatIDInt, _ := strconv.ParseInt(chatID, 10, 64)
	resultStr := result.String()
	if result.Len() > 4096 {
		resultStr = resultStr[0:4096]
	}
	msg := tgbotapi.NewMessage(chatIDInt, resultStr)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = false

	callbackData := fmt.Sprintf("read/%s", entry.Hash)
	if len(callbackData) > 64 {
		callbackData = callbackData[:64]
	}
	buttonRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonURL("Open", entry.URL),
		tgbotapi.NewInlineKeyboardButtonData("Mark as read", callbackData),
	)
	if entry.CommentsURL != "" {
		commentButton := tgbotapi.NewInlineKeyboardButtonURL("Comments", entry.CommentsURL)
		buttonRow = append(buttonRow, commentButton)
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttonRow)
	client.SendMessage(msg)

	return nil
}
