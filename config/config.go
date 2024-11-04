package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	MongoIPAddress string `json:"mongo_ip"`
	MongoPort      int    `json:"mongo_port"`
	MongoUsername  string `json:"mongo_user"`
	MongoPassword  string `json:"mongo_password"`
}

func (c Config) BuildUri() string {
	s := []string{"mongodb://", c.MongoUsername, ":", c.MongoPassword, "@", c.MongoIPAddress, ":", strconv.Itoa(c.MongoPort)}
	return strings.Join(s, "")
}

func Parse(path string) Config { // convert to abs path
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

	var ret Config
	errors := json.Unmarshal([]byte(bytes), &ret)
	if errors != nil {
		fmt.Println(errors.Error())
		panic(1)
	}

	return ret
}

func ParseFromEnv() Config {
	var ret Config

	var envs [4]string = [4]string{"MONGO_IP", "MONGO_PORT", "MONGO_USER", "MONGO_PASS"}
	for index, env := range envs {
		env_set, exists := os.LookupEnv(env)
		if !exists {
			fmt.Printf("[error] Failed to find %d of %d environmental variables: %s\n", index, len(envs), env)
			panic(1)
		}

		if env == "MONGO_IP" {
			ret.MongoIPAddress = env_set
		} else if env == "MONGO_PORT" {
			port, err := strconv.Atoi(env_set)
			if err != nil {
				fmt.Println("[error] Failed to convert Mongo Port to string")
				panic(1)
			}
			ret.MongoPort = port
		} else if env == "MONGO_USER" {
			ret.MongoUsername = env_set
		} else if env == "MONGO_PASS" {
			ret.MongoPassword = env_set
		}
	}

	return ret
}
