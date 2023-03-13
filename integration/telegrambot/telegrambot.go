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
	"miniflux.app/logger"
	"miniflux.app/model"
	"miniflux.app/telegrambot/client"
)

func renderTemplate(tplStr string, data interface{}) string {
	tpl, err := template.New("footer").Parse(tplStr)
	if err != nil {
		logger.Error("[telegrambot]: template parsing failed: %w", err)
		return ""
	}
	var result bytes.Buffer
	if err := tpl.Execute(&result, data); err != nil {
		logger.Error("[telegrambot]: template execution failed: %w", err)
		return ""
	}
	return result.String()
}

// PushEntry pushes entry to telegram chat using integration settings provided
func PushEntry(entry *model.Entry, botToken, chatID string, previewLengthStr string) error {
	title := renderTemplate("<b>{{ .Title }}</b>", entry)
	message := title
	previewLength, err := strconv.Atoi(previewLengthStr)
	if err != nil {
		return fmt.Errorf("[telegrambot]: preview length type conversion failed: %w", err)
	}
	if previewLength > 0 {
		content := renderTemplate("\n\n{{ .Content }}", entry)
		if len(content) > previewLength {
			content = content[0:previewLength]
		}
		message += content
	}
	footer := renderTemplate("\n\n<i>{{ .Date }} - {{ .Author }}</i>", entry)
	message += footer

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
	client.SendMessage(chatID, msg)

	return nil
}
