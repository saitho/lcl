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
	return Rule{Action: "default"}, nil
}

func getSubdomain(host string) (string, error) {
	var re = regexp.MustCompile(`^(?:([a-zA-Z0-9](?:[-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?)\.)?(?:localhost|(?:[a-zA-Z0-9]{1,2}(?:[-a-zA-Z0-9]{0,252}[a-zA-Z0-9])?)\.(?:[a-zA-Z]{2,63}))(?::(?:\d+))?$`)

	matches := re.FindStringSubmatch(host)
	if len(matches) < 2 || matches[1] == "" {
		return "", fmt.Errorf("no subdomain found")
	}
	return matches[1], nil
}

func determineAction(aliases map[string]Alias, request *http.Request) (string, string, http.FileSystem) {
	subdomain, err := getSubdomain(request.Host)

	if err != nil {
		fsys, err := fs.Sub(staticPath, "static")
		if err != nil {
			panic(err)
		}
		return "serve", "", http.FS(fsys)
	}

	path := request.URL.Path
	port := subdomain

	// Check if port is a number
	_, err = strconv.Atoi(port)
	if err != nil {
		// NaN - check if subdomain is alias
		if value, ok := aliases[port]; ok {
			port = strconv.Itoa(value.Port)
			rule, err := getRuleForPath(value, path)
			if err != nil {
				return "", "", nil
			}
			if rule.Action == "default" {
			} else if rule.Action == "noop" {
				return "noop", "", nil
			} else {
				return rule.Action, rule.Target, nil
			}
		} else {
			return "", "", nil
		}
	}

	// Default redirect to localhost
	query := request.URL.RawQuery
	if query != "" {
		query = "?" + query
	}
	return "redirect", "http://localhost:" + port + path + query, nil
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
		log.Println(fmt.Sprintf("Loaded %d subdomain aliases.", len(aliases)))
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Tool", "lcl")
		subdomain, err := getSubdomain(request.Host)
		if err != nil {
			fsys, err := fs.Sub(staticPath, "static")
			if err != nil {
				panic(err)
			}
			log.Println(fmt.Sprintf("Received request on main domain. Serving lcl website"))
			http.FileServer(http.FS(fsys)).ServeHTTP(writer, request)
			return
		}

		action, path, serveFs := determineAction(aliases, request)

		switch action {
		case "noop":
			log.Println(fmt.Sprintf("Received request on subdomain %s <NOOP>", subdomain))
			writer.WriteHeader(http.StatusOK)
			break
		case "serve":
			if serveFs != nil {
				http.FileServer(serveFs).ServeHTTP(writer, request)
				log.Println(fmt.Sprintf("Received request on subdomain %s <SERVE:%s>", subdomain, serveFs))
			} else {
				// serve path
				log.Println(fmt.Sprintf("Received request on subdomain %s <SERVE:%s>", subdomain, path))
			}
			break
		case "redirect":
			log.Println(fmt.Sprintf("Received request on subdomain %s <REDIRECT:%s>", subdomain, path))
			http.Redirect(writer, request, path, http.StatusTemporaryRedirect)
			break
		default:
			log.Println(fmt.Sprintf("Received request on subdomain %s <NOTFOUND>", subdomain))
			http.NotFound(writer, request)
			break
		}
	})

	log.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
