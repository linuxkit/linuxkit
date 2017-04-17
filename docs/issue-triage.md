Triaging of issues
------------------

Triage provides an important way to contribute to an open source project.
Triage helps ensure issues resolve quickly by:

- Describing the issue's intent and purpose is conveyed precisely. This is
  necessary because it can be difficult for an issue to explain how an end user
  experiences a problem and what actions they took.
- Giving a contributor the information they need before they commit to
  resolving an issue.
- Lowering the issue count by preventing duplicate issues.
- Streamlining the development process by preventing duplicate discussions.

If you don't have time to code, consider helping with triage. The community
will thank you for saving them time by spending some of yours.

### 1. Ensure the issue contains basic information

Before triaging an issue very far, make sure that the issue's author provided
the standard issue information. This will help you make an educated
recommendation on how this to categorize the issue.

If you cannot triage an issue using what its author provided, explain kindly to
the author that they must provide additional information to clarify the problem.

If the author does not respond requested information within the timespan of a
week, close the issue with a kind note stating that the author can request for
the issue to be reopened when the necessary information is provided.

### 2. Classify the Issue

An issue can have multiple of the following labels. Typically, a properly
classified issue should have:

- One label identifying its kind (`kind/*`).
- One or multiple labels identifying the functional areas of interest (`area/*`).
- Where applicable, one label categorizing its difficulty (`exp/*`).

#### Issue kind

| Kind             | Description                                                                                                                     |
|------------------|---------------------------------------------------------------------------------------------------------------------------------|
| kind/bug         | Bugs are bugs. The cause may or may not be known at triage time so debugging should be taken account into the time estimate.    |
| kind/enhancement | Enhancements are not bugs or new features but can drastically improve usability or performance of a project component.          |
| kind/feature     | Functionality or other elements that the project does not currently support.  Features are new and shiny.                       |
| kind/question    | Contains a user or contributor question requiring a response.                                                                   |

#### Functional area

| Area                      |
|---------------------------|
| area/build                |
| area/cli                  |
| area/containerd           |
| area/docs                 |
| area/kernel               |
| area/logging              |
| area/networking           |
| area/security             |
| area/testing              |
| area/time                 |
| area/unikernel            |
| area/usability            |

#### Platform

| Platform                  |
|---------------------------|
| platform/arm              |
| platform/aws              |
| platform/azure            |
| platform/gcp              |
| platform/osx              |
| platform/windows          |


#### Experience level

Experience level is a way for a contributor to find an issue based on their
skill set.  Experience types are applied to the issue or pull request using
labels.

| Level            | Experience level guideline                                                                                                                                            |
|------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| exp/beginner     | New to LinuxKit, and is looking to help while learning the basics.                                                                                                    |
| exp/intermediate | Comfortable with the project and understands the core concepts, and looking to dive deeper into the project.                                                          |
| exp/expert       | Proficient with the project and has been following, and active in, the community to understand the rationale behind design decisions and where the project is headed. |

As the table states, these labels are meant as guidelines.

#### Triage status

To communicate the triage status with other collaborators, you can apply status
labels to issues. These labels prevent duplicating effort.

| Status                        | Description                                                                                                                                                                 |
|-------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| status/confirmed              | You triaged the issue, and were able to reproduce the issue. Always leave a comment how you reproduced, so that the person working on resolving the issue has a way to set up a test-case.
| status/accepted               | Apply to enhancements / feature requests that we think are good to have. Adding this label helps contributors find things to work on.
| status/more-info-needed       | Apply this to issues that are missing information (e.g. no steps to reproduce), or require feedback from the reporter. If the issue is not updated after a week, it can generally be closed.
| status/needs-attention        | Apply this label if an issue (or PR) needs more eyes.

### 3. Prioritizing issue

When, and only when, an issue is attached to a specific milestone, the issue can be labeled with the
following labels to indicate their degree of priority (from more urgent to less urgent).

| Priority    | Description                                                                                                                       |
|-------------|-----------------------------------------------------------------------------------------------------------------------------------|
| priority/P0 | Urgent: Security, critical bugs, blocking issues. P0 basically means drop everything you are doing until this issue is addressed. |
| priority/P1 | Important: P1 issues are a top priority and a must-have for the next release.                                                     |
| priority/P2 | Normal priority: default priority applied.                                                                                        |
| priority/P3 | Best effort: those are nice to have / minor issues.                                                                               |

And that's it. That should be all the information required for a new or
existing contributor to come in a resolve an issue.
