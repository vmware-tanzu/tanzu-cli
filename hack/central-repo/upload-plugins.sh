#!/usr/bin/env bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

ROOT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
# password to be used while signing the image using cosign
# (cosign binary internally would use this environment variable while signing the image)
export COSIGN_PASSWORD=test-cli-registry

usage() {
    echo "upload-plugins.sh [-h | -d | --dry-run]"
    echo
    echo "Creates test central repositories:"
    echoImages
    echo
    echo "  -h               Print this help"
    echo "  -d, --dry-run    Only print the commands that would be executed"
    exit 0
}

echoImages() {
    echo "=> localhost:9876/tanzu-cli/plugins/central:small"
    echo "    - a small amount of plugins matching product plugin names"
    echo "    - only versions v0.0.1 and v9.9.9 can be installed"
    echo "=> localhost:9876/tanzu-cli/plugins/central:large with 100 plugins"
    echo "    - the same content as the small image with extra plugins for a total of 100"
    echo "    - none of the 'stubXY' plugins can be installed"
    echo "    - only versions v0.0.1 and v9.9.9 can be installed"
    echo "=> localhost:9876/tanzu-cli/plugins/sandbox1:small"
    echo "    - an extra v11.11.11 version of the plugins of the small image"
    echo "=> localhost:9876/tanzu-cli/plugins/sandbox2:small"
    echo "    - an extra v22.22.22 version of the plugins of the small image"
    echo "=> localhost:9876/tanzu-cli/plugins/airgapped:small"
    echo "    - a small amount of plugins matching product plugin names"
    echo "    - contains only versions v0.0.1 and v9.9.9 of plugins and all of them can be installed"
    echo "=> localhost:9876/tanzu-cli/plugins/airgapped:large"
    echo "    - all plugins matching product plugin names"
    echo "    - contains only versions v0.0.1 and v9.9.9 of plugins and all of them can be installed"
    echo "=> localhost:9876/tanzu-cli/plugins/shas:small"
    echo "    - a small amount of plugins matching product plugin names"
    echo "    - contains only versions v0.0.1 and v9.9.9 of plugins and all of them can be installed"
    echo "    - SHAs are used to reference the v9.9.9 plugin binaries and tags for v0.0.1"
}
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    usage
fi

if [ "$1" = "info" ]; then
    echoImages
    exit 0
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
sanboxImage1=sandbox1:small
sanboxImage2=sandbox2:small
smallAirgappedImage=airgapped:small
largeAirgappedImage=airgapped:large
smallImageUsingSHAs=shas:small
database=/tmp/plugin_inventory.db
publisher="vmware/tkg"

# Push an empty image
echo "======================================"
echo "Creating an empty test Central Repository: $repoBasePath/central:empty"
echo "======================================"
rm -f $database
touch $database
${dry_run} imgpkg push -i $repoBasePath/central:empty -f $database  --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/central:empty


# Push an image with an empty table
echo "======================================"
echo "Creating a test Central Repository with no entries: $repoBasePath/central:emptytable"
echo "======================================"
# Create db table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database
${dry_run} imgpkg push -i $repoBasePath/central:emptytable -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/central:emptytable

addPlugin() {
    local name=$1
    local target=$2
    local pushBinary=$3
    local versions=$4
    local useSHAs=$5

    local tmpPluginPhase1="/tmp/fakeplugin1.sh"
    local tmpPluginPhase2="/tmp/fakeplugin2.sh"

    # Start preparing the plugin file with the name and target.
    # Start here to avoir repeating in the loop.
    sed -e "s/__NAME__/$name/" -e "s/__TARGET__/$target/" $ROOT_DIR/fakeplugin.sh > $tmpPluginPhase1

    # If the version is not specified, we create 10 of them
    # We include version v0.0.1 to match the real TMC plugin versions
    recommended=""
    if [ -z "$versions" ]; then
        versions="v0.0.1 v1.1.1 v2.2.2 v3.3.3 v4.4.4 v5.5.5 v6.6.6 v7.7.7 v8.8.8 v9.9.9"
    fi

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

                # For efficiency, only push the plugin binaries that have requested it, and for
                # those, still only push versions v0.0.1 (to match real TMC plugins) and v9.9.9
                if [ $pushBinary = "true" ]; then
                    if [ $version = "v0.0.1" ] || [ $version = "v9.9.9" ] || [ $version = "v11.11.11" ] || [ $version = "v22.22.22" ]; then
                        echo "Pushing binary for $name version $version for target $target, $os-$arch"
                        if [ "$dry_run" = "echo" ]; then
                            sha="12345"
                        else
                            sha=$(imgpkg push -i $repoBasePath/$image_path:$version -f $tmpPluginPhase2 --registry-verify-certs=false --json | grep sha256 | cut -d@ -f2 | cut -d\' -f1)
                        fi
                    fi
                fi

                local image_ref="$image_path:$version"
                [ -n "$useSHAs" ] && image_ref="$image_path@$sha"
                local sql_cmd="INSERT INTO PluginBinaries VALUES('$name','$target','$recommended','$version','false','Desc for $name','TKG','VMware','$os','$arch','$digest','$image_ref');"
                if [ "$dry_run" = "echo" ]; then
                    echo $sql_cmd
                else
                    echo $sql_cmd | sqlite3 -batch $database
                fi
            done
        done
    done
}

addGroup() {
    local vendor=$1
    local publisher=$2
    local name=$3
    local groupVersion=$4
    local plugin=$5
    local target=$6
    local pluginVersion=$7
    local mandatory='true'
    local hidden='false'

    echo "Adding $plugin/$target version $pluginVersion to plugin group $vendor-$publisher/$name:$groupVersion"

    local sql_cmd="INSERT INTO PluginGroups VALUES('$vendor','$publisher','$name','$groupVersion','Desc for $vendor-$publisher/$name','$plugin','$target','$pluginVersion', '$mandatory', '$hidden');"
    if [ "$dry_run" = "echo" ]; then
        echo $sql_cmd
    else
        echo $sql_cmd | sqlite3 -batch $database
    fi
}

k8sPlugins=(cluster feature management-cluster package secret telemetry kubernetes-release)
tmcPlugins=(account apply audit cluster clustergroup data-protection ekscluster events iam
            inspection integration management-cluster policy workspace helm secret
            continuousdelivery tanzupackage)
globalPlugins=(isolated-cluster pinniped-auth)

echo "======================================"
echo "Creating small test Central Repository: $repoBasePath/$smallImage"
echo "======================================"

for name in ${globalPlugins[*]}; do
    addPlugin $name global true
done

for name in ${k8sPlugins[*]}; do
    addPlugin $name kubernetes true
    addGroup vmware tkg default v0.0.1 $name kubernetes v0.0.1
    addGroup vmware tkg default v9.9.9 $name kubernetes v9.9.9
done

for name in ${tmcPlugins[*]}; do
    addPlugin $name mission-control true
    addGroup vmware tmc tmc-user v0.0.1 $name mission-control v0.0.1
    addGroup vmware tmc tmc-user v9.9.9 $name mission-control v9.9.9
done

# Push small inventory file
${dry_run} imgpkg push -i $repoBasePath/$smallImage -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$smallImage

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
    1) target=kubernetes
       ;;
    2) target=mission-control
       ;;
    esac
    addPlugin stub$idx $target false
done

# Push large inventory file
${dry_run} imgpkg push -i $repoBasePath/$largeImage -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$largeImage

# Create an additional image as if it was a sandbox build
echo "======================================"
echo "Creating a first test Central Repository as a sandbox build: $repoBasePath/$sanboxImage1"
echo "======================================"
# Reset the DB
rm -f $database
touch $database
# Create the DB table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

# Create a new version of the plugins for the sandbox repo
for name in ${k8sPlugins[*]}; do
    addPlugin $name kubernetes true v11.11.11
    addGroup vmware tkg default v11.11.11 $name kubernetes v11.11.11
done

for name in ${tmcPlugins[*]}; do
    addPlugin $name mission-control true v11.11.11
    addGroup vmware tmc tmc-user v11.11.11 $name mission-control v11.11.11
done

# Push sanbox inventory file
${dry_run} imgpkg push -i $repoBasePath/$sanboxImage1 -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$sanboxImage1

# Create a second additional image as if it was a sandbox build
echo "======================================"
echo "Creating a second test Central Repository as a sandbox build: $repoBasePath/$sanboxImage2"
echo "======================================"
# Reset the DB
rm -f $database
touch $database
# Create the DB table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

# Create a new version of the plugins for the sandbox repo
for name in ${k8sPlugins[*]}; do
    addPlugin $name kubernetes true v22.22.22
    addGroup vmware tkg default v22.22.22 $name kubernetes v22.22.22
    # Redefine a group that exists in the central repo
    addGroup vmware tkg default v9.9.9 $name kubernetes v22.22.22
done

for name in ${tmcPlugins[*]}; do
    addPlugin $name mission-control true v22.22.22
    addGroup vmware tmc tmc-user v22.22.22 $name mission-control v22.22.22
done

# Push sandbox inventory file
${dry_run} imgpkg push -i $repoBasePath/$sanboxImage2 -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$sanboxImage2

echo "======================================"
echo "Creating small airgapped test Central Repository: $repoBasePath/$smallAirgappedImage"
echo "======================================"
# Reset the DB
rm -f $database
touch $database
# Create the DB table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

for name in ${globalPlugins[0]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9
done

for name in ${k8sPlugins[0]}; do
    addPlugin $name kubernetes true v0.0.1
    addPlugin $name kubernetes true v9.9.9
    addGroup vmware tkg default v0.0.1 $name kubernetes v0.0.1
    addGroup vmware tkg default v9.9.9 $name kubernetes v9.9.9
done

for name in ${tmcPlugins[0]}; do
    addPlugin $name mission-control true v0.0.1
    addPlugin $name mission-control true v9.9.9
    addGroup vmware tmc tmc-user v0.0.1 $name mission-control v0.0.1
    addGroup vmware tmc tmc-user v9.9.9 $name mission-control v9.9.9
done

# Push airgapped inventory file
${dry_run} imgpkg push -i $repoBasePath/$smallAirgappedImage -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$smallAirgappedImage

echo "======================================"
echo "Creating large airgapped test Central Repository: $repoBasePath/$largeAirgappedImage"
echo "======================================"
# Reset the DB
rm -f $database
touch $database
# Create the DB table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

for name in ${globalPlugins[*]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9
done

for name in ${k8sPlugins[*]}; do
    addPlugin $name kubernetes true v0.0.1
    addPlugin $name kubernetes true v9.9.9
    addGroup vmware tkg default v0.0.1 $name kubernetes v0.0.1
    addGroup vmware tkg default v9.9.9 $name kubernetes v9.9.9
done

for name in ${tmcPlugins[*]}; do
    addPlugin $name mission-control true v0.0.1
    addPlugin $name mission-control true v9.9.9
    addGroup vmware tmc tmc-user v0.0.1 $name mission-control v0.0.1
    addGroup vmware tmc tmc-user v9.9.9 $name mission-control v9.9.9
done

# Push airgapped inventory file
${dry_run} imgpkg push -i $repoBasePath/$largeAirgappedImage -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry  -y $repoBasePath/$largeAirgappedImage

echo "======================================"
echo "Creating small test Central Repository using SHAs: $repoBasePath/$smallImageUsingSHAs"
echo "======================================"
# Reset the DB
rm -f $database
touch $database
# Create the DB table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

for name in ${globalPlugins[0]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9 useSha
done

for name in ${k8sPlugins[0]}; do
    addPlugin $name kubernetes true v0.0.1
    addPlugin $name kubernetes true v9.9.9 useSha
    addGroup vmware tkg default v0.0.1 $name kubernetes v0.0.1
    addGroup vmware tkg default v9.9.9 $name kubernetes v9.9.9
done

for name in ${tmcPlugins[0]}; do
    addPlugin $name mission-control true v0.0.1
    addPlugin $name mission-control true v9.9.9 useSha
    addGroup vmware tmc tmc-user v0.0.1 $name mission-control v0.0.1
    addGroup vmware tmc tmc-user v9.9.9 $name mission-control v9.9.9
done

# Push airgapped inventory file
${dry_run} imgpkg push -i $repoBasePath/$smallImageUsingSHAs -f $database --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$smallImageUsingSHAs

rm -f $database
