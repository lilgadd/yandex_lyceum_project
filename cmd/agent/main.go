package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"calc/models"
)

var taskMap = make(map[string]models.Task) // Мапа для хранения задач, ключ - ID задачи

// Функция для получения результата задачи
func getTaskResult(taskId string) (float64, error) {
	// Проверка, существует ли задача в мапе
	task, exists := taskMap[taskId]
	if !exists {
		return 0, fmt.Errorf("задача с ID %s не найдена", taskId)
	}

	// Ждем, пока задача не будет выполнена
	for !task.Status {
		log.Printf("Задача %s еще не выполнена. Ожидание...", taskId)
		time.Sleep(1 * time.Second) // Ждем 1 секунду перед повторной проверкой
		task, exists = taskMap[taskId] // Повторно получаем задачу из мапы
		if !exists {
			return 0, fmt.Errorf("задача с ID %s не найдена", taskId)
		}
	}

	// Возвращаем результат задачи, если она выполнена
	return task.Result, nil
}

// Функция для выполнения операции (например, сложение, вычитание)
func PerformOperation(task *models.Task) float64 {
	switch task.Operation {
	case "+":
		task.Result = task.Arg1 + task.Arg2
	case "-":
		task.Result = task.Arg1 - task.Arg2
	case "*":
		task.Result = task.Arg1 * task.Arg2
	case "/":
		if task.Arg2 != 0 {
			task.Result = task.Arg1 / task.Arg2
		} else {
			log.Println("Ошибка: деление на ноль!")
			task.Result = 0
		}
	default:
		log.Println("Неизвестная операция!")
		task.Result = 0
	}

	// Обновляем статус задачи на выполненную
	task.Status = true
	taskMap[task.Id] = *task // Обновляем задачу в мапе
	log.Printf("Задача %s выполнена. Результат: %f", task.Id, task.Result)

	return task.Result
}

// Функция для воркера
func worker(id int, pollInterval time.Duration, wg *sync.WaitGroup) {
	defer wg.Done() // Уменьшаем счётчик после завершения работы горутины

	client := &http.Client{Timeout: 5 * time.Second}
	orchestratorURL := "http://localhost:8080/internal/task"

	for {
		resp, err := client.Get(orchestratorURL)
		if err != nil {
			log.Printf("[Worker %d] Ошибка при получении задачи: %v", id, err)
			time.Sleep(pollInterval)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			time.Sleep(pollInterval)
			continue
		}

		var task models.Task
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			log.Printf("[Worker %d] Ошибка при декодировании задачи: %v", id, err)
			resp.Body.Close()
			time.Sleep(pollInterval)
			continue
		}
		resp.Body.Close()

		// Добавляем задачу в мапу (или обновляем её, если она уже есть)
		taskMap[task.Id] = task

		// Проверяем зависимости и ждём, пока они не будут выполнены
		for _, depId := range task.Dependencies {
			// Получаем результат зависимости
			for {
				result, err := getTaskResult(depId)
				if err != nil {
					log.Printf("[Worker %d] Ошибка при получении результата зависимости %s: %v", id, depId, err)
					time.Sleep(pollInterval) // Ждём, если зависимость ещё не выполнена
					continue
				}
				log.Printf("[Worker %d] Зависимость %s выполнена с результатом %f", id, depId, result)

				// Устанавливаем аргумент из зависимости
				if depId == task.Dependencies[0] { // Если это первая зависимость (которая используется в аргументе)
					task.Arg1 = result
				} else if depId == task.Dependencies[1] { // Если это вторая зависимость
					task.Arg2 = result
				}

				// Все зависимости выполнены
				break
			}
		}

		log.Printf("[Worker %d] Выполнение задачи %s", id, task.Id)
		// Выполняем операцию
		result := PerformOperation(&task)

		log.Printf("[Worker %d] Результат задачи %s: %f", id, task.Id, result)

		// Отправляем результат только если задача финальная
		if task.IsFinal {
			payload := models.Responce2{
			Id:     task.ExpressionID,
			Result: result,
			}
			data, _ := json.Marshal(payload)

			res, err := client.Post(orchestratorURL, "application/json", bytes.NewReader(data))
			if err != nil {
				log.Printf("[Worker %d] Ошибка при отправке финального результата: %v", id, err)
				continue
			}
			res.Body.Close()
			log.Printf("[Worker %d] Финальный результат задачи %s отправлен", id, task.Id)

			// Вывод финального результата в консоль
			fmt.Printf("ФИНАЛЬНЫЙ РЕЗУЛЬТАТ (%s): %.2f\n", task.ExpressionID, result)
}
	}
}

func main() {
	// Задать переменную окружения внутри программы (для локальной разработки или тестирования)
	os.Setenv("COMPUTING_POWER", "2")  // Пример, тут можно поставить любое значение

	computingPower := 1
	if val := os.Getenv("COMPUTING_POWER"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			computingPower = n
		}
	}

	log.Printf("Агент запущен с %d воркерами", computingPower)

	// Создаём объект sync.WaitGroup для ожидания завершения всех горутин
	var wg sync.WaitGroup

	// Запуск воркеров
	for i := 0; i < computingPower; i++ {
		wg.Add(1) // Увеличиваем счётчик горутин
		go worker(i+1, 500*time.Millisecond, &wg)
	}

	// Ожидаем завершения всех горутин
	wg.Wait()
}
