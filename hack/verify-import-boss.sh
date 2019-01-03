
set -o errexit
set -o nounset
set -o pipefail

if ! which import-boss > /dev/null; then
  echo "Can not find import-boss"
  exit 1
fi

import-boss -i k8s.io/cloud-provider-vra/... --verify-only
