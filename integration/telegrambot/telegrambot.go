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
func PushEntry(entry *model.Entry, botToken, chatID string, previewLengthStr string) error {
	tplTitle, err := template.New("title").Parse("<b>{{ .Title }}</b>")
	if err != nil {
		return fmt.Errorf("telegrambot: template parsing failed: %w", err)
	}
	var resultTitle bytes.Buffer
	if err := tplTitle.Execute(&resultTitle, entry); err != nil {
		return fmt.Errorf("telegrambot: template execution failed: %w", err)
	}
	message := resultTitle.String()
	previewLength, err := strconv.Atoi(previewLengthStr)
	if err != nil {
		panic(err)
	}
	if previewLength > 0 {
		tplContent, err := template.New("content").Parse("\n\n{{ .Content }}")
		if err != nil {
			return fmt.Errorf("telegrambot: template parsing failed: %w", err)
		}
		var resultTitle bytes.Buffer
		if err := tplContent.Execute(&resultTitle, entry); err != nil {
			return fmt.Errorf("telegrambot: template execution failed: %w", err)
		}
		content := resultTitle.String()
		if resultTitle.Len() > previewLength {
			content = content[0:previewLength]
		}
		message += content
	}

	chatIDInt, _ := strconv.ParseInt(chatID, 10, 64)
	msg := tgbotapi.NewMessage(chatIDInt, message)
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
