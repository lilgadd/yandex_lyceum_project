package orchestrator

import (
	"calc/models"
	"encoding/json"
	"fmt"
	"net/http"
	"log"
    "github.com/go-chi/chi/v5"
    "calc/database"
	"github.com/golang-jwt/jwt/v5"
    "time"
    "database/sql"
    "golang.org/x/crypto/bcrypt"
    "strings"
)

func GetTaskHandler(w http.ResponseWriter, r *http.Request) {

	mutex.Lock()
	defer mutex.Unlock()

	// Проверяем, что в очереди есть задачи
	if len(TaskQueue) == 0 {
		http.Error(w, "нет доступных задач", http.StatusNotFound)
		return
	}

	// Получаем первую задачу из очереди
	task := TaskQueue[0]
	TaskQueue = TaskQueue[1:]

	// Отправляем задачу агенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, fmt.Sprintf("ошибка при отправке задачи: %v", err), http.StatusInternalServerError)
	}

	// log.Printf("Задача с ID %s была отправлена агенту", task.Id)
}


func PostTaskResultHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// log.Println("Получен POST запрос для записи результата")

	var taskResult models.Responce2
	if err := json.NewDecoder(r.Body).Decode(&taskResult); err != nil {
		http.Error(w, "невалидные данные", http.StatusUnprocessableEntity)
		return
	}

	// Обновляем в БД
	err := UpdateExpressionResultAndStatus(db, taskResult.Id, taskResult.Result, "завершено")
	if err != nil {
		http.Error(w, fmt.Sprintf("что-то пошло не так"), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("результат успешно записан"))
}

func UpdateExpressionResultAndStatus(db *sql.DB, id string, result float64, status string) error {
	res, err := db.Exec(`UPDATE expressions SET result = ?, status = ? WHERE id = ?`, result, status, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	log.Printf("обновлено строк: %d", rowsAffected)
	return nil
}



func GetExpressionsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Извлекаем токен
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "невалидные данные", http.StatusInternalServerError)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(database.JwtSecret), nil // замени на свой секрет
	})
	if err != nil {
		http.Error(w, "невалидные данные", http.StatusInternalServerError)
		return
	}

	// 2. Достаём user_id из токена
	userID, ok := claims["user_id"].(string)
	if !ok {
		http.Error(w, "невалидные данные", http.StatusInternalServerError)
		return
	}

	// 3. Готовим результат
	var response struct {
		Expressions []models.Expression `json:"expressions"`
	}

	// 4. Выполняем запрос с фильтрацией по user_id
	rows, err := db.Query(`SELECT id, status, result FROM expressions WHERE user_id = ?`, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка при получении данных: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var expr models.Expression
		var result sql.NullFloat64

		if err := rows.Scan(&expr.Id, &expr.Status, &result); err != nil {
			http.Error(w, fmt.Sprintf("ошибка при сканировании данных: %v", err), http.StatusInternalServerError)
			return
		}

		// Если результат не null — присваиваем, иначе 0.0 (или пропускаем)
		if result.Valid {
			expr.Result = result.Float64
		}

		response.Expressions = append(response.Expressions, expr)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "ошибка при отправке данных", http.StatusInternalServerError)
	}
}




func GetExpressionByID(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	id := chi.URLParam(r, "id")

	// Проверка токена
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "невалидные данные", http.StatusInternalServerError)
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return database.JwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "невалидные данные", http.StatusInternalServerError)
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		http.Error(w, "невалидные данные", http.StatusInternalServerError)
		return
	}

	// Получаем выражение из БД
	expr, err := database.GetExpressionByID(db, id, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("невалидные данные"), http.StatusInternalServerError)
		return
	}
	if expr == nil {
		http.Error(w, "выражение не найдено", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(expr)
}



// Структура для данных регистрации
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Структура для запроса логина
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Функция для генерации JWT токена
func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID, 
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(database.JwtSecret)
}


// Обработчик регистрации пользователя
func RegisterHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "неверный формат данных", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Login) == "" || strings.TrimSpace(req.Password) == "" {
		http.Error(w, "невалидные данные", http.StatusBadRequest)
		return
	}

	_, err := database.RegisterUser(db, req.Login, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func LoginHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "невалидные данные", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Login) == "" || strings.TrimSpace(req.Password) == "" {
		http.Error(w, "невалидные данные", http.StatusBadRequest)
		return
	}

	var userID, hashedPassword string
	err := db.QueryRow("SELECT id, password FROM users WHERE login = ?", req.Login).Scan(&userID, &hashedPassword)
	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)) != nil {
		http.Error(w, "налидные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "ошибка сервера", http.StatusInternalServerError)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(database.JwtSecret)
	if err != nil {
		http.Error(w, "ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenStr,
	})
}

