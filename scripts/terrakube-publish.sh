#!/usr/bin/env bash
# Registers the release goreleaser just built in dist/ as a version of the
# `unifi` provider in Terrakube's private registry, with one implementation per
# platform zip. Terrakube stores only pointers; the files stay at RELEASE_URL.
set -euo pipefail

api() {
  curl -sS -f -H "Authorization: Bearer ${TERRAKUBE_TOKEN}" "$@"
}

post() {
  api -X POST -H "Content-Type: application/vnd.api+json" -d "$2" "$1" | jq -r '.data.id'
}

provider=unifi
sums=$(ls dist/*_SHA256SUMS)
sums_file=$(basename "${sums}")
version=${sums_file#terraform-provider-unifi_}
version=${version%_SHA256SUMS}
ascii_armor=$(gpg --armor --export "${GPG_FINGERPRINT}")
key_id=${GPG_FINGERPRINT: -16}

org_url="${TERRAKUBE_API}/api/v1/organization/$(api "${TERRAKUBE_API}/api/v1/organization" \
  | jq -r --arg n "${TERRAKUBE_ORG}" '.data[] | select(.attributes.name == $n) | .id')"

provider_id=$(api "${org_url}/provider" \
  | jq -r --arg n "${provider}" '.data[] | select(.attributes.name == $n) | .id')
if [ -z "${provider_id}" ]; then
  provider_id=$(post "${org_url}/provider" "$(jq -nc --arg n "${provider}" \
    '{data: {type: "provider",
             attributes: {name: $n, description: "drewinglis fork of resnickio/terraform-provider-unifi"}}}')")
  echo "Created provider ${provider} (${provider_id})"
fi

# Reuses an existing version and skips platforms it already has, so a run that
# failed partway can be repeated.
version_id=$(api "${org_url}/provider/${provider_id}/version" \
  | jq -r --arg v "${version}" '.data[] | select(.attributes.versionNumber == $v) | .id')
if [ -z "${version_id}" ]; then
  version_id=$(post "${org_url}/provider/${provider_id}/version" "$(jq -nc --arg v "${version}" \
    '{data: {type: "version", attributes: {versionNumber: $v, protocols: "5.0"}}}')")
  echo "Created version ${version} (${version_id})"
fi
existing=$(api "${org_url}/provider/${provider_id}/version/${version_id}/implementation" \
  | jq -r '.data[] | "\(.attributes.os)_\(.attributes.arch)"')

count=0
while read -r shasum filename; do
  platform=${filename#terraform-provider-unifi_"${version}"_}
  platform=${platform%.zip}
  os=${platform%%_*}
  arch=${platform#*_}
  if grep -qx "${os}_${arch}" <<<"${existing}"; then
    echo "Skipping ${os}/${arch}: already registered"
    continue
  fi
  post "${org_url}/provider/${provider_id}/version/${version_id}/implementation" "$(jq -nc \
    --arg os "${os}" --arg arch "${arch}" --arg f "${filename}" --arg shasum "${shasum}" \
    --arg url "${RELEASE_URL}" --arg sums "${sums_file}" \
    --arg key "${key_id}" --arg armor "${ascii_armor}" \
    '{data: {type: "implementation",
             attributes: {os: $os, arch: $arch, filename: $f, shasum: $shasum,
                          downloadUrl: "\($url)/\($f)",
                          shasumsUrl: "\($url)/\($sums)",
                          shasumsSignatureUrl: "\($url)/\($sums).sig",
                          keyId: $key, asciiArmor: $armor,
                          trustSignature: "", source: "drewinglis",
                          sourceUrl: "https://git.app.drw.io/drewinglis/terraform-provider-unifi"}}}')" >/dev/null
  echo "Registered ${os}/${arch}"
  count=$((count + 1))
done < <(grep '\.zip$' "${sums}")

echo "Registered ${count} platforms for ${provider} ${version}"
