package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const configDirectoryPath = "/usr/local/etc/autocrypt/"
const configFilename = "paths.txt"

var refreshInterval int64
var gracePeriod int64
var extendTime int64
var autolockTime int64

func main() {
	fmt.Println("Initializing")
	if runtime.GOOS != "linux" {
		fmt.Println("Error: Compatible with Linux only")
		os.Exit(3)
	}
	if _, err := os.Stat(filepath.Join(configDirectoryPath, configFilename)); os.IsNotExist(err) {
		fmt.Println("Configuration file does not exist. Creating one")
		createConfig()
	}
	fileHandle, err := os.Open(filepath.Join(configDirectoryPath, configFilename))
	if err != nil {
		fmt.Println("Failed to read config")
		os.Exit(1)
	}
	fileBytes, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		fmt.Println("Failed to read config")
		os.Exit(1)
	}
	_ = fileHandle.Close()
	var config Config
	err = toml.Unmarshal(fileBytes, &config)
	if err != nil {
		fmt.Println("Failed to read config")
		os.Exit(1)
	}
	refreshInterval = int64(config.RefreshInterval)
	gracePeriod = int64(config.GracePeriod)
	extendTime = int64(config.ExtendTime)
	autolockTime = int64(config.AutoLockTime)
	var watchList = make([]string, 0)
	for _, path := range config.WatchPaths {
		if strings.Trim(path, " ") == "" {
			continue
		}
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Println(fmt.Sprintf("Error: %s does not exist. Will not watch", path))
		} else {
			watchList = append(watchList, path)
		}
	}
	fmt.Println(fmt.Sprintf("Directories to watch: %d", len(watchList)))
	if len(watchList) == 0 {
		fmt.Println("No directories to watch. Exiting")
		os.Exit(0)
	}
	for _, watchPath := range config.WatchPaths {
		go watcher(watchPath)
	}
	fmt.Println("Listening to filesystem changes...")
	lock := make(chan bool)
	<-lock
}

func createConfig() {
	if _, err := os.Stat(configDirectoryPath); os.IsNotExist(err) {
		err := os.Mkdir(configDirectoryPath, 0755)
		if err != nil {
			fmt.Println("Failed to create config directory")
			os.Exit(127)
		}
	}
	f, err := os.OpenFile(filepath.Join(configDirectoryPath, configFilename), os.O_CREATE|os.O_RDWR, 0644)
	config := Config{
		GracePeriod:     1800,
		WatchPaths:      []string{""},
		AutoLockTime:    300,
		ExtendTime:      300,
		RefreshInterval: 5,
	}
	if err != nil {
		fmt.Println("Failed to create config file")
		os.Exit(1)
	}
	encoder := toml.NewEncoder(f)
	err = encoder.Encode(config)
	if err != nil {
		fmt.Println("Failed to write config")
		os.Exit(1)
	}
	_ = f.Close()
	fmt.Println(fmt.Sprintf("Generated config at %s", filepath.Join(configDirectoryPath, configFilename)))
}
