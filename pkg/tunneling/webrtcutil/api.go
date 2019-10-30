package webrtcutil

import (
	"github.com/pion/webrtc/v2"
)

// API is a shared global WebRTC API for tunnels, preconfigured to enable data
// channel stream detaching.
var API *webrtc.API

func init() {
	// Configure WebRTC behavior.
	settings := webrtc.SettingEngine{}
	settings.DetachDataChannels()

	// Create the WebRTC API instance.
	API = webrtc.NewAPI(webrtc.WithSettingEngine(settings))
}
