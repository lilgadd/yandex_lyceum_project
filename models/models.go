package models

import(
	// "database/sql"
)

var(
	//Время указано в миллисекундах
	Tadd = 5
	Tsub = 5 
	Tmul = 10
	Tdiv = 10
)

type ExpressionInput struct {
    Expression string `json:"expression"`
}

type Expression struct{
	Id string		`json:"id"`
    Status string	`json:"status"` 
    Result float64	`json:"result"`
}

type Responce1 struct{
	Id string	`json:"id"`
}

type ASTNode struct {
	Value         float64
	Operator      string
	IsLeaf        bool
	Left, Right   *ASTNode
	TaskID        string
	TaskScheduled bool
}

type Task struct {
	Id                string	`json:"id"`
	Arg1              float64 	`json:"arg1"`
	Arg2              float64  	`json:"arg2"`
	Operation         string   	`json:"operation"`
	Result float64				`json:"result"`
	Operation_time_ms float64	`json:"operation_time"`
	Dependencies      []string	`json:"dependence"`
	Status bool					`json:"status"`
	ExpressionID    string   	`json:"expression_id"`
	IsFinal bool 				`json:"is_final"`
}

type Responce2 struct{
	Id string		`json:"id"`
  	Result float64	`json:"result"`
}

type User struct {
	ID       int64  `json:"id"`       // Идентификатор пользователя
	Login    string `json:"login"`    // Логин пользователя
	Password string `json:"password"` // Хэшированный пароль
}
