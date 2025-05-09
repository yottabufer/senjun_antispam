package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

// управляет белым списком и счетчиком сообщений пользователей
type SpamFilter struct {
	whiteList        map[int64]bool // белый список пользователей (защищён мьютексом)
	userMessageCount map[int64]int  // счетчик сообщений для каждого пользователя (защищён мьютексом)
	mutex            sync.Mutex     // для безопасного доступа из разных горутин
}

// проверяет наличие пользователя в белом списке
func (filter *SpamFilter) isInWhiteList(userID int64) bool {
	filter.mutex.Lock() // Блокируем доступ к данным
	defer filter.mutex.Unlock()
	return filter.whiteList[userID]
}

// увеличивает счетчик сообщений пользователя
func (filter *SpamFilter) incrementMessageCount(userID int64) int {
	filter.mutex.Lock()
	defer filter.mutex.Unlock()
	filter.userMessageCount[userID]++
	return filter.userMessageCount[userID]
}

// добавляет пользователя в белый список и сохраняет в файл
func (filter *SpamFilter) addToWhiteList(userID int64) error {
	filter.mutex.Lock()
	defer filter.mutex.Unlock()
	log.Print(filter.whiteList)

	// Проверяем, существует ли уже в мапе
	if filter.whiteList[userID] {
		return nil
	}

	filter.whiteList[userID] = true
	file, err := os.OpenFile("src/data_text/white_list.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(fmt.Sprintf("%d\n", userID))
	return err
}

// загружает белый список из файла
func loadWhiteList(filename string) (map[int64]bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int64]bool), nil
		}
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	list := make(map[int64]bool)
	for scanner.Scan() {
		if id, err := strconv.ParseInt(scanner.Text(), 10, 64); err == nil {
			list[id] = true
		}
	}
	return list, scanner.Err()
}
