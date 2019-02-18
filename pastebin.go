package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"github.com/hako/durafmt"
)

type Paste struct {
	Url                 string
	Title               string
	Body                string
	Syntax              string
	Mime                string
	Size                int
	Clicks              int
	Expiration          int64
	CreatedOn           int64
	CreatedOnFormatted  time.Time `redis:"-"`
	ExpirationFormatted string    `redis:"-"`
}

type Configuration struct {
	RedisHost string `json:"redisHost"`
	RedisPass string `json:"redisPass"`
	HttpPort  string `json:"httpPort"`
	AppName   string `json:"AppName"`
	AppUrl    string `json:"AppUrl"`
}

var (
	//validPath = regexp.MustCompile("^(|/(raw|p|clone))/([a-zA-Z0-9]+)$")
	validPath   = regexp.MustCompile("^(|/(p|clone))/([a-zA-Z0-9]+)$")
	templates   = template.Must(template.ParseFiles("templates/index.html", "templates/paste.html", "templates/info.html"))
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	mimeTypes   = map[string]string{
		"text/plain":         "text",
		"text/generic":       "generic",
		"text/javascript":    "javascript",
		"text/x-csrc":        "clike",
		"text/x-cmake":       "cmake",
		"text/css":           "css",
		"text/x-d":           "d",
		"text/x-diff":        "diff",
		"text/x-dockerfile":  "dockerfile",
		"text/x-erlang":      "erlang",
		"text/x-go":          "go",
		"text/x-haskell":     "haskell",
		"text/html":          "xml",
		"text/x-java":        "clike",
		"jinja2":             "jinja2",
		"text/x-kotlin":      "clike",
		"text/x-lua":         "lua",
		"text/x-markdown":    "markdown",
		"text/x-perl":        "perl",
		"text/x-php":         "php",
		"text/x-python":      "python",
		"text/x-rpm-changes": "rpm",
		"text/x-rst":         "rst",
		"text/x-ruby":        "ruby",
		"text/x-rust":        "rust",
		"text/x-sh":          "shell",
		"text/x-sql":         "sql",
		"text/x-swift":       "swift",
		"text/x-yaml":        "yaml",
		"application/xml":    "xml",
	}
	RedisPool *redis.Pool
	config    Configuration
)

func panicOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%v: %v", msg, err)
		panic(fmt.Sprintf("%v: %v", msg, err))
	}
}

func newRedisPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func getRedisPool() {
	RedisPool = newRedisPool(config.RedisHost, config.RedisPass)
	c := RedisPool.Get()
	defer c.Close()

	pong, err := redis.String(c.Do("PING"))
	panicOnError(err, "Cannot ping Redis")
	log.Infof("Redis PING: %s", pong)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p Paste) {
	type Response struct {
		Paste   Paste
		AppName string
		AppUrl  string
	}
	var response Response
	response.Paste = p
	response.AppName = config.AppName
	response.AppUrl = config.AppUrl

	err := templates.ExecuteTemplate(w, tmpl+".html", response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func generateUrl() string {
	c := RedisPool.Get()
	defer c.Close()

	url := randomString(4)
	var err error
	for reply := 1; reply != 0; reply, err = redis.Int(c.Do("EXISTS", "paste:"+url)) {
		url = randomString(4)
		panicOnError(err, "Redis error checking for existing pastes")
	}
	return url
}

func handleAndValidateForm(r *http.Request) (Paste, error) {
	println(r.FormValue("p"))
	title := r.FormValue("title")
	body := r.FormValue("p")
	mime := r.FormValue("mime")
	syntax := mimeTypes[r.FormValue("mime")]
	expire := r.FormValue("expire")

	return createPaste(title, body, syntax, mime, expire)
}

func createPaste(title string, body string, syntax string, mime string, expire string) (Paste, error) {
	var p Paste

	if len(title) == 0 {
		title = "Untitled paste"
	} else if len(title) > 50 {
		return p, errors.New("Title can not be longer than 50 characters.")
	}

	if len(body) == 0 {
		return p, errors.New("Paste is empty.")
	} else if len(body) > 100000 {
		return p, errors.New("Paste is too big.")
	}

	if syntax == "" || mime == "" {
		syntax = "text"
		mime = "text/plain"
	}

	expireInt, _ := strconv.ParseInt(expire, 10, 64)

	p = Paste{
		Url:        generateUrl(),
		Title:      title,
		Body:       body,
		Syntax:     syntax,
		Mime:       mime,
		Expiration: expireInt,
		CreatedOn:  time.Now().Unix(),
	}
	return p, nil
}

func savePaste(p Paste) {
	c := RedisPool.Get()
	defer c.Close()

	key := "paste:" + p.Url
	c.Send("HMSET", redis.Args{}.Add(key).AddFlat(&p)...)

	if p.Expiration != 0 {
		c.Send("EXPIRE", key, p.Expiration)
	}

	c.Flush()
	_, err := c.Receive()
	panicOnError(err, "Redis error trying to create new paste")
}

func getPaste(r *http.Request) (Paste, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)

	if m == nil {
		return Paste{}, errors.New("Invalid route")
	}
	url := m[3]

	c := RedisPool.Get()
	defer c.Close()

	var p Paste
	reply, err := redis.Values(c.Do("HGETALL", "paste:"+url))
	if len(reply) == 0 || err != nil {
		err = errors.New("Paste not found")
		return Paste{}, err
	}
	redis.ScanStruct(reply, &p)

	c.Do("HINCRBY", "paste:"+p.Url, "Clicks", 1)

	p.CreatedOnFormatted = time.Unix(p.CreatedOn, 0).UTC()
	if p.Expiration != 0 {
		p.ExpirationFormatted = durafmt.Parse(time.Until(time.Unix(p.CreatedOn+p.Expiration, 0).UTC())).String()
	}
	if p.Syntax == "generic" {
		p.Syntax = ""
	}

	return p, nil
}

func pasteViewHandler(w http.ResponseWriter, r *http.Request) {
	p, err := getPaste(r)
	if err != nil {
		errorHandler(w, r, http.StatusNotFound)
	} else {
		renderTemplate(w, "paste", p)
	}
}

func cloneHandler(w http.ResponseWriter, r *http.Request) {
	p, err := getPaste(r)
	if err != nil {
		errorHandler(w, r, http.StatusNotFound)
	} else {
		p.Title += " Copy"
		renderTemplate(w, "index", p)
	}
}

func documentHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		contentType := r.Header.Get("Content-Type")

		log.Printf("%#v", r)

		var data string

		switch contentType {
		case "multipart/form-data":
			log.Print("multipart/form-data received")

			data = r.FormValue("data")
			log.Printf("data: %#v", data)
		default:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			data = string(body)

			log.Printf("%#v", data)
		}

		p, err := createPaste("", data, "", "", "0")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"key\":\"" + p.Url + "\"}"))

		log.Printf("created url: %s", p.Url)

		return
	default:
		http.Error(w, "Invalid request", http.StatusBadRequest)
	}
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if r.URL.Path == "/" {
			renderTemplate(w, "index", Paste{})
		} else {
			p, err := getPaste(r)
			if err != nil {
				errorHandler(w, r, http.StatusNotFound)
			} else {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, "%s", p.Body)
				//template.HTMLEscape(w, []byte(p.Body))
			}
		}
	case "POST":
		p, err := handleAndValidateForm(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		savePaste(p)

		if r.FormValue("raw") == "0" {
			if r.FormValue("nojs") == "1" {
				http.Redirect(w, r, config.AppUrl+"/p/"+p.Url, http.StatusMovedPermanently)
			} else {
				http.Error(w, config.AppUrl+"/p/"+p.Url, http.StatusOK)
			}
		} else {
			http.Error(w, config.AppUrl+"/"+p.Url, http.StatusOK)
		}

	default:
		http.Error(w, "Invalid request", http.StatusBadRequest)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fn(w, r)
	}
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "info", Paste{})
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, "404: Wrong url or deleted.")
	}
}

func loadConfiguration() Configuration {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal("Cannot load config.json")
	}

	var c Configuration
	err = json.Unmarshal(file, &c)
	if err != nil {
		if err != nil {
			log.Fatal("Cannot decode config.json")
		}
	}
	log.Info("config.json loaded")
	return c
}

func main() {
	config = loadConfiguration()
	rand.Seed(time.Now().UnixNano())
	getRedisPool()

	// Routing
	http.HandleFunc("/p/", makeHandler(pasteViewHandler))
	http.HandleFunc("/clone/", makeHandler(cloneHandler))
	http.HandleFunc("/info", makeHandler(infoHandler))
	http.HandleFunc("/documents", makeHandler(documentHandler))
	http.HandleFunc("/", makeHandler(pasteHandler))
	http.Handle("/assets/", gziphandler.GzipHandler(http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/public")))))
	log.Fatal(http.ListenAndServe(":"+config.HttpPort, nil))
}
