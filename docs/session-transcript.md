# Session Transcripts

Record of the working sessions for the cloud engineer assessment, in chronological order. The user prompts have been lightly edited for grammar and to keep the tone consistent and coherent; the substance is unchanged.

---

## Session 1: Build, Deploy, and Verify the Media Processing Service

### Project setup and tooling

1. **Prompt:** Please help me install Git, Terraform, and Docker on this machine.
   - **Outcome:** Checked installed tooling, then installed Terraform and Docker Desktop via winget (Git was already present). Confirmed WSL2 was available for Docker's backend.

2. **Prompt:** Let's lay out the repo as a full application with a proper Terraform folder structure.
   - **Outcome:** Agreed on a common monorepo layout where the application lives alongside the Terraform infrastructure code.

3. **Prompt:** I'm planning an `app` and a `worker`: the app serves the public API and the worker handles the background processing.
   - **Outcome:** Confirmed the split between the web-facing service and the job consumer, and agreed the worker was justified for this workload.

4. **Prompt:** It's a public API that could use ffmpeg to process video editing, so I think the worker is needed.
   - **Outcome:** Confirmed the architecture direction and resolved a few clarifying points (cloud provider, language, compute). Choices recorded: AWS, Go, ECS/Fargate.

### Planning

5. **Prompt:** Let me walk through the architecture before we build it:
   - **Main choices:** ECS Fargate in private subnets behind a public ALB, modular Terraform per concern, RDS Postgres serving as both the database and the job queue, S3 for media, ECR for images, Secrets Manager for credentials, CloudWatch for centralized logs.
   - **Secrets:** the DB password is generated at apply time and injected into the containers through ECS secrets from Secrets Manager, so nothing sensitive sits in images or config.
   - **Worker communication:** the API and worker don't talk directly. The API inserts job rows into Postgres, and the worker polls the jobs table with `FOR UPDATE SKIP LOCKED` to claim work.
   - **Deploys and rollbacks:** versioned image tags pinned in `terraform.tfvars`, so a redeploy or rollback is just changing the tag and reapplying, which produces a new task definition revision.
   - **Outcome:** Locked in the architecture and used it to shape the three-stage plan and the module layout.

6. **Prompt:** Good job on the plan. I'd like to execute it in multiple stages: first plan the folder structure and `git init` with a first commit using conventional commit messages (`feat:`, `fix:`, `chore:`, ...), then commit the app, then commit Terraform. AWS isn't configured on this machine yet, so please write a setup guideline and the Dockerfiles now. Here is the detailed app plan for an async media processing service:

   - Upload a file, apply operations one at a time (each building on the last), download the result, with async processing via a worker pool (similar to ezgif).
   - **Flow:** `POST /upload` returns `session_id` and `version: 1`; `POST /sessions/{id}/apply` queues an operation on the current version; `GET /sessions/{id}/status` polls until done; `GET /sessions/{id}/download` returns a pre-signed S3 URL; repeat `apply` to chain more operations.
   - **Operations (via ffmpeg):** convert format, resize, compress, grayscale filter.
   - **API surface:** `POST /upload`, `POST /sessions/{id}/apply`, `GET /sessions/{id}/status`, `GET /sessions/{id}/download`. The API never runs ffmpeg; it only writes to Postgres and returns.
   - **Data model (PostgreSQL):** `sessions` (id, current_version, created_at) and `jobs` (id, session_id, operation, params, input_version, output_version, status, created_at). S3 key pattern: `sessions/{id}/v{version}.{ext}`.
   - **Worker:** loop claims a `pending` job using `SELECT ... FOR UPDATE SKIP LOCKED`, then download, ffmpeg, upload, update `session.current_version`, mark done; otherwise sleep. SKIP LOCKED gives safe concurrent workers with no double-processing; `desired_count = 2`.
   - **Infra mapping:** public ALB, ECS Fargate (private subnets) for API/worker, RDS PostgreSQL (private), S3, ECR, Secrets Manager to ECS, IAM roles with tiered security groups, CloudWatch logs.
   - **Key decisions:** one Postgres database for both state and job queuing; Fargate rather than Lambda (CPU-heavy ffmpeg); versioned image tags to enable rollback via task definition revisions; worker 1vCPU/2GB, API 0.25vCPU/0.5GB.
   - **Outcome:** Produced a comprehensive three-stage plan with the commit sequence, folder tree, Dockerfile strategy, and AWS setup guidance.

7. **Prompt:** Please execute Stage 1.
   - **Outcome:** Scaffolded the folder structure, created `.gitignore`, ran `git init`, and committed `chore: scaffold project structure`.

8. **Prompt:** One correction: the region is EU, not US. Please proceed to Stage 2.
   - **Outcome:** Presented the Stage 2 plan (Go app + Dockerfiles), noting the EU region default (eu-central-1) for later stages.

9. **Prompt:** Please go ahead.
   - **Outcome:** Installed Go, built the API, worker, shared packages, and multi-stage Dockerfiles; ran `go build` and `go vet`; committed `feat: add async media processing api and worker`.

### AWS configuration

10. **Prompt:** Could you provide an AWS setup guide?
    - **Outcome:** Created `docs/AWS_SETUP.md` covering CLI install, IAM user, `aws configure`, and Terraform preparation steps.

11. **Prompt:** Actually, please remove that commit. It was meant as an answer for me, not something to live in the repo.
    - **Outcome:** Removed the commit and its file.

12. **Prompt:** AWS isn't installed yet, could you install it for me?
    - **Outcome:** Installed the AWS CLI and verified it.

13. **Prompt:** I'm not able to run `aws configure` myself.
    - **Outcome:** Offered to complete the configuration and asked for the access key and secret.

14. **Prompt:** I've made the access key and secret available to you. Please finish the AWS configuration.
    - **Outcome:** Completed `aws configure` with eu-central-1 as the default region.

### Infrastructure (Stage 3)

15. **Prompt:** OK, please go ahead with Stage 3 and prepare the Terraform.
    - **Outcome:** Created the state bucket and lock table, wrote the Terraform modules (network, storage, database, secrets, IAM, cluster, ALB, ECR, service) plus dev/prod environments, added a `/healthz` endpoint for the ALB health check, validated, and committed `feat: add terraform infrastructure`.

16. **Prompt:** Are these requirements met?
    - Do not store secrets in Git, container images, or plaintext configuration.
    - The worker, database, and queue must not be publicly accessible.
    - Apply reasonable least-privilege IAM and network rules.
    - Use versioned container tags rather than relying only on `latest`.
    - The Terraform should pass `terraform fmt` and `terraform validate`.
    - **Outcome:** Audited each requirement. Found no secrets in Git, a private worker/database, and reasonable IAM and SG tiering; flagged that `terraform fmt` needed a fix on `backend.tf` and that versioned tags should be strengthened.

17. **Prompt:** OK, please proceed with the fixes.
    - **Outcome:** Ran `terraform fmt`, made the prod ECR repositories IMMUTABLE, and pinned versioned image tags (1.0.0). Both environments passed fmt and validate.

### Deployment and testing

18. **Prompt:** We need to test this properly. Walk me through the testing options and I'll pick one.
    - **Outcome:** Presented two options: a local docker-compose harness (Postgres + MinIO) or a full AWS deployment (apply, build/push, test against the ALB).

19. **Prompt:** Let's go with the full AWS deployment.
    - **Outcome:** Applied the Terraform, logged into ECR, built and pushed images, and prepared to test the live API.

20. **Prompt:** Please continue.
    - **Outcome:** Resolved partial-apply state issues (stale lock, orphaned ALB and route tables, RDS free-tier constraints, CPU/memory type fixes) and completed the apply.

21. **Prompt:** Let's pick this up where it left off and continue.
    - **Outcome:** Imported the remaining state, re-applied, forced new ECS deployments, and confirmed the live `/healthz` and the full upload, process, download flow.

22. **Prompt:** Please write me the commands to test using the video in the demo folder.
    - **Outcome:** Located the demo video and prepared test commands.

23. **Prompt:** Could you write a bash script inside the demo folder, run it, and, if it succeeds, add the outputs to the gitignore of that app folder?
    - **Outcome:** Created and ran a bash test script; fixed a URL-decoding bug (`\u0026`) in the download step.

24. **Prompt:** Actually, I meant just a few commands to test rather than waiting for every operation. The gitignore part was also overlooked, and since this is Windows, a `.bat` file would be more appropriate.
    - **Outcome:** Replaced the bash script with a quick `test-demo.bat` and added `app/.gitignore` for the demo outputs.

25. **Prompt:** Yes, and could the `.bat` poll the status frequently, printing the status every second?
    - **Outcome:** Added a 1-second polling loop.

26. **Prompt:** Do you see the problem with the errors?
    - **Outcome:** Diagnosed the `timeout` input-redirection error and replaced it with a `ping -n 2` delay.

27. **Prompt:** Is it really that long? Can you confirm the workers are processing correctly? I'd also like to push two videos in parallel.
    - **Outcome:** Checked the CloudWatch worker logs, confirmed both workers process jobs correctly in parallel (about two minutes per operation on a 30MB video), and demonstrated two jobs running concurrently.

28. **Prompt:** Nice. Please update the `.bat` to poll every 10 seconds and print the expected time for each operation at the start.
    - **Outcome:** Updated polling to 10 seconds with an expected-time table per operation and verified the run.

29. **Prompt:** Continue with any next steps, or stop and ask for clarification if anything is unclear.
    - **Outcome:** Verified the updated script end to end.

30. **Prompt:** Please change the script to use the snail and squirrel videos.
    - **Outcome:** Rewrote the script to upload both videos and process them in parallel.

31. **Prompt:** Where are the outputs? Being gitignored doesn't mean the files were removed.
    - **Outcome:** Explained that the script only printed download URLs without saving files, and offered to add a real download step plus clean up the git index.

---

## Session 2: Architecture Review

### Review and confirmation

1. **Prompt:** Would you verify this Terraform configuration covers the following?
   - Networking and subnets
   - A public entry point for the API
   - Container execution for the API and worker
   - A database or queue
   - Object storage
   - Container image storage
   - Secure handling of application secrets
   - Basic IAM roles and security groups
   - Centralized application logs
   - **Outcome:** Static review of all modules confirmed all nine requirements are covered, and flagged minor issues (global S3 bucket naming, HTTP-only ALB, single NAT gateway, `use_lockfile` requiring Terraform 1.9+, no `force_new_deployment`).

2. **Prompt:** Please review the repo and lay out the architecture on these five points: main architecture choices, how the containers receive secrets securely, how the worker communicates with the API, database, or queue, how container versions are deployed and rolled back, and the three checks I'd perform if the API works but the worker processes no jobs.
   - **Outcome:** Reviewed the modules and app code and confirmed each point with references (Fargate in private subnets, Secrets Manager via ECS `valueFrom`, Postgres-as-queue with `FOR UPDATE SKIP LOCKED`, tag-driven rolling deploy and rollback, and the three worker-troubleshooting checks).

3. **Prompt:** Now, is this production ready, or is it dev only?
   - **Outcome:** Gave a concise verdict on readiness with the key blockers.

4. **Prompt:** I'd like to move to production. Please plan what I need to do.
   - **Outcome:** Outlined the production hardening plan and asked clarifying questions on the big tradeoffs (domain/HTTPS, SQS vs DB-as-queue, CI/CD, multi-AZ RDS).

---

## Session 3: Documentation and README

### Key management review

1. **Prompt:** Is this repo dev only? How are keys and things managed?
   - **Outcome:** Confirmed both dev and prod environments exist. Secrets are generated at apply time (`random_password`), stored in AWS Secrets Manager as a JSON blob, and injected into ECS tasks via `secrets` with `valueFrom` references (`${secret_arn}:KEY::`). The app builds `DATABASE_URL` from the `DB_*` env vars. `.gitignore` excludes `.env`, `*.tfvars` (except examples), `.terraform/`, and tfstate files; no secrets are committed, and the state is stored in an encrypted S3 backend.

### README

2. **Prompt:** Please write a README for this repo. It should explain the main architecture choices, how the containers receive secrets securely, how the worker communicates with the API, database, or queue, how container versions are deployed and rolled back, and the three checks I'd perform if the API works but the worker processes no jobs. I also want a current limitations section highlighting the things we know are not production ready: the app is deliberately simple, how keys are managed, no CI/CD yet, a single NAT gateway, HTTP only, and the DB-as-queue approach.
   - **Outcome:** Wrote `README.md` covering all five architecture review sections plus the current limitations the user described.

3. **Prompt:** On GPU inference: it's a direction I'd like to explore for the video processing, but it's not implemented yet. I think we should mention it in the README as an intended future feature rather than a current capability.
   - **Outcome:** Updated the README to note GPU inference as a planned, not-yet-implemented direction.

4. **Prompt:** Could we mention the DynamoDB + SQS intention, or is it already mentioned?
   - **Outcome:** Confirmed SQS was only mentioned in passing and DynamoDB not at all; added the DynamoDB + SQS note to the limitations and intentions.

5. **Prompt:** Please show the base API for testing at `http://media-dev-alb-1228361591.eu-central-1.elb.amazonaws.com`, and make clear that everything is dev, not production ready.
   - **Outcome:** Added a "Testing the dev API" section with the endpoint list and a dev-only caveat.

6. **Prompt:** OK, please commit the changes.
   - **Outcome:** Committed the README updates.
