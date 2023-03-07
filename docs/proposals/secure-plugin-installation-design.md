# Secure Plugin Installation

## Abstract

The core Tanzu CLI supports a plugin model where the developers of different
Tanzu services can distribute plugins that target functionalities of the
services they own. Plugin developers would publish the plugins to a repository
using a publishing tool.Currently, CLI ensures plugin is coming from a trusted
source, however there is no validation performed by the Tanzu CLI to ensure the
integrity of the plugins downloaded
from the repository.

## Key Concepts

- Repository - plugin repository whose plugin artifacts are indexed in a database OCI image

## Background

Currently, CLI ensures plugin is coming from a trusted source, but it is
highly desirable for CLI to verify the integrity of the plugins downloaded
from the repository

## Goals

This proposal is based on the proposal for the Central repository for Tanzu
plugins. It proposes changes in order to address the below use case

- Secure download of the plugins from the repository by making the below changes
  - Changes to the CLI to validate the Identity of the plugin.

## Non Goals

- **Transport security:**

  This proposal doesn’t address security issues that could result due to
  transport security vulnerabilities(eg: DNS/proxy vulnerabilities that could
  repoint to another registry).

## High-Level Design

This proposal ensures the identity and integrity of the plugin binary to be
installed, but not the transport security or registry. Transport security
shouldn’t be a concern as long as the CLI can verify the identity and integrity
of the plugin binary.

The metadata needed to verify the identity and integrity of every plugin
(ex: digest of plugin binary) in the plugin repository is captured in the
Plugin Database image. The integrity of the plugin database image is in turn
verifiable by first signing the database image with a VMware private key every
time it is updated and then validating with the CLI using VMware’s
corresponding public key.

## Detailed Design

The plugin publishing workflow would update the Plugin Database image
and sign the Plugin Database image using cosign.

### Plugin download and validation by the Tanzu CLI

The Tanzu CLI would perform the validation of the plugin identity during the
“plugin install” command.

Proposed steps for plugin download and validation:

1. The Tanzu CLI would download the Plugin Database Image and ensure its
   integrity by downloading the signed OCI image (containing the signature of
   the plugin database image) and validate the signature using the
   pre-configured public-key
2. Later, the CLI would ensure the digest value of the downloaded plugin
   matches with the digest value of the plugin entry in the database

### Air-gapped environment

The Tanzu CLI will allow air-gapped operators to download the contents of the
repository as a tar file, transfer it to their internal network, and upload it
to a private repository. It includes copying the Plugin Database Image and its
corresponding signed OCI image which would enable CLI to validate the Plugin
Database Image signature.

### Key rotation

#### Unplanned rotation

If the private key is compromised, repository admins should push the updated
signed OCI image (containing the signature of the plugin database image)
created with a new key-pair.

To fix CLI rejecting plugins when validating with expired public key the below
two approaches would be used

1. Produce new core CLI releases that updates the public key used (for all
   the supported versions as per support policy)
2. To enable the existing CLI versions to validate the new signature, users
   can update the tanzu configuration setting with the new public key. (Users
   should download the new public key posted in a well known secure location
   to their local file system and update the tanzu configuration settings with
   proper key)

#### Planned rotation

If the key-pair needs to be changed for some reason (for example to strengthen
the encryption), the key rotation should be handled gracefully. Cosign supports
signing images with multiple keys. So the Plugin Database image should be signed
with 2 keys, so that existing CLI users don't experience a disruptive change
when there is a planned key rotation.
