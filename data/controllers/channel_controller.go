package controllers

import (
	"log/slog"
	"sync/atomic"

	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type ChannelController struct {
	// InputController sends data here
	InputChan chan *models.ChannelStuff

	// HotstringController sends data here, intended for OutputController
	OutputChan chan *models.ChannelStuff

	// Reset signal for treatments
	ResetChan chan bool

	// TypeStr 출력 중 hotstring 버퍼 입력 차단 플래그
	IsMuting atomic.Bool
}

func NewChannelController() *ChannelController {
	slog.Debug("Creating ChannelController")
	return &ChannelController{
		InputChan:  make(chan *models.ChannelStuff, 100),
		OutputChan: make(chan *models.ChannelStuff, 100),
		ResetChan:  make(chan bool, 10),
	}
}
