package main

import (
	"flag"
	"fmt"
	"github.com/go-zoo/bone"
	jww "github.com/spf13/jwalterweatherman"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	"github.com/valleycamp/weathermoss/api"
)

func main() {
	flgVerbose := flag.Bool("verbose", false, "Output additional debugging information to both STDOUT and the log file")
	flgPortNum := flag.Int("port", 8777, "The port to run the HTTP server on.") // 8777 = "WM"
	flgConfigPath := flag.String("conf", "weathermoss-conf.json", "Path to the config JSON file")
	flag.Parse()

	// Note at this point only WARN or above is actually logged to file, and ERROR or above to console.
	jww.SetLogFile("weathermoss.log")

	// Set extra logging if the command line flag was set
	if *flgVerbose {
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetStdoutThreshold(jww.LevelInfo)
		jww.INFO.Println("Verbose debug level set.")
	} else {
		// Set custom default logging verbosity.
		jww.SetLogThreshold(jww.LevelWarn)
		jww.SetStdoutThreshold(jww.LevelError)
	}

	// Read config file
	appconf, err := getConfigFromFile(*flgConfigPath)
	if err != nil {
		jww.FATAL.Println("Configuration Error:", err)
		os.Exit(0)
	}

	// Set up something to handle ctrl-c/kill cleanup!
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("")
		// TODO: Graceful cleanup http's listenAndServe runner?
		// TODO: Any other cleanup needed?
		fmt.Println("Cleaned up and shut down.")
		os.Exit(0)
	}()

	jww.DEBUG.Println(fmt.Sprintf("Connecting to db: %s:%s@tcp(%s:%s)/%s?parseTime=true", appconf.DB.Username, appconf.DB.Password, appconf.DB.Host, appconf.DB.Port, appconf.DB.Database))
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", appconf.DB.Username, appconf.DB.Password, appconf.DB.Host, appconf.DB.Port, appconf.DB.Database))
	if err != nil {
		jww.FATAL.Println("Failed to open database. Error was:", err)
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		jww.FATAL.Println("Failed to open database. Error was:", err)
		os.Exit(1)
	}

	// Somewhat arbitrary. TODO: Tune as necessary.
	db.SetMaxIdleConns(500)
	db.SetMaxOpenConns(1000)

	// Set up the HTTP router, followed by all the routes
	router := bone.New()

	/*
		// Redirect static resources, and then handle the static resources (/gui/) routes with the static asset file
		router.Handle("/", http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			http.Redirect(response, request, "/gui/", 302)
		}))
		router.Get("/gui/", http.StripPrefix("/gui/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""})))
	*/
	router.GetFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Weathermoss"))
	})

	// Define the API (JSON) routes
	api := api.NewApiHandlers(db)
	router.GetFunc("/api/current", api.Current)
	router.GetFunc("/api/ws", api.WsCombinedHandler)
	router.GetFunc("/api/ws/10min", api.WsTenMinuteHandler)
	router.GetFunc("/api/ws/15sec", api.WsFifteenSecHandler)

	// Start the HTTP server
	fmt.Println("Starting API server on port", *flgPortNum, ". Press Ctrl-C to quit.")
	http.ListenAndServe(fmt.Sprintf(":%d", *flgPortNum), router)
}
