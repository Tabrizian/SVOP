/**
 * File              : main.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 01.04.2019
 * Last Modified Date: 01.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package main

import (
	"log"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath("./configs/")
	viper.AddConfigPath("/etc/SVOP/")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	log.Print(viper.Get("hello"))
}
