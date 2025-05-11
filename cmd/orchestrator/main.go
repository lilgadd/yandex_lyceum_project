package main

import (
	"log"
	"net/http"
	"github.com/go-chi/chi/v5"
	"calc/orchestrator"
)


func main() {

	r := chi.NewRouter()

	// Эндпоинты API
	r.Post("/api/v1/calculate", orchestrator.CalculateHandler)
	r.Get("/internal/task", orchestrator.GetTaskHandler)
	r.Post("/internal/task", orchestrator.PostTaskResultHandler)
	r.Get("/api/v1/expressions", orchestrator.GetExpressionsHandler)
	r.Get("/api/v1/expressions/{id}", orchestrator.GetExpressionByID)
	// Запуск сервера
	log.Println("Сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}