package controllers

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
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
			// Check if we received an OutputStuff update
			if output, ok := stuff.Object.(*models.OutputStuff); ok {
				slog.Info("OutputController: Output Updated")

				// Show final output result before reset
				result := output.ToString()
				slog.Info("Final Output:", "result", result)
				fmt.Printf("✅ [COMPLETED] %s\n", result)

				// 방금 입력한 트리거 문자열 길이만큼 백스페이스 눌러서 지우기
				if deleteSpace := output.GetDeleteHostring(); deleteSpace > 0 {
					for i := 0; i < deleteSpace; i++ {
						robotgo.KeyTap("backspace")
						time.Sleep(10 * time.Millisecond) // 백스페이스 연속 입력간 아주 짧은 딜레이
					}
					time.Sleep(50 * time.Millisecond) // 다 지우고 약간 대기
				}

				if chartText := output.GetChartText(); chartText != "" {
					if coord, exists := hotstrings.K_Coordinates["chartWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						// 키보드 입력을 통해 글자 입력 (로봇고 사용)
						robotgo.TypeStr(chartText)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if specificText := output.GetSpecificText(); specificText != "" {
					if coord, exists := hotstrings.K_Coordinates["specificWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						robotgo.KeyTap("end", "ctrl") // Ctrl + End 입력
						time.Sleep(50 * time.Millisecond)

						robotgo.TypeStr(specificText)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if orderCodes := output.GetOrderCode(); len(orderCodes) > 0 {
					if coord, exists := hotstrings.K_Coordinates["orderWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						// 먼저 Down 키 20번 입력
						for i := 0; i < 20; i++ {
							robotgo.KeyTap("down")
							time.Sleep(10 * time.Millisecond)
						}
						time.Sleep(50 * time.Millisecond)

						for _, code := range orderCodes {
							robotgo.TypeStr(code)
							time.Sleep(500 * time.Millisecond)
							robotgo.KeyTap("enter")
							time.Sleep(500 * time.Millisecond)
						}
						time.Sleep(50 * time.Millisecond)
					}
				}

				if mx999Text := output.GetMx999Text(); mx999Text != "" {
					if coord, exists := hotstrings.K_Coordinates["mx999Window"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이
						robotgo.TypeStr(mx999Text)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if memoText := output.GetMemoText(); memoText != "" {
					if coord, exists := hotstrings.K_Coordinates["memoWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이
						robotgo.TypeStr(memoText)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if simpleText := output.GetSimpleText(); simpleText != "" {
					// 윈도우 클릭 없이 바로 입력
					robotgo.TypeStr(simpleText)
					time.Sleep(50 * time.Millisecond)
				}

				// Send reset signal to clear
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
