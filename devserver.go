package ortfomk

import (
	"fmt"
	"net/http"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
	"github.com/omeid/go-livereload"
)

var (
	livereloadAddr = ":8357"
)

func StartDevServer(port int, distFolder string) {
	files, err := doublestar.Glob(distFolder + "/**")
	if err != nil {
		printerr("could not get files to watch for changes", err)
	}

	watch, err := fsnotify.NewWatcher()
	if err != nil {
		printerr("could not start the watcher", err)
	}
	defer watch.Close()

	for _, file := range files {
		err = watch.Add(file)
		if err != nil {
			printerr("could not add "+file+" to the files to watch", err)
			return
		}
	}

	if distFolder != "" {
		go func() {
			static := http.StripPrefix(distFolder, http.FileServer(http.Dir(distFolder)))
			err = http.ListenAndServe(":"+fmt.Sprint(port), static)
			if err != nil {
				printerr("An error occured while starting the server", err)
			}
		}()
	}
	lr := livereload.New("go-livereload")
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/livereload.js", livereload.LivereloadScript)
		mux.Handle("/", lr)
		// log.Infof("Serving livereload at %s", *livereloadAddr)
		err = http.ListenAndServe(livereloadAddr, mux)
		if err != nil {
			printerr("An error occured while starting the server", err)
		}
	}()

	for {
		select {
		case event := <-watch.Events:
			if event.Op&(fsnotify.Rename|fsnotify.Create|fsnotify.Write) > 0 {
				printfln("Reloading %s\n", event.Name)
				lr.Reload(event.Name, true)
			}
		case err := <-watch.Errors:
			if err != nil {
				printerr("An error occured", err)
			}
		}
	}
}
