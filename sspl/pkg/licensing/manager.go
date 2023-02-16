//go:build mutagensspl

// Copyright (c) 2022-present Mutagen IO, Inc.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the Server Side Public License, version 1, as published by
// MongoDB, Inc.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the Server Side Public License for more
// details.
//
// You should have received a copy of the Server Side Public License along with
// this program. If not, see
// <http://www.mongodb.com/licensing/server-side-public-license>.

package licensing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/state"
	"github.com/mutagen-io/mutagen/pkg/timeutil"
)

// managerRegistryLock serializes access to managerRegistry.
var managerRegistryLock sync.RWMutex

// managerRegistry maps product identifiers to their corresponding license
// manager instances.
var managerRegistry = map[string]*Manager{}

// RegisterManager registers a license manager in the global registry, making it
// the canonical source of license information for its associated product.
func RegisterManager(manager *Manager) {
	managerRegistryLock.Lock()
	managerRegistry[manager.product] = manager
	managerRegistryLock.Unlock()
}

// Manager implements license management for a single product.
type Manager struct {
	// logger is the logger for the license manager.
	logger *logging.Logger
	// product is the product identifier for the product that the license
	// manager is managing.
	product string
	// userAgent is the user agent to specify when performing API requests.
	userAgent string
	// tracker tracks changes to the license state.
	tracker *state.Tracker
	// stateLock guards and tracks changes to the license state.
	stateLock *state.TrackingLock
	// state is the current license state.
	state *State
	// stateInitialized tracks whether or not state initialization has reached a
	// point where the license information is usable.
	stateInitialized bool
	// keys is the channel used to submit API key changes to the run loop.
	keys chan<- string
	// runCancel cancels the run loop execution context.
	runCancel context.CancelFunc
	// runDone is closed when the run loop exits.
	runDone chan struct{}
}

// NewManager creates a new license manager for the specified product. By the
// time this function has returned, the Manager will have loaded any existing
// license information from disk. The specified user agent string will be used
// for API requests when requesting license tokens. If userAgent is empty, then
// a default value will be used.
func NewManager(logger *logging.Logger, product, userAgent string) (*Manager, error) {
	// Compute the storage path for the product.
	storagePath, err := pathToProductStorage(product)
	if err != nil {
		return nil, fmt.Errorf("unable to compute license storage path: %w", err)
	}

	// Create the state tracker and associated lock.
	tracker := state.NewTracker()
	stateLock := state.NewTrackingLock(tracker)

	// Create the API key handoff channel.
	keys := make(chan string)

	// Create a cancelable context to regulate the lifetime of the run loop.
	runCtx, runCancel := context.WithCancel(context.Background())

	// Create a channel that the run loop can use to signal completion.
	runDone := make(chan struct{})

	// Create the licensing manager.
	manager := &Manager{
		logger:    logger,
		product:   product,
		userAgent: userAgent,
		tracker:   tracker,
		stateLock: stateLock,
		state:     &State{},
		keys:      keys,
		runCancel: runCancel,
		runDone:   runDone,
	}

	// Start the run loop.
	go manager.run(runCtx, storagePath, keys)

	// Poll until the run loop has loaded existing license information. We also
	// watch for the case that the run loop has an early exit, in which case we
	// know that a warning will be set.
	var previousStateIndex uint64
	for {
		previousStateIndex, err = manager.tracker.WaitForChange(context.Background(), previousStateIndex)
		if err != nil {
			manager.Shutdown()
			return nil, fmt.Errorf("polling failed during initial license loading: %w", err)
		}
		manager.stateLock.Lock()
		initialized := manager.stateInitialized || manager.state.Warning != ""
		manager.stateLock.UnlockWithoutNotify()
		if initialized {
			break
		}
	}

	// Success.
	return manager, nil
}

// Shutdown gracefully terminates the license manager's operations.
func (m *Manager) Shutdown() {
	// Terminate the run loop context and wait for the run loop to exit. This
	// will also terminate state tracking.
	m.runCancel()
	<-m.runDone
}

// SetKey sets the API key for the license manager. Setting the value to an
// empty string will clear any existing license information.
func (m *Manager) SetKey(ctx context.Context, key string) error {
	select {
	case m.keys <- key:
		return nil
	case <-ctx.Done():
		return context.Canceled
	case <-m.runDone:
		return errors.New("license manager is shut down")
	}
}

// Poll polls for changes to the licensing state.
func (m *Manager) Poll(ctx context.Context, previousStateIndex uint64) (uint64, *State, error) {
	// Wait for a state change from the previous index.
	stateIndex, err := m.tracker.WaitForChange(ctx, previousStateIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("unable to track state changes: %w", err)
	}

	// Grab the state lock and defer its release.
	m.stateLock.Lock()
	defer m.stateLock.UnlockWithoutNotify()

	// Return the current state.
	return stateIndex, proto.Clone(m.state).(*State), nil
}

// run is the run loop entry point for license managers.
func (m *Manager) run(ctx context.Context, storagePath string, keys <-chan string) {
	// Defer closure of the termination signaling channel. We intentionally
	// avoid terminating state tracking here because we want pollers to still be
	// able to see the last state (with a warning) if the run loop exits.
	defer func() {
		m.logger.Info("License manager terminating")
		close(m.runDone)
	}()

	// Log startup.
	m.logger.Info("Licensing manager starting")

	// Compute storage paths.
	keyStoragePath := filepath.Join(storagePath, keyStorageName)
	licenseStoragePath := filepath.Join(storagePath, licenseStorageName)

	// Load any existing API key.
	var key string
	if t, err := os.ReadFile(keyStoragePath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			m.logger.Error("Unable to load existing key:", err)
			m.stateLock.Lock()
			m.state.Status = Status_Unlicensed
			m.state.Warning = fmt.Sprintf("Unable to load existing key: %v", err)
			m.stateLock.Unlock()
			return
		}
	} else if !utf8.Valid(t) {
		os.Remove(keyStoragePath)
	} else {
		key = string(t)
	}
	if key != "" {
		m.logger.Info("Found existing key")
	} else {
		m.logger.Info("No existing key found")
	}

	// If there's no valid API key, then any license token that exists should be
	// voided.
	if key == "" {
		os.Remove(licenseStoragePath)
	}

	// Load any existing license token.
	var licenseTokenExpiration time.Time
	if l, err := os.ReadFile(licenseStoragePath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			m.logger.Error("Unable to load existing license token:", err)
			m.stateLock.Lock()
			m.state.Status = Status_Unlicensed
			m.state.Warning = fmt.Sprintf("Unable to load existing license token: %v", err)
			m.stateLock.Unlock()
			return
		}
	} else if !utf8.Valid(l) {
		os.Remove(licenseStoragePath)
	} else if e, err := parseAndValidateLicenseToken(string(l), m.product); err != nil {
		os.Remove(licenseStoragePath)
	} else {
		licenseTokenExpiration = e
	}
	if !licenseTokenExpiration.IsZero() {
		m.logger.Info("Found existing license token")
	} else {
		m.logger.Info("No existing license token found")
	}

	// Create a utility function to clear the current licensing state.
	clearLicensingState := func() error {
		if err := os.Remove(keyStoragePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("unable to remove key storage: %w", err)
		} else if err = os.Remove(licenseStoragePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("unable to remove license token storage: %w", err)
		}
		key = ""
		licenseTokenExpiration = time.Time{}
		return nil
	}

	// Create a renewal timer, set to fire immediately, and defer its
	// termination.
	renewalTimer := time.NewTimer(0)
	defer renewalTimer.Stop()

	// Loop and manage the licensing state until cancellation.
	for {
		// Wait for an event that would change the licensing state.
		newToken := false
		select {
		case <-ctx.Done():
			return
		case <-renewalTimer.C:
			m.logger.Info("License renewal timer triggered")
		case k := <-keys:
			if k == "" {
				m.logger.Info("Key clearing request received")
			} else {
				m.logger.Info("Key update request received")
			}
			if err := clearLicensingState(); err != nil {
				m.logger.Error("Unable to clear licensing state on key update:", err)
				m.stateLock.Lock()
				m.state.Status = Status_Unlicensed
				m.state.Warning = fmt.Sprintf("Unable to clear licensing state on key update: %v", err)
				m.stateLock.Unlock()
				return
			}
			key = k
			newToken = key != ""
		}

		// Attempt a license renewal if we have an API token.
		keyInvalid := false
		if key != "" {
			m.logger.Info("Attempt to acquire license token")
			if t, err := getLicenseToken(m.product, key, m.userAgent); err != nil {
				m.logger.Warn("Unable to acquire license token:", err)
				if err == errInvalidKey || err == errNoSubscription || newToken {
					keyInvalid = true
				}
			} else if e, err := parseAndValidateLicenseToken(t, m.product); err != nil {
				// This is an unlikely path that indicates the server is
				// returning invalid or rapidly expiring tokens. We'll treat
				// this as an invalid token situation for now.
				m.logger.Warn("Received invalid license token:", err)
				keyInvalid = true
			} else {
				m.logger.Info("Received license token expiring", e.Format(time.RFC3339))
				licenseTokenExpiration = e
				if newToken {
					if err := os.WriteFile(keyStoragePath, []byte(key), 0600); err != nil {
						m.logger.Error("Unable to store key:", err)
						m.stateLock.Lock()
						m.state.Status = Status_Unlicensed
						m.state.Warning = fmt.Sprintf("Unable to store key: %v", err)
						m.stateLock.Unlock()
						return
					}
				}
				if err := os.WriteFile(licenseStoragePath, []byte(t), 0600); err != nil {
					m.logger.Error("Unable to store license token:", err)
					m.stateLock.Lock()
					m.state.Status = Status_Unlicensed
					m.state.Warning = fmt.Sprintf("Unable to store license token: %v", err)
					m.stateLock.Unlock()
					return
				}
			}
		}

		// If the API token was invalid, then clear the licensing state and fall
		// back to more basic licenses.
		if keyInvalid {
			m.logger.Warn("Clearing invalid key")
			if err := clearLicensingState(); err != nil {
				m.logger.Error("Unable to clear licensing state on invalid key:", err)
				m.stateLock.Lock()
				m.state.Status = Status_Unlicensed
				m.state.Warning = fmt.Sprintf("Unable to clear licensing state on invalid key: %v", err)
				m.stateLock.Unlock()
				return
			}
		}

		// Determine the best available license.
		now := time.Now()
		if now.Before(licenseTokenExpiration) {
			var warning string
			remainingTime := licenseTokenExpiration.Sub(now)
			if remainingTime < 48*time.Hour {
				warning = "Unable to renew license token, less than 48 hours remain"
			}
			m.stateLock.Lock()
			m.state.Status = Status_Licensed
			m.state.Warning = warning
			m.stateLock.Unlock()
			timeutil.StopAndDrainTimer(renewalTimer)
			renewalTime := 24 * time.Hour
			if remainingTime < renewalTime {
				renewalTime = remainingTime
			}
			renewalTimer.Reset(renewalTime)
		} else if !licenseTokenExpiration.IsZero() {
			m.stateLock.Lock()
			m.state.Status = Status_ValidKey
			m.state.Warning = "License token expired, unable to renew"
			m.stateLock.Unlock()
			timeutil.StopAndDrainTimer(renewalTimer)
			renewalTimer.Reset(24 * time.Hour)
		} else {
			m.stateLock.Lock()
			m.state.Status = Status_Unlicensed
			m.state.Warning = ""
			m.stateLock.Unlock()
			timeutil.StopAndDrainTimer(renewalTimer)
		}

		// After our first pass through the loop, mark the state as initialized.
		m.stateLock.Lock()
		if m.stateInitialized {
			m.stateLock.UnlockWithoutNotify()
		} else {
			m.stateInitialized = true
			m.stateLock.Unlock()
		}
	}
}
