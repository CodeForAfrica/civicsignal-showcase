# CivicSignal rendered website

Final HTML, assets and published dataset for CivicSignal. The build/pipeline
source is [CodeForAfrica/civicsignal-web](https://github.com/CodeForAfrica/civicsignal-web).

- [Local container and tests](CONTAINER.md)
- [Manual Docker Hub / Dokku deployment and rollback](MANUAL_DEPLOY.md)

Images are built from this repository, then pushed to Docker Hub and explicitly
deployed by an operator. Git pushes do not automatically deploy the website.
The live Media Cloud portal is a separate app and stays unchanged.
