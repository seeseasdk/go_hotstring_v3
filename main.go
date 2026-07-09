package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/seeseasdk/go_hotstring_v3/data/controllers"
	"github.com/seeseasdk/go_hotstring_v3/utils/prettylog"
)

func main() {
	logger := prettylog.InitLogger()
	logger.Info("Application started")

	// Initialize Channel Controller (The Bus)
	cc := controllers.NewChannelController()

	// Initialize Controllers
	outputController := controllers.NewOutputController(cc)
	hotstringController := controllers.NewHotstringController(cc)
	inputController := controllers.NewInputController(cc)

	// Start Controllers
	// Start Output first to be ready to receive
	outputController.Start()
	// Start Hotstring logic
	hotstringController.Start()
	// Start Input last (starts listening) - 고루틴으로 실행
	go inputController.Start()

	// Block until signal
	logger.Info("System ready. Waiting for input (Simulated via Stdin)...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Application finished")
}
