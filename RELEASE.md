[comment]: # ( Copyright Red Hat )

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Release Process](#release-process)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Release Process IDP

The XXX is released on an as-needed basis. The process is as follows:

1. An issue is proposing a new release with a changelog since the last release
1. All [OWNERS](OWNERS) must LGTM this release
1. An OWNER runs `git tag -s $VERSION` and inserts the changelog and pushes the tag with `git push $VERSION`
1. The release issue is closed
1. An announcement email is sent to `xxx` with the subject `[ANNOUNCE] xxx $VERSION is released`

# Release Process IDP documentation

In order to keep available previous version of the documetation for customer, a ebook is created at each push of the [idp-mgmt-docs repository](https://github.com/identitatem/idp-mgmt-docs) in [ebook](https://github.com/identitatem/idp-mgmt-docs/tree/gh-pages/ebook.pdf) and is available for download [here](https://identitatem.github.io/idp-mgmt-docs/ebook.pdf). This ebook must be added to a github release as asset.

1. Merge [Upstream repository](https://github.com/identitatem/idp-mgmt-docs-upstream) into this repository
1. Update the version in [idp_mgmt_docs](https://github.com/identitatem/idp-mgmt-docs-upstream/blob/main/idp_mgmt_docs.adoc)
1. Create PR for the above
1. Once the PR is merged, from [idp-mgmt-docs repository](https://github.com/identitatem/idp-mgmt-docs) click `create a releae`
1. Add a the same tag as in the above `Release Process IDP`
1. Fill the Release title and the description
1. Download the [ebook](https://identitatem.github.io/idp-mgmt-docs/ebook.pdf) on your local system.
1. Attach the ebook to the release
1. Click `Publish Release`