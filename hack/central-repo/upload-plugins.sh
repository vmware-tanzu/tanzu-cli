#!/usr/bin/env bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

ROOT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)

usage() {
    echo "upload-plugins.sh [-h | -d | --dry-run]"
    echo
    echo "Creates two test central repositories:"
    echo "- localhost:9876/tanzu-cli/plugins/central:small with a small amount of plugins"
    echo "- localhost:9876/tanzu-cli/plugins/central:large with 100 plugins"
    echo
    echo "  -h               Print this help"
    echo "  -d, --dry-run    Only print the commands that would be executed"
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

if [ $# -gt 0 ]; then
    usage
fi

repoBasePath=host.docker.internal:9876/tanzu-cli/plugins
smallImage=central:small
largeImage=central:large
database=/tmp/plugin_inventory.db
publisher="vmware/tkg"

# Push an empty image
echo "======================================"
echo "Creating an empty test Central Repository: $repoBasePath/central:empty"
echo "======================================"
rm -f $database
touch $database
${dry_run} imgpkg push -i $repoBasePath/central:empty -f $database --registry-insecure

# Create db table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

# Push an image with an empty table
echo "======================================"
echo "Creating a test Central Repository with no entries: $repoBasePath/central:emptytable"
echo "======================================"
${dry_run} imgpkg push -i $repoBasePath/central:emptytable -f $database --registry-insecure

addPlugin() {
    local name=$1
    local target=$2
    local pushBinary=$3

    local tmpPluginPhase1="/tmp/fakeplugin1.sh"
    local tmpPluginPhase2="/tmp/fakeplugin2.sh"

    # Start preparing the plugin file with the name and target.
    # Start here to avoir repeating in the loop.
    sed -e "s/__NAME__/$name/" -e "s/__TARGET__/$target/" $ROOT_DIR/fakeplugin.sh > $tmpPluginPhase1

    # Define 10 versions for the plugin
    # We include version v0.0.1 to match the real TMC plugin versions
    local versions="v0.0.1 v1.1.1 v2.2.2 v3.3.3 v4.4.4 v5.5.5 v6.6.6 v7.7.7 v8.8.8 v9.9.9"
    for version in $versions; do

        # Put printout to show progress
        echo "Inserting $name version $version for target $target"

        # Create the plugin file with the correct version
        sed -e "s/__VERSION__/$version/" $tmpPluginPhase1 > $tmpPluginPhase2
        local digest=$(sha256sum $tmpPluginPhase2 | cut -f1 -d' ')

        for os in darwin linux windows; do
            for arch in amd64 arm64; do
                if [ $arch = arm64 ] && [ $os != darwin ]; then
                    # Only support darwin with arm64 for now
                    continue
                fi

                local image_path=$publisher/$os/$arch/$target/$name

                local sql_cmd="INSERT INTO PluginBinaries VALUES('$name','$target','v9.9.9','$version','FALSE','Desc for $name','TKG','VMware','$os','$arch','$digest','$image_path:$version');"
                if [ "$dry_run" = "echo" ]; then
                    echo $sql_cmd 
                else 
                    echo $sql_cmd | sqlite3 -batch $database
                fi

                # For efficiency, only push the plugin binaries that have requested it, and for
                # those, still only push versions v0.0.1 (to match real TMC plugins) and v9.9.9
                if [ $pushBinary = "true" ]; then
                    if [ $version = "v0.0.1" ] || [ $version = "v9.9.9" ]; then
                        echo "Pushing binary for $name version $version for target $target, $os-$arch"
                        ${dry_run} imgpkg push -i $repoBasePath/$image_path:$version -f $tmpPluginPhase2 --registry-insecure
                    fi
                fi
            done
        done
    done
}

addGroup() {
    local vendor=$1
    local publisher=$2
    local name=$3
    local plugin=$4
    local target=$5
    local version=$6

    echo "Adding $plugin/$target version $version to plugin group $vendor-$publisher/$name"

    local sql_cmd="INSERT INTO PluginGroups VALUES('$vendor','$publisher','$name','$plugin','$target','$version');"
    if [ "$dry_run" = "echo" ]; then
        echo $sql_cmd 
    else 
        echo $sql_cmd | sqlite3 -batch $database
    fi
}

k8sPlugins=(cluster feature management-cluster package secret telemetry kubernetes-release)
tmcPlugins=(account apply audit cluster clustergroup data-protection ekscluster events iam 
            inspection integration management-cluster policy workspace)
globalPlugins=(isolated-cluster pinniped-auth)

echo "======================================"
echo "Creating small test Central Repository: $repoBasePath/$smallImage"
echo "======================================"

for name in ${globalPlugins[*]}; do
    addPlugin $name global true
done

for name in ${k8sPlugins[*]}; do
    addPlugin $name k8s true
    addGroup vmware tkg v1.0.0 $name k8s v0.0.1
    addGroup vmware tkg v2.1.0 $name k8s v9.9.9
done

for name in ${tmcPlugins[*]}; do
    addPlugin $name tmc true
    addGroup vmware tmc v1.2.3 $name tmc v0.0.1
    addGroup vmware tmc v9.0.0 $name tmc v9.9.9
done

# Push small inventory file
${dry_run} imgpkg push -i $repoBasePath/$smallImage -f $database --registry-insecure

echo "======================================"
echo "Creating large test Central Repository: $repoBasePath/$largeImage"
echo "======================================"

# Push generic plugins to get to 100 total plugins in the DB.
# Those plugins will not be installable as we won't push their binaries.
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

# Push large inventory file
${dry_run} imgpkg push -i $repoBasePath/$largeImage -f $database --registry-insecure
rm -f $database
