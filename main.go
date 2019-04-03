/**
 * File              : main.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 01.04.2019
 * Last Modified Date: 03.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package main

import (
    "log"
    "github.com/spf13/viper"
    "github.com/Tabrizian/SVOP/models"

    "github.com/Tabrizian/SVOP/utils"
)

func main() {
    var configuration models.Configuration
    viper.SetConfigName("config")
    viper.AddConfigPath("./configs/")
    viper.AddConfigPath("/etc/SVOP/")
    if err := viper.ReadInConfig(); err != nil {
        log.Fatalf("Error reading config file, %s", err)
    }
    err := viper.Unmarshal(&configuration)
    if err != nil {
        log.Fatalf("Error while unmarshaling the configuration file %s", err)
    }

    log.Print(configuration.Auth.Username)

    osClient, _ := utils.NewOpenStackClient(configuration.Auth)
    osClient.GetAuthToken()
    log.Print(osClient.NovaURL)

}
