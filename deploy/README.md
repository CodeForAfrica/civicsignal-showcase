# Manual GitHub deployment through AWS OIDC and SSM

`deploy_to_prod` in this repository replaces the operator's image-push/SSH steps.
It only runs from main via **Actions → deploy_to_prod → Run workflow**. It does
not synchronize rendered output from the build/source repository.

## What it does

1. On one native ARM64 runner, test the Go deployment helper, build the website,
   run both browser suites with the real dataset, and push the tested image to
   Docker Hub under a unique commit/run tag.
2. In the GitHub `production` environment, exchange GitHub OIDC for a temporary
   AWS role session. No SSH key or long-lived AWS access key is used.
3. Send the digest to SSM document `civicsignal-web-deploy`, version 1, targeting
   only EC2 instance `i-046fe9e345ae1424e` in eu-west-1/account 499665620971.
4. The document invokes `/usr/local/libexec/civicsignal-deploy`. It accepts only
   a SHA-256 digest from the fixed `codeforafrica/civicsignal-web` repository,
   checks the current release, runs Dokku, and verifies the new container/image,
   HTTPS health, homepage login link and JSON dataset.
5. A deploy or verification failure triggers redeployment of the previous pinned
   image and verifies it. The action fails even after successful rollback, so
   operators see that the requested release failed. A failed rollback is reported
   explicitly and requires operator intervention.

The app name, repository, instance, region and document are fixed. The workflow
has no role permission to run arbitrary shell documents, start SSM sessions,
change IAM, update SSM documents, or edit domains. IAM GetCommandInvocation uses
Resource `*` because AWS does not offer per-instance resource scoping for that
read operation; do not place sensitive output in other command logs.

## One-time setup (not applied by this code change)

This existing manually-managed host has no instance profile/SSM registration.
The app-specific CloudFormation template owns only the new IAM roles/profile and
SSM document. It does not import, replace or manage the EC2 instance, Dokku apps,
networking, databases, DNS, or existing Pulumi stacks.

1. Have an AWS administrator review and deploy `aws.yml` in eu-west-1. Pass the
   existing GitHub OIDC provider ARN; if one is missing, have the account identity
   administrator establish it first. Do not create a duplicate provider.

   ```sh
   aws cloudformation deploy --region eu-west-1 \
     --stack-name civicsignal-web-github-deploy \
     --template-file deploy/aws.yml --capabilities CAPABILITY_IAM \
     --parameter-overrides GitHubOIDCProviderArn=arn:aws:iam::499665620971:oidc-provider/token.actions.githubusercontent.com
   aws cloudformation describe-stacks --region eu-west-1 \
     --stack-name civicsignal-web-github-deploy --query 'Stacks[0].Outputs'
   ```

2. Recheck the instance's profile before associating the output InstanceProfileName.
   If one now exists, stop and integrate SSM into that role instead of replacing it.

   ```sh
   aws ec2 describe-iam-instance-profile-associations --region eu-west-1 \
     --filters Name=instance-id,Values=i-046fe9e345ae1424e
   aws ec2 associate-iam-instance-profile --region eu-west-1 \
     --instance-id i-046fe9e345ae1424e --iam-instance-profile Name=OUTPUT_PROFILE_NAME
   ```

3. Install/start a current Amazon SSM Agent on the host through your approved
   bootstrap access. Agent must support ENV_VAR interpolation (3.3.2746.0 or newer).
   Allow outbound HTTPS to regional SSM/ssmmessages endpoints and Docker Hub;
   no inbound port needs to be opened. Confirm this exact instance is Online in
   Systems Manager before continuing.

4. Build and install the reviewed Go helper once using administrator access:

   ```sh
   cd deploy
   go test ./...
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o civicsignal-deploy ./cmd/deploy
   # Transfer this reviewed binary to the host using approved bootstrap access.
   # On the host:
   sudo install -d -o root -g root -m 0755 /usr/local/libexec
   sudo install -o root -g root -m 0755 /tmp/civicsignal-deploy /usr/local/libexec/civicsignal-deploy
   ```

   Do not download an arbitrary branch script at deployment time. Future helper
   changes require the same administrator review/install. It needs the existing
   `/usr/bin/dokku` and `/usr/bin/docker`. SSM document version is pinned to 1;
   document changes need review and an explicit workflow version update.

5. Create the GitHub environment `production`. Restrict deployment branches to
   **main only**. Protect main and workflow/deploy files from unreviewed edits;
   reviewers can optionally be required for production. This branch restriction
   is mandatory: the OIDC environment subject does not itself contain a branch.
   Set environment variable `AWS_DEPLOY_ROLE_ARN` to the stack's DeployRoleArn.
   Set repository Actions secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (a
   dedicated write-scoped Docker Hub token). The build job uses repository secrets;
   it does not have AWS OIDC permissions. The deployment job has no Docker Hub
   secret. The existing public image repository can be pulled without login.

6. Check the deployed baseline is a healthy pinned image from this repository.
   Run the first workflow under observation, then verify its SSM command and the
   website. AWS template syntax, local unit tests and compilation do not replace
   this first live integration test.

## Cancellation, concurrency and recovery

GitHub concurrency serializes production runs; `cancel-in-progress: false`
prevents a newer request interrupting a release. The helper additionally uses a
host-level nonblocking flock. Manual deployments should not run concurrently.
A user cancelling a workflow does NOT cancel its SSM command; the host finishes
verification/rollback independently. Inspect the command ID in the Actions log
before retrying. The helper and SSM have bounded execution times. A machine crash
or agent termination can still interrupt rollback; use MANUAL_DEPLOY.md for
recovery. No workflow is authorized to delete the old app or its data.

## Cost and scope

Manual dispatch only: no push, PR, schedule, or workflow_run trigger. One native
`ubuntu-24.04-arm` build/test runner (25-minute limit) and one `ubuntu-latest`
deployment runner (45-minute limit); no matrix, QEMU or multi-architecture fanout.
No GitHub artifact upload or build cache is configured. Each successful build
adds one image tag to Docker Hub; retain known-good release digests and manage
old tags separately. SSM polling consumes deployment-runner minutes. No live
workflow has been triggered by this code change.

Only CivicSignal's website deployment is automated. Existing portal, tools,
certificate renewal and DNS stay as configured. The action does not renew or
transfer domains. For fallback instructions see ../MANUAL_DEPLOY.md.

References: [AWS ENV_VAR parameters](https://docs.aws.amazon.com/systems-manager/latest/userguide/documents-syntax-data-elements-parameters.html),
[GitHub OIDC on AWS](https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-aws).
