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
    echo "=> localhost:9876/tanzu-cli/plugins2/airgapped:large"
    echo "    - same as plugins/airgapped:large except with a different central config content"
    echo "=> localhost:9876/tanzu-cli/plugins/shas:small"
    echo "    - a small amount of plugins matching product plugin names"
    echo "    - contains only versions v0.0.1 and v9.9.9 of plugins and all of them can be installed"
    echo "    - SHAs are used to reference the v9.9.9 plugin binaries and tags for v0.0.1"
    echo "=> localhost:9876/tanzu-cli/plugins/extra:small"
    echo "    - same DB as central:small but with an extra column both for plugins and groups"
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

# The names for the different images part of the test repo
smallImage=central:small
largeImage=central:large
sanboxImage1=sandbox1:small
sanboxImage2=sandbox2:small
smallAirgappedImage=airgapped:small
largeAirgappedImage=airgapped:large
smallImageUsingSHAs=shas:small
extraColumn=extra:small

# Constants
repoBasePath=host.docker.internal:9876/tanzu-cli/plugins
imageContentPath=/tmp/tanzu-test-central-repo
database=$imageContentPath/plugin_inventory.db
centralConfigFile=$imageContentPath/central_config.yaml
publisher="vmware/tkg"

resetImageContentDir() {
    # Use a two step rm -rf so that we don't risk deleting the wrong thing
    # This avoids a potential issue if the imageContentPath is empty or corrupt
    [ -d $imageContentPath ] && mv $imageContentPath $imageContentPath.old
    rm -rf "${imageContentPath}.old"
    mkdir -p $imageContentPath
}

# Start clean
resetImageContentDir

# Push an empty image
echo "======================================"
echo "Creating an empty test Central Repository: $repoBasePath/central:empty"
echo "======================================"
touch $database
${dry_run} imgpkg push -i $repoBasePath/central:empty -f $imageContentPath  --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/central:empty

# Push an image with an empty table and central config file
echo "======================================"
echo "Creating a test Central Repository with no entries: $repoBasePath/central:emptytable"
echo "======================================"
# Create db table
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database
# Create empty central config file
touch $centralConfigFile
# Push and sign the image which will contain both the DB and the central config file
${dry_run} imgpkg push -i $repoBasePath/central:emptytable -f $imageContentPath --registry-verify-certs=false
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
                    if [ $version = "v0.0.1" ] || [ $version = "v9.9.9" ] || [ $version = "v11.11.11" ] || [ $version = "v22.22.22" ] || [[ $version = v1.9* ]] || [[ $version = v1.10* ]] || [[ $version = v1.11* ]] || [[ $version = v2.3* ]]; then
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
                local sql_cmd="INSERT INTO PluginBinaries VALUES('$name','$target','$recommended','$version','false','$name functionality','TKG','VMware','$os','$arch','$digest','$image_ref');"

                # With "extra" specified, we need to insert an extra value for the extra column
                if [ -n "$extra" ]; then
                    sql_cmd="INSERT INTO PluginBinaries VALUES('$name','$target','$extra','$recommended','$version','false','$name functionality','TKG','VMware','$os','$arch','$digest','$image_ref');"
                fi
                
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

    local sql_cmd="INSERT INTO PluginGroups VALUES('$vendor','$publisher','$name','$groupVersion','Desc for $vendor-$publisher/$name:$groupVersion','$plugin','$target','$pluginVersion', '$mandatory', '$hidden');"

    # With "extra" specified, we need to insert an extra value for the extra column
    if [ -n "$extra" ]; then
        local sql_cmd="INSERT INTO PluginGroups VALUES('$vendor','$publisher','$name','$groupVersion','$extra','Desc for $vendor-$publisher/$name:$groupVersion','$plugin','$target','$pluginVersion', '$mandatory', '$hidden');"
    fi

    if [ "$dry_run" = "echo" ]; then
        echo $sql_cmd
    else
        echo $sql_cmd | sqlite3 -batch $database
    fi
}

createCentralConfigFile() {
    local centralCfgFile=$1
    # This allows to change the content of the central config
    local specialRecommendedVersion=${2:-"v2.1.0-alpha.2"}

    echo "Creating central config file $centralCfgFile"
    # Create some content that is rich enough to write
    # different types of tests
    cat <<EOF > $centralConfigFile
cli.core.cli_recommended_versions: 
- version: $specialRecommendedVersion
- version: v2.0.2
- version: v1.5.0-beta.0
- version: v1.4.4
- version: v1.3.3
- version: v1.2.2
- version: v1.1.1
- version: v0.90.0
cli.core.tanzu_application_platform_scopes:
- scope: tap:viewer
- scope: tap:admin
- scope: tap:member
cli.core.tanzu_hub_metadata:
  cspProductIdentifier: "TANZU-SAAS"
  cspDisplayName: "Tanzu Platform"
  endpointProduction: https://www.production.fake.vmware.com/hub
  endpointStaging: https://www.staging.fake.vmware.com/hub
  useCentralConfig: false
cli.core.some-string: "the meaning of life, the universe, and everything"
cli.core.some-int: 42
cli.core.some-bool: true
cli.core.some-yaml:
  # This chunk of yaml will be parsed automatically
  description: Build Tanzu components
  target: global
  version: v1.2.0
  buildSHA: f3abe62e  # This is a comment
  group: Admin
cli.core.some-yaml-as-a-string: |-
  # This chunk of yaml will NOT be parsed automatically
  # because of the use of the |- operator
  # It will be kept as a string.  The caller
  # can parse it into its own yaml object
  description: Another
  target: global
  version: v1.8.0
  buildSHA: f3abe62e  # This is a comment
  group: Build
cli.core.some-json-as-a-string: |- 
    {
        "this": "json",
        "data": "will not be parsed"
        "but" : "will be returned as a string"
        "which": "the caller can parse itself"
    }

cli.core.tanzu_default_endpoint: https://api.tanzu.cloud.vmware.com

cli.core.tanzu_endpoint_map:
  https://api.tanzu.cloud.vmware.com:
    ucp: https://api.tanzu.cloud.vmware.com
    tmc: https://tmc.tanzu.cloud.vmware.com
    hub: https://api.mgmt.cloud.vmware.com

cli.core.tanzu_cli_platform_saas_endpoints_as_regular_expression:
  - https://(www.)?platform(.)*.tanzu.broadcom.com
  - https://api.tanzu(.)*.cloud.vmware.com

EOF
}

k8sPlugins=(cluster feature management-cluster package secret kubernetes-release telemetry)
tmcPlugins=(account apply audit cluster clustergroup data-protection ekscluster events iam
            inspection integration management-cluster policy workspace helm secret
            continuousdelivery tanzupackage)
opsPlugins=(clustergroup)
globalPlugins=(isolated-cluster pinniped-auth)
essentialPlugins=(telemetry)
pluginUsingSha=(plugin-with-sha)
multiversionPlugins=(cluster package secret)

defaultDB=$database
defaultCentralConfigFile=$centralConfigFile
defaultImageContentPath=$imageContentPath
# Build two DBs with the same set of plugins and groups, except one DB will have
# an extra column in both tables.  We do this in a loop to make sure both DB have
# the exact same set of plugins, so we can run the same tests on both.
for db in small extra; do
    if [ $db = extra ]; then
        # Use a special directory for the extra-column tests
        imageContentPath=${defaultImageContentPath}-extra
        # Set the new location for the DB and central config file
        database=$imageContentPath/plugin_inventory.db
        centralConfigFile=$imageContentPath/central_config.yaml
        # Clean any old directory and create the new one
        resetImageContentDir

        # Create the special DB
        touch $database
        # Create the DB tables with the extra column
        cat $ROOT_DIR/create_tables_extra.sql | sqlite3 -batch $database

        image=$repoBasePath/$extraColumn
        extra="extra value"

        # Nothing special to do for the central config file since it
        # is the same for both images

        echo "======================================"
        echo "Creating a test Central Repository as an extracolumn build: $image"
        echo "======================================"
    else
        # Note that we don't call resetImageContentDir because we
        # need to keep the previous content which comes from previous steps above
        imageContentPath=$defaultImageContentPath
        database=$defaultDB
        centralConfigFile=$defaultCentralConfigFile

        image=$repoBasePath/$smallImage
        extra=""

        echo "======================================"
        echo "Creating small test Central Repository: $image"
        echo "======================================"
    fi

    for name in ${globalPlugins[*]}; do
        addPlugin $name global true
    done

    for name in ${essentialPlugins[*]}; do
        addPlugin $name global true
        addGroup vmware tanzucli essentials v0.0.1 $name global v0.0.1
        addGroup vmware tanzucli essentials v9.9.9 $name global v9.9.9
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

    for name in ${opsPlugins[*]}; do
        addPlugin $name operations true
    done

    for name in ${pluginUsingSha[*]}; do
        addPlugin $name global true v0.0.1 useSha
        addPlugin $name global true v9.9.9 useSha
    done

    for name in ${multiversionPlugins[*]}; do
        addPlugin $name kubernetes true "v1.9.1 v1.9.2-beta.1 v1.10.1 v1.10.2 v1.11.2 v1.11.3 v2.3.0 v2.3.4 v2.3.5"
    done

    additionalPluginGroupInfo=("shortversion;v0.0.1;v1.9.1" "shortversion;v1.1.0;v1.9" "shortversion;v1.1.0-beta.1;v1.9.2-beta.1" "shortversion;v1.1.1;v1" "shortversion;v1.2.0;v2.3" "shortversion;v9.9.9;v2.3.5")
    for pluginGroupInfo in ${additionalPluginGroupInfo[*]}; do
        groupName=$(echo $pluginGroupInfo | cut -d ";" -f 1)
        groupVersion=$(echo $pluginGroupInfo | cut -d ";" -f 2)
        pluginVersion=$(echo $pluginGroupInfo | cut -d ";" -f 3)
        for name in ${multiversionPlugins[*]}; do
            addGroup vmware tkg $groupName $groupVersion $name kubernetes $pluginVersion
        done
    done

    createCentralConfigFile $centralConfigFile

    # Push small inventory file
    ${dry_run} imgpkg push -i $image -f $imageContentPath --registry-verify-certs=false
    ${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $image
done

# Reset to the default settings to continue the setup
imageContentPath=$defaultImageContentPath
database=$defaultDB
centralConfigFile=$defaultCentralConfigFile
extra=""

echo "======================================"
echo "Creating large test Central Repository: $repoBasePath/$largeImage"
echo "======================================"

# Push generic plugins to get to 100 total plugins in the DB.
# Those plugins will not be installable as we won't push their binaries.
pluginCount=$((${#k8sPlugins[@]} + ${#tmcPlugins[@]} + ${#opsPlugins[@]} + ${#globalPlugins[@]}))
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
${dry_run} imgpkg push -i $repoBasePath/$largeImage -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$largeImage

# Create an additional image as if it was a sandbox build
echo "======================================"
echo "Creating a first test Central Repository as a sandbox build: $repoBasePath/$sanboxImage1"
echo "======================================"

resetImageContentDir

# Create the DB table
touch $database
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

for name in ${opsPlugins[*]}; do
    addPlugin $name operations true v11.11.11
done

createCentralConfigFile $centralConfigFile

# Push sanbox inventory file
${dry_run} imgpkg push -i $repoBasePath/$sanboxImage1 -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$sanboxImage1

# Create a second additional image as if it was a sandbox build
echo "======================================"
echo "Creating a second test Central Repository as a sandbox build: $repoBasePath/$sanboxImage2"
echo "======================================"

resetImageContentDir

# Create the DB table
touch $database
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

for name in ${opsPlugins[*]}; do
    addPlugin $name operations true v22.22.22
done

createCentralConfigFile $centralConfigFile

# Push sandbox inventory file
${dry_run} imgpkg push -i $repoBasePath/$sanboxImage2 -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$sanboxImage2

echo "======================================"
echo "Creating small airgapped test Central Repository: $repoBasePath/$smallAirgappedImage"
echo "======================================"

resetImageContentDir

# Create the DB table
touch $database
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

for name in ${globalPlugins[0]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9
done

for name in ${essentialPlugins[*]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9
    addGroup vmware tanzucli essentials v0.0.1 $name global v0.0.1
    addGroup vmware tanzucli essentials v9.9.9 $name global v9.9.9
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

for name in ${opsPlugins[0]}; do
    addPlugin $name operations true v0.0.1
    addPlugin $name operations true v9.9.9
done

for name in ${pluginUsingSha[0]}; do
    addPlugin $name global true v0.0.1 useSha
    addPlugin $name global true v9.9.9 useSha
done

createCentralConfigFile $centralConfigFile

# Push airgapped inventory file
${dry_run} imgpkg push -i $repoBasePath/$smallAirgappedImage -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$smallAirgappedImage

echo "======================================"
echo "Creating large airgapped test Central Repository: $repoBasePath/$largeAirgappedImage"
echo "======================================"

resetImageContentDir

# Create the DB table
touch $database
cat $ROOT_DIR/create_tables.sql | sqlite3 -batch $database

for name in ${globalPlugins[*]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9
done

for name in ${essentialPlugins[*]}; do
    addPlugin $name global true v0.0.1
    addPlugin $name global true v9.9.9
    addGroup vmware tanzucli essentials v0.0.1 $name global v0.0.1
    addGroup vmware tanzucli essentials v9.9.9 $name global v9.9.9
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

for name in ${opsPlugins[*]}; do
    addPlugin $name operations true v0.0.1
    addPlugin $name operations true v9.9.9
done

for name in ${pluginUsingSha[*]}; do
    addPlugin $name global true v0.0.1 useSha
    addPlugin $name global true v9.9.9 useSha
done

createCentralConfigFile $centralConfigFile

# Push airgapped inventory file
${dry_run} imgpkg push -i $repoBasePath/$largeAirgappedImage -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry  -y $repoBasePath/$largeAirgappedImage

# Create a second large image with a different config file content
airgapWithDiffConfigRepoBasePath=${repoBasePath}2
echo "======================================"
echo "Creating large airgapped test Central Repository with different config: $airgapWithDiffConfigRepoBasePath/$largeAirgappedImage"
echo "======================================"

createCentralConfigFile $centralConfigFile v2.1.0-beta.1

# Push this second airgapped inventory file
${dry_run} imgpkg push -i $airgapWithDiffConfigRepoBasePath/$largeAirgappedImage -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $airgapWithDiffConfigRepoBasePath/$largeAirgappedImage

echo "======================================"
echo "Creating small test Central Repository using SHAs: $repoBasePath/$smallImageUsingSHAs"
echo "======================================"

resetImageContentDir

# Create the DB table
touch $database
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

for name in ${opsPlugins[0]}; do
    addPlugin $name operations true v0.0.1
    addPlugin $name operations true v9.9.9 useSha
done

createCentralConfigFile $centralConfigFile

# Push shas inventory file
${dry_run} imgpkg push -i $repoBasePath/$smallImageUsingSHAs -f $imageContentPath --registry-verify-certs=false
${dry_run} cosign sign --key $ROOT_DIR/cosign-key-pair/cosign.key --allow-insecure-registry -y $repoBasePath/$smallImageUsingSHAs

resetImageContentDir
