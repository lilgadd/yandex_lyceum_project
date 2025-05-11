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

func CalculateHandler(w http.ResponseWriter, r *http.Request) {
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

	// Генерим ID
	id := uuid.New().String()

	// Блокировка при работе с мапой
	mutex.Lock()
	defer mutex.Unlock()

	expr := &models.Expression{
		Id:     id,
		Status: "ожидает выполнения",
		Result: 0,
	}

	// Сохраняем выражение в хранилище
	Expressions[id] = expr

	// Отправляем ответ с ID
	resp := models.Responce1{Id: id}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
    http.Error(w, fmt.Sprintf("Ошибка при отправке ответа: %v", err), http.StatusInternalServerError)
    return
}


	// Отправляем на обработку
	go ProcessExpression(id, input.Expression)
}


func ProcessExpression(id string, input string) {
	mutex.Lock()
	Expressions[id].Status = "выполняется"
	mutex.Unlock()

	rpn := convertToRPN(input)
	tree := createExpressionTree(rpn)

	// Функция вернёт true, когда все задачи будут добавлены в очередь
	if createTasksForTree(tree, id) {
	
		
	}


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

	// Новая регулярка: распознаёт отрицательные числа
	re := regexp.MustCompile(`(-?\d+\.?\d*|\+|\-|\*|\/|\(|\))`)
	tokens := re.FindAllString(expression, -1)

	for i, token := range tokens {
		switch token {
		case "+", "-":
			// Проверка на унарный минус (если токен "-" и перед ним ничего или "(" или оператор)
			if token == "-" && (i == 0 || tokens[i-1] == "(" || tokens[i-1] == "+" || tokens[i-1] == "-" || tokens[i-1] == "*" || tokens[i-1] == "/") {
				// Объединяем с числом справа
				if i+1 < len(tokens) {
					tokens[i+1] = "-" + tokens[i+1]
					continue // пропускаем унарный минус, число станет отрицательным
				}
			}
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
			stack = stack[:len(stack)-1] // удаляем "("
		default:
			output = append(output, token)
		}
	}

	for len(stack) > 0 {
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output
}

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


func createExpressionTree(rpn []string) *models.ASTNode {
	var stack []*models.ASTNode

	for _, token := range rpn {
		if token == "+" || token == "-" || token == "*" || token == "/" {
			right := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			left := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			node := &models.ASTNode{
				Operator: token,
				Left:     left,
				Right:    right,
			}
			stack = append(stack, node)
		} else {
			var value float64
			fmt.Sscanf(token, "%f", &value)
			node := &models.ASTNode{
				Value:  value,
				IsLeaf: true,
			}
			stack = append(stack, node)
		}
	}

	return stack[0]
}

func createTasksForTree(node *models.ASTNode, id string) bool {
	var finalTaskID string

	var traverse func(n *models.ASTNode)
	traverse = func(n *models.ASTNode) {
		if n == nil {
			return
		}

		traverse(n.Left)
		traverse(n.Right)

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
					IsFinal:           false, // по умолчанию false
				}

				if !n.Left.IsLeaf {
					task.Dependencies = append(task.Dependencies, n.Left.TaskID)
				}
				if !n.Right.IsLeaf {
					task.Dependencies = append(task.Dependencies, n.Right.TaskID)
				}

				Tasks[taskIDStr] = task
				TaskQueue = append(TaskQueue, task)

				n.Value = 0
				n.IsLeaf = false
				n.TaskID = taskIDStr
				n.TaskScheduled = true

				// Сохраняем ID задачи у корня
				if n == node {
					finalTaskID = taskIDStr
				}
				TaskMutex.Unlock()
			}
		}
	}

	traverse(node)

	// Устанавливаем флаг IsFinal = true у корневой задачи
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
