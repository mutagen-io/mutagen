package tunneling

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/pion/webrtc/v2"

	"github.com/mutagen-io/mutagen/pkg/tunneling/webrtcutil"
)

// webrtcAPIOnce restricts the global WebRTC API to a single initialization.
var webrtcAPIOnce sync.Once

// webrtcAPI is a shared global WebRTC API for tunnels, preconfigured to enable
// data channel stream detaching. It is lazily initialized and should be
// accessed using the loadWebRTCAPI function.
var webrtcAPI *webrtc.API

// webrtcAPIError is any error that occurs during the initialization of the
// global WebRTC API.
var webrtcAPIError error

// loadWebRTCAPI performs lazy allocation of the global WebRTC API for tunnels.
func loadWebRTCAPI() (*webrtc.API, error) {
	// Perform initialization once.
	webrtcAPIOnce.Do(func() {
		// Creating the setting engine.
		settings := webrtc.SettingEngine{}

		// Set up logging.
		settings.LoggerFactory = webrtcutil.NewLoggerFactory("webrtc")

		// Enable data channel detaching.
		settings.DetachDataChannels()

		// Check if a UDP port range has been specified in the environment and
		// enforce that any specification is not one-sided. If a UDP port range
		// has been specified, then parse the ports and set the range in the
		// setting engine.
		envMinimumPort := os.Getenv("MUTAGEN_TUNNEL_UDP_PORT_MINIMUM")
		envMaximumPort := os.Getenv("MUTAGEN_TUNNEL_UDP_PORT_MAXIMUM")
		if envMinimumPort != "" && envMaximumPort == "" {
			webrtcAPIError = errors.New("minimum port specified in environment without maximum port")
			return
		} else if envMaximumPort != "" && envMinimumPort == "" {
			webrtcAPIError = errors.New("maximum port specified in environment without minimum port")
			return
		}
		if envMinimumPort != "" {
			// Parse ports.
			var minimumPort, maximumPort uint16
			if port64, err := strconv.ParseUint(envMinimumPort, 10, 16); err != nil {
				webrtcAPIError = fmt.Errorf("invalid minimum port value specified in environment: %w", err)
				return
			} else {
				minimumPort = uint16(port64)
			}
			if port64, err := strconv.ParseUint(envMaximumPort, 10, 16); err != nil {
				webrtcAPIError = fmt.Errorf("invalid maximum port value specified in environment: %w", err)
				return
			} else {
				maximumPort = uint16(port64)
			}

			// Sanity check the port range.
			if maximumPort < minimumPort {
				webrtcAPIError = errors.New("maximum port specified in environment is less than minimum port specified in environment")
				return
			} else if minimumPort < 49152 {
				webrtcAPIError = errors.New("minimum port specified in environment falls below ephemeral port range ([49152, 65535])")
				return
			}

			// Set the port range.
			if err := settings.SetEphemeralUDPPortRange(minimumPort, maximumPort); err != nil {
				webrtcAPIError = fmt.Errorf("unable to set UDP port range: %w", err)
				return
			}
		}

		// Create the WebRTC API instance.
		webrtcAPI = webrtc.NewAPI(webrtc.WithSettingEngine(settings))
	})

	// Done.
	return webrtcAPI, webrtcAPIError
}
