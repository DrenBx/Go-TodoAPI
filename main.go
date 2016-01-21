package main

import (
    "os"
    "time"
    "errors"
    "net/http"
    "strings"
    "math/rand"
    "encoding/json"
    "github.com/julienschmidt/httprouter"
    "database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// Task ...
type Task struct {
    ID          int     `json:"id"`
    Content     string  `json:"content"`
    Completed   bool    `json:"completed"`
    CreditCard  string  `json:"creditcard"`
}

var db *sql.DB
const signinChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."

func constructTask(content, completed, creditcard string) (*Task, error) {
    
    b := false
    if len(content) < 5 {
        return nil, errors.New("task: Content size must have greater 5 characters")
    }
    if strings.Contains(completed, "1") || strings.Contains(completed, "true") {
        b = true
    }
    return &Task{Content:content, Completed:b, CreditCard:creditcard}, nil
}

func getAccess(w http.ResponseWriter, r, token string) bool {
    count := db.QueryRow("SELECT COUNT(*) as count FROM perms INNER JOIN users ON users.id = perms.user_id WHERE perms.name=? AND users.token=?", r, token)
    if checkCount(count) > 0 {
        return true
    }
    sendJSONResponse(w, 401, "error: You don't have permision!")
    return false
}

func checkCount(row *sql.Row) (count int) {
    err:= row.Scan(&count)
    checkErr(err)
    return count
}

func generateRandomString(n int) string {

    b := make([]byte, n)
    l := len(signinChars)
    rand.Seed(time.Now().UnixNano())
    
    for i := range b {
        b[i] = signinChars[rand.Intn(l)]
    }
    
    return string(b)
}

func userConnect(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    
    var id int
    row := db.QueryRow("SELECT id FROM users WHERE name=?", r.PostFormValue("name"))
    if err := row.Scan(&id); err == nil {
        token := generateRandomString(128)
        prepareAndExec("UPDATE users set token=? WHERE id=?", token, id)
        sendJSONResponse(w, 201, token)
    } else {
        sendJSONResponse(w, 401, "error: You don't have permision!")   
    }
}

func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
    d, err := json.Marshal(data)
    if err != nil {
        return
    }
    w.WriteHeader(status)
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Write(d)
}

func getTasks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

    if ret := getAccess(w, "show", r.FormValue("token")); ret == false {
        return
    }
    
    query := "SELECT * FROM tasks"
    if r.FormValue("completed") == "1" || r.FormValue("completed") == "true" {
        query += " WHERE completed=1"
    } else if r.FormValue("completed") != "" {
        query += " WHERE completed=0"
    }
    if r.FormValue("size") != "" {
        query += " LIMIT " + r.FormValue("size")
    } else {
        query += " LIMIT 20"
    }
    rows, _ := db.Query(query)
    defer rows.Close()

    tasks := []Task{}
	for rows.Next() {
		t := Task{}
		err := rows.Scan(&t.ID, &t.Content, &t.Completed, &t.CreditCard)
        checkErr(err)
        if err == nil {
            tasks = append(tasks, t)            
        }
	}
    if len(tasks) <= 0 {
        sendJSONResponse(w, 400, "No Task!")
    } else {
        sendJSONResponse(w, 200, tasks)   
    }
}

func showTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
    
    if ret := getAccess(w, "show", r.FormValue("token")); ret == false {
        return
    }
    
    println(params.ByName("id"))
    row := db.QueryRow("SELECT * FROM tasks WHERE id=?", params.ByName("id"))

    var t Task
    if err := row.Scan(&t.ID, &t.Content, &t.Completed, &t.CreditCard); err != nil {
        sendJSONResponse(w, 404, "Task not found!")
        return
    }
    sendJSONResponse(w, 200, t)
}

func createTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

    if ret := getAccess(w, "create", r.PostFormValue("token")); ret == false {
        return
    }

    t, err := constructTask(r.PostFormValue("content"), r.PostFormValue("completed"), r.PostFormValue("creditcard"))
    if err != nil {
        sendJSONResponse(w, 304, err.Error())
        return
    }
    res, err := prepareAndExec("INSERT INTO tasks(content, creditcard, completed) values(?,?,?)", t.Content, t.CreditCard, t.Completed)

    aff, err := res.RowsAffected()
    affectedResponse(w, aff, err, "created!")
}

func updateTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

    if ret := getAccess(w, "edit", r.PostFormValue("token")); ret == false {
        return
    }

    t, err := constructTask(r.PostFormValue("content"), r.PostFormValue("completed"), r.PostFormValue("creditcard"))
    if err != nil {
        sendJSONResponse(w, 304, err.Error())
        return
    }
    res, _ := prepareAndExec("UPDATE tasks set content=?, creditcard=?, completed=? WHERE id=?", t.Content, t.CreditCard, t.Completed, params.ByName("id"))

    aff, err := res.RowsAffected()
    affectedResponse(w, aff, err, "updated!")
}

func deleteTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

    if ret := getAccess(w, "delete", r.PostFormValue("token")); ret == false {
        return
    }

    res, _ := prepareAndExec("DELETE FROM tasks WHERE id=?", params.ByName("id"))

    aff, err := res.RowsAffected()
    affectedResponse(w, aff, err, "deleted!")
}

func affectedResponse(w http.ResponseWriter, affected int64, err error, action string) {
    if affected > 0 || err != nil {
        sendJSONResponse(w, 304, errors.New("Task wasn't "+action))
    } else {
        sendJSONResponse(w, 201, "Task was "+action)
    }
}

func prepareAndExec(q string, params ...interface{}) (sql.Result, error) {
    query, _ := db.Prepare(q)
    res, err := query.Exec(params...)

    return res, err
}

func createRouteAndListen() {
    router := httprouter.New()
    router.POST("/connect", userConnect)
    router.GET("/tasks", getTasks)
    router.POST("/tasks", createTask)
    router.GET("/tasks/:id", showTask)
    router.PUT("/tasks/:id", updateTask)
    router.DELETE("/tasks/:id", deleteTask)

	http.ListenAndServe(":8000", router)
}

func initDB() {
    db.Exec("CREATE TABLE IF NOT EXISTS `tasks` ( `id` INTEGER PRIMARY KEY AUTOINCREMENT, `content` TEXT NOT NULL, `completed` BOOLEAN DEFAULT 0, `creditcard` VARCHAR(64) NOT NULL);")
    db.Exec("CREATE TABLE IF NOT EXISTS `users` ( `id` INTEGER PRIMARY KEY AUTOINCREMENT, `name` VARCHAR(64) NOT NULL, `token` TEXT);")
    db.Exec("CREATE TABLE IF NOT EXISTS `perms` ( `id` INTEGER PRIMARY KEY AUTOINCREMENT, `user_id` INTEGER NOT NULL, `name` VARCHAR(64) NOT NULL);")
    
    db.Exec("INSERT INTO tasks(content) values(?);", "This is a task")
    db.Exec("INSERT INTO tasks(content, completed) values(?,?);", "This is a task completed", true)
    
    db.Exec("INSERT INTO users(name) values(?);", "A")
    db.Exec("INSERT INTO users(name) values(?);", "B")
    
    db.Exec("INSERT INTO perms(name, user_id) values(?,?);", "show", 1)
    db.Exec("INSERT INTO perms(name, user_id) values(?,?);", "show", 2)
    db.Exec("INSERT INTO perms(name, user_id) values(?,?);", "create", 2)
    db.Exec("INSERT INTO perms(name, user_id) values(?,?);", "edit", 2)
    db.Exec("INSERT INTO perms(name, user_id) values(?,?);", "delete", 2)
}

func main() {

    d, err := sql.Open("sqlite3", "./task.db")
    defer d.Close()
    
    if err == nil {
        db = d
        if len(os.Args) > 1 && os.Args[1] == "db:init" {
            initDB()
        }
        createRouteAndListen()
    }
    checkErr(err)
}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}