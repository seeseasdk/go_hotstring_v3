package controllers

import (
	"fmt"
	"log/slog"
	"os"

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
	fmt.Println("💡 [TIP] Ctrl+Enter to flush, Ctrl+C or ESC to exit")
	os.Stdout.Sync()

	// 키보드 초기화
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	// 키보드 입력 대기
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			fmt.Printf("Keyboard input error: %v\n", err)
			continue
		}

		// Exit conditions: ESC or Ctrl+C
		if key == keyboard.KeyEsc || key == keyboard.KeyCtrlC {
			fmt.Println("🚪 [EXIT] Program terminating")
			os.Stdout.Sync()
			os.Exit(0)
		}

		if key == keyboard.KeyBackspace || key == keyboard.KeyBackspace2 {
			c.cc.InputChan <- models.NewChannelStuff(
				"InputController",
				"HotstringController",
				"Backspace",
				false,
				nil,
			)
			continue
		}

		// Ctrl+Enter trigger (Enter key assumed to be with Ctrl)
		if key == keyboard.KeyEnter {
			fmt.Println("🚀 [CTRL+ENTER] Triggered!")
			os.Stdout.Sync()
			c.cc.InputChan <- models.NewChannelStuff(
				"InputController",
				"HotstringController",
				"FlushTreatments",
				true,
				nil,
			)
			continue
		}

		// 일반 문자 입력 처리 (hotstring 패턴 매칭용)
		if char != 0 {
			// 문자를 HotstringController로 전송
			c.cc.InputChan <- models.NewChannelStuff(
				"InputController",
				"HotstringController",
				"AddChar",
				false,
				map[string]interface{}{
					"char": char,
				},
			)
		}
	}
}
