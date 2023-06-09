# Tanzu CLI policy

This document defines the version policy, plugin/tanzu-cli compatibility policy, support policy, and deprecation policy of the Tanzu CLI.

## Tanzu CLI Version policy

Tanzu CLI and [Tanzu Plugin Runtime Library](https://github.com/vmware-tanzu/tanzu-plugin-runtime) use a three-part version scheme: `<major>.<minor>.<patch>`.
Changes to each of these versions have a very specific meaning:

- major: indicates significant feature changes and possible breaking changes
- minor: indicates functional changes (primary feature delivery)
- patch: indicates minimal risk changes (critical security vulnerabilities and bug fixes)

Given this scheme, the primary vehicle for changes is minor releases. The deprecation policy assumes that all modifications and removals happen as part of minor releases (possibly rolled up into a major release).

This section specifies the version, support, and deprecation policies for Alpha, Beta, and GA releases.

## Tanzu-Plugin-Runtime/Tanzu-CLI Compatibility policy

- Each Tanzu Core CLI release should be compatible with all the existing plugins at the time of release and also with plugins developed with any future patch of the existing minor releases of Tanzu Plugin Runtime.
  - For example, the below table shows the compatible Tanzu Plugin Runtime versions with different releases of Tanzu CLI.
  - ***Note***: In the below table v0.11, 0.25, 0.28, 0.29 versions are `legacy` library versions from Tanzu-Framework and `x` is any existing or future patch version.

    | Tanzu CLI Version | Compatible Tanzu Plugin Runtime Versions |
    |-------------------|------------------------------------------|
    | v0.90.x           | v0.11.x                                  |
    |                   | v0.25.x                                  |
    |                   | v0.28.x                                  |
    |                   | v0.29.x                                  |
    |                   | v0.90.x                                  |

- ***Important***: Based on this compatibility policy, the user can always upgrade the Tanzu CLI to the latest version without worrying about existing plugin compatibility.
- This means any change in the contract between Tanzu CLI and Tanzu Plugin Runtime must be done in a backward-compatible manner.
- ***FAQ***:
  - What happens if I upgrade to the CLI v0.90.0 and use a plugin developed with Plugin Runtime v0.28.0?
    - As CLI is always compatible with all existing plugins, plugins developed with Plugin Runtime v0.28.0 should continue to work.
  - If I have developed a plugin with (v0.11, 0.25, 0.28, 0.29) of Tanzu Plugin Runtime (from Tanzu-Framework repository), Will my plugins be compatible with v0.90 Tanzu CLI?
    - Yes. As shown in the above diagram and the backward compatibility guarantees provided with the v0.90 release of Tanzu CLI, all plugins developed with (v0.11, 0.25, 0.28, 0.29) of Tanzu Plugin Runtime will be compatible with v0.90 Tanzu CLI.
  - When will my plugin be incompatible with the CLI?
    - The user may be using a CLI version that is old and some new functionality implemented in a plugin with newer Plugin Runtime is not compatible with the old CLI. However, the old CLI will still be able to invoke the plugin with all the compatible features. Also, upgrading CLI to a newer version should resolve this new feature incompatibility issue.
    - Example: If a plugin is developed with Tanzu Plugin Runtime v1.3.1 which introduced a new feature X which works in combination with a newer version of Tanzu CLI then if the user is using old Tanzu CLI v1.2.0 (which doesn’t support this new feature X introduced in Plugin Runtime v1.3.1) while invoking a plugin, that new functionality might not work when using old CLI. However, other functionalities of that plugin will continue to work as expected with the old CLI. This will make the new feature in the plugin incompatible with the installed CLI. However, the user can upgrade the CLI to the latest v1.3.0 version to make the new feature compatible again. This type of situation will always be communicated with the above table when this situation arises.

## Tanzu CLI Support policy

Tanzu Core CLI and Tanzu Plugin Runtime are bound to provide security fixes and bug fixes as soon as possible.

Starting with v1.0 releases of Tanzu Core CLI and Tanzu Plugin Runtime, only the latest available minor release will be supported. This means that

- Any bug fixes and security fixes will be done as a patch release of the latest minor release.
- Once the new minor version gets released, the previous minor version goes out of support.

Important: Based on the backward compatibility guarantees and deprecation policy listed above, supporting only the latest minor version makes sense because of the below reasons:

- Users can easily upgrade Tanzu CLI to newer versions considering it will support all older versions of the plugins.
- Plugin developers can easily upgrade to a newer version of Tanzu Plugin Runtime considering the backward compatibility guarantees of public APIs. More details about this are available in the Deprecation Policy section below.

Example:

- If the latest available version of Tanzu Plugin Runtime is v1.1.0, and if any security issues are encountered, it will be fixed and released as v1.1.1
- Once the v1.2.0 release is out. v1.1 goes out of support and any new security fixes or bug fixes will be done as part of the v1.2 release cadence as v1.2.1, v1.2.2, etc until v1.3.0 is out.

***FAQ***:

- As a plugin developer, If a bug is encountered in Tanzu Plugin Runtime, what will it take to get that bug fixed as part of my plugin?
  - A new minor or patch release will be done for Tanzu Plugin Runtime with the bug fixed. As a plugin developer, you can use this newly released version of Tanzu Plugin Runtime to build the plugin.
- As a user, if a bug is encountered in Tanzu Core CLI, what will it take to get that bug fixed?
  - A new minor or patch release will be done for Tanzu Core CLI. Users can install a newer version of the CLI to get the bug fixed.
- As a plugin developer, If I have created my plugin with v1.1.0. Now Once the v1.2.0 release is out, v1.1 goes out of support. If a bug or CVE is encountered in v1.1 what will happen?
  - As mentioned above in the example, a new v1.2.1 release will be published and plugin developers will need to consume this new v1.2.1 release to get the bug or CVE fix.

Important: The team will continue to support v0.25 and v0.28 releases that are there on the Tanzu-Framework repository to do any bug fixes or security fixes during the support window of these releases.

- As a Tanzu CLI user, When I switch from pre-v1.0 CLI to v1.0 CLI, what changes should I expect?
  - When using v1.0 CLI, all the existing plugins should continue to work and will be compatible with the new CLI.
  - However, users should expect some potential changes in the UX around plugin lifecycle operation. The change in the UX will be kept as minimal as possible.

## Tanzu CLI Deprecation policy

Since the Tanzu CLI offers a varied set of functionality, any breaking changes or removal of functionality must follow a clear deprecation policy.

Any CLI feature or API can be marked as deprecated as part of a minor release but cannot be removed before the deprecation time window once the feature is marked as deprecated.

### Tanzu Plugin Runtime

- Deprecation window: 12 months

- ***FAQ***:

  - What happens to the existing plugins that are using the deprecated API during the deprecation window?
    - If the existing plugin is using the deprecated functionality, that plugin should continue to work without any issues
    - Once the plugin is upgraded to a newer version of the Tanzu Plugin Runtime, the developer should switch to an alternative API. Usage of deprecated API is strongly discouraged
  - What happens to the existing plugins that are using the deprecated API after the deprecation window and when the deprecated API is removed from Tanzu Plugin Runtime?
    - If the existing plugin is using the deprecated functionality, that plugin should continue to work without any issues
    - Once the plugin is upgraded to a newer version of the Tanzu Plugin Runtime, the developer MUST switch to an alternative API as the deprecated API has been removed
  - As a plugin developer, what should I do when an API is marked as deprecated as part of Tanzu Plugin Runtime?
    - As mentioned above, the deprecated API will be removed from the code after 12 months. So, as a plugin developer, you should plan and switch to an alternate API or functionality before the deprecated API is removed from the library.
    - Usage of deprecated API is strongly discouraged
  - If a bug/CVE is filed against the deprecated code/API, how will it be tackled?
    - If the deprecated code/API is still under the deprecation window, a new patch release with the bug-fix/CVE -fix will be done.
    - If the deprecated code/API is outside the deprecation window, the deprecated code/API will be removed from the library in subsequent releases and the fix won’t be implemented.
  - Example:
    - If an API is marked as deprecated starting with the v1.0.0 (June 2023) release, that API will be available and functional for all the releases happening until May 2024. After that, as part of the next minor release of Tanzu Plugin Runtime, the API will be removed and will no longer be available for developers to consume.

### Tanzu Core CLI

- Deprecation window: 6 months

- ***FAQ***:
  - What happens if the user uses a deprecated feature during the deprecation window?
    - The deprecated feature should continue to work as expected with a potential warning message about deprecated feature usage
    - At this time, the user is encouraged to update any scripts or automation to not use deprecated features and use any alternatives if available.
  
  - What happens if the user uses a deprecated feature after the deprecation window?
    - The deprecated feature might throw an error or it might not exist in the newer version of the Tanzu CLI after the deprecation window. So, the user will be forced to update the scripts or automation to use alternatives.
  
  - Example:
    - If a feature is marked as deprecated starting with the v1.0.0 (June 2023) release, that API will be available and functional for all the releases happening until December 2023. After that, as part of the next minor release of Tanzu CLI, the feature will be removed and will no longer be available for users to consume.

Important: Between the v0.28.0 and v1.0.0 releases, the team will be updating(adding/removing) the Public APIs and commands available in the Tanzu Plugin Runtime Library and Tanzu Core CLI respectively to make them more stable.

- This means, Plugin developers might be forced to update/adjust their code to use the new Public API interface available with the v1.0.0 Tanzu Plugin Runtime Library.
- Also, Users might be forced to update their scripts as the v1.0.0 CLI may have removed/renamed/changed some commands.
- Note: The required updates will be documented, and we expect the number of changes required to be minimal.

#### Deprecation Notices & Communications

- Document as part of Release Notes
- Tanzu CLI documentation should specify the alternatives available along with the deprecation notice.

#### Checklist for any Feature/API deprecation

- Documentation and release notes should state that the Feature/API has been deprecated and describe the alternative if applicable.
- It is expected that the Feature/API will continue to operate without change for all the releases that happen during the deprecation time window.
- Use of the deprecated Feature/API should generate feedback indicating that the Feature/API is deprecated.

#### Checklist for any Feature/API removal after the deprecation time window

- Documentation and release notes should state that the Feature/API has been removed.
