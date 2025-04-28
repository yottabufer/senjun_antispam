package main

import (
	"bufio"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/viper"
)

// читает все строки из файла по указанному пути, принимает путь к файлу
func read_file_lines(path string) ([]string, error) {
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
func load_data() ([]string, []bool, error) {
	spam, err := read_file_lines("data_text/spam_data.txt")
	if err != nil {
		return nil, nil, err
	}
	ham, err := read_file_lines("data_text/ham_data.txt")
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

// delete_message удаляет указанное сообщение в телеге
// Принимает экземпляр бота и сообщение, которое нужно удалить
func delete_message(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	delete_config := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
	if _, err := bot.Request(delete_config); err != nil {
		log.Printf("Ошибка удаления соощения:\n%v", err)
	}
}

func main() {
	viper.SetConfigFile("../config.yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Ошибка чтения конфига:\n%v", err)
	}

	// горячая перезагрузка конфига
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Конфиг обновлен: %s", e.Name)
	})

	// Настройка наивного байеса
	bayes := NaiveBayes{
		exclude: make(map[string]struct{}),
	}
	exclude_content, _ := os.ReadFile("data_text/exclude_data.txt")
	exclude_words := strings.FieldsFunc(string(exclude_content), func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	for _, word := range exclude_words {
		bayes.exclude[strings.TrimSpace(word)] = struct{}{}
	}

	// Инициализация спам-фильтра
	filter := &SpamFilter{
		white_list:         make(map[int64]bool),
		user_message_count: make(map[int64]int),
	}

	// Загрузка белого списка
	whiteList, err := read_white_list("data_text/white_list.txt")
	if err != nil {
		log.Printf("Ошибка загрузки белого списка:\n%v", err)
	} else {
		filter.white_list = whiteList
	}

	// Обучение модели на исторических данных
	messages, labels, err := load_data()
	if err != nil {
		log.Fatal("Ошибка загрузки данных:\n", err)
	}
	bayes.train_model(messages, labels)

	// Инициализация Telegram-бота
	token := viper.GetString("telegram.token")
	if token == "" {
		log.Fatal("Токен не найден в конфиге")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Инициализация %s", bot.Self.UserName)
	chat_id := viper.GetInt64("telegram.chat_id")
	if chat_id == 0 {
		log.Fatal("chat_id не найден в конфиге")
	}

	// Настройка получения обновлений от телеграмма, можно увеличить цикл обновления
	tg_update := tgbotapi.NewUpdate(0)
	tg_update.Timeout = 60
	updates := bot.GetUpdatesChan(tg_update)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		user_id := update.Message.From.ID

		// Пропускаем пользователей из белого списка
		if filter.is_white_list(user_id) {
			log.Printf("Пользоватеть %d в белом списке", user_id)
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
		prediction := bayes.predict_for_message(update.Message.Text)
		log.Printf("Сообщение: %s | Спам: %v", update.Message.Text, prediction)
		if prediction {
			delete_message(bot, update.Message)
			continue
		}

		// Обновление счетчика сообщений, добавление в белый список после 5 сообщений
		count := filter.increment_message_count(user_id)
		if count >= viper.GetInt("filter.message_count_to_white_list") {
			if err := filter.add_to_white_list(user_id); err != nil {
				log.Printf("Ошибка добавления в белый список:\n%v", err)
			} else {
				log.Printf("Пользователь %d добавлен в белый список", user_id)
			}
		}
	}
}
