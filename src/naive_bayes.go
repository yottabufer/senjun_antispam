package main

import (
	"math"
	"strings"
	"unicode"
)

type NaiveBayes struct {
	spam_count int                 // количество спам-сообщений
	ham_count  int                 // количество не-спам-сообщений
	spam_words map[string]float64  // словарь с частотами слов в спам-сообщениях
	ham_words  map[string]float64  // словарь с частотами слов в не-спам сообщениях
	exclude    map[string]struct{} // Словарь исключений
	all_unique int                 // общее количество уникальных слов
}

// preprocess_for_text выполняет предобработку текста:
// приводит к нижнему регистру
// удаляет пунктуацию
// разбивает на слова
func (bayes *NaiveBayes) preprocess_for_text(text string) []string {
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

// train_model обучает модель на наборе сообщений
// :param:messages: список текстовых сообщений
// :param:labels: список меток (true — спам, false — неспам)
// метод вычисляет частоты слов для каждого класса и применяет сглаживание Лапласа
func (bayes *NaiveBayes) train_model(messages []string, labels []bool) {
	bayes.spam_words = make(map[string]float64)
	bayes.ham_words = make(map[string]float64)
	spam_total := 0
	ham_total := 0
	unique_word := make(map[string]bool)

	for i, msg := range messages {
		words := bayes.preprocess_for_text(msg)
		for _, word := range words {
			unique_word[word] = true
			if labels[i] {
				bayes.spam_words[word]++
				spam_total++
			} else {
				bayes.ham_words[word]++
				ham_total++
			}
		}
	}

	bayes.spam_count = spam_total
	bayes.ham_count = ham_total
	bayes.all_unique = len(unique_word)

	// Применяем сглаживание Лапласа для избежания нулевых вероятностей
	for word := range bayes.spam_words {
		bayes.spam_words[word] = (bayes.spam_words[word] + 1) / (float64(spam_total) + float64(bayes.all_unique))
	}
	for word := range bayes.ham_words {
		bayes.ham_words[word] = (bayes.ham_words[word] + 1) / (float64(ham_total) + float64(bayes.all_unique))
	}
}

// predict_for_message определяет, является ли сообщение спамом.
// Возвращает true, если вероятность спама выше вероятности не-спама
// Использует https://ru.wikipedia.org/wiki/Преобразование_Лапласа для устойчивости вычислений
func (bayes *NaiveBayes) predict_for_message(msg string) bool {

	words := bayes.preprocess_for_text(msg)

	total := bayes.spam_count + bayes.ham_count
	// Начальные вероятности классов (логарифм для устойчивости)
	spam_prob := math.Log(float64(bayes.spam_count) / float64(total))
	ham_prob := math.Log(float64(bayes.ham_count) / float64(total))

	for _, word := range words {

		// Вероятность слова в спаме (с сглаживанием)
		if prob, exists := bayes.spam_words[word]; exists {
			spam_prob += math.Log(prob)
		} else {
			spam_prob += math.Log(1.0 / float64(bayes.spam_count+bayes.all_unique))
		}

		// Вероятность слова в не-спаме
		if prob, exists := bayes.ham_words[word]; exists {
			ham_prob += math.Log(prob)
		} else {
			ham_prob += math.Log(1.0 / float64(bayes.ham_count+bayes.all_unique))
		}
	}

	return spam_prob > ham_prob
}
