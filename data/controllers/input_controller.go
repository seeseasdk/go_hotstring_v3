package controllers

import (
	"bufio"
	"log/slog"
	"os"

	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type InputController struct {
	cc *ChannelController
}

func NewInputController(cc *ChannelController) *InputController {
	return &InputController{
		cc: cc,
	}
}

func (c *InputController) Start() {
	slog.Info("InputController started. Listening for input...")

	// Simulation: Reading from Stdin for now since no keyboard hook library is configured
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				slog.Error("Error reading input", "error", err)
				break
			}

			// Ignore newlines for cleaner testing
			char := string(r)
			if char == "\n" || char == "\r" {
				continue
			}

			slog.Debug("Input received", "char", char)

			// Create ChannelStuff
			stuff := models.NewChannelStuff(
				"InputController",
				"HotstringController",
				"InputReceived",
				false,
				char,
			)

			// Send to ChannelController
			c.cc.InputChan <- stuff
		}
	}()
}
