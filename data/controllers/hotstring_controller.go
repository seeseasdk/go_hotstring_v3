package controllers

import (
	"log/slog"
	"strings"

	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type HotstringController struct {
	cc         *ChannelController
	buffer     string
	treatments *models.Treatments
}

func NewHotstringController(cc *ChannelController) *HotstringController {
	return &HotstringController{
		cc:         cc,
		buffer:     "",
		treatments: models.NewTreatments(),
	}
}

func (c *HotstringController) Start() {
	slog.Info("HotstringController started")
	go func() {
		for {
			select {
			case stuff := <-c.cc.InputChan:
				if stuff.Do == "FlushTreatments" {
					slog.Info("HotstringController: Flushing treatments (Processing Buffer)")
					// Trigger processing of the accumulated buffer
					c.processBuffer()

					// Then output
					c.cc.OutputChan <- models.NewChannelStuff("HotstringController", "OutputController", "UpdateTreatment", true, c.treatments)

					// Clear buffer after processing
					c.buffer = ""
					continue
				}

				// Extract key from input
				char, ok := stuff.Object.(string)
				if !ok {
					continue
				}

				// Only accumulate buffer, do not process yet
				c.buffer += char

			case <-c.cc.ResetChan:
				// Reset treatments when output is complete
				c.treatments = models.NewTreatments()
				slog.Debug("HotstringController: Treatments reset")
			}
		}
	}()
}

// processBuffer scans the buffer and extracts matches sequentially
func (c *HotstringController) processBuffer() {
	// Loop to process buffer for multiple matches
	matched := true
	for matched {
		matched = false

		// 1. K_ESWT_ONLY (Prefix 'e')
		for k, v := range hotstrings.K_ESWT_ONLY {
			target := "e" + k
			if strings.HasSuffix(c.buffer, target) {
				slog.Debug("Hotstring Triggered (ESWT)", "trigger", target)
				c.treatments.SetESWT(*v)
				c.buffer = c.buffer[:len(c.buffer)-len(target)]
				matched = true
				goto NextLoop
			}
		}

		// 2. K_FirstMeeting (Prefix 'z')
		for k, v := range hotstrings.K_FirstMeeting {
			target := "z" + k
			if strings.HasSuffix(c.buffer, target) {
				slog.Debug("Hotstring Triggered (FirstMeeting)", "trigger", target)
				c.treatments.SetAddExtraTreatments(v)
				c.buffer = c.buffer[:len(c.buffer)-len(target)]
				matched = true
				goto NextLoop
			}
		}

		// 3. K_Xrays (Prefix 'x')
		for k, v := range hotstrings.K_Xrays {
			target := "x" + k
			if strings.HasSuffix(c.buffer, target) {
				slog.Debug("Hotstring Triggered (Xrays)", "trigger", target)
				c.treatments.SetAddExtraTreatments(v)
				c.buffer = c.buffer[:len(c.buffer)-len(target)]
				matched = true
				goto NextLoop
			}
		}

		// 4. K_Sonos (Prefix 's')
		for k, v := range hotstrings.K_Sonos {
			target := "s" + k
			if strings.HasSuffix(c.buffer, target) {
				slog.Debug("Hotstring Triggered (Sonos)", "trigger", target)
				c.treatments.SetAddExtraTreatments(v)
				c.buffer = c.buffer[:len(c.buffer)-len(target)]
				matched = true
				goto NextLoop
			}
		}

		// 5. K_Blocks (Direct keys and Prefix 'c' or 'p')
		// First check direct keys in K_Blocks
		for k, v := range hotstrings.K_Blocks {
			// Exact match check requires suffix check, but since we are processing backwards from end of string usually?
			// Wait, the logic is: buffer builds up "clm4bshr".
			// If we process from end (HasSuffix), then "shr" matches first.
			// buffer becomes "clm4b".
			// Then "clm4b" (prefixed match) matches.
			// buffer becomes "".
			// This order (LIFO-like extraction from end) works for sequential inputs if the components are distinct enough.

			if strings.HasSuffix(c.buffer, k) {
				slog.Debug("Hotstring Triggered (Blocks-Direct)", "trigger", k, "site", v.GetSite())
				c.treatments.SetAddInjection(*v)
				c.buffer = c.buffer[:len(c.buffer)-len(k)]
				matched = true
				goto NextLoop
			}
		}

		// Then check prefix-based matches
		for k, v := range hotstrings.K_Blocks {
			target_c := "c" + k
			target_p := "p" + k
			if strings.HasSuffix(c.buffer, target_c) {
				slog.Debug("Hotstring Triggered (Blocks-C)", "trigger", target_c, "site", v.GetSite())
				c.treatments.SetAddInjection(*v)
				c.buffer = c.buffer[:len(c.buffer)-len(target_c)]
				matched = true
				goto NextLoop
			} else if strings.HasSuffix(c.buffer, target_p) {
				slog.Debug("Hotstring Triggered (Blocks-P)", "trigger", target_p, "site", v.GetSite())
				c.treatments.SetAddInjection(*v)
				c.buffer = c.buffer[:len(c.buffer)-len(target_p)]
				matched = true
				goto NextLoop
			}
		}

		// 6. K_Simples (Others)
		for k, v := range hotstrings.K_Simples {
			if strings.HasSuffix(c.buffer, k) {
				slog.Debug("Hotstring Triggered (Simples)", "trigger", k)
				c.treatments.SetAddExtraTreatments(v)
				c.buffer = c.buffer[:len(c.buffer)-len(k)]
				matched = true
				goto NextLoop
			}
		}
	NextLoop:
	}
}
