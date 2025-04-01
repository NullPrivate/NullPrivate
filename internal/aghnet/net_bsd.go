//go:build darwin || freebsd || openbsd

package aghnet

import "github.com/AdGuardPrivate/AdGuardPrivate/internal/aghos"

func canBindPrivilegedPorts() (can bool, err error) {
	return aghos.HaveAdminRights()
}
