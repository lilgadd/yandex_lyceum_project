# Распределённый вычислитель арифметических выражений

##### Проект представляет собой распределённую систему вычисления арифметических выражений. 
##### Он состоит из двух компонентов: 
* ##### Оркестратор, который принимает выражения, разбивает их на подзадачи и управляет выполнением
* ##### Агент, который с помощью нескольких воркеров выполняет вычисления. Количество воркеров определяется переменной окружения `COMPUTING_POWER`

 ##### Пользователи могут регистрироваться, входить в систему и отправлять выражения через API. Все данные хранятся в SQLite, а взаимодействие между компонентами происходит через HTTP

##### В проекте используются три базы данных SQLite:

1. ###### user_store.db — хранит данные пользователей:
    ```
    id (уникальный идентификатор)

    login

    password
    ```

2. ###### expression_store.db — хранит выражения, отправленные пользователями:
    ```
    id 

    user_id

    expression

    result

    status
    ```

3. ###### store.db — временное хранилище задач для вычисления:
    ```
    id (идентификатор задачи)

    expression_id (идентификатор родительского выражения)

    subexpression (подвыражение)

    result (результат выполнения подзадачи)
    ```
status (ожидает, выполняется, завершен)
> ###### Базы создаются автоматически при запуске сервера, если отсутствуют.
**Схема эндпоинтов**
![схема эндпоинтов](images\dgrm.png)
## 📡 API-эндпоинты

#### 🔐 Аутентификация

* #### `POST /api/v1/register`
Регистрация нового пользователя.  
**Тело запроса:**
```
{
  "login": "your_username",
  "password": "your_password"
}
```

* #### `POST /api/v1/login`
Вход пользователя в систему, возвращает JWT-токен
**Тело запроса:**
```
{
  "login": "your_username",
  "password": "your_password"
}
```
**Ответ:**
```
{
  "token": "jwt_token"
}
```

#### 🧮 Работа с выражениями (нужен JWT в заголовке Authorization: Bearer <token>)

* #### `POST /api/v1/calculate`
Добавление выражения для вычисления
**Тело запроса:**
```
{
  "expression": "2 + 3 * (4 - 1)"
}
```

* #### `GET /api/v1/expressions`
Выводит все вводимые пользователем выражения
**Пример ответа:**
```
[
  {
    "id": "expression_id_1",
    "expression": "2 + 3 * (4 - 1)",
    "status": "выполняется"
  },
  {
    "id": "expression_id_2",
    "expression": "5 * (6 + 2)",
    "status": "завершен",
    "result": 40
  }
]
```

* #### `GET /api/v1/expressions/{id}`
Выводит конкретное выражение, айди которого указан в запросе
**Пример ответа:**
```
{
  "id": "expression_id",
  "expression": "2 + 3 * (4 - 1)",
  "result": 11,
  "status": "завершен"
}
```

#### ⚙️ Работа агента

* #### `GET /internal/task`
Запрашивает задачу у оркестратора
**Пример ответа:**
```
{
  "task_id": "task_id",
  "expression_id": "expression_id",
  "subexpression": "3 * (4 - 1)"
}
```
* #### `POST /internal/task`
Отправляет результат вычислений оркестратору
**Пример ответа:**
```
{
  "id": "expression_id",
  "result": 9
}
```
___
## 🚀 Запуск проекта

Перед запуском убедитесь, что у вас установлен [Go](https://go.dev/) и переменная окружения `COMPUTING_POWER` задана.
> `COMPUTING_POWER` находится в cmd\agent\main.go в функции: main.go(){}, по дефолту `COMPUTING_POWER` = "2", но можно изменять значение 
### 1. Клонировать репозиторий
```
git clone https://github.com/lilgadd/yandex_lyceum_project.git
```
```
cd yandex_lyceum_project
```
### 2. Запустить оркестратор и агента
>##### ‼️ВАЖНО‼️Сначала запускается оркестратор, потом агент
```
go run cmd/orchestrator/main.go
```
```
go run cmd/agent/main.go
``` 

#### ✅ Теперь сервис доступен по адресу:
```
http://localhost:8080 — оркестратор
http://localhost:8081 — агент
```
___
#### Примеры запросов для проверки

* #### `POST /api/v1/register`
>Придумайте логин и пароль

`Статус 200 OK:`
```
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "exampleuser",
    "password": "examplepassword"
  }'
```
`Для получения ошибки: "пользователь с таким логином уже существует" и статусом: 400 Bad Request повторите запрос с теми же данными`

`Ошибка: "невалидные данные" и статус: 400 Bad Request:`
>Поле login или password(или оба вместе) должны быть пусты
```
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "",
    "password": "examplepassword"
  }'
```
```
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "exampleuser",
    "password": ""
  }'
```
```
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "",
    "password": ""
  }'
```
`Ошибка: "неверный формат данных" и статус 400 Bad Request:`
```
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "exampleuser",
  }'
```
* #### `POST /api/v1/login`
`Статус: 200 OK, введите логин и пароль, что вводили при регистрации`
```
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "login": "exampleuser",
    "password": "examplepassword"
  }'
```
**Ответ:**
```
{
  "token": "jwt_token"
}
```
`Ошибка: "невалидные данные", статус: 400 Bad Request`
```
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "login": "exampleuser",
  }'
```

* #### `POST /api/v1/calculate`
`Статус: 201`
>‼️Не забудьте вставить JWT-токен
```
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE" \
  -d '{
    "expression": "(2 + 3) * 4"
  }'
```
**Ответ:**
```
{
    "id": "expression_id"
}
```
`Ошибка: "выражение пустое", статус: 422 Unprocessable Entity`
```
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE" \
  -d '{
    "expression": ""
  }'
```
`Ошибка: "невалидные данные", статус: 422 Unprocessable Entity`
```
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE" \
  -d '{
    "expression": "abc"
  }'
```
`Ошибка: "отсутствует токен авторизации", статус: 422`
```
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer " \
  -d '{
    "expression": "abc"
  }'
```
* #### `GET /api/v1/expressions`
Выводит все вводимые пользователем выражения
`Статус: 200 OK`
```
curl -X GET http://localhost:8080/api/v1/expressions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE"
```
``Ошибка: "невалидные данные", статус: 500 Internal Server Error`
```
curl -X GET http://localhost:8080/api/v1/expressions \
  -H "Authorization: Bearer "
```
* #### `GET /api/v1/expressions/{id}`
`Статус: 200 OK`
>‼️Не забудьте указать айди выражения
```
curl -X GET http://localhost:8080/api/v1/expressions/EXPRESSION_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE"
```
`Ошибка: "невалидные данные", статус: 500 Internal Server Error`
```
curl -X GET http://localhost:8080/api/v1/expressions/EXPRESSION_ID \
  -H "Authorization: Bearer "
```
`Ошибка: "выражение не найдено", статус: 404 Not Found`
>Уже готовый запрос, ничего изменять не нужно
```
curl -X GET http://localhost:8080/api/v1/expressions/0 \
  -H "Authorization: Bearer "
```