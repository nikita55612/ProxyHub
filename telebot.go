package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var telebotOwner int64
var telebotAccessCode string

type TelebotParams struct {
	Token         string
	OwnerID       string
	AccessCode    string
	WebApp        string
	UsersFilePath string
}

var usersFilePath string
var usersFileMu sync.Mutex
var usersFileCache map[int64]*TelebotUserInfo

type TelebotUserInfo struct {
}

func RunTelebot(ctx context.Context, stop context.CancelFunc, params *TelebotParams) {
	defer stop()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
		bot.WithCallbackQueryDataHandler("", bot.MatchTypePrefix, сallbackHandler),
	}

	b, err := bot.New(params.Token, opts...)
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// _, err = b.SetChatMenuButton(ctx, &bot.SetChatMenuButtonParams{
	// 	ChatID: nil,
	// 	MenuButton: models.MenuButtonWebApp{
	// 		Type: "web_app",
	// 		Text: "🌐",
	// 		WebApp: models.WebAppInfo{
	// 			URL: params.WebApp,
	// 		},
	// 	},
	// })
	if err != nil {
		log.Fatalf("Failed to set chat menu button: %v", err)
	}

	telebotOwnerInt, err := strconv.Atoi(params.OwnerID)
	if err != nil {
		log.Fatalf("Parse Int OwnerID: %v", err)
	}

	telebotOwner = int64(telebotOwnerInt)
	telebotAccessCode = params.AccessCode
	usersFilePath = params.UsersFilePath

	_, err = os.Stat(usersFilePath)

	if os.IsNotExist(err) {
		file, err := os.Create(usersFilePath)
		if err != nil {
			log.Fatalf("Failed to create a file: %v", err)
		}
		defer file.Close()
	}

	if err = ReadTelebotUsersFromFile(); err != nil {
		log.Fatalf("ReadTelebotUsersFromFile error: %v", err)
	}

	log.Println("Telegram bot started.")
	b.Start(ctx)
	log.Println("Telegram bot stopped gracefully.")
}

func ReadTelebotUsersFromFile() error {
	usersFileMu.Lock()
	defer usersFileMu.Unlock()

	if usersFileCache == nil {
		usersFileCache = make(map[int64]*TelebotUserInfo)
	}

	file, err := os.Open(usersFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if parts == nil {
			continue
		}
		if len(parts) < 2 {
			continue
		}
		userID, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		usersFileCache[int64(userID)] = nil
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func WriteNewTelebotUser(userId int64, info *TelebotUserInfo) error {
	usersFileMu.Lock()
	defer usersFileMu.Unlock()

	file, err := os.OpenFile(usersFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, userId, "?")
	if err != nil {
		return err
	}

	usersFileCache[userId] = info

	return nil
}

func DelTelebotUser(userId int64) error {
	tmpPath := usersFilePath + ".tmp"

	in, err := os.Open(usersFilePath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	scanner := bufio.NewScanner(in)
	writer := bufio.NewWriter(out)

	prefix := strconv.Itoa(int(userId)) + " "

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, prefix) {
			writer.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, usersFilePath); err != nil {
		return err
	}

	delete(usersFileCache, userId)

	return nil
}

func IsExistTelebotUser(userId int64) bool {
	usersFileMu.Lock()
	defer usersFileMu.Unlock()

	_, ok := usersFileCache[userId]

	return ok
}

func GetAllUserIDs() []int64 {
	usersFileMu.Lock()
	defer usersFileMu.Unlock()

	result := make([]int64, len(usersFileCache))
	for userID := range usersFileCache {
		result = append(result, userID)
	}

	return result
}

func GetUsersCount() int {
	usersFileMu.Lock()
	defer usersFileMu.Unlock()

	return len(usersFileCache)
}

func TrimCommand(text string, command string) string {
	return strings.TrimSpace(strings.TrimPrefix(text, command))
}

func GetClientForUser(ctx context.Context, b *bot.Bot, userID int64) (*models.Message, error) {
	clientText := fmt.Sprintf(`<u><i><b>🤡 Client</b></i></u>

Количество участников: <b>%d</b>`, GetUsersCount())

	return b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      clientText,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text: "🌐 ProxyHub",
						URL:  serverFullExternalURL,
					},
				},
				{

					{
						Text:         "🚫 Удалить авторизацию",
						CallbackData: "del_auth",
					},
				},
			},
		},
	})
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	if update.Message.From.IsBot {
		return
	}
	replay := func(text string) (*models.Message, error) {
		return b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		})
	}
	if strings.HasPrefix(update.Message.Text, "/start") {
		if IsExistTelebotUser(update.Message.Chat.ID) {
			GetClientForUser(ctx, b, update.Message.Chat.ID)
			return
		}

		userAccessCode := TrimCommand(update.Message.Text, "/start")
		if userAccessCode == telebotAccessCode {
			WriteNewTelebotUser(update.Message.Chat.ID, nil)
			replay("Пользователь авторизован")
			GetClientForUser(ctx, b, update.Message.Chat.ID)
		} else {
			replay("Доступ отклонен")
		}

		return
	}
	if update.Message.Chat.ID == telebotOwner {
		if strings.HasPrefix(update.Message.Text, "/help") {
			replay("/send - отправить всем")

			return
		}
		if strings.HasPrefix(update.Message.Text, "/send") {
			for _, userID := range GetAllUserIDs() {
				b.ForwardMessage(ctx, &bot.ForwardMessageParams{
					ChatID:     userID,
					FromChatID: telebotOwner,
					MessageID:  update.Message.ID,
				})
			}
			return
		}
		if strings.HasPrefix(update.Message.Caption, "/send") {
			for _, userID := range GetAllUserIDs() {
				b.ForwardMessage(ctx, &bot.ForwardMessageParams{
					ChatID:     userID,
					FromChatID: telebotOwner,
					MessageID:  update.Message.ID,
				})
			}
			return
		}
	}
	if !IsExistTelebotUser(update.Message.Chat.ID) && update.Message.Chat.ID != telebotOwner {
		replay("Доступ отклонен")

		return
	}
	if strings.HasPrefix(update.Message.Text, "/send") {
		replay("🤡")
	}
	if strings.HasPrefix(update.Message.Text, "/client") {
		GetClientForUser(ctx, b, update.Message.Chat.ID)
	}

	strUpd, _ := json.MarshalIndent(update, "", "     ")
	fmt.Printf("%s\n", string(strUpd))

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}

func сallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	replay := func(text string) (*models.Message, error) {
		return b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.From.ID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		})
	}
	if !IsExistTelebotUser(update.CallbackQuery.From.ID) && update.CallbackQuery.From.ID != telebotOwner {
		replay("Доступ отклонен")

		return
	}

	if update.CallbackQuery.Data == "del_auth" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.From.ID,
			Text:      "Подтвердить удаление авторизации",
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{

						{
							Text:         "OK",
							CallbackData: "del_auth_",
						},
					},
				},
			},
		})
	}
	if update.CallbackQuery.Data == "del_auth_" {
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    update.CallbackQuery.From.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
		})
		DelTelebotUser(update.CallbackQuery.From.ID)
	}

	strUpd, _ := json.MarshalIndent(update, "", "     ")
	fmt.Printf("%s\n\n-------------", string(strUpd))
}
