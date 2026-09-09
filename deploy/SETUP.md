# Installed AWS/GitHub deployment setup

Configured 9 September 2026 for CodeForAfrica/civicsignal-showcase.

- CloudFormation stack: `civicsignal-web-github-deploy`, eu-west-1/account 499665620971.
- GitHub role: `arn:aws:iam::499665620971:role/civicsignal-web-github-deploy-GitHubDeployRole-uP7qlwpLCOZB`.
- EC2: `i-046fe9e345ae1424e`.
- Instance profile: `civicsignal-web-github-deploy-InstanceProfile-Aqdn6Yzah6NZ`.
- Profile association: `iip-assoc-0b20803f76cacba53`.
- Existing SSM Agent 3.3.4793.0 is active and registered Online. Only this agent
  was restarted for registration; no application restart was required for setup.
- SSM document: `civicsignal-web-deploy`, version 1.
- Root-owned helper: `/usr/local/libexec/civicsignal-deploy`, mode 0755.
- Helper SHA-256: `9c5a49b04d1cc83d368fa097317eb0faf6e015c72e7b405026bd14a73f4c65b7`.

GitHub `production` environment permits only the main branch. Its
`AWS_DEPLOY_ROLE_ARN` variable contains the role above. Repository secrets
`DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` were populated from the existing
provided Docker Hub login. Secret values are not in this repository. A new
limited-scope Docker Hub token was not created; the token-named secret accepts
the supplied registry credential.

Main requires one approving PR review for ordinary collaborators, dismisses
stale approvals, and disallows force-pushes/deletion. Administrators retain
GitHub's override capability. There is no extra production reviewer gate.

IAM simulation verified that SendCommand is allowed for this instance and the
custom deployment document, and denied for AWS-RunShellScript and the separate
CivicSignal API instance. The role cannot install or replace the root-owned
helper. Helper updates remain an administrator operation.

## Verified first deployment

[GitHub Actions run 34388337811](https://github.com/CodeForAfrica/civicsignal-showcase/actions/runs/34388337811)
completed successfully from commit `096c717dbf8a404ddb6186b057eeb56625ebf4b6`.
The first attempt exposed an OIDC subject mismatch before deployment. The
CloudFormation trust policy was corrected to the repository's exact immutable
subject, and rerunning the failed deployment job succeeded using the tested image.

- Image tag: `codeforafrica/civicsignal-web:sha-096c717dbf8a404ddb6186b057eeb56625ebf4b6-34388337811`.
- Deployed digest: `sha256:de578ae15fefccc6d8520b254194cbea41aa9a2ed6aa28d2e066082a0fbdc7c1`.
- SSM command: `98b74666-cc1a-4d64-a84e-a360352b278f`, Success, exit code 0.
- Go tests and both local browser suites passed in the workflow. The host helper
  verified the deployed image, container health, HTTPS, login link and dataset.
- Portal, Explorer, Sources, Topics and database container IDs stayed unchanged.
- Previous website digest, retained for rollback:
  `sha256:2a8d13c22b5ad352a32c721d0315b323ec3c74dfc563bb1f9aab4a8545974139`.

For subsequent releases, publish the rendered changes to main, then choose
**Actions → deploy_to_prod → Run workflow → main** in this repository.
Pushing commits alone does not deploy. Automatic rollback on a failed release
is implemented and unit-tested; a production failure/rollback was not induced.

See [README.md](README.md) for operation, cancellation behavior and rollback.
