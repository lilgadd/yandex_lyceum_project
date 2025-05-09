package models

var(
	//Время указано в миллисекундах
	tadd = 5
	tsub = 5 
	tmul = 10
	tdiv = 10
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

type Task struct{
	Id string 					`json:"id"`
    Arg1 float64				`json:"arg1"`
    Arg2 float64				`json:"arg2"`
    Operation string			`json:"operation"`
    Operation_time_ms float64	`json:"operation_time"`
}

type Responce2 struct{
	Id string		`json:"id"`
  	Result float64	`json:"result"`
}

