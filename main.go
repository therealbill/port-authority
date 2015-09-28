package main // import "github.com/therealbill/port-authority"

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/cli"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/therealbill/airbrake-go"
	"github.com/therealbill/port-authority/actions"
	"github.com/therealbill/port-authority/handlers"
	"github.com/zenazn/goji"
)

var Build string
var key string

type LaunchConfig struct {
	Name              string
	Port              int
	RPCPort           int
	BindAddress       string
	TemplateDirectory string
}

var (
	config LaunchConfig
	app    *cli.App
)

func init() {
	// Register consul store to libkv
	consul.Register()

}

func serve(c *cli.Context) {
	client := c.String("consuladdress")
	name := c.String("name")

	// Initialize a new store with consul
	kv, err := libkv.NewStore(
		store.CONSUL, // or "consul"
		[]string{client},
		&store.Config{
			ConnectionTimeout: 10 * time.Second,
		},
	)
	if err != nil {
		log.Fatal("Cannot create store consul")
	}

	prefix := "app/port-authority/config"
	key_apiport := fmt.Sprintf("%s/api_port", prefix)
	//key_uiport := fmt.Sprintf("%s/ui_port", prefix)
	key_rpcport := fmt.Sprintf("%s/rpc_port", prefix)
	key_templatedir := fmt.Sprintf("%s/template_directory", prefix)
	key_airbrakeapikey := fmt.Sprintf("%s/airbrake/api_key", prefix)
	key_airbrakeendpoint := fmt.Sprintf("%s/airbrake/endpoint", prefix)
	havestore := true
	tmp, err := kv.Get(key_apiport)
	if err != nil {
		havestore = false
		log.Printf("Error: %v", err)
	} else {
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("key not found in config store: %s/api_port", prefix)
			} else {
				log.Print("Error on connection: %v", err)
			}
		} else {
			port, err := strconv.Atoi(string(tmp.Value))
			if err != nil {
				config.Port = int(port)
			}
		}
	}
	var port_start = 30000
	var port_end = 40000
	if havestore {
		//config.RPCPort, err = kv.Get(key_rpcport)
		log.Printf("Connected to config store")
		tmp, err = kv.Get(key_templatedir)
		if err != nil {
			log.Print("template_directory key not foumd, using /tmp as default!")
			config.TemplateDirectory = "/tmp"
		} else {
			config.TemplateDirectory = string(tmp.Value)
		}

		tmp, err = kv.Get(key_rpcport)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("key not found in config store: %s/rpc_port", prefix)
			} else {
				log.Print("Error on connection: %v", err)
			}
		} else {
			fmt.Printf("rpcport: %s", string(tmp.Value))
			port, err := strconv.Atoi(string(tmp.Value))
			log.Printf("%s %v", port, err)
			if err != nil {
				config.RPCPort = int(port)
			}
		}
		var my_key string
		if name != "" {
			my_key = fmt.Sprintf("%s/%s", prefix, name)
		} else {
			my_key = prefix
		}

		key_portstart := fmt.Sprintf("%s/ports_begin", my_key)
		tmp, err = kv.Get(key_portstart)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("key not found in config store: %s", key_portstart)
			} else {
				log.Print("Error on connection: %v", err)
			}
		} else {
			port, err := strconv.Atoi(string(tmp.Value))
			if err == nil {
				port_start = int(port)
			}
		}

		key_portend := fmt.Sprintf("%s/ports_end", my_key)
		tmp, err = kv.Get(key_portend)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("key not found in config store: %s", key_portend)
			} else {
				log.Print("Error on connection: %v", err)
			}
		} else {
			port, err := strconv.Atoi(string(tmp.Value))
			if err == nil {
				port_end = int(port)
			}
		}
	}
	if actions.InitializeRedisClient("127.0.0.1:6379", "") != nil {
		log.Fatal("Can not connect to Redis!")
	}
	log.Printf("Initializing with ports from %d to %d", port_start, port_end)
	err = actions.InitializePorts(port_start, port_end)
	if err != nil {
		if strings.Contains(err.Error(), "already been init") {
			log.Print(err.Error())
		} else {
			log.Printf("Error on init: %v", err)
		}
	}

	if len(config.BindAddress) != 0 {
		flag.Set("bind", config.BindAddress)
	}
	if config.Port == 0 {
		log.Print("ENV contained no port, using default")
		config.Port = 8080
		flag.Set("bind", fmt.Sprintf("%s:%d", config.BindAddress, config.Port))
	}
	if config.RPCPort == 0 {
		config.RPCPort = config.Port + 1
	}

	if config.TemplateDirectory > "" {
		if !strings.HasSuffix(config.TemplateDirectory, "/") {
			config.TemplateDirectory += "/"
		}
	}
	handlers.TemplateBase = config.TemplateDirectory

	config_json, _ := json.Marshal(config)
	if havestore {
		keypair, err := kv.Get(key_airbrakeapikey)
		abendpoint, err := kv.Get(key_airbrakeendpoint)
		if err == nil {
			key := keypair.Value
			//airbrake.Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"
			airbrake.Endpoint = string(abendpoint.Value)
			airbrake.ApiKey = string(key)
			airbrake.Environment = os.Getenv("RVT_ENVIRONMENT")
			if len(airbrake.Environment) == 0 {
				airbrake.Environment = "Development"
			}
		}
	}
	log.Printf("Config: %s", config_json)
	// HTML Interface URLS
	// API URLS
	goji.Put("/api/service/:id", handlers.APIGetOpenPort)
	goji.Get("/api/service/:id", handlers.APIGetPortFromInstance)
	goji.Delete("/api/service/:id", handlers.APIRemoveService)
	goji.Get("/api/port/:port", handlers.APIGetInstanceFromPort)
	goji.Get("/api/ports/inventory/count", handlers.APIGetPortCapacity)
	goji.Get("/api/ports/inventory/list", handlers.APIGetAvailableInventory)
	goji.Get("/api/ports/assigned/count", handlers.APIGetAssignedCount)
	goji.Get("/api/ports/assigned/list", handlers.APIGetAssignedList)
	goji.Serve()
}

func main() {
	app = cli.NewApp()
	app.Name = "port-authority"
	app.Usage = "Manage, assign, and track port assignments"
	app.Version = "0.1"
	app.EnableBashCompletion = true
	author := cli.Author{Name: "Bill Anderson", Email: "therealbill@me.com"}
	app.Authors = append(app.Authors, author)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "consuladdress,c",
			Usage:  "Address of the Consul server",
			EnvVar: "PA_CONSUL",
			Value:  "localhost:8500",
		},
		cli.StringFlag{
			Name:   "name,n",
			Usage:  "Name of the running server",
			EnvVar: "PA_NAME",
			Value:  "apiserver-01",
		},
		cli.StringFlag{
			Name:   "envirnmemt,e",
			Usage:  "Name of the airbrake environment for this server",
			EnvVar: "PA_ENV",
			Value:  "apiserver-01",
		},
	}
	app.Action = serve
	app.Run(os.Args)
}
