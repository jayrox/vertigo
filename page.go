package main

import "github.com/martini-contrib/sessions"

type Page struct {
	Session sessions.Session
	Data    interface{}
	Err     string
}
