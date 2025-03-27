# Mattermost Voting Bot

Бот для создания и управления голосованиями в Mattermost.

## Функциональность

- Создание голосований с множественным выбором
- Голосование за предложенные варианты
- Просмотр текущих результатов голосования
- Завершение голосования
- Удаление голосования

## Требования

- Docker и Docker Compose
- Go 1.21 или выше
- Mattermost Server 5.37.0 или выше

## Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/yourusername/mattermost-voting-bot.git
cd mattermost-voting-bot
```

2. Соберите плагин:
```bash
go build -o server/dist/plugin.exe
```

3. Запустите dev-среду:
```bash
docker-compose up -d
```

4. Откройте Mattermost (http://localhost:8065) и войдите с учетными данными:
   - Email: admin@example.com
   - Password: admin123

5. Перейдите в System Console -> Plugins -> Plugin Management
6. Загрузите собранный плагин (server/dist/plugin.exe)
7. Включите плагин

## Использование

### Создание голосования
```
/vote create "Какой день недели?" "Понедельник" "Вторник" "Среда" "Четверг" "Пятница"
```

### Голосование
```
/vote vote [ID_голосования] [номер_варианта]
```

### Просмотр результатов
```
/vote results [ID_голосования]
```

### Завершение голосования
```
/vote end [ID_голосования]
```

### Удаление голосования
```
/vote delete [ID_голосования]
```

## Разработка

1. Внесите изменения в код
2. Соберите плагин:
```bash
go build -o server/dist/plugin.exe
```
3. Перезагрузите плагин в Mattermost

## Лицензия

MIT