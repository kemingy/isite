# isite

Convert **i**ssues to a web**site**.

## Usage

```bash
isite generate --help
```

After generating the Markdown based documents, you can build the website with the following engines.

## Examples

- [WithCode](https://github.com/kemingy/withcode/)
- [GitBlog](https://github.com/yihong0618/gitblog)

## Engines

- [x] [zola](https://github.com/getzola/zola)
- [ ] [hugo](https://github.com/gohugoio/hugo)

## Installation

- GitHub Releases: download the pre-built binaries from the [releases](https://github.com/kemingy/isite/releases) page.
- Docker Image: [`docker pull ghcr.io/kemingy/isite`](https://github.com/kemingy/isite/pkgs/container/isite)

## GitHub Actions

> [!IMPORTANT]
> Please remember to enable the GitHub Pages with GitHub Actions as the source.

TODO: provide a GitHub Actions workflow to automate the process.

```yaml
name: Deploy static content to Pages

on:
  issues:
    types:
      - opened
      - edited
      - closed
      - reopened
      - labeled
      - unlabeled
  workflow_dispatch:

# Sets permissions of the GITHUB_TOKEN to allow deployment to GitHub Pages
permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: ${{ github.workflow }}
  cancel-in-progress: true

jobs:
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: ${{ github.token }}
      ISITE_VERSION: v0.2.0
      ZOLA_VERSION: v0.20.0
      USER: ${{ github.repository_owner }}
      REPO: ${{ github.event.repository.name }}
      BASE_URL: https://${{ github.repository_owner }}.github.io/${{ github.event.repository.name }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Generate markdown
        run: |
          gh release download $ISITE_VERSION --repo kemingy/isite -p '*Linux_x86_64*' -O- | tar -xz -C /tmp && mv /tmp/isite /usr/local/bin
          isite generate --user $USER --repo $REPO
          gh release download $ZOLA_VERSION --repo getzola/zola -p '*x86_64-unknown-linux*' -O- | tar -xz -C /tmp && mv /tmp/zola /usr/local/bin
          cd output && zola build --base-url $BASE_URL
      - name: Setup Pages
        uses: actions/configure-pages@v5
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: 'output/public'
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```
