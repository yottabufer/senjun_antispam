package main

import (
	"math"
	"strings"
	"unicode"
)

type naiveBayes struct {
	spamCount        int                 // количество спам-сообщений
	hamCount         int                 // количество не-спам-сообщений
	spamWords        map[string]float64  // словарь с частотами слов в спам-сообщениях
	hamWords         map[string]float64  // словарь с частотами слов в не-спам сообщениях
	exclude          map[string]struct{} // Словарь исключений
	totalUniqueWords int                 // общее количество уникальных слов
}

// preprocessText выполняет предобработку текста:
// приводит к нижнему регистру
// удаляет пунктуацию
// разбивает на слова
func (bayes *naiveBayes) preprocessText(text string) []string {
	text = strings.ToLower(text)
	text = strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, text)

	words := strings.Fields(text)
	filtered := make([]string, 0, len(words))

	for _, word := range words {
		if _, excluded := bayes.exclude[word]; !excluded {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// trainModel обучает модель на наборе сообщений
// :param:messages: список текстовых сообщений
// :param:labels: список меток (true — спам, false — неспам)
// метод вычисляет частоты слов для каждого класса и применяет сглаживание Лапласа
func (bayes *naiveBayes) trainModel(messages []string, labels []bool) {
	bayes.spamWords = make(map[string]float64)
	bayes.hamWords = make(map[string]float64)
	spamTotal := 0
	hamTotal := 0
	uniqueWords := make(map[string]struct{})

	for i, msg := range messages {
		words := bayes.preprocessText(msg)
		for _, word := range words {
			uniqueWords[word] = struct{}{}
			if labels[i] {
				bayes.spamWords[word]++
				spamTotal++
			} else {
				bayes.hamWords[word]++
				hamTotal++
			}
		}
	}

	bayes.spamCount = spamTotal
	bayes.hamCount = hamTotal
	bayes.totalUniqueWords = len(uniqueWords)

	// Применяем сглаживание Лапласа для избежания нулевых вероятностей
	for word := range bayes.spamWords {
		bayes.spamWords[word] = (bayes.spamWords[word] + 1) / (float64(spamTotal) + float64(bayes.totalUniqueWords))
	}
	for word := range bayes.hamWords {
		bayes.hamWords[word] = (bayes.hamWords[word] + 1) / (float64(hamTotal) + float64(bayes.totalUniqueWords))
	}
}

// predictMessage определяет, является ли сообщение спамом.
// Возвращает true, если вероятность спама выше вероятности не-спама
// Использует https://ru.wikipedia.org/wiki/Преобразование_Лапласа для устойчивости вычислений
func (bayes *naiveBayes) predictMessage(msg string) bool {

	words := bayes.preprocessText(msg)

	if len(words) == 0 {
		return false
	}

	total := bayes.spamCount + bayes.hamCount
	// Начальные вероятности классов (логарифм для устойчивости)
	spam_prob := math.Log(float64(bayes.spamCount) / float64(total))
	ham_prob := math.Log(float64(bayes.hamCount) / float64(total))

	for _, word := range words {

		// Вероятность слова в спаме (с сглаживанием)
		if prob, exists := bayes.spamWords[word]; exists {
			spam_prob += math.Log(prob)
		} else {
			spam_prob += math.Log(1.0 / float64(bayes.spamCount+bayes.totalUniqueWords))
		}

		// Вероятность слова в не-спаме
		if prob, exists := bayes.hamWords[word]; exists {
			ham_prob += math.Log(prob)
		} else {
			ham_prob += math.Log(1.0 / float64(bayes.hamCount+bayes.totalUniqueWords))
		}
	}

	return spam_prob > ham_prob
}
