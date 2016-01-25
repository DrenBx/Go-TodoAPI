package main

import (
    "net/http"
    "io"
    "io/ioutil"
    "errors"
    "strconv"
    "encoding/json"
    "github.com/julienschmidt/httprouter"
    "google.golang.org/appengine"
    "google.golang.org/appengine/datastore"
)

//Task ...
type Task struct {
    ID          int64   `datastore:"id" json:"id"`
    Content     string  `datastore:"content" json:"content"`
    Creditcard  string  `datastore:"creditcard" json:"creditcard"`
    Completed   bool    `datastore:"completed" json:"completed"`
}

func maketask(id int64, body io.Reader) (*Task, error) {
    var task Task
    b, err := ioutil.ReadAll(io.LimitReader(body, 4096))
    if err != nil {
        return nil, errors.New("An error was occured")
    }
    if err := json.Unmarshal(b, &task); err != nil {
        return nil, errors.New("An error was occured")
    }
    task.ID = id
    return &task, nil
}

//--------------------------------------------------------------------------

func getTasks(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
    ctx := appengine.NewContext(r)
    tasks := []Task{}
    q := datastore.NewQuery("tasks")

    if r.FormValue("completed") != "" {
        q.Filter("completed =", atob(r.FormValue("completed")))
    }
    if r.FormValue("size") != "" {
        s, _ := strconv.Atoi(r.FormValue("size"))
        q.Limit(s)
    } else {
        q.Limit(20)
    }
    _, err := q.GetAll(ctx, &tasks)
    if err != nil {
        sendJSONResponse(w, http.StatusInternalServerError, nil)
        return
    }
    sendJSONResponse(w, http.StatusAccepted, tasks)
}

func showTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
    ctx := appengine.NewContext(r)
    tasks := []Task{}
    id, _ := strconv.Atoi(params.ByName("id"))
    _, err := datastore.NewQuery("tasks").Filter("id =", id).GetAll(ctx, &tasks)
    if err != nil || len(tasks) < 1{
        sendJSONResponse(w, http.StatusNotFound, nil)
        return
    }
    sendJSONResponse(w, http.StatusAccepted, tasks[0])
}

func createTask(w http.ResponseWriter, r *http.Request, _ httprouter.Params)  {
    ctx := appengine.NewContext(r)
    low, _, err := datastore.AllocateIDs(ctx, "tasks", nil, 1)
    if err != nil {
        sendJSONResponse(w, http.StatusNotModified, nil)
        return
    }
    t, err := maketask(low, r.Body)
    if err != nil {
        sendJSONResponse(w, http.StatusNotModified, nil)
        return
    }
    if _, e := datastore.Put(ctx, datastore.NewKey(ctx, "tasks", "", low, nil), t); e != nil {
        sendJSONResponse(w, http.StatusNotModified, nil)
        return
    }
    sendJSONResponse(w, http.StatusCreated, nil)
}

func updateTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
    ctx := appengine.NewContext(r)
    id, _ := strconv.ParseInt(params.ByName("id"), 10, 64)
    t, err := maketask(id, r.Body)
    if err != nil {
        sendJSONResponse(w, http.StatusNotModified, nil)
        return
    }
    if _, e := datastore.Put(ctx, datastore.NewKey(ctx, "tasks", "", id, nil), t); e != nil {
        sendJSONResponse(w, http.StatusNotModified, nil)
        return
    }
}

func deleteTask(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
    c := appengine.NewContext(r)
    id, _ := strconv.ParseInt(params.ByName("id"), 10, 64)
	key := datastore.NewKey(c, "tasks", "", id, nil)
	if err := datastore.Delete(c, key); err != nil {
        sendJSONResponse(w, http.StatusNotModified, nil)
    } else {
        sendJSONResponse(w, http.StatusAccepted, nil)
    }
}