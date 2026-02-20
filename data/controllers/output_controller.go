package controllers

import (
	"fmt"
	"log/slog"

	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type OutputController struct {
	cc *ChannelController
}

func NewOutputController(cc *ChannelController) *OutputController {
	return &OutputController{
		cc: cc,
	}
}

func (c *OutputController) Start() {
	slog.Info("OutputController started")
	go func() {
		for stuff := range c.cc.OutputChan {
			// Check if we received a Treatment update
			if treatment, ok := stuff.Object.(*models.Treatments); ok {
				slog.Info("OutputController: Treatment Updated")

				// Show final treatment result before reset
				slog.Info("Final Treatment:", "result", treatment.ToString())

				// Send reset signal to clear treatments
				c.cc.ResetChan <- true
				slog.Info("OutputController: Reset signal sent")

				slog.Debug("--------------------------------------------------")
				continue
			}

			// Fallback for direct object passing (if ever used)
			slog.Debug("OutputController received raw object", "object_type", fmt.Sprintf("%T", stuff.Object))
		}
	}()
}
