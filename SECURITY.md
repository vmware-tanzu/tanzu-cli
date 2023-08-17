# Security Release Process

Tanzu CLI is the primary product surface that enables other products leveraging the Tanzu portfolio to extend the CLI in a powerful way to meet customer needs.

The community has adopted this security disclosure and response policy to ensure we responsibly handle critical issues.

## Supported Versions

Minor versions of the Tanzu CLI are expected to be released every three months. These releases, along with possible patch releases will be supported under this [support policy](https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/full/policy.md#tanzu-cli-support-policy).

## Reporting a Vulnerability - Private Disclosure Process

Security is of the highest importance and all security vulnerabilities or suspected security vulnerabilities should be reported to Tanzu CLI privately, to minimize attacks against current users of Tanzu CLI before they are fixed. Vulnerabilities will be investigated and patched on the next patch (or minor) release as soon as possible. This information could be kept entirely internal to the project.

If you know of a publicly disclosed security vulnerability for Tanzu CLI, please **IMMEDIATELY** contact the VMware Security Team (security@vmware.com). The use of encrypted email is encouraged. The public PGP key can be found [here](https://kb.vmware.com/kb/1055). **IMPORTANT: Do not file public issues on GitHub for security vulnerabilities**

To report a vulnerability or a security-related issue, please contact the VMware email address with the details of the vulnerability. The email will be fielded by the VMware Security Team and then shared with the Tanzu CLI maintainers who have committer and release permissions. Emails will be addressed within 3 business days, including a detailed plan to investigate the issue and any potential workarounds to perform in the meantime. Do not report non-security-impacting bugs through this channel. Use [GitHub issues](https://github.com/vmware-tanzu/tanzu-cli/issues) instead.

## Proposed Email Content

Provide a descriptive subject line and in the body of the email include the following information:

* Basic identity information, such as your name and your affiliation or company.
* Detailed steps to reproduce the vulnerability  (POC scripts, screenshots, and logs are all helpful to us).
* Description of the effects of the vulnerability on Tanzu CLI and the related hardware and software configurations, so that the VMware Security Team can reproduce it.
* How the vulnerability affects Tanzu CLI usage and an estimation of the attack surface, if there is one.
* List other projects or dependencies that were used in conjunction with Tanzu CLI to produce the vulnerability.

## When to report a vulnerability

* When you think Tanzu CLI has a potential security vulnerability.
* When you suspect a potential vulnerability but you are unsure that it impacts Tanzu CLI.
* When you know of or suspect a potential vulnerability on another project that is used by Tanzu CLI.

## Patch, Release, and Disclosure

The VMware Security Team will respond to vulnerability reports as follows:

1. The Security Team will investigate the vulnerability and determine its effects and criticality.
2. If the issue is not deemed to be a vulnerability, the Security Team will follow up with a detailed reason for rejection.
3. The Security Team will initiate a conversation with the reporter within 3 business days.
4. If a vulnerability is acknowledged and the timeline for a fix is determined, the Security Team will work on a plan to communicate with the appropriate community, including identifying mitigating steps that affected users can take to protect themselves until the fix is rolled out.
5. The Security Team will also create a [CVSS](https://www.first.org/cvss/specification-document) using the [CVSS Calculator](https://www.first.org/cvss/calculator/3.0). The Security Team makes the final call on the calculated CVSS; it is better to move quickly than making the CVSS perfect. Issues may also be reported to [Mitre](https://cve.mitre.org/) using this [scoring calculator](https://nvd.nist.gov/vuln-metrics/cvss/v3-calculator). The CVE will initially be set to private.
6. The Security Team will work on fixing the vulnerability and perform internal testing before preparing to roll out the fix.
7. The Security Team will provide early disclosure of the vulnerability by emailing the Tanzu CLI Distributors (`tanzu-cli-distributors@googlegroups.com`) mailing list. Distributors can initially plan for the vulnerability patch ahead of the fix, and later can test the fix and provide feedback to the Tanzu CLI team. See the section **Early Disclosure to Tanzu CLI Distributors List** for details about how to join this mailing list.
8. A public disclosure date is negotiated by the VMware Security Team, the bug submitter, and the distributors list. We prefer to fully disclose the bug as soon as possible once a user mitigation or patch is available. It is reasonable to delay disclosure when the bug or the fix is not yet fully understood, the solution is not well-tested, or for distributor coordination. The timeframe for disclosure is from immediate (especially if it’s already publicly known) to a few weeks. For a critical vulnerability with a straightforward mitigation, we expect the report date for the public disclosure date to be on the order of 14 business days. The VMware Security Team holds the final say when setting a public disclosure date.
9. Once the fix is confirmed, the Security Team will patch the vulnerability in the next patch or minor release, and backport a patch release into all earlier supported releases. Upon release of the patched version of Tanzu CLI, we will follow the **Public Disclosure Process**.

## Public Disclosure Process

The Security Team publishes a public advisory to the Tanzu CLI community via GitHub. In most cases, additional communication via Slack, Twitter, mailing lists, blog and other channels will assist in educating Tanzu CLI users and rolling out the patched release to affected users.

The Security Team will also publish any mitigating steps users can take until the fix can be applied to their Tanzu CLI instances. Tanzu CLI distributors will handle creating and publishing their own security advisories.

## Mailing lists

* Use security@vmware.com to report security concerns to the VMware Security Team, who uses the list to privately discuss security issues and fixes prior to disclosure.
* Join the Tanzu CLI Distributors (`tanzu-cli-distributors@googlegroups.com`) mailing list for early private information and vulnerability disclosure. Early disclosure may include mitigating steps and additional information on security patch releases. See below for information on how Tanzu CLI distributors or vendors can apply to join this list.

## Early Disclosure to Tanzu CLI Distributors List

The private list is intended to be used primarily to provide actionable information to multiple distributor projects at once. This list is not intended to inform individuals about security issues.

## Membership Criteria

To be eligible to join the Tanzu CLI Distributors mailing list, you should:

1. Be an active distributor of the Tanzu CLI.
2. Have a user base that is not limited to your own organization.
3. Have a publicly verifiable track record up to the present day of fixing security issues.
4. Not be a downstream or rebuild of another distributor.
5. Be a participant and active contributor in the Tanzu CLI community.
6. Accept the Embargo Policy that is outlined below.
7. Have someone who is already on the list vouch for the person requesting membership on behalf of your distribution.

**The terms and conditions of the Embargo Policy apply to all members of this mailing list. A request for membership represents your acceptance to the terms and conditions of the Embargo Policy.**

## Embargo Policy

The information that members receive on the Tanzu CLI Distributors mailing list must not be made public, shared, or even hinted at anywhere beyond those who need to know within your specific team, unless you receive explicit approval to do so from the VMware Security Team. This remains true until the public disclosure date/time agreed upon by the list. Members of the list and others cannot use the information for any reason other than to get the issue fixed for your respective distribution's users.

Before you share any information from the list with members of your team who are required to fix the issue, these team members must agree to the same terms, and only be provided with information on a need-to-know basis.

In the unfortunate event that you share information beyond what is permitted by this policy, you must urgently inform the VMware Security Team (security@vmware.com) of exactly what information was leaked and to whom. If you continue to leak information and break the policy outlined here, you will be permanently removed from the list.

## Requesting to Join

Send new membership requests to `tanzu-cli-distributors@googlegroups.com`. In the body of your request please specify how you qualify for membership and fulfill each criterion listed in the Membership Criteria section above.

## Confidentiality, integrity and availability

We consider vulnerabilities leading to the compromise of data confidentiality, elevation of privilege, or integrity to be our highest priority concerns. Availability, in particular in areas relating to DoS and resource exhaustion, is also a serious security concern. The VMware Security Team takes all vulnerabilities, potential vulnerabilities, and suspected vulnerabilities seriously and will investigate them in an urgent and expeditious manner.

Note that we do not currently consider the default settings for Tanzu CLI to be secure-by-default. It is necessary for operators to explicitly configure settings, role based access control, and other resource related features in Tanzu CLI to provide a hardened CLI environment. We will not act on any security disclosure that relates to a lack of safe defaults. Over time, we will work towards improved safe-by-default configuration, taking into account backwards compatibility.
