package handler

import (
	"html/template"
	"net/http"
)

func ServeUI(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("internal/templates/index.html"))
	tmpl.Execute(w, nil)
}
