#!/bin/bash -eu

root_dir=$(cd $(dirname $0)/../; pwd)

if [ $# -ne 2 ]; then
    echo "usage ./scripts/bump-go-version.sh prev-ver latest-ver"
    exit 1
fi
prev=$1
latest=$2

prev_major_minor=$(echo ${prev} | perl -wlne '/(\d+.\d+).*/ and print $1')
latest_major_minor=$(echo ${latest} | perl -wlne '/(\d+.\d+).*/ and print $1')

perl -pi -e "s/go_version='.+'/go_version='go${latest}'/" ${root_dir}/README.md

for p in $(find ${root_dir} -name 'go.mod'); do
    echo ${p}
    cd $(dirname ${p})
    go mod edit -go=${prev_major_minor}
done

for p in $(find ${root_dir}/.github/workflows -name '*.yml'); do
    echo ${p}
    perl -pi -e "s/go-version:\s*${prev_major_minor}\.x/go-version: ${latest_major_minor}\.x/" ${p}
    perl -pi -e "s/go-version:\s*\[${prev_major_minor}\.x,\s*\d+\.\d+\.x\]/go-version: [${latest_major_minor}\.x, ${prev_major_minor}\.x]/" ${p}
done

perl -pi -e "s/semver\.NewVersion\(\"\d*\.\d*.0\"\)/semver\.NewVersion\(\"${prev_major_minor}.0\"\)/" ${root_dir}/scripts/cross-build/build-matrix/main.go
