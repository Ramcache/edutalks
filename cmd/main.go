// @title Edutalks API
// @version 1.0
// @description Документация API Edutalks (регистрация, логин, токены и т.д.).
// @host localhost:8080
// @BasePath /
package main

import (
	_ "edutalks/docs"
	"edutalks/internal/app"
	"edutalks/internal/config"
	"fmt"
	"log"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Ошибка загрузки конфига:", err)
	}

	router, err := app.InitApp(cfg)
	if err != nil {
		log.Fatal("Ошибка инициализации приложения:", err)
	}
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	fmt.Println("Сервер запущен на порту", cfg.Port)
	http.ListenAndServe(":"+cfg.Port, router)
}
