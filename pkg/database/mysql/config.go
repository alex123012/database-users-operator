/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mysql

import (
	"fmt"
	"net/url"

	"github.com/xo/dburl"
)

type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DatabaseName string

	usersHostname string
}

func NewConfig(host string, port int, user, pass, dbname, usersHostname string) *Config {
	return &Config{
		Host:          host,
		User:          user,
		Password:      pass,
		Port:          port,
		DatabaseName:  dbname,
		usersHostname: usersHostname,
	}
}

func (c *Config) ConnString() (string, error) {
	userInfo := url.User(c.User)
	if c.Password != "" {
		userInfo = url.UserPassword(c.User, c.Password)
	}

	return dburl.GenMysql(&dburl.URL{
		URL: url.URL{
			Scheme:   "mysql",
			Host:     fmt.Sprintf("%s:%d", c.Host, c.Port),
			User:     userInfo,
			Path:     c.DatabaseName,
			RawQuery: "interpolateParams=true",
		},
		Transport: "tcp",
	})
}

func (c *Config) UsersHostname() string {
	if c.usersHostname == "" {
		return "*"
	}
	return c.usersHostname
}
