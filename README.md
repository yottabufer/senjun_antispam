# SenJun antispam bot 

## Бот для удаления плохих сообщений 0.1

### Пример запуска бота из корневой директории, поддерживает флаг "--env-path" для указания где лежит config
```go
go run ./src --env-path="src/config.yaml"
```

## Пример файла config.yaml, который по умолчанию лежит в src/:
```yaml
telegram:
  token: "str_token"
  chat_id: int_chat_id
filter:
  message_count_to_white_list: 555
  min_message_length: 444
```

## Зависимости:
- [fsnotify](https://github.com/fsnotify/fsnotify)
- [telegram-bot-api/v5](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [viper](https://github.com/spf13/viper)
