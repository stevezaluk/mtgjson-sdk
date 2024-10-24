package server

import (
	"fmt"
	"os"
	"errors"
	"io"
	"encoding/json"
	"strings"
	"strconv"
)

type Config struct {
	MongoIPAddress string
	MongoPort int

	MongoUsername string
	MongoPassword string
}

func (c* Config) Parse(path string) { // convert to abs path
	if strings.HasPrefix(path, "~") {
		home := os.Getenv("HOME")
		path = strings.ReplaceAll(path, "~", home)
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("[error] Failed to find config file: ", path)
		panic(1)
	}

	if errors.Is(err, os.ErrPermission) {
		fmt.Println("[error] Invalid permissions for open the config file: ", path)
		panic(1)
	}

	bytes, _ := io.ReadAll(file)

	errors := json.Unmarshal([]byte(bytes), &c)
	if errors != nil {
		fmt.Println(errors.Error())
		panic(1)
	}
}

func (c* Config) ParseFromEnv() {
	var envs [4]string = [4]string{"MONGO_IP", "MONGO_PORT", "MONGO_USER", "MONGO_PASS"}
	for index, env := range envs {
		env_set, exists := os.LookupEnv(env)
		if !exists {
			fmt.Printf("[error] Failed to find %d of %d environmental variables: %s\n", index, len(envs), env)
			panic(1)
		}

		if env == "MONGO_IP" {
			c.MongoIPAddress = env_set
		} else if env == "MONGO_PORT" {
			port, err := strconv.Atoi(env_set)
			if (err != nil) {
				fmt.Println("[error] Failed to convert Mongo Port to string")
				panic(1)
			}
			c.MongoPort = port
		} else if env == "MONGO_USER" {
			c.MongoUsername = env_set
		} else if env == "MONGO_PASS" {
			c.MongoPassword = env_set
		}
	}
}

func (c Config) BuildUri() (string) {
	s := []string{"mongodb://", c.MongoUsername, ":", c.MongoPassword, "@", c.MongoIPAddress, ":", strconv.Itoa(c.MongoPort)}
	return strings.Join(s, "")
}