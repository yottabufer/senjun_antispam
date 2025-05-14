package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/viper"

	"senjun_antispam/internal/classifier"
	"senjun_antispam/internal/filter"
	"senjun_antispam/internal/logger"
)

// читает все строки из файла по указанному пути, принимает путь к файлу
func readFileLines(path string) ([]string, error) {
	var lines []string
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// загружает обучающие данные (спам и ham) для модели
func loadData() ([]string, []bool, error) {
	spam, err := readFileLines("data_text/spam_data.txt")
	if err != nil {
		return nil, nil, err
	}
	ham, err := readFileLines("data_text/ham_data.txt")
	if err != nil {
		return nil, nil, err
	}
	messages := append(spam, ham...)
	labels := make([]bool, len(messages))
	for i := range spam {
		labels[i] = true
	}
	return messages, labels, nil
}

// deleteMessage удаляет указанное сообщение в телеге
// Принимает экземпляр бота и сообщение, которое нужно удалить
func deleteMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	delete_config := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
	if _, err := bot.Request(delete_config); err != nil {
		log.Printf("Message deletion error:\n%v", err)
	}
}

func main() {
	// Достаём параметры запуска
	configPath := flag.String("env-path", "configs/config.yaml", "Path to config file")
	flag.Parse()
	viper.SetConfigFile(*configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config.yaml file:\n%v", err)
	}
	logger := logger.Logger{
		Level: strings.ToLower(viper.GetString("logging.level")),
	}
	// горячая перезагрузка конфига
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logger.Info("config.yml has been updated: %s", e.Name)
	})

	// Настройка наивного байеса
	bayes := classifier.NaiveBayes{
		Exclude: make(map[string]struct{}),
	}
	exclude_content, _ := os.ReadFile("data_text/exclude_data.txt")
	exclude_words := strings.FieldsFunc(string(exclude_content), func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	for _, word := range exclude_words {
		bayes.Exclude[strings.TrimSpace(word)] = struct{}{}
	}

	// Инициализация спам-фильтра
	spamFilter := filter.SpamFilter{
		WhiteList:        make(map[int64]bool),
		UserMessageCount: make(map[int64]int),
	}

	// Загрузка белого списка
	whiteList, err := filter.LoadWhiteList("data_text/white_list.txt")
	if err != nil {
		logger.Error("Error loading the whitelist:\n%v", err)
	} else {
		spamFilter.WhiteList = whiteList
	}

	// Обучение модели на исторических данных
	messages, labels, err := loadData()
	if err != nil {
		logger.Error("Error loading data for training the model:\n%v", err)
	}
	bayes.TrainModel(messages, labels)

	// Инициализация Telegram-бота
	token := viper.GetString("telegram.token")
	if token == "" {
		logger.Error("Telegram bot token not found in config.yaml")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("%v", err)
	}
	log.Printf("Initialization %s", bot.Self.UserName)
	chatID := viper.GetInt64("telegram.chat_id")
	if chatID == 0 {
		logger.Error("chatID not found in config.yaml")
	}

	// Настройка получения обновлений от телеграмма, можно увеличить цикл обновления
	tgUpdate := tgbotapi.NewUpdate(0)
	tgUpdate.Timeout = 60
	updates := bot.GetUpdatesChan(tgUpdate)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID

		// Пропускаем пользователей из белого списка
		if spamFilter.IsInWhiteList(userID) {
			logger.Debug("User %d is on the whitelist", userID)
			continue
		}

		// Пропускаем медиафайлы и короткие сообщения
		// TODO: Пока не знаю как обрабатывать всевозможные файлы, но обязательно вернусь
		if update.Message.Photo != nil ||
			update.Message.Video != nil ||
			update.Message.Document != nil {
			continue
		}
		if len(update.Message.Text) < viper.GetInt("filter.min_message_length") {
			continue
		}

		// Классификация сообщения и удаление сообщения
		prediction := bayes.PredictMessage(update.Message.Text)
		log.Printf("Message: %s | Spam: %v", update.Message.Text, prediction)
		if prediction {
			deleteMessage(bot, update.Message)
			continue
		}

		// Обновление счетчика сообщений, добавление в белый список после 5 сообщений
		count := spamFilter.IncrementMessageCount(userID)
		if count >= viper.GetInt("filter.message_count_to_white_list") {
			if err := spamFilter.AddToWhiteList(userID); err != nil {
				logger.Error("The error of adding to the whitelist:\n%v", err)
			} else {
				logger.Debug("User %d added to the whitelist", userID)
			}
		}
	}
}
