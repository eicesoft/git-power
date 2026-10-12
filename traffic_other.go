//go:build !darwin

package main

import "context"

// centerTrafficLights 在非 darwin 平台上无操作。
func centerTrafficLights(_ context.Context) {}
