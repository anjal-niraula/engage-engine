// Package integration contains end-to-end and integration tests for the application.
//
// Integration tests verify that domain use cases, delivery handlers, and infrastructure
// implementations (PostgreSQL, MongoDB, Redis, Kafka) interact correctly against real
// or testcontainer-backed dependencies.
//
// Conventions:
//  1. Files in this directory should use the build tag:
//     //go:build integration
//  2. Run integration tests with:
//     go test -v -tags=integration ./tests/integration/...
package integration
