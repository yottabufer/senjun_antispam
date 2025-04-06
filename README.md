# SenJun antispam bot 

## Бот для удаления плохих сообщений 0.1

### Бот запускает из /src
```go
go run .
```

## В корне в config.yaml нужно разместить:
```yaml
telegram:
  token: "str_token"
  chat_id: int_chat_id

```

## Зависимости:
- [fsnotify](https://github.com/fsnotify/fsnotify)
- [telegram-bot-api/v5](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [viper](https://github.com/spf13/viper)