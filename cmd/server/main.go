package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/valyala/fasthttp"
)

// Location blok
type Location struct {
	Path  string
	Root  string
	Index string
}

// Server blok
type Server struct {
	Listen     string
	ServerName string
	Locations  []Location
}

// Config
type Config struct {
	Servers []Server
}

// Config faylni o'qish (nginx tarzida)
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	scanner := bufio.NewScanner(file)

	var currentServer *Server
	var currentLocation *Location
	var inServer, inLocation bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "server {" {
			currentServer = &Server{}
			inServer = true
			continue
		}

		if line == "}" && inLocation {
			// location blok tugadi
			currentServer.Locations = append(currentServer.Locations, *currentLocation)
			currentLocation = nil
			inLocation = false
			continue
		}

		if line == "}" && inServer {
			// server blok tugadi
			config.Servers = append(config.Servers, *currentServer)
			currentServer = nil
			inServer = false
			continue
		}

		if strings.HasPrefix(line, "location ") && strings.HasSuffix(line, "{") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "location"))
			path = strings.TrimSpace(strings.TrimSuffix(path, "{"))
			currentLocation = &Location{Path: path}
			inLocation = true
			continue
		}

		// direktivalar key value;
		if strings.HasSuffix(line, ";") {
			parts := strings.Fields(strings.TrimSuffix(line, ";"))
			if len(parts) >= 2 {
				key := parts[0]
				value := strings.Join(parts[1:], " ")
				if inLocation {
					switch key {
					case "root":
						currentLocation.Root = value
					case "index":
						currentLocation.Index = value
					}
				} else if inServer {
					switch key {
					case "listen":
						currentServer.Listen = value
					case "server_name":
						currentServer.ServerName = value
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &config, nil
}

// HTTP handler
func makeHandler(location Location) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/html; charset=utf-8")
		log.Printf("Request: %s %s", ctx.Method(), ctx.Path())
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString(fmt.Sprintf("<h1>Path: %s, Root: %s, Index: %s</h1>", location.Path, location.Root, location.Index))
	}
}

func main() {
	config, err := LoadConfig("config.cfm")
	if err != nil {
		log.Fatalf("Config yuklanmadi: %v", err)
	}

	for _, server := range config.Servers {
		fmt.Printf("Server %s:%s ishlayapti\n", server.ServerName, server.Listen)
		mux := func(ctx *fasthttp.RequestCtx) {
			for _, loc := range server.Locations {
				if string(ctx.Path()) == loc.Path {
					makeHandler(loc)(ctx)
					return
				}
			}
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}

		go func(s Server) {
			fasthttp.ListenAndServe(s.ServerName+":"+s.Listen, mux)
		}(server)
	}

	select {} // serverlarni ishlatib turish uchun bloklash
}
