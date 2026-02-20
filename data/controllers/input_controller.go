package controllers

import (
	"log/slog"

	"github.com/eiannone/keyboard"

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

	go func() {
		// Open keyboard connection
		if err := keyboard.Open(); err != nil {
			slog.Error("Failed to open keyboard", "error", err)
			return
		}
		defer func() {
			_ = keyboard.Close()
		}()

		for {
			r, key, err := keyboard.GetKey()
			if err != nil {
				slog.Error("Error reading input", "error", err)
				break
			}

			// Quit on Ctrl+C or Esc
			if key == keyboard.KeyEsc || key == keyboard.KeyCtrlC {
				slog.Info("Exiting InputController...")
				return
			}

			// Simulation: Ctrl+Enter (Checking for rune 10 which is LF, often produced by Ctrl+Enter in terminals)
			// Or if user presses Enter (rune 13/KeyEnter), we might treat it as flush if desired.
			// User asked for Ctrl+Enter. Let's try rune 10.
			if r == 10 || (key == keyboard.KeyEnter && r == 10) {
				slog.Info("Ctrl+Enter detected - Flushing treatments")
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"FlushTreatments",
					true,
					nil,
				)
				continue
			}

			// Also support Ctrl+Space as alternate flush if Ctrl+Enter is tricky
			if key == keyboard.KeySpace && r == 0 { // Ctrl+Space often yields rune 0
				slog.Info("Ctrl+Space detected - Flushing treatments")
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"FlushTreatments",
					true,
					nil,
				)
				continue
			}

			// Ignore normal Enter/Return for now unless it's explicitly needed
			// But wait, if user types Enter, maybe they want a newline in output?
			// For hotstrings, usually we ignore whitespace or treat as separator.
			if key == keyboard.KeyEnter || r == 13 {
				continue
			}

			// Normal character input
			if r != 0 {
				char := string(r)
				// Create ChannelStuff for normal input
				stuff := models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"InputReceived",
					false,
					char,
				)
				c.cc.InputChan <- stuff
			}
		}
	}()
}
