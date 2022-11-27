package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/labstack/echo"
	"golang.org/x/net/websocket"

	"tinystatic/routes"
)

var build func()
var watcher fsnotify.Watcher

func hello(c echo.Context) error {
	fmt.Println("Hello s")

	watcher, _ := fsnotify.NewWatcher()
	// watcher.Add(".")
	// watcher.Add("./static")
	// watcher.Add("./examples/blog/routes")
	// watcher.Add("./examples/blog/routes/static")
	// watcher.Add("./examples/blog/partials")
	// watcher.Add("./examples/blog/templates")

	root_dir := "./examples/blog"
	var add_dirs func(string)

	add_dirs = func(dir string) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		for _, f := range files {
			if f.IsDir() && !strings.Contains(f.Name(), "output") {
				path := dir + "/" + f.Name()
				fmt.Println(path)
				watcher.Add(path)
				add_dirs(path)
			}
		}
	}
	add_dirs(root_dir)

	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			e, ok := <-watcher.Events
			fmt.Println(e, ok)
			build()

			err := websocket.Message.Send(ws, "page")
			if err != nil {
				c.Logger().Error(err)
			}
			err = websocket.Message.Send(ws, "css")
			if err != nil {
				c.Logger().Error(err)
			}
			// time.Sleep(4 * time.Second)``
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

var (
	partialDir  string
	templateDir string
)

func main() {
	var outputDir string
	var routeDir string
	var clean bool
	flag.StringVar(&outputDir, "output", "./examples/blog/output", "The directory to write the generated outputs to")
	flag.StringVar(&routeDir, "routes", "./examples/blog/routes", "The directory from which to read the routes")
	flag.StringVar(&partialDir, "partials", "./examples/blog/partials", "The directory from which to read the partials")
	flag.StringVar(&templateDir, "templates", "./examples/blog/templates", "The directory from which to read the templates")
	flag.BoolVar(&clean, "clean", false, "Whether to delete the output directory before regenerating")
	flag.Parse()

	if clean {
		log.Println("Removing previous output from", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			log.Fatalln(err)
		}
	}

	build = func() {

		log.Println("Loading routes from", routeDir)
		rootRoute, err := routes.LoadRoutes("/", routeDir)
		if err != nil {
			log.Fatalln(err)
		}

		if err := routes.ExpandRoutes(&rootRoute); err != nil {
			log.Fatalln(err)
		}

		log.Println("Writing output to", outputDir)
		allRoutes := rootRoute.AllRoutes()
		for _, r := range allRoutes {
			fmt.Println("output", r)
			if r.Href != "" {
				log.Println("∟", r.FilePath, "->", r.Href)
			}
			if err := r.Generate(outputDir, partialDir, templateDir, allRoutes); err != nil {
				log.Fatalln(err)
			}
		}
		log.Println("Finished")
	}

	build()

	e := echo.New()
	e.Static("/", "./static")
	e.Static("/", outputDir)
	e.GET("/.gostatic.hotreload", hello)
	e.GET("/", func(c echo.Context) error {

		return nil
	})
	e.Start("localhost:8080")

}
