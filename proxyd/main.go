package main

import (
	"context"
	"log"

	"os"

	"github.com/aidanhopper/reverse-proxy/proxy-engine/engine"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	// "gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Entrypoints map[string]string
	HTTP        struct {
		Middlewares []string
		Routes      map[string]struct {
			Rule        string
			Service     string
			Middlewares []string
			Entrypoints []string
			TLS         string
		}
		Services map[string]struct {
			ReverseProxy string `yaml:"reverse-proxy"`
			LoadBalancer struct {
				Method   string
				Services []struct {
					ReverseProxy string `yaml:"reverse-proxy"`
				}
			} `yaml:"load-balancer"`
		}
	}
}

type State struct {
	Config ServerConfig
	Server *engine.Server
}

type MapItem[T any] struct {
	key  string
	item T
}

func UpdatedMapItems[T comparable](old map[string]T, updated map[string]T) []MapItem[T] {
	var ret []MapItem[T]
	for key, updatedValue := range updated {
		oldValue, present := old[key]
		if !present || oldValue != updatedValue {
			ret = append(ret, MapItem[T]{
				key,
				updatedValue,
			})
			continue
		}
	}
	return ret
}

func DeletedMapItems[T comparable](old map[string]T, updated map[string]T) []MapItem[T] {
	var ret []MapItem[T]
	for key, oldValue := range old {
		_, present := updated[key]
		if !present {
			ret = append(ret, MapItem[T]{
				key,
				oldValue,
			})
			continue
		}
	}
	return ret
}

func ChangedMapItems[T comparable](old map[string]T, updated map[string]T) ([]MapItem[T], []MapItem[T]) {
	return UpdatedMapItems(old, updated), DeletedMapItems(old, updated)
}

func (state *State) Reconsile(newConfig ServerConfig) {
	log.Println(newConfig)

	// Update entrypoints
	updated, deleted := ChangedMapItems(state.Config.Entrypoints, newConfig.Entrypoints)
	for _, value := range deleted {
		state.Server.DeregisterEntryPoint(value.key)
	}

	for _, value := range updated {
		// Will need to change this to look if its a http/tcp, udp, or unix entrypoint
		state.Server.RegisterEntryPoint(engine.TCPEntryPoint{
			Identifier: value.key,
			Address:    value.item,
		})
	}

	state.Config = newConfig
}

func ReadServerConfig(path string) (ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServerConfig{}, err
	}

	newConfig := ServerConfig{}
	err = yaml.Unmarshal(data, &newConfig)
	if err != nil {
		return ServerConfig{}, err
	}

	return newConfig, nil
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Fatal("Please specify path to config file")
	}
	configPath := args[1]

	config, err := ReadServerConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	state := &State{
		Server: engine.NewServer(),
	}

	go state.Server.Serve(context.Background())

	state.Reconsile(config)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Name == configPath && event.Has(fsnotify.Create) {
					newConfig, err := ReadServerConfig(configPath)
					if err != nil {
						log.Println(err)
						continue
					}
					state.Reconsile(newConfig)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}

	select {}
}
