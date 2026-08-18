// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"regexp"
	"time"
)

// Validation patterns, kept identical to the ones terraform-api enforces. The
// duplication is deliberate: catching a bad value at plan time is worth far
// more to a practitioner than the same rejection halfway through an apply.
var (
	machineNamePattern     = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	networkPattern         = regexp.MustCompile(`^(hostonly|bridge|nat|intnet)(,(hostonly|bridge|nat|intnet)){0,3}$`)
	serviceNamePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _-]{0,62}$`)
	provisionerNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	paramNamePattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	portSpecPattern        = regexp.MustCompile(`^\d{1,5}(-\d{1,5})?$`)
	ipv4Pattern            = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	paramRIDPattern        = regexp.MustCompile(`^param:\d+$`)
	secretRIDPattern       = regexp.MustCompile(`^secret:\d+$`)
	networkRIDPattern      = regexp.MustCompile(`^cs_net:\d+$`)
	storeRIDPattern        = regexp.MustCompile(`^store:\d+$`)
)

// Default timeouts. They are the ceiling for the whole operation, including
// the server-side wait, and are sized for what the work actually takes: a
// machine boots a full guest image, a container service boots several and then
// forms a Swarm cluster on top.
const (
	defaultMachineCreateTimeout = 30 * time.Minute
	defaultMachineUpdateTimeout = 15 * time.Minute
	defaultMachineDeleteTimeout = 15 * time.Minute

	defaultServiceCreateTimeout = 60 * time.Minute
	defaultServiceUpdateTimeout = 30 * time.Minute
	defaultServiceDeleteTimeout = 30 * time.Minute

	defaultStorageTimeout = 10 * time.Minute
)
