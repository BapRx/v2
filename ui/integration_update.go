// Copyright 2018 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package ui // import "miniflux.app/ui"

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"

	"miniflux.app/http/request"
	"miniflux.app/http/response/html"
	"miniflux.app/http/route"
	"miniflux.app/locale"
	"miniflux.app/logger"
	"miniflux.app/service/telegrambot"
	"miniflux.app/ui/form"
	"miniflux.app/ui/session"
)

func (h *handler) updateIntegration(w http.ResponseWriter, r *http.Request) {
	printer := locale.NewPrinter(request.UserLanguage(r))
	sess := session.New(h.store, request.SessionID(r))
	user, err := h.store.UserByID(request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	integration, err := h.store.Integration(user.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	integrationForm := form.NewIntegrationForm(r)
	integrationForm.Merge(integration)

	if integration.FeverUsername != "" && h.store.HasDuplicateFeverUsername(user.ID, integration.FeverUsername) {
		sess.NewFlashErrorMessage(printer.Printf("error.duplicate_fever_username"))
		html.Redirect(w, r, route.Path(h.router, "integrations"))
		return
	}

	if integration.FeverEnabled {
		if integrationForm.FeverPassword != "" {
			integration.FeverToken = fmt.Sprintf("%x", md5.Sum([]byte(integration.FeverUsername+":"+integrationForm.FeverPassword)))
		}
	} else {
		integration.FeverToken = ""
	}

	if integration.GoogleReaderUsername != "" && h.store.HasDuplicateGoogleReaderUsername(user.ID, integration.GoogleReaderUsername) {
		sess.NewFlashErrorMessage(printer.Printf("error.duplicate_googlereader_username"))
		html.Redirect(w, r, route.Path(h.router, "integrations"))
		return
	}

	if integration.GoogleReaderEnabled {
		if integrationForm.GoogleReaderPassword != "" {
			integration.GoogleReaderPassword = integrationForm.GoogleReaderPassword
		}
	} else {
		integration.GoogleReaderPassword = ""
	}

	if integration.TelegramBotChatID != "" || integration.TelegramBotToken != "" {
		if integration.TelegramBotChatID != "" {
			if h.store.HasDuplicateTelegramChatID(user.ID, integration.TelegramBotChatID) {
				sess.NewFlashErrorMessage(printer.Printf("error.duplicate_telegram_chat_id"))
				html.Redirect(w, r, route.Path(h.router, "integrations"))
				return
			}
			// Check Telegram chat ID format
			_, err := strconv.ParseInt(integration.TelegramBotChatID, 10, 64)
			if err != nil {
				sess.NewFlashErrorMessage(printer.Printf("error.invalid_format_expected_number"))
				html.Redirect(w, r, route.Path(h.router, "integrations"))
				return
			}
		}
		if integration.TelegramBotToken != "" && h.store.HasDuplicateTelegramBotToken(user.ID, integration.TelegramBotToken) {
			sess.NewFlashErrorMessage(printer.Printf("error.duplicate_telegram_bot_token"))
			html.Redirect(w, r, route.Path(h.router, "integrations"))
			return
		}
		// Check if the bot is already running
		if integration.TelegramBotChatID != "" && integration.TelegramBotToken != "" {
			if err := telegrambot.AddBot(h.store, integration.TelegramBotToken, integration.TelegramBotChatID); err != nil {
				logger.Error("[Integrations] Failed configuring the new bot: %v", err)
			}
		}
	}

	if integration.TelegramBotPreviewLength != "" {
		i, err := strconv.ParseInt(integration.TelegramBotPreviewLength, 10, 64)
		if err != nil {
			sess.NewFlashErrorMessage(printer.Printf("error.invalid_format_expected_number"))
			html.Redirect(w, r, route.Path(h.router, "integrations"))
			return
		}
		if i < 0 || i > 4096 {
			sess.NewFlashErrorMessage(printer.Printf("error.telegram_preview_length_out_of_bound"))
			html.Redirect(w, r, route.Path(h.router, "integrations"))
			return
		}
	}

	err = h.store.UpdateIntegration(integration)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess.NewFlashMessage(printer.Printf("alert.prefs_saved"))
	html.Redirect(w, r, route.Path(h.router, "integrations"))
}
