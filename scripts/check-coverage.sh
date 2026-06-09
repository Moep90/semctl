#!/usr/bin/env bash
#
# Copyright 2026 The semctl authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Fail if total test coverage drops below the committed floor.
#
# The floor is a ratchet: it only ever moves up. When coverage rises
# meaningfully, bump .github/coverage-baseline so the gain can't silently
# regress later. New code therefore can never lower the bar, and there is no
# arbitrary target that blocks unrelated work.

set -euo pipefail

floor_file="${COVERAGE_BASELINE_FILE:-.github/coverage-baseline}"
profile="${COVERAGE_PROFILE:-coverage.out}"

if [ ! -f "$profile" ]; then
  echo "generating coverage profile ($profile)..."
  go test -covermode=atomic -coverprofile="$profile" ./... >/dev/null
fi

total=$(go tool cover -func="$profile" | awk '/^total:/ { gsub(/%/, "", $NF); print $NF }')
floor=$(tr -d '[:space:]' < "$floor_file")

printf 'total coverage: %s%%   floor: %s%%\n' "$total" "$floor"

if awk -v t="$total" -v f="$floor" 'BEGIN { exit !(t + 0 < f + 0) }'; then
  echo "::error::coverage ${total}% is below the floor ${floor}% — add tests, or lower ${floor_file} only if the drop is intentional and justified"
  exit 1
fi

echo "coverage floor OK"
if awk -v t="$total" -v f="$floor" 'BEGIN { exit !(t + 0 >= f + 1.0) }'; then
  echo "note: coverage ${total}% is >=1% above the floor — consider bumping ${floor_file} to lock in the gain"
fi
