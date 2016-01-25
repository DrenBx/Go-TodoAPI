package main

import (
    "net/http"
    "encoding/json"
    "github.com/julienschmidt/httprouter"
)

func init() {
    router := httprouter.New()
    router.GET("/tasks", getTasks)
    router.POST("/tasks", createTask)
    router.GET("/tasks/:id", showTask)
    router.PUT("/tasks/:id", updateTask)
    router.DELETE("/tasks/:id", deleteTask)

    http.Handle("/", router)
}

func checkErr(err error) {
    if err != nil {
        panic(err)
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

func atob(str string) bool {
    if str == "true" || str == "1" {
        return true
    }
    return false
}