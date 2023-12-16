# isite

Convert **i**ssues to a web**site**.

## Usage

```bash
isite generate --help
```

After generating the Markdown based documents, you can build the website with the following engines.

## Engines

- [x] [zola](https://github.com/getzola/zola)
- [ ] [hugo](https://github.com/gohugoio/hugo)

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
      ISITE_VERSION: v0.1.1
      ZOLA_VERSION: v0.17.2
      USER: ${{ github.repository_owner }}
      REPO: ${{ github.event.repository.name }}
      BASE_URL: https://${{ github.repository_owner }}.github.io/${{ github.event.repository.name }}
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Generate markdown
        run: |
          gh release download $ISITE_VERSION --repo kemingy/isite -p '*Linux_x86_64*' --output isite.tar.gz
          tar zxf isite.tar.gz && mv isite /usr/local/bin
          isite generate --user $USER --repo $REPO
          gh release download $ZOLA_VERSION --repo getzola/zola -p '*linux*' --output zola.tar.gz
          tar zxf zola.tar.gz && mv zola /usr/local/bin
          cd output && zola build --base-url $BASE_URL
      - name: Setup Pages
        uses: actions/configure-pages@v3
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v2
        with:
          path: 'output/public'
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v2
```
