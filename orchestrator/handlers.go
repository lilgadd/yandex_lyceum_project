package orchestrator

import (
	"calc/models"
	"encoding/json"
	"fmt"
	"net/http"
	"log"
    "github.com/go-chi/chi/v5"

)

func GetTaskHandler(w http.ResponseWriter, r *http.Request) {

	mutex.Lock()
	defer mutex.Unlock()

	// Проверяем, что в очереди есть задачи
	if len(TaskQueue) == 0 {
		http.Error(w, "Нет доступных задач", http.StatusNotFound)
		return
	}

	// Получаем первую задачу из очереди
	task := TaskQueue[0]
	TaskQueue = TaskQueue[1:]

	// Отправляем задачу агенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при отправке задачи: %v", err), http.StatusInternalServerError)
	}

	log.Printf("Задача с ID %s была отправлена агенту", task.Id)
}




func PostTaskResultHandler(w http.ResponseWriter, r *http.Request) {
    // Декодируем тело запроса в структуру TaskResult
    var taskResult models.Responce2
    if err := json.NewDecoder(r.Body).Decode(&taskResult); err != nil {
        http.Error(w, "Невалидные данные", http.StatusUnprocessableEntity) 
        return
    }

    // Используем мьютекс для синхронизации доступа к данным
    mutex.Lock()
    defer mutex.Unlock()

    // Проверяем, есть ли такое выражение в мапе
    expression, exists := Expressions[taskResult.Id]
    if !exists {
        http.Error(w, "Задача не найдена", http.StatusNotFound) 
        return
    }

    // Обновляем результат вычисления
    expression.Result = taskResult.Result
    expression.Status = "завершено"

    // Отправляем успешный ответ
    w.WriteHeader(http.StatusOK) 
    w.Write([]byte("Результат успешно записан"))
}

func GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Предположим, что Expressions - это мапа, где ключ - ID выражения, а значение - указатель на структуру Expression
	mutex.Lock()
	defer mutex.Unlock()

	// Создаём срез для хранения всех выражений
	var response struct {
		Expressions []models.Expression `json:"expressions"`
	}

	// Перебираем все выражения в мапе и добавляем их в срез
	for _, expr := range Expressions {
		// Убедимся, что указатель на выражение не nil
		if expr != nil {
			response.Expressions = append(response.Expressions, *expr)
		}
	}

	// Отправляем ответ в формате JSON
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func GetExpressionByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем параметр {id} из URL
	id := chi.URLParam(r, "id")

	// Ищем выражение по ID в мапе
	expr, ok := Expressions[id]
	if !ok {
		http.Error(w, "expression not found", http.StatusNotFound)
		return
	}

	// Формируем ответ
	response := map[string]interface{}{
		"expression": map[string]interface{}{
			"id":     expr.Id,
			"status": expr.Status,
			"result": expr.Result,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}


