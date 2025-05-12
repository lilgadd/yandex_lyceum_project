package database

import (
	"database/sql"
	"errors"
	"calc/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
)

var JwtSecret = []byte("supersecretkey") // Ключ можно менять

// Инициализация базы данных
func InitDB() (*sql.DB, *sql.DB, error) {
	// Инициализация базы данных для пользователей
	userDB, err := sql.Open("sqlite3", "./user_store.db")
	if err != nil {
		return nil, nil, err
	}

	// Создаём таблицу пользователей, если она ещё не существует
	createUsersTable := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			login TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL
		);
	`
	_, err = userDB.Exec(createUsersTable)
	if err != nil {
		return nil, nil, err
	}

	// Инициализация базы данных для выражений
	expressionDB, err := sql.Open("sqlite3", "./expression_store.db")
	if err != nil {
		return nil, nil, err
	}

	createExpressionsTable := `CREATE TABLE IF NOT EXISTS expressions (
	id TEXT PRIMARY KEY,
	user_id TEXT,
	expression TEXT NOT NULL,
	status TEXT NOT NULL,
	result FLOAT,
	FOREIGN KEY (user_id) REFERENCES users(id)
	);`
	_, err = expressionDB.Exec(createExpressionsTable)
	if err != nil {
		return nil, nil, err
	}

	return userDB, expressionDB, nil
}

func RegisterUser(db *sql.DB, login, password string) (string, error) {
	// Проверка существования
	var existingID string
	err := db.QueryRow("SELECT id FROM users WHERE login = ?", login).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if existingID != "" {
		return "", errors.New("пользователь с таким логином уже существует")
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Генерируем UUID
	userID := uuid.New().String()

	// Вставка
	_, err = db.Exec("INSERT INTO users (id, login, password) VALUES (?, ?, ?)", userID, login, hashedPassword)
	if err != nil {
		return "", err
	}

	return userID, nil
}


func SaveExpressionForUser(dbConn *sql.DB, userID string, id, expression string) error {
	// SQL запрос для сохранения
	insertStmt := `INSERT INTO expressions (user_id, id, expression, status) VALUES (?, ?, ?, ?)`
	_, err := dbConn.Exec(insertStmt, userID, id, expression, "ожидает выполнения")
	return err
}


func GetExpressionsByUser(db *sql.DB, userID string) ([]models.Expression, error) {
	rows, err := db.Query(`SELECT id, status, result FROM expressions WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expressions []models.Expression
	for rows.Next() {
		var expr models.Expression
		err := rows.Scan(&expr.Id, &expr.Status, &expr.Result)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}
	return expressions, nil
}

func GetExpressionByID(db *sql.DB, id string, userID string) (*models.Expression, error) {
	query := `SELECT id, status, result FROM expressions WHERE id = ? AND user_id = ?`
	row := db.QueryRow(query, id, userID)

	var expr models.Expression
	err := row.Scan(&expr.Id, &expr.Status, &expr.Result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // не найдено
		}
		return nil, err
	}
	return &expr, nil
}

