package main

import (
	"fmt"
	"net/http"

	"tinygo.org/x/bluetooth"
)

// Only hardware operations cross this boundary. Tests supply an adapter without
// Bluetooth hardware or system D-Bus access.
type scanner interface {
	Enable() error
	Scan(func(*bluetooth.Adapter, bluetooth.ScanResult)) error
}

// run returns when scanning or HTTP serving ends. The executable then exits;
// retrying and coordinated graceful shutdown remain future work.
func run(adapter scanner, exporter *exporter, handler http.Handler, serve func(http.Handler) error) error {
	if err := adapter.Enable(); err != nil {
		return fmt.Errorf("enable BLE stack: %w", err)
	}
	results := make(chan error, 2)
	go func() {
		err := serve(handler)
		if err != nil {
			err = fmt.Errorf("serve metrics: %w", err)
		}
		results <- err
	}()
	go func() {
		err := adapter.Scan(exporter.handleAdvertisement)
		if err != nil {
			err = fmt.Errorf("scan sensors: %w", err)
		}
		results <- err
	}()
	return <-results
}
