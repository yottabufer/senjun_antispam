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
	spam, err := readFileLines("src/data_text/spam_data.txt")
	if err != nil {
		return nil, nil, err
	}
	ham, err := readFileLines("src/data_text/ham_data.txt")
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
	configPath := flag.String("env-path", "../config.yaml", "Path to config file")
	flag.Parse()
	viper.SetConfigFile(*configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config.yaml file:\n%v", err)
	}
	logger := &logger{
		level: strings.ToLower(viper.GetString("logging.level")),
	}
	// горячая перезагрузка конфига
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logger.info("config.yml has been updated: %s", e.Name)
	})

	// Настройка наивного байеса
	bayes := naiveBayes{
		exclude: make(map[string]struct{}),
	}
	exclude_content, _ := os.ReadFile("src/data_text/exclude_data.txt")
	exclude_words := strings.FieldsFunc(string(exclude_content), func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	for _, word := range exclude_words {
		bayes.exclude[strings.TrimSpace(word)] = struct{}{}
	}

	// Инициализация спам-фильтра
	spamFilter := &SpamFilter{
		whiteList:        make(map[int64]bool),
		userMessageCount: make(map[int64]int),
	}

	// Загрузка белого списка
	whiteList, err := loadWhiteList("src/data_text/white_list.txt")
	if err != nil {
		logger.error("Error loading the whitelist:\n%v", err)
	} else {
		spamFilter.whiteList = whiteList
	}

	// Обучение модели на исторических данных
	messages, labels, err := loadData()
	if err != nil {
		logger.error("Error loading data for training the model:\n%v", err)
	}
	bayes.trainModel(messages, labels)

	// Инициализация Telegram-бота
	token := viper.GetString("telegram.token")
	if token == "" {
		logger.error("Telegram bot token not found in config.yaml")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.error("%v", err)
	}
	log.Printf("Initialization %s", bot.Self.UserName)
	chatID := viper.GetInt64("telegram.chat_id")
	if chatID == 0 {
		logger.error("chatID not found in config.yaml")
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
		if spamFilter.isInWhiteList(userID) {
			logger.debug("User %d is on the whitelist", userID)
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
		prediction := bayes.predictMessage(update.Message.Text)
		log.Printf("Message: %s | Spam: %v", update.Message.Text, prediction)
		if prediction {
			deleteMessage(bot, update.Message)
			continue
		}

		// Обновление счетчика сообщений, добавление в белый список после 5 сообщений
		count := spamFilter.incrementMessageCount(userID)
		if count >= viper.GetInt("filter.message_count_to_white_list") {
			if err := spamFilter.addToWhiteList(userID); err != nil {
				logger.error("The error of adding to the whitelist:\n%v", err)
			} else {
				logger.debug("User %d added to the whitelist", userID)
			}
		}
	}
}
