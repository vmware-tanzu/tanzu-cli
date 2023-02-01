#!/usr/bin/env bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

ROOT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)

usage() {
    echo "generate_central.sh [-h | -d | --dry-run | --fast] REPO_URI"
    echo
    echo "Push 99 plugins to a test repository located at REPO_URI (e.g., localhost:9998/central:small"
    echo "  -h               Print this help"
    echo "  -d, --dry-run    Only print the commands that would be executed"
    echo "      --fast       Only install 4 plugins"
    exit 0
}

if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    usage
fi

dry_run=""
if [ "$1" = "-d" ] || [ "$1" = "--dry-run" ]; then
    dry_run=echo
    shift
fi

fast=off
if [ "$1" = "--fast" ]; then
    fast=on
    shift
fi

if [ $# -eq 0 ] || [[ $1 == "-"* ]]; then
    usage
fi

content_image=$1
repoBasePath=$(dirname $content_image)
database=/tmp/central.db
publisher="vmware/tkg"

# Create db table
cat << EOF | sqlite3 -batch $database
CREATE TABLE IF NOT EXISTS "PluginBinaries" (
		"PluginName"         TEXT NOT NULL,
		"Target"             TEXT NOT NULL,
		"RecommendedVersion" TEXT NOT NULL,
		"Version"            TEXT NOT NULL,
		"Hidden"             INTEGER NOT NULL,
		"Description"        TEXT NOT NULL,
		"Publisher"          TEXT NOT NULL,
		"Vendor"             TEXT NOT NULL,
		"OS"                 TEXT NOT NULL,
		"Architecture"       TEXT NOT NULL,
		"Digest"             TEXT NOT NULL,
		"URI"                TEXT NOT NULL,
		PRIMARY KEY("PluginName", "Target", "Version", "OS", "Architecture")
	);
EOF

addPlugin() {
    name=$1
    target=$2
    pushBinary=$3

    tmpPluginPhase1="/tmp/fakeplugin1.sh"
    tmpPluginPhase2="/tmp/fakeplugin2.sh"

    # Start preparing the plugin file with the name and target.
    # Start here to avoir repeating in the loop.
    sed -e "s/__NAME__/$name/" -e "s/__TARGET__/$target/" $ROOT_DIR/fakeplugin.sh > $tmpPluginPhase1

    # Define 10 versions for the plugin
    for v in {0..9}; do
        version="v$v.$v.$v"

        # Put printout to show progress
        echo "Inserting $name version $version for target $target"

        # Create the plugin file with the correct version
        sed -e "s/__VERSION__/$version/" $tmpPluginPhase1 > $tmpPluginPhase2
        digest=$(sha256sum $tmpPluginPhase2 | cut -f1 -d' ')

        for os in darwin linux windows; do
            for arch in amd64 arm64; do
                if [ $arch = arm64 ] && [ $os != darwin ]; then
                    # Only support darwin with arm64 for now
                    continue
                fi

                image_path=$publisher/$os/$arch/$target/$name

                sql_cmd="INSERT INTO PluginBinaries VALUES('$name','$target','v9.9.9','$version','FALSE','Desc for $name','TKG','VMware','$os','$arch','$digest','$image_path:$version');"
                if [ "$dry_run" = "echo" ]; then
                    echo $sql_cmd 
                else 
                    echo $sql_cmd | sqlite3 -batch $database
                fi

                # For efficiency, only push the plugin binaries that have requested it, and for
                # those, still only push versions v8.8.8 and v9.9.9
                if [ $pushBinary = "true" ]; then
                    if [ $version = "v8.8.8" ] || [ $version = "v9.9.9" ]; then
                        echo "Pushing binary for $name version $version for target $target, $os-$arch"
                        ${dry_run} imgpkg push -i $repoBasePath/$image_path:$version -f $tmpPluginPhase2 --registry-insecure
                    fi
                fi
            done
        done
    done
}

k8sPlugins=(cluster feature management-cluster package secret telemetry kubernetes-release)
tmcPlugins=(account apply audit cluster clustergroup data-protection ekscluster events iam 
            inspection integration management-cluster policy workspace)
globalPlugins=(isolated-cluster pinniped-auth)

echo "======================================================================="
echo "Creating test Central Repository: $content_image"
echo "======================================================================="

for name in ${globalPlugins[*]}; do
    addPlugin $name global true
done

for name in ${k8sPlugins[*]}; do
    addPlugin $name k8s true
done

for name in ${tmcPlugins[*]}; do
    addPlugin $name tmc true
done


# Push generic plugins to get to 100 total plugins in the DB.
# Those plugins will not be installable as we won't push their binaries.
if [ $fast = "off" ]; then
    pluginCount=$((${#k8sPlugins[@]} + ${#tmcPlugins[@]} + ${#globalPlugins[@]}))
    numPluginsToCreate=$((100-$pluginCount))
    
    for (( idx=1; idx<=$numPluginsToCreate; idx++ )); do
        target_rand=$(($RANDOM % 3))
        case $target_rand in
        0) target=global
           ;;
        1) target=k8s
           ;;
        2) target=tmc
           ;;
        esac

        addPlugin stub$idx $target false
    done
fi

# Push content file
${dry_run} imgpkg push -i $content_image -f $database --registry-insecure
rm -f $database

