// Main.go contains settings related to the web server, such as
// template helper functions, HTTP routes and Martini settings.
package main

import (
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/jinzhu/gorm"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"github.com/martini-contrib/strict"
	"github.com/pkg/browser"
)

// NewServer spaws a new Vertigo server
func NewServer() *martini.ClassicMartini {

	helpers := template.FuncMap{
		// Unescape unescapes and parses HTML from database objects.
		// Used in templates such as "/post/display.tmpl"
		"unescape": func(s string) template.HTML {
			return template.HTML(html.UnescapeString(s))
		},
		// Title renders post name as a page title.
		// Otherwise it defaults to Vertigo.
		"title": func(t interface{}) string {
			post, exists := t.(Post)
			if exists {
				return post.Title
			}
			return Settings.Name
		},
		// Page Title renders page title.
		"pagetitle": func(t interface{}) string {
			if Settings.Name == "" {
				return "Vertigo"
			}
			return Settings.Name
		},
		// Description renders page description.
		"description": func(t interface{}) string {
			if Settings.Description == "" {
				return "Blog in Go"
			}
			return Settings.Description
		},
		// Hostname renders page hostname.
		"hostname": func(t interface{}) string {
			return urlHost()
		},
		// Date helper returns unix date as more readable one in string format. Format of YYYY-MM-DD
		// https://html.spec.whatwg.org/multipage/semantics.html#datetime-value
		"date": func(d int64) time.Time {
			return time.Unix(d, 0)
		},
		"datef": func(t time.Time, f string) string {
			if f == "" {
				f = "2006-01-02"
			}
			return t.Format(f)
		},
		"loggedin": func(session sessions.Session) bool {
			return sessionIsAlive(session)
		},
		"allowregistrations": func(t interface{}) bool {
			return Settings.AllowRegistrations
		},
		"ismarkdown": func(s string) bool {
			if len(s) > 0 {
				return true
			}
			return false
		},
		// Env helper returns environment variable of s.
		"env": func(s string) string {
			if s == "MAILGUN_SMTP_LOGIN" {
				return strings.TrimLeft(os.Getenv(s), "postmaster@")
			}
			return os.Getenv(s)
		},
		// Markdown returns whether user has Markdown enabled from settings.
		"Markdown": func() bool {
			if Settings.Markdown {
				return true
			}
			return false
		},
		// ReadOnly checks whether a post is safe to edit with current settings.
		"ReadOnly": func(p Post) bool {
			if Settings.Markdown && p.Markdown == "" {
				return true
			}
			return false
		},
	}

	m := martini.Classic()
	store := sessions.NewCookieStore([]byte(Settings.CookieHash))
	m.Use(sessions.Sessions("user", store))
	m.Use(middleware())
	m.Use(sessionchecker())
	m.Use(strict.Strict)
	m.Use(martini.Static("public", martini.StaticOptions{
		SkipLogging: true,
		Expires: func() string {
			return "Cache-Control: max-age=31536000"
		},
	}))
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
		Funcs:  []template.FuncMap{helpers}, // Specify helper function maps for templates to access.
	}))

	m.Get("/", Homepage)

	m.Group("/feeds", func(r martini.Router) {
		r.Get("", func(res render.Render) {
			res.Redirect("/feeds/rss", 302)
		})
		r.Get("/atom", ReadFeed)
		r.Get("/rss", ReadFeed)
	})

	m.Group("/post", func(r martini.Router) {

		// Please note that `/new` route has to be before the `/:slug` route. Otherwise the program will try
		// to fetch for Post named "new".
		// For now I'll keep it this way to streamline route naming.
		r.Get("/new", ProtectedPage, func(res render.Render, s sessions.Session) {
			var post Post
			res.HTML(200, "post/new", Page{Session: s, Data: post})
		})
		r.Get("/:slug", ReadPost)

		r.Get("/:slug/edit", ProtectedPage, EditPost)
		r.Post("/:slug/edit", ProtectedPage, strict.ContentType("application/x-www-form-urlencoded"), binding.Form(Post{}), binding.ErrorHandler, UpdatePost)
		r.Get("/:slug/delete", ProtectedPage, DeletePost)
		r.Get("/:slug/publish", ProtectedPage, PublishPost)
		r.Get("/:slug/unpublish", ProtectedPage, UnpublishPost)
		r.Post("/new", ProtectedPage, strict.ContentType("application/x-www-form-urlencoded"), binding.Form(Post{}), binding.ErrorHandler, CreatePost)
		r.Post("/search", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(Search{}), binding.ErrorHandler, SearchPost)
	})

	m.Group("/user", func(r martini.Router) {

		r.Get("", ProtectedPage, ReadUser)
		//r.Post("/delete", strict.ContentType("application/x-www-form-urlencoded"), ProtectedPage, binding.Form(User{}), DeleteUser)

		r.Get("/blogsettings", ProtectedPage, ReadBlogSettings)
		r.Post("/blogsettings", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(Vertigo{}), binding.ErrorHandler, ProtectedPage, UpdateBlogSettings)

		r.Post("/installation", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(Vertigo{}), binding.ErrorHandler, UpdateSettings)

		r.Get("/register", SessionRedirect, func(res render.Render, s sessions.Session) {
			res.HTML(200, "user/register", Page{Session: s, Data: nil})
		})
		r.Post("/register", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(User{}), binding.ErrorHandler, CreateUser)

		r.Get("/settings", ProtectedPage, ReadSettings)
		r.Post("/settings", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(User{}), binding.ErrorHandler, ProtectedPage, UpdateSettings)

		r.Get("/recover", SessionRedirect, func(res render.Render, s sessions.Session) {
			res.HTML(200, "user/recover", Page{Session: s, Data: nil})
		})
		r.Post("/recover", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(User{}), RecoverUser)
		r.Get("/reset/:id/:recovery", SessionRedirect, func(res render.Render, s sessions.Session) {
			res.HTML(200, "user/reset", Page{Session: s, Data: nil})
		})
		r.Post("/reset/:id/:recovery", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(User{}), ResetUserPassword)

		r.Get("/login", SessionRedirect, func(res render.Render, s sessions.Session) {
			res.HTML(200, "user/login", Page{Session: s, Data: nil})
		})
		r.Post("/login", strict.ContentType("application/x-www-form-urlencoded"), binding.Form(User{}), LoginUser)
		r.Get("/logout", LogoutUser)

	})

	m.Group("/api", func(r martini.Router) {

		r.Get("", func(res render.Render) {
			res.HTML(200, "api/index", nil)
		})
		r.Get("/settings", ProtectedPage, ReadSettings)
		r.Post("/settings", strict.ContentType("application/json"), binding.Json(Vertigo{}), binding.ErrorHandler, ProtectedPage, UpdateSettings)
		r.Post("/installation", strict.ContentType("application/json"), binding.Json(Vertigo{}), binding.ErrorHandler, UpdateSettings)
		r.Get("/users", ReadUsers)
		r.Get("/user/logout", LogoutUser)
		r.Get("/user/:id", ReadUser)
		//r.Delete("/user", DeleteUser)
		r.Post("/user", strict.ContentType("application/json"), binding.Json(User{}), binding.ErrorHandler, CreateUser)
		r.Post("/user/login", strict.ContentType("application/json"), binding.Json(User{}), binding.ErrorHandler, LoginUser)
		r.Post("/user/recover", strict.ContentType("application/json"), binding.Json(User{}), RecoverUser)
		r.Post("/user/reset/:id/:recovery", strict.ContentType("application/json"), binding.Json(User{}), ResetUserPassword)

		//r.Get("/posts", ReadPosts)
		r.Get("/posts.json", ReadPosts)

		//r.Get("/post/:slug", ReadPost)
		r.Get("/post/:slug.json", ReadPost)

		r.Post("/post", strict.ContentType("application/json"), binding.Json(Post{}), binding.ErrorHandler, ProtectedPage, CreatePost)
		r.Get("/post/:slug/publish", ProtectedPage, PublishPost)
		r.Get("/post/:slug/unpublish", ProtectedPage, UnpublishPost)
		r.Post("/post/:slug/edit", strict.ContentType("application/json"), binding.Json(Post{}), binding.ErrorHandler, ProtectedPage, UpdatePost)
		r.Get("/post/:slug/delete", ProtectedPage, DeletePost)
		r.Post("/post", strict.ContentType("application/json"), binding.Json(Post{}), binding.ErrorHandler, ProtectedPage, CreatePost)
		r.Post("/post/search", strict.ContentType("application/json"), binding.Json(Search{}), binding.ErrorHandler, SearchPost)

		r.Post("/images.json", ProtectedPage, UploadImage)
		r.Post("/save", func(req *http.Request, db *gorm.DB, s sessions.Session) {
			log.Printf("\n%+v\n", req)
			log.Printf("\n%+v\n", req.Body)
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				// err
			}
			sb := string(body[:])

			CreateDraft(sb, db, s)
			log.Printf("\n%+v\n", s)
		})
	})

	m.Router.NotFound(strict.MethodNotAllowed, strict.NotFound)
	return m
}

/*
func redir(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, urlHost()+req.RequestURI, http.StatusMovedPermanently)
}
*/

func main() {
	server := NewServer()
	if os.Getenv("PORT") != "" {
		browser.OpenURL("http://localhost:" + os.Getenv("PORT"))
	} else {
		browser.OpenURL("http://localhost:3000")
	}

	//http.ListenAndServe(":80", http.HandlerFunc(redir))
	server.Run()
}
