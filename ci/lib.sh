# Copyright (c) 2022-2024 The authors (see the AUTHORS file)
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

# Shared helpers for the CI scripts. Sourced, not executed.
#
# Everything CI does lives in ci/, so that CircleCI and GitHub Actions run the
# same checks rather than two overlapping subsets of them.

# REPO_ROOT is derived from this file's location, so the scripts behave the same
# whatever the working directory.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT

log() { printf '\n=== %s\n' "$*" >&2; }

die() { printf 'error: %s\n' "$*" >&2; exit 1; }

# require_tool reports every missing tool at once, rather than failing on the
# first one twenty lines into a build.
require_tool() {
    local missing=()
    local tool
    for tool in "$@"; do
        command -v "$tool" >/dev/null 2>&1 || missing+=("$tool")
    done
    if [ ${#missing[@]} -ne 0 ]; then
        die "missing from PATH: ${missing[*]}
These come from the CI image, see docker/ci/Dockerfile."
    fi
}

# tree_state / require_unchanged_tree catch what a step leaves behind. They
# compare before and after rather than demanding a pristine checkout, so the
# scripts are usable on a working tree that already has edits in it; on a CI
# checkout the two are the same thing.
tree_state() {
    git -C "$REPO_ROOT" status --porcelain | LC_ALL=C sort
}

require_unchanged_tree() {
    local what="$1"
    local before="$2"
    local after
    after="$(tree_state)"
    if [ "$before" != "$after" ]; then
        printf 'error: %s changed the working tree:\n' "$what" >&2
        diff <(printf '%s\n' "$before") <(printf '%s\n' "$after") >&2 || true
        die "a build must not modify tracked files, nor leave untracked ones behind"
    fi
}
