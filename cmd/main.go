package main

import (
	"edutalks/internal/app"
	"edutalks/internal/config"
	"fmt"
	"log"
	"net/http"
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

	fmt.Println("Сервер запущен на порту", cfg.Port)
	http.ListenAndServe(":"+cfg.Port, router)
}
