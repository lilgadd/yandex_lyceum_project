package orchestrator

import (
	"calc/models"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"github.com/google/uuid"
	"unicode"
	"regexp"
	"fmt"
	"database/sql"
	"calc/database"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"strconv"
)

var(
	Expressions = make(map[string]*models.Expression)
	mutex = &sync.Mutex{}

	taskID    int
	Tasks     = make(map[string]*models.Task)
	TaskQueue []*models.Task
	TaskMutex sync.Mutex

	TaskReady = make(chan bool, 1)
	ComputingPowerChannel = make(chan int)

)

// Хендлер для вычислений
func CalculateHandler(w http.ResponseWriter, r *http.Request, dbConn *sql.DB) {
	var input models.ExpressionInput

	// Декодируем JSON
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "что-то пошло не так", http.StatusInternalServerError)
		return
	}

	// Убираем пробелы
	cleaned := strings.ReplaceAll(input.Expression, " ", "")
	if cleaned == "" {
		http.Error(w, "выражение пустое", http.StatusUnprocessableEntity)
		return
	}

	// Проверяем корректность выражения
	if !isValidExpression(input.Expression) {
		http.Error(w, "некорректные данные", http.StatusUnprocessableEntity)
		return
	}

	// Достаём user_id из токена
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Отсутствует токен авторизации", http.StatusUnauthorized)
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return database.JwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Недействительный токен", http.StatusUnauthorized)
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		http.Error(w, "user_id отсутствует или некорректен в токене", http.StatusUnauthorized)
		return
	}

	// Генерим ID для выражения
	id := uuid.New().String()

	// Добавляем в БД
	err = database.SaveExpressionForUser(dbConn, userID, id, cleaned)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка сохранения выражения: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ с ID
	resp := models.Responce1{Id: id}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при отправке ответа: %v", err), http.StatusInternalServerError)
		return
	}

	// Параллельно обрабатываем выражение
	go ProcessExpression(dbConn, id, cleaned)
}





func ProcessExpression(db *sql.DB, id string, input string) {
	// Обновляем статус в БД
	if err := UpdateExpressionStatus(db, id, "выполняется"); err != nil {
		log.Printf("Ошибка обновления статуса выражения: %v", err)
	}

	rpn := convertToRPN(input)
	tree := createExpressionTree(rpn)

	if createTasksForTree(tree, id) {
	}
}


func UpdateExpressionStatus(db *sql.DB, id string, status string) error {
	_, err := db.Exec(`UPDATE expressions SET status = ? WHERE id = ?`, status, id)
	return err
}

func isValidExpression(expression string) bool {
	openBrackets := 0
	previousChar := rune(0)

	// Проверяем, что выражение не пустое
	if len(expression) == 0 {
		return false
	}

	for i, char := range expression {
		// Пропускаем пробелы
		if unicode.IsSpace(char) {
			continue
		}

		// Проверка на допустимые символы
		if !strings.ContainsRune("0123456789+-*/().", char) {
			return false
		}

		// Проверка на баланс скобок
		if char == '(' {
			openBrackets++

			// Следующий символ не может быть оператором (кроме унарного минуса)
			if i+1 < len(expression) {
				next := rune(expression[i+1])
				if strings.ContainsRune("*/+)", next) {
					return false
				}
			}

			// Предыдущий символ не может быть цифрой или ')'
			if previousChar != 0 && (unicode.IsDigit(previousChar) || previousChar == ')') {
				return false
			}
		} else if char == ')' {
			openBrackets--

			// Скобки не сбалансированы
			if openBrackets < 0 {
				return false
			}

			// Предыдущий символ не может быть оператором или '('
			if strings.ContainsRune("+-*/(", previousChar) {
				return false
			}
		}

		// Два подряд оператора (исключая унарный минус)
		if strings.ContainsRune("+-*/", char) && strings.ContainsRune("+-*/", previousChar) {
			// Разрешаем унарный минус после '('
			if !(char == '-' && previousChar == '(') {
				return false
			}
		}

		// Проверка начала выражения
		if i == 0 && (char == '*' || char == '/' || char == '+') {
			return false
		}

		// Проверка конца выражения
		if i == len(expression)-1 && strings.ContainsRune("+-*/(", char) {
			return false
		}

		previousChar = char
	}

	// Проверка, что нет пустого выражения или только операторов
	if previousChar == '+' || previousChar == '-' || previousChar == '*' || previousChar == '/' {
		return false
	}

	// Проверка баланса скобок
	return openBrackets == 0
}


func convertToRPN(expression string) []string {
	var output []string
	var stack []string

	// Регулярное выражение: числа, операторы, скобки
	re := regexp.MustCompile(`(\d+\.?\d*|\+|\-|\*|\/|\(|\))`)
	tokens := re.FindAllString(expression, -1)

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token {
		case "+", "-":
			// Обработка унарного минуса: если "-" в начале или после "(", оператора
			if token == "-" && (i == 0 || tokens[i-1] == "(" || isOperator(tokens[i-1])) {
				// Объединяем "-" и следующее число
				if i+1 < len(tokens) && isNumber(tokens[i+1]) {
					tokens[i+1] = "-" + tokens[i+1]
					continue // пропускаем текущий унарный минус
				}
			}

			// Обычные операторы + и -
			for len(stack) > 0 && precedence(stack[len(stack)-1]) >= precedence(token) {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)

		case "*", "/":
			for len(stack) > 0 && precedence(stack[len(stack)-1]) >= precedence(token) {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)

		case "(":
			stack = append(stack, token)

		case ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 && stack[len(stack)-1] == "(" {
				stack = stack[:len(stack)-1]
			}

		default:
			// Число
			output = append(output, token)
		}
	}

	// Оставшиеся операторы в стек
	for len(stack) > 0 {
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output
}

// Дополнительные функции
func precedence(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}

func isOperator(s string) bool {
	return s == "+" || s == "-" || s == "*" || s == "/"
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}




func createExpressionTree(rpn []string) *models.ASTNode {
	var stack []*models.ASTNode

	for _, token := range rpn {
		switch token {
		case "+", "-", "*", "/":
			if len(stack) < 2 {
				panic("invalid expression: not enough operands")
			}

			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			node := &models.ASTNode{
				Operator: token,
				Left:     left,
				Right:    right,
			}

			// (необязательная проверка/отладка)
			_ = getOperatorPriority(token)

			stack = append(stack, node)

		case "u-":
			if len(stack) < 1 {
				panic("invalid expression: not enough operands for unary minus")
			}

			operand := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			node := &models.ASTNode{
				Operator: "u-",
				Left:     operand,
			}

			_ = getOperatorPriority("u-")

			stack = append(stack, node)

		default:
			var value float64
			_, err := fmt.Sscanf(token, "%f", &value)
			if err != nil {
				panic(fmt.Sprintf("invalid number token: %s", token))
			}
			node := &models.ASTNode{
				Value:  value,
				IsLeaf: true,
			}
			stack = append(stack, node)
		}
	}

	if len(stack) != 1 {
		panic("invalid RPN expression: remaining elements in stack")
	}

	return stack[0]
}



// Функция для получения приоритета операции
func getOperatorPriority(operator string) int {
    switch operator {
    case "u-":
        return 3 // самый высокий приоритет
    case "*", "/":
        return 2
    case "+", "-":
        return 1
    default:
        return 0
    }
}


// Функция для рекурсивного обхода дерева и создания задач
func createTasksForTree(node *models.ASTNode, id string) bool {
	var finalTaskID string

	var traverse func(n *models.ASTNode)
	traverse = func(n *models.ASTNode) {
		if n == nil {
			return
		}

		traverse(n.Left)
		traverse(n.Right)

		// === Обработка бинарных операторов ===
		if n.Left != nil && n.Right != nil {
			leftReady := n.Left.IsLeaf || n.Left.TaskScheduled
			rightReady := n.Right.IsLeaf || n.Right.TaskScheduled

			if leftReady && rightReady && !n.TaskScheduled {
				TaskMutex.Lock()
				taskID++
				taskIDStr := fmt.Sprintf("%d", taskID)

				task := &models.Task{
					Id:                taskIDStr,
					Arg1:              n.Left.Value,
					Arg2:              n.Right.Value,
					Operation:         n.Operator,
					Operation_time_ms: float64(getOperationTime(n.Operator)),
					ExpressionID:      id,
					IsFinal:           false,
				}

				if !n.Left.IsLeaf {
					task.Dependencies = append(task.Dependencies, n.Left.TaskID)
				}
				if !n.Right.IsLeaf {
					task.Dependencies = append(task.Dependencies, n.Right.TaskID)
				}

				priority := getOperatorPriority(n.Operator)
				log.Printf("Приоритет оператора %s: %d", n.Operator, priority)

				Tasks[taskIDStr] = task
				TaskQueue = append(TaskQueue, task)

				n.Value = 0
				n.IsLeaf = false
				n.TaskID = taskIDStr
				n.TaskScheduled = true

				if n == node {
					finalTaskID = taskIDStr
				}
				TaskMutex.Unlock()
			}
		}

		// === Обработка унарного минуса ===
		if n.Operator == "u-" && n.Left != nil && n.Right == nil && !n.TaskScheduled {
			leftReady := n.Left.IsLeaf || n.Left.TaskScheduled

			if leftReady {
				TaskMutex.Lock()
				taskID++
				taskIDStr := fmt.Sprintf("%d", taskID)

				task := &models.Task{
					Id:                taskIDStr,
					Arg1:              n.Left.Value,
					Operation:         "u-",
					Operation_time_ms: float64(getOperationTime("u-")),
					ExpressionID:      id,
					IsFinal:           false,
				}

				if !n.Left.IsLeaf {
					task.Dependencies = append(task.Dependencies, n.Left.TaskID)
				}

				priority := getOperatorPriority("u-")
				log.Printf("Приоритет унарного оператора %s: %d", n.Operator, priority)

				Tasks[taskIDStr] = task
				TaskQueue = append(TaskQueue, task)

				n.Value = 0
				n.IsLeaf = false
				n.TaskID = taskIDStr
				n.TaskScheduled = true

				if n == node {
					finalTaskID = taskIDStr
				}
				TaskMutex.Unlock()
			}
		}
	}

	traverse(node)

	if finalTaskID != "" {
		TaskMutex.Lock()
		if finalTask, ok := Tasks[finalTaskID]; ok {
			finalTask.IsFinal = true
		}
		TaskMutex.Unlock()
	}

	return true
}


func getOperationTime(operator string) int {
    switch operator {
    case "+":
        return models.Tadd
    case "-":
        return models.Tsub
    case "*":
        return models.Tmul
    case "/":
        return models.Tdiv
    default:
        return 0
    }
}
