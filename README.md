# SenJun antispam bot 

## Бот для удаления плохих сообщений 0.1

### Пример запуска бота из корневой директории, поддерживает флаг "--env-path" для указания где лежит config
```go
go run ./src --env-path="configs/config.yaml"
```

## Зависимости:
- [fsnotify](https://github.com/fsnotify/fsnotify)
- [telegram-bot-api/v5](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [viper](https://github.com/spf13/viper)
