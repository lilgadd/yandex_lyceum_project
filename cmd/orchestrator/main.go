package main

import (
	"log"
	"net/http"
	"github.com/go-chi/chi/v5"
	"calc/orchestrator"
	"calc/database"
	"fmt"
)

func main() {

	r := chi.NewRouter()


	// Инициализация базы данных
	userDB, expressionDB, err := database.InitDB()
	if err != nil {
		log.Fatal("ошибка при инициализации баз данных:", err)
	}
	defer userDB.Close()
	defer expressionDB.Close()

	fmt.Println("База данных пользователей успешно инициализирована")
	fmt.Println("База данных выражений успешно инициализирована")

	// Эндпоинты API
	r.Post("/api/v1/calculate", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        orchestrator.CalculateHandler(w, r, expressionDB)
    }))
	r.Get("/internal/task", orchestrator.GetTaskHandler)
	r.Post("/internal/task", func(w http.ResponseWriter, r *http.Request) {
	orchestrator.PostTaskResultHandler(w, r, expressionDB)
	})

	r.Get("/api/v1/expressions", func(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Получен запрос:", r.Method, r.URL.Path)
    if r.Method == http.MethodGet {
        orchestrator.GetExpressionsHandler(w, r, expressionDB)
    } else {
        http.Error(w, "метод недоступен", http.StatusMethodNotAllowed)
    }
	})


	r.Get("/api/v1/expressions/{id}", func(w http.ResponseWriter, r *http.Request) {
	orchestrator.GetExpressionByID(w, r, expressionDB)
	})

	r.Post("/api/v1/register", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	orchestrator.RegisterHandler(w, r, userDB)
	}))
	r.Post("/api/v1/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	orchestrator.LoginHandler(w, r, userDB)
	}))

	// Запуск сервера
	log.Println("Сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}