package main

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v2"
)

//go:embed static
var staticPath embed.FS

type Rule struct {
	Path   string
	Action string
	Target string
}

type Alias struct {
	Alias       string
	Port        int
	Certificate struct {
		UseCustom bool
	}
	Rules []Rule
}

type Config struct {
	Aliases []Alias
}

func getRuleForPath(alias Alias, path string) (Rule, error) {
	for _, rule := range alias.Rules {
		if rule.Path != path {
			continue
		}
		return rule, nil
	}
	return Rule{}, nil
}

func determineAction(aliases map[string]Alias, request *http.Request) (string, string, http.FileSystem) {
	var re = regexp.MustCompile(`^(?:([a-zA-Z0-9](?:[-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?)\.)?(?:localhost|(?:[a-zA-Z0-9]{1,2}(?:[-a-zA-Z0-9]{0,252}[a-zA-Z0-9])?)\.(?:[a-zA-Z]{2,63}))(?::(?:\d+))?$`)

	matches := re.FindStringSubmatch(request.Host)

	if len(matches) < 2 || matches[1] == "" {
		fsys, err := fs.Sub(staticPath, "static")
		if err != nil {
			panic(err)
		}
		return "serve", "", http.FS(fsys)
	}

	path := request.URL.Path
	port := matches[1]

	// Check if port is a number
	_, err := strconv.Atoi(port)
	if err != nil {
		// NaN - check if subdomain is alias
		if value, ok := aliases[port]; ok {
			port = strconv.Itoa(value.Port)
			rule, err := getRuleForPath(value, path)
			if err != nil {
				return "", "", nil
			}
			if rule.Action == "noop" {
				return "noop", "", nil
			} else {
				return rule.Action, rule.Target, nil
			}
		} else {
			return "", "", nil
		}
	}

	// Default redirect to localhost
	return "redirect", "http://localhost:" + port + path, nil
}

func main() {
	// Load alias config
	config := Config{}
	aliases := map[string]Alias{}

	data, err := ioutil.ReadFile("./subdomain_aliases.yml")
	if err == nil {
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			panic("Unable to parse config file")
		}
		for _, alias := range config.Aliases {
			aliases[alias.Alias] = alias
		}
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Tool", "lcl")
		var re = regexp.MustCompile(`^(?:([a-zA-Z0-9](?:[-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?)\.)?(?:localhost|(?:[a-zA-Z0-9]{1,2}(?:[-a-zA-Z0-9]{0,252}[a-zA-Z0-9])?)\.(?:[a-zA-Z]{2,63}))(?::(?:\d+))?$`)

		matches := re.FindStringSubmatch(request.Host)
		if len(matches) < 2 || matches[1] == "" {
			fsys, err := fs.Sub(staticPath, "static")
			if err != nil {
				panic(err)
			}
			http.FileServer(http.FS(fsys)).ServeHTTP(writer, request)
			return
		}

		action, path, serveFs := determineAction(aliases, request)

		switch action {
		case "noop":
			writer.WriteHeader(http.StatusOK)
			break
		case "serve":
			if serveFs != nil {
				http.FileServer(serveFs).ServeHTTP(writer, request)
			} else {
				// serve path
			}
			break
		case "redirect":
			http.Redirect(writer, request, path, http.StatusTemporaryRedirect)
			break
		default:
			http.NotFound(writer, request)
			break
		}
	})

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
