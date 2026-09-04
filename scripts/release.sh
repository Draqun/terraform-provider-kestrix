#!/usr/bin/env bash
set -Eeuo pipefail
version="${1:?usage: provider-release.sh VERSION}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="${repo_root}/dist/provider/${version}"
rm -rf "${out}"
mkdir -p "${out}"
for os in linux darwin windows; do
  for arch in amd64 arm64; do
    name="terraform-provider-kestrix_${version}_${os}_${arch}"
    binary_dir="${out}/${name}"
    mkdir -p "${binary_dir}"
    GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${version}" -o "${binary_dir}/terraform-provider-kestrix" "${repo_root}"
    (cd "${binary_dir}" && zip -q -j "../${name}.zip" terraform-provider-kestrix)
    rm -rf "${binary_dir}"
  done
done
(cd "${out}" && sha256sum ./*.zip > SHA256SUMS)
cat > "${out}/provenance.json" <<EOF
{"provider":"registry.terraform.io/kestrix/kestrix","version":"${version}","source":"kestrix/terraform-provider-kestrix","reproducible":true}
EOF
echo "Provider release artifacts written to ${out}"
