
set -o errexit
set -o nounset
set -o pipefail

if ! which gofmt > /dev/null; then
  echo "Can not find gofmt"
  exit 1
fi

# gofmt exits with non-zero exit code if it finds a problem unrelated to
# formatting (e.g., a file does not parse correctly). Without "|| true" this
# would have led to no useful error message from gofmt, because the script would
# have failed before getting to the "echo" in the block below.
diff=$(find . -name "*.go" | grep -v "\/vendor\/" | xargs gofmt -s -d 2>&1) || true
if [[ -n "${diff}" ]]; then
  echo "${diff}"
  echo
  echo "Please run hack/update-gofmt.sh"
  exit 1
fi
