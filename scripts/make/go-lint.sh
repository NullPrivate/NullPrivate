#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 14

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '0' ]; then
	set -x
fi

# Set $EXIT_ON_ERROR to zero to see all errors.
if [ "${EXIT_ON_ERROR:-1}" -eq '0' ]; then
	set +e
else
	set -e
fi

set -f -u

# Source the common helpers, including not_found and run_linter.
. ./scripts/make/helper.sh

# Simple analyzers

# blocklist_imports is a simple check against unwanted packages.  The following
# packages are banned:
#
#   *  Package errors is replaced by our own package in the
#   github.com/AdguardTeam/golibs module.
#
#   *  Packages log and github.com/AdguardTeam/golibs/log are replaced by
#      stdlib's new package log/slog and AdGuard's new utilities package
#      github.com/AdguardTeam/golibs/logutil/slogutil.
#
#   *  Package github.com/prometheus/client_golang/prometheus/promauto is not
#      recommended, as it encourages reliance on global state.
#
#   *  Packages golang.org/x/exp/maps, golang.org/x/exp/slices, and
#      golang.org/x/net/context have been moved into stdlib.
#
#   *  Package io/ioutil is soft-deprecated.
#
#   *  Package reflect is often an overkill, and for deep comparisons there are
#      much better functions in module github.com/google/go-cmp.  Which is
#      already our indirect dependency and which may or may not enter the stdlib
#      at some point.
#
#      See https://github.com/golang/go/issues/45200.
#
#   *  Package sort is replaced by package slices.
#
#   *  Package unsafe is… unsafe.
#
# Currently, the only standard exception are files generated from protobuf
# schemas, which use package reflect.  If your project needs more exceptions,
# add and document them.
#
# NOTE:  Flag -H for grep is non-POSIX but all of Busybox, GNU, macOS, and
# OpenBSD support it.
#
# NOTE:  Exclude the security_windows.go, because it requires unsafe for the OS
# APIs.
#
# TODO(a.garipov): Add golibs/log.
blocklist_imports() {
	find . \
		-type 'f' \
		-name '*.go' \
		'!' '(' \
		-name '*.pb.go' \
		-o -path './internal/permcheck/security_windows.go' \
		')' \
		-exec \
		'grep' \
		'-H' \
		'-e' '[[:space:]]"errors"$' \
		'-e' '[[:space:]]"github.com/prometheus/client_golang/prometheus/promauto"$' \
		'-e' '[[:space:]]"golang.org/x/exp/maps"$' \
		'-e' '[[:space:]]"golang.org/x/exp/slices"$' \
		'-e' '[[:space:]]"golang.org/x/net/context"$' \
		'-e' '[[:space:]]"io/ioutil"$' \
		'-e' '[[:space:]]"log"$' \
		'-e' '[[:space:]]"reflect"$' \
		'-e' '[[:space:]]"sort"$' \
		'-e' '[[:space:]]"unsafe"$' \
		'-n' \
		'{}' \
		';'
}

# method_const is a simple check against the usage of some raw strings and
# numbers where one should use named constants.
method_const() {
	find . \
		-type 'f' \
		-name '*.go' \
		-exec \
		'grep' \
		'-H' \
		'-e' '"DELETE"' \
		'-e' '"GET"' \
		'-e' '"PATCH"' \
		'-e' '"POST"' \
		'-e' '"PUT"' \
		'-n' \
		'{}' \
		';'
}

# underscores is a simple check against Go filenames with underscores.  Add new
# build tags and OS as you go.  The main goal of this check is to discourage the
# use of filenames like client_manager.go.
underscores() {
	underscore_files="$(
		find . \
			-type 'f' \
			-name '*_*.go' \
			'!' '(' -name '*_bsd.go' \
			-o -name '*_darwin.go' \
			-o -name '*_freebsd.go' \
			-o -name '*_generate.go' \
			-o -name '*_linux.go' \
			-o -name '*_next.go' \
			-o -name '*_openbsd.go' \
			-o -name '*_others.go' \
			-o -name '*_test.go' \
			-o -name '*_unix.go' \
			-o -name '*_windows.go' \
			')' \
			-exec 'printf' '\t%s\n' '{}' ';'
	)"
	readonly underscore_files

	if [ "$underscore_files" != '' ]; then
		printf \
			'found file names with underscores:\n%s\n' \
			"$underscore_files"
	fi
}

# TODO(a.garipov): Add an analyzer to look for `fallthrough`, `goto`, and `new`?

# govulncheck_filtered runs govulncheck and ignores advisories that currently
# have no released fixed version.  Keep the allowlist narrow so that new
# vulnerabilities still fail CI.
govulncheck_filtered() {
	output="$(govulncheck ./...)"
	exitcode="$?"

	if [ "$exitcode" -eq '0' ]; then
		[ "$output" = '' ] || printf '%s\n' "$output"

		return '0'
	fi

	advisory_ids="$(
		printf '%s\n' "$output" \
			| sed -n 's/^Vulnerability #[0-9][0-9]*: \(GO-[0-9-]*\)$/\1/p' \
			| sort -u
	)"
	readonly advisory_ids

	known_unfixed='GO-2026-4923'
	readonly known_unfixed

	unexpected_ids="$(printf '%s\n' "$advisory_ids" | grep -F -x -v "$known_unfixed")"

	if [ "$advisory_ids" = '' ] || [ "$unexpected_ids" != '' ]; then
		printf '%s\n' "$output"

		return "$exitcode"
	fi

	printf '%s\n' \
		"ignoring known unfixed advisory ${known_unfixed} for go.etcd.io/bbolt" \
		"see https://pkg.go.dev/vuln/${known_unfixed}" \
		"upstream fix exists, but no released fixed version is available yet"
}

# Checks

run_linter -e blocklist_imports

run_linter -e method_const

run_linter -e underscores

run_linter -e gofumpt --extra -e -l .

run_linter "${GO:-go}" vet ./...

run_linter govulncheck_filtered

# 将 dnsforward 和 home 的圈复杂度阈值适度放宽，其它目录仍保持 10。
run_linter gocyclo --over 10 -ignore '^internal/(dnsforward|home)/' .
run_linter gocyclo --over 15 ./internal/home/
run_linter gocyclo --over 15 ./internal/dnsforward/

# TODO(a.garipov): Enable 10 for all.
run_linter gocognit --over='20' \
	./internal/querylog/ \
	;

run_linter gocognit --over='19' \
	./internal/home/ \
	;

run_linter gocognit --over='18' \
	./internal/aghtls/ \
	;

run_linter gocognit --over='15' \
	./internal/aghos/ \
	./internal/filtering/ \
	;

run_linter gocognit --over='14' \
    ./internal/dhcpd \
    ;

# 对 dnsforward 适度放宽认知复杂度阈值
run_linter gocognit --over='18' \
    ./internal/dnsforward/ \
    ;

run_linter gocognit --over='13' \
	./internal/aghnet/ \
	;

run_linter gocognit --over='12' \
	./internal/filtering/rewrite/ \
	;

run_linter gocognit --over='11' \
	./internal/updater/ \
	;

run_linter gocognit --over='10' \
    ./internal/aghalg/ \
    ./internal/aghhttp/ \
    ./internal/aghrenameio/ \
    ./internal/aghtest/ \
    ./internal/aghuser/ \
	./internal/arpdb/ \
	./internal/client/ \
	./internal/configmigrate/ \
    ./internal/dhcpsvc \
    ./internal/filtering/hashprefix/ \
    ./internal/filtering/rulelist/ \
    ./internal/filtering/safesearch/ \
    ./internal/ipset \
    ./internal/next/ \
	./internal/rdns/ \
	./internal/schedule/ \
	./internal/stats/ \
	./internal/version/ \
	./internal/whois/ \
	./scripts/ \
	;

run_linter ineffassign ./...

run_linter unparam ./...

find . \
	'(' \
	-name 'node_modules' \
	-type 'd' \
	-prune \
	')' \
	-o \
	-type 'f' \
	'(' \
	-name 'Makefile' \
	-o -name '*.conf' \
	-o -name '*.go' \
	-o -name '*.mod' \
	-o -name '*.sh' \
	-o -name '*.yaml' \
	-o -name '*.yml' \
	')' \
	-exec 'misspell' '--error' '{}' '+'

if [ "${SKIP_HEAVY_LINTERS:-0}" -eq '0' ]; then
    run_linter nilness ./...

    # TODO(a.garipov): Enable for all.
    run_linter fieldalignment \
		./internal/aghalg/ \
		./internal/aghhttp/ \
		./internal/aghos/ \
		./internal/aghrenameio/ \
		./internal/aghtest/ \
		./internal/aghtls/ \
		./internal/aghuser/ \
		./internal/arpdb/ \
		./internal/client/ \
		./internal/configmigrate/ \
		./internal/dhcpsvc/ \
		./internal/filtering/hashprefix/ \
		./internal/filtering/rewrite/ \
		./internal/filtering/rulelist/ \
		./internal/filtering/safesearch/ \
		./internal/ipset/ \
		./internal/next/... \
		./internal/querylog/ \
		./internal/rdns/ \
		./internal/schedule/ \
		./internal/stats/ \
		./internal/updater/ \
		./internal/version/ \
		./internal/whois/ \
		;

    run_linter -e shadow --strict ./...

    # TODO(a.garipov): Enable for all.
    # TODO(e.burkov):  Re-enable G115.
    run_linter gosec --exclude G115 --quiet \
		./internal/aghalg/ \
		./internal/aghhttp/ \
		./internal/aghnet/ \
		./internal/aghos/ \
		./internal/aghrenameio/ \
		./internal/aghtest/ \
		./internal/aghuser/ \
		./internal/arpdb/ \
		./internal/client/ \
		./internal/configmigrate/ \
		./internal/dhcpd/ \
		./internal/dhcpsvc/ \
		./internal/dnsforward/ \
		./internal/filtering/hashprefix/ \
		./internal/filtering/rewrite/ \
		./internal/filtering/rulelist/ \
		./internal/filtering/safesearch/ \
		./internal/ipset/ \
		./internal/next/ \
		./internal/rdns/ \
		./internal/schedule/ \
		./internal/stats/ \
		./internal/version/ \
		./internal/whois/ \
		;

    run_linter errcheck ./...
else
    printf 'skip heavy linters: nilness/fieldalignment/shadow/gosec/errcheck\n' 1>&2
fi

# staticcheck 构建与分析较重；仅在显式跳过时不运行（SKIP_HEAVY_LINTERS=1）。
if [ "${SKIP_HEAVY_LINTERS:-0}" -eq '0' ]; then
	staticcheck_matrix='
linux:   GOOS=linux
'
	readonly staticcheck_matrix

	printf '%s' "$staticcheck_matrix" | run_linter staticcheck --matrix ./...
fi
