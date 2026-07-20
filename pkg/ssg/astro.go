package ssg

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/pelletier/go-toml/v2"

	"github.com/kemingy/isite/pkg/models"
)

const astroConfigTemplate = `import { defineConfig } from "astro/config";
import optimizeImages from "./src/lib/image-optimizer.mjs";
%s
export default defineConfig({
%s	integrations: [optimizeImages()],
	base: %s,%s
});
`

const astroImageOptimizer = `import { readdir, readFile, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const imageTag = /<img\b[^>]*>/gi;
const loadingAttribute = /\sloading\s*=/i;
const decodingAttribute = /\sdecoding\s*=/i;
const earlyImageTextLimit = 320;

function addImageAttributes(tag, lazy) {
  const attributes = [];
  if (lazy && !loadingAttribute.test(tag)) attributes.push('loading="lazy"');
  if (!decodingAttribute.test(tag)) attributes.push('decoding="async"');
  if (attributes.length === 0) return tag;

  const closing = tag.endsWith("/>") ? "/>" : ">";
  return tag.slice(0, -closing.length).trimEnd() + " " + attributes.join(" ") + " " + closing;
}

function optimizePostBody(html) {
  const start = html.indexOf('<div class="post__body">');
  if (start === -1) return html;

  const end = [
    html.indexOf('<div class="post-reactions"', start),
    html.indexOf('<section class="post-comments"', start),
    html.indexOf('<footer class="post-footer"', start),
  ].filter((position) => position > start).sort((left, right) => left - right)[0];
  if (end == null) return html;

  const body = html.slice(start, end);
  let cursor = 0;
  let textLength = 0;
  let imageCount = 0;
  const optimizedBody = body.replace(imageTag, (tag, offset) => {
    textLength += body
      .slice(cursor, offset)
      .replace(/<[^>]+>/g, "")
      .replace(/&(?:#\d+|#x[\da-f]+|\w+);/gi, "x")
      .trim().length;
    cursor = offset + tag.length;
    const lazy = imageCount > 0 || textLength > earlyImageTextLimit;
    imageCount += 1;
    return addImageAttributes(tag, lazy);
  });

  return html.slice(0, start) + optimizedBody + html.slice(end);
}

async function htmlFiles(directory) {
  const files = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = directory + "/" + entry.name;
    if (entry.isDirectory()) files.push(...await htmlFiles(path));
    else if (entry.isFile() && entry.name.endsWith(".html")) files.push(path);
  }
  return files;
}

export default function optimizeImages() {
  return {
    name: "isite:image-optimizer",
    hooks: {
      "astro:build:done": async ({ dir }) => {
        for (const path of await htmlFiles(fileURLToPath(dir))) {
          const html = await readFile(path, "utf8");
          const optimized = optimizePostBody(html);
          if (optimized !== html) await writeFile(path, optimized);
        }
      },
    },
  };
}
`

const astroContentConfig = `import { defineCollection } from "astro:content";
import { glob } from "astro/loaders";
import { z } from "astro/zod";

const issues = defineCollection({
  loader: glob({ base: "./src/content/issues", pattern: "**/*.md" }),
  schema: z.object({
    number: z.number().int().positive(),
    title: z.string(),
    description: z.string(),
    createdAt: z.coerce.date(),
    updatedAt: z.coerce.date(),
    author: z.object({
      name: z.string(),
      url: z.string().url(),
      avatarUrl: z.string().url(),
    }),
    issueUrl: z.string().url(),
    tags: z.array(z.string()),
    reactions: z.object({
      thumbsUp: z.number(),
      thumbsDown: z.number(),
      laugh: z.number(),
      hooray: z.number(),
      confused: z.number(),
      heart: z.number(),
      rocket: z.number(),
      eyes: z.number(),
    }),
    comments: z.array(z.object({
      url: z.string().url(),
      authorName: z.string(),
      authorAvatar: z.string().url(),
      content: z.string(),
      updatedAt: z.coerce.date(),
    })),
  }),
});

export const collections = { issues };
`

const astroBaseLayout = `---
import "../styles/global.css";
%s
import { FEED, MENU, SITE_DESCRIPTION, SITE_TITLE } from "../lib/site";
import { withBase } from "../lib/urls";

interface Props {
  title?: string;
  description?: string;
}

const { title = SITE_TITLE, description = SITE_DESCRIPTION } = Astro.props;
const pageTitle = title === SITE_TITLE ? title : title + " | " + SITE_TITLE;
---

<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content={description} />
    <meta name="generator" content={Astro.generator} />
    {FEED && <link rel="alternate" type="application/rss+xml" title={SITE_TITLE} href={withBase("rss.xml")} />}
    <title>{pageTitle}</title>
  </head>
  <body>
    <div class="container">
      <header id="header">
        <a class="logo" href={withBase("")} aria-label={SITE_TITLE}></a>
        <nav class="menu" aria-label="Main navigation">
          <ul>{MENU.map((item) => <li><a href={withBase(item.url)}>{item.name}</a></li>)}</ul>
        </nav>
        <details class="mobile-menu">
          <summary aria-label="Open navigation"><span></span><span></span><span></span></summary>
          <nav aria-label="Mobile navigation">
            <ul>{MENU.map((item) => <li><a href={withBase(item.url)}>{item.name}</a></li>)}</ul>
          </nav>
        </details>
      </header>
      <main><div class="content"><slot /></div></main>
    </div>
  </body>
</html>
`

const astroPostList = `---
import type { CollectionEntry } from "astro:content";
import { withBase } from "../lib/urls";

interface Props {
  issues: CollectionEntry<"issues">[];
  page: number;
  pageCount: number;
}

const { issues, page, pageCount } = Astro.props;
const pageURL = (number: number) => number === 1 ? withBase("") : withBase("page/" + number + "/");
---

<div class="posts">
  {issues.length === 0 ? <p class="empty">No issues matched the selected filters.</p> : issues.map((issue) => (
    <article class="post post-list-item">
      <header class="post__header">
        <h1 class="post__title"><a href={withBase("issue-" + issue.data.number + "/")}>{issue.data.title}</a></h1>
        <div class="post__meta">
          <time datetime={issue.data.createdAt.toISOString()}>{issue.data.createdAt.toISOString().slice(0, 10)}</time>
          <span class="post__source"><a href={issue.data.issueUrl}>read the source issue</a></span>
        </div>
      </header>
      <div class="read-more"><a href={withBase("issue-" + issue.data.number + "/")}>Read more...</a></div>
    </article>
  ))}
</div>
{pageCount > 1 && (
  <nav class="pagination" aria-label="Pagination">
    {page > 1 && <a class="previous" href={pageURL(page - 1)}>‹ Newer</a>}
    {page < pageCount && <a class="next" href={pageURL(page + 1)}>Older ›</a>}
  </nav>
)}
`

const astroIndexPage = `---
import { getCollection } from "astro:content";
import PostList from "../components/PostList.astro";
import Base from "../layouts/Base.astro";
import { PAGE_SIZE } from "../lib/site";

const allIssues = (await getCollection("issues")).sort(
  (a, b) => b.data.createdAt.valueOf() - a.data.createdAt.valueOf(),
);
const pageCount = Math.ceil(allIssues.length / PAGE_SIZE);
---

<Base><PostList issues={allIssues.slice(0, PAGE_SIZE)} page={1} pageCount={pageCount} /></Base>
`

const astroPaginationPage = `---
import { getCollection } from "astro:content";
import PostList from "../../components/PostList.astro";
import Base from "../../layouts/Base.astro";
import { PAGE_SIZE } from "../../lib/site";

export async function getStaticPaths() {
  const issues = (await getCollection("issues")).sort(
    (a, b) => b.data.createdAt.valueOf() - a.data.createdAt.valueOf(),
  );
  const pageCount = Math.ceil(issues.length / PAGE_SIZE);
  return Array.from({ length: Math.max(0, pageCount - 1) }, (_, index) => {
    const page = index + 2;
    return {
      params: { page: page.toString() },
      props: { issues: issues.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), page, pageCount },
    };
  });
}

const { issues, page, pageCount } = Astro.props;
---

<Base title={"Page " + page}><PostList issues={issues} page={page} pageCount={pageCount} /></Base>
`

const astroIssuePage = `---
import { getCollection, render, type CollectionEntry } from "astro:content";
import Base from "../layouts/Base.astro";
import { tagSlug, withBase } from "../lib/urls";

export async function getStaticPaths() {
  const issues = (await getCollection("issues")).sort(
    (a, b) => b.data.createdAt.valueOf() - a.data.createdAt.valueOf(),
  );
  return issues.map((issue, index) => ({
    params: { number: issue.data.number.toString() },
    props: {
      issue,
      newer: index > 0 ? issues[index - 1] : null,
      older: index < issues.length - 1 ? issues[index + 1] : null,
    },
  }));
}

interface Props {
  issue: CollectionEntry<"issues">;
  newer: CollectionEntry<"issues"> | null;
  older: CollectionEntry<"issues"> | null;
}

const { issue, newer, older } = Astro.props;
const { Content } = await render(issue);
const avatarURL = (value: string) => {
  const url = new URL(value);
  if (url.hostname === "avatars.githubusercontent.com") url.searchParams.set("s", "64");
  return url.href;
};
const reactions = [
  ["👍", issue.data.reactions.thumbsUp],
  ["👎", issue.data.reactions.thumbsDown],
  ["😄", issue.data.reactions.laugh],
  ["🎉", issue.data.reactions.hooray],
  ["😕", issue.data.reactions.confused],
  ["❤️", issue.data.reactions.heart],
  ["🚀", issue.data.reactions.rocket],
  ["👀", issue.data.reactions.eyes],
] as const;
---

<Base title={issue.data.title} description={issue.data.description}>
  <article class="post post-detail">
    <header class="post__header">
      <h1 class="post__title">{issue.data.title}</h1>
      <div class="post__meta">
        <time datetime={issue.data.createdAt.toISOString()}>{issue.data.createdAt.toISOString().slice(0, 10)}</time>
        <span class="post__source"><a href={issue.data.issueUrl}>read the source issue</a></span>
      </div>
    </header>

    <div class="post__body"><Content /></div>

    {reactions.some(([, count]) => count > 0) && (
      <div class="post-reactions" aria-label="Reactions">
        {reactions.filter(([, count]) => count > 0).map(([emoji, count]) => (
          <span class="reaction-emoji">{emoji} <span class="reaction-count">{count}</span></span>
        ))}
      </div>
    )}

    {issue.data.comments.length > 0 && (
      <section class="post-comments" aria-label="Comments">
        {issue.data.comments.map((comment) => (
          <article class="comment-item">
            <div class="comment-author">
              <img src={avatarURL(comment.authorAvatar)} alt="" width="64" height="64" loading="lazy" decoding="async" />
              <strong>{comment.authorName}</strong>
              <a class="comment-date" href={comment.url}>
                <time datetime={comment.updatedAt.toISOString()}>{comment.updatedAt.toISOString().slice(0, 10)}</time>
              </a>
            </div>
            <p class="comment-content">{comment.content}</p>
          </article>
        ))}
      </section>
    )}

    <footer class="post-footer">
      {issue.data.tags.length > 0 && (
        <div class="post-tags">
          {issue.data.tags.map((tag) => <a href={withBase("tags/" + tagSlug(tag) + "/")}>#{tag}</a>)}
        </div>
      )}
      <nav class="post-nav" aria-label="Adjacent posts">
        {newer && <a class="previous" href={withBase("issue-" + newer.data.number + "/")}>‹ {newer.data.title}</a>}
        {older && <a class="next" href={withBase("issue-" + older.data.number + "/")}>{older.data.title} ›</a>}
      </nav>
    </footer>
  </article>
</Base>
`

const astroTagsPage = `---
import { getCollection } from "astro:content";
import Base from "../../layouts/Base.astro";
import { tagSlug, withBase } from "../../lib/urls";

const issues = await getCollection("issues");
const tags = [...new Set(issues.flatMap((issue) => issue.data.tags))].sort((a, b) => a.localeCompare(b));
---

<Base title="Tags" description="Browse issues by GitHub label">
  <section class="taxonomies">
    <h1 class="taxonomies__title">Tags</h1>
    <div class="taxonomies__items">
      {tags.length === 0 ? <p class="empty">No labels were found.</p> : tags.map((tag) => {
        const count = issues.filter((issue) => issue.data.tags.includes(tag)).length;
        return <a href={withBase("tags/" + tagSlug(tag) + "/")}>{tag}<span class="count">{count}</span></a>;
      })}
    </div>
  </section>
</Base>
`

const astroTagPage = `---
import { getCollection } from "astro:content";
import Base from "../../layouts/Base.astro";
import { tagSlug, withBase } from "../../lib/urls";

export async function getStaticPaths() {
  const issues = await getCollection("issues");
  const tags = [...new Set(issues.flatMap((issue) => issue.data.tags))];
  return tags.map((tag) => ({ params: { tag: tagSlug(tag) }, props: { tag, issues } }));
}

const { tag, issues } = Astro.props;
const taggedIssues = issues
  .filter((issue) => issue.data.tags.includes(tag))
  .sort((a, b) => b.data.createdAt.valueOf() - a.data.createdAt.valueOf());
---

<Base title={tag} description={"Issues tagged " + tag}>
  <section class="taxonomy">
    <h1 class="taxonomies__title">{tag}</h1>
    {taggedIssues.map((issue) => (
      <div class="taxonomy__item">
        <time class="taxonomy__item__time" datetime={issue.data.createdAt.toISOString()}>{issue.data.createdAt.toISOString().slice(0, 10)}</time>
        <span class="taxonomy__item__title"><a href={withBase("issue-" + issue.data.number + "/")}>{issue.data.title}</a></span>
      </div>
    ))}
  </section>
</Base>
`

const astroRSSPage = `import { getCollection } from "astro:content";
import rss from "@astrojs/rss";
import { SITE_DESCRIPTION, SITE_TITLE } from "../lib/site";
import { withBase } from "../lib/urls";

export async function GET(context) {
  const issues = (await getCollection("issues")).sort(
    (a, b) => b.data.createdAt.valueOf() - a.data.createdAt.valueOf(),
  );
  return rss({
    title: SITE_TITLE,
    description: SITE_DESCRIPTION,
    site: context.site ?? context.url.origin,
    items: issues.map((issue) => ({
      title: issue.data.title,
      description: issue.data.description,
      pubDate: issue.data.createdAt,
      link: withBase("issue-" + issue.data.number + "/"),
      categories: issue.data.tags,
    })),
  });
}
`

const astroGlobalCSS = `:root {
  --ink: #34495e;
  --muted: #757575;
  --accent: #bb5649;
  --line: #e6e6e6;
  --paper: #fefefe;
  --soft: #f8f5ec;
  font-family: Athelas, "Songti SC", STHeiti, "Microsoft Yahei", serif;
  color: var(--ink);
  background: var(--paper);
}
* { box-sizing: border-box; }
html { border-top: 3px solid var(--accent); }
body { margin: 0; min-width: 280px; line-height: 1.5; background: var(--paper); }
a { color: var(--accent); text-decoration: none; }
a:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; }
img { max-width: 100%; height: auto; vertical-align: middle; }
.container { width: 800px; margin: 0 auto; padding-bottom: 4rem; }
#header { display: flex; justify-content: space-between; min-height: 88px; padding: 20px; }
.logo { flex: 1 1 auto; }
.menu { align-self: flex-start; }
.menu ul { display: flex; flex-wrap: wrap; gap: 0 8px; margin: 0; padding: 0 25px 0 0; list-style: none; }
.menu li, .post__title { position: relative; overflow: hidden; }
.menu li::before, .post__title::before {
  position: absolute;
  z-index: 0;
  right: 51%;
  bottom: 0;
  left: 51%;
  height: 2px;
  background: var(--accent);
  content: "";
  transition: right .2s ease-out, left .2s ease-out;
}
.menu li:hover::before, .post__title:hover::before { right: 0; left: 0; }
.menu a { position: relative; z-index: 1; color: var(--ink); font-size: 18px; }
.mobile-menu { display: none; }
.content { padding: 0 20px; }
.posts { margin-bottom: 20px; border-bottom: 1px solid var(--line); }
.post { padding: 1.5rem 0; }
.post + .post { border-top: 1px solid var(--line); }
.post__title { display: inline-block; margin: 0; color: var(--ink); font-size: 26px; font-weight: 400; vertical-align: middle; }
.post__title a { position: relative; z-index: 1; color: var(--ink); }
.post__meta { color: var(--muted); font-family: ui-sans-serif, system-ui, sans-serif; font-size: 14px; }
.post__source { padding: 0 12px; }
.read-more { margin-top: 1rem; }
.read-more a { font-size: 1.1rem; }
.read-more a:hover { text-decoration: underline; }
.post__body { margin-top: 2rem; overflow-wrap: anywhere; }
.post__body h2, .post__body h3, .post__body h4 { font-weight: 400; }
.post__body a:hover { border-bottom: 1px solid var(--accent); }
.post__body img { display: inline-block; }
.post__body pre { overflow: auto; padding: 1rem; background: #2f3f4d; color: #e8edf1; }
.post__body :not(pre) > code { padding: 3px 5px; border-radius: 4px; background: var(--soft); color: #c7254e; }
.post__body blockquote { margin: 2em 0; padding: 10px 20px; border-left: 3px solid rgb(192 91 77 / 30%); background: rgb(192 91 77 / 5%); color: rgb(52 73 94 / 80%); }
.post__body table { width: 100%; margin: 10px 0; border-spacing: 0; box-shadow: 2px 2px 3px rgb(0 0 0 / 12.5%); }
.post__body th, .post__body td { padding: 5px 15px; border: 1px double #f4efe1; }
.post__body thead, .post__body tr:hover { background: var(--soft); }
.post-reactions { display: flex; flex-wrap: wrap; margin: 1rem 0; }
.reaction-emoji { display: inline-block; margin: 0 4px; padding: 2px 8px; border: 1px solid #cacaca; border-radius: 1rem; }
.reaction-count { color: #666; }
.post-comments { margin: 2rem 0; border-top: 1px solid var(--line); }
.comment-item { display: flex; gap: 1rem; margin: 2rem 1rem; content-visibility: auto; contain-intrinsic-size: auto 110px; }
.comment-author { width: 84px; flex: 0 0 84px; overflow-wrap: anywhere; font-family: ui-sans-serif, system-ui, sans-serif; font-size: 12px; }
.comment-author img { display: block; width: 64px; border-radius: 2px; }
.comment-date { display: block; color: var(--muted); }
.comment-content { flex: 1; margin: 0; padding: 8px 1rem; border: 1px solid #d7d7d7; border-radius: 4px; box-shadow: 0 2px 4px rgb(0 0 0 / 20%); white-space: pre-wrap; }
.post-footer { margin-top: 20px; border-top: 1px solid var(--line); }
.post-tags { padding: 15px 0; }
.post-tags a { margin-right: 5px; overflow-wrap: anywhere; }
.post-nav, .pagination { min-height: 2rem; margin: 1rem 0; }
.post-nav::after, .pagination::after { display: block; clear: both; content: ""; }
.next, .previous { max-width: 48%; color: var(--ink); font-size: 20px; font-weight: 600; transition: transform .2s ease-out; }
.next { float: right; text-align: right; }
.previous { float: left; }
.next:hover { transform: translateX(4px); color: var(--accent); }
.previous:hover { transform: translateX(-4px); color: var(--accent); }
.taxonomies { margin: 2em 0 3em; text-align: center; }
.taxonomies__title { display: inline-block; margin: 0 0 1rem; border-bottom: 2px solid var(--accent); color: var(--accent); font-size: 18px; font-weight: 400; }
.taxonomies__items a { position: relative; display: inline-block; margin: 5px 10px; }
.taxonomies__items .count { position: relative; top: -8px; right: -2px; font-size: 12px; }
.taxonomy { margin: 2em 0; }
.taxonomy__item { padding: 3px 20px; border-left: 1px solid #cacaca; transition: transform .2s ease-out, border-color .2s ease-out; }
.taxonomy__item:hover { transform: translateX(4px); border-left: 3px solid var(--accent); }
.taxonomy__item__time { margin-right: 10px; color: var(--muted); }
.empty { color: var(--muted); }
@media (max-width: 800px) {
  html { border-top: 0; }
  .container { width: 100%; }
  #header { min-height: 50px; padding: 0; box-shadow: 0 2px 2px #cacaca; }
  .logo, .menu { display: none; }
  .mobile-menu { display: block; width: 100%; }
  .mobile-menu summary { position: relative; width: 50px; height: 50px; cursor: pointer; list-style: none; }
  .mobile-menu summary::-webkit-details-marker { display: none; }
  .mobile-menu summary span { position: absolute; left: 15px; width: 20px; height: 1px; background: var(--accent); transition: transform .2s ease; }
  .mobile-menu summary span:nth-child(1) { top: 19px; }
  .mobile-menu summary span:nth-child(2) { top: 25px; }
  .mobile-menu summary span:nth-child(3) { top: 31px; }
  .mobile-menu[open] summary span:nth-child(1) { top: 25px; transform: rotate(45deg); }
  .mobile-menu[open] summary span:nth-child(2) { opacity: 0; }
  .mobile-menu[open] summary span:nth-child(3) { top: 25px; transform: rotate(-45deg); }
  .mobile-menu nav { border-top: 1px solid var(--soft); background: var(--paper); }
  .mobile-menu ul { margin: 0; padding: 0; list-style: none; }
  .mobile-menu li { padding: 10px 30px; border-bottom: 1px solid var(--soft); }
  .content { padding-top: 2rem; }
  .comment-item { margin-inline: 0; }
  .comment-author { width: 70px; flex-basis: 70px; }
}
@media (max-width: 480px) {
  .post__title { font-size: 23px; }
  .post__source { display: block; padding: 0; }
  .comment-item { display: block; }
  .comment-author { width: auto; margin-bottom: .75rem; }
  .next, .previous { max-width: 100%; float: none; display: block; margin: .75rem 0; text-align: left; }
}
`

type Astro struct {
	Title       string
	BaseURL     string
	Description string
	Feed        bool
	Katex       bool
	Theme       string
	ThemeRepo   string
	ConfigPath  string
	Menu        []astroMenuItem
}

type astroMenuItem struct {
	URL  string `json:"url" toml:"url"`
	Name string `json:"name" toml:"name"`
}

type astroLegacyConfig struct {
	Extra struct {
		EvenTitle string          `toml:"even_title"`
		EvenMenu  []astroMenuItem `toml:"even_menu"`
	} `toml:"extra"`
}

type astroAuthor struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	AvatarURL string `json:"avatarUrl"`
}

type astroComment struct {
	URL          string `json:"url"`
	AuthorName   string `json:"authorName"`
	AuthorAvatar string `json:"authorAvatar"`
	Content      string `json:"content"`
	UpdatedAt    string `json:"updatedAt"`
}

type astroReactions struct {
	ThumbsUp   int `json:"thumbsUp"`
	ThumbsDown int `json:"thumbsDown"`
	Laugh      int `json:"laugh"`
	Hooray     int `json:"hooray"`
	Confused   int `json:"confused"`
	Heart      int `json:"heart"`
	Rocket     int `json:"rocket"`
	Eyes       int `json:"eyes"`
}

type astroPost struct {
	Number      int            `json:"number"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
	Author      astroAuthor    `json:"author"`
	IssueURL    string         `json:"issueUrl"`
	Tags        []string       `json:"tags"`
	Reactions   astroReactions `json:"reactions"`
	Comments    []astroComment `json:"comments"`
}

func NewAstro(cmd *models.Command, meta *models.Repository) *Astro {
	description := cmd.Title
	if meta != nil && meta.Description != "" {
		description = meta.Description
	}
	return &Astro{
		Title:       cmd.Title,
		BaseURL:     cmd.BaseURL,
		Description: description,
		Feed:        cmd.Feed,
		Katex:       cmd.Katex,
		Theme:       cmd.Theme,
		ThemeRepo:   cmd.ThemeRepo,
		ConfigPath:  cmd.Config,
		Menu: []astroMenuItem{
			{URL: "", Name: "Home"},
			{URL: "tags/", Name: "Tags"},
		},
	}
}

func (a *Astro) Generate(issues []models.Issue, outputDir string) error {
	if a.Theme != "" || a.ThemeRepo != "" {
		return errors.New("custom themes are only supported by the zola engine")
	}
	if err := a.loadConfig(); err != nil {
		return err
	}

	path, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get the output absolute path for %s", outputDir)
	}
	for _, dir := range []string{
		"src/components", "src/content/issues", "src/layouts", "src/lib", "src/pages/page", "src/pages/tags", "src/styles",
	} {
		if err := os.MkdirAll(filepath.Join(path, dir), 0755); err != nil {
			return errors.Wrapf(err, "failed to create astro directory %s", dir)
		}
	}

	if err := a.generateProjectFiles(path); err != nil {
		return err
	}
	return a.generatePosts(path, issues)
}

func (a *Astro) generateProjectFiles(path string) error {
	dependencies := map[string]string{"astro": "^7.1.1"}
	if a.Feed {
		dependencies["@astrojs/rss"] = "^4.0.19"
	}
	if a.Katex {
		dependencies["@astrojs/markdown-remark"] = "^7.2.1"
		dependencies["katex"] = "^0.18.1"
		dependencies["rehype-katex"] = "^7.0.1"
		dependencies["remark-math"] = "^6.0.0"
	}
	packageJSON, err := json.MarshalIndent(map[string]any{
		"name":    "isite-astro-site",
		"type":    "module",
		"private": true,
		"engines": map[string]string{"node": ">=22.12.0"},
		"scripts": map[string]string{
			"dev": "astro dev", "build": "astro build", "preview": "astro preview", "astro": "astro",
		},
		"dependencies": dependencies,
	}, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to encode astro package.json")
	}
	packageJSON = append(packageJSON, '\n')

	site, base := astroDeployment(a.BaseURL)
	configImports := ""
	markdownConfig := ""
	if a.Katex {
		configImports = "import rehypeKatex from \"rehype-katex\";\nimport remarkMath from \"remark-math\";\n"
		markdownConfig = "\n  markdown: { remarkPlugins: [remarkMath], rehypePlugins: [rehypeKatex] },"
	}
	siteConfig := ""
	if site != "" {
		siteConfig = fmt.Sprintf("\tsite: %s,\n", jsonString(site))
	}
	astroConfig := fmt.Sprintf(astroConfigTemplate, configImports, siteConfig, jsonString(base), markdownConfig)
	katexImport := ""
	if a.Katex {
		katexImport = "import \"katex/dist/katex.min.css\";"
	}
	baseLayout := fmt.Sprintf(astroBaseLayout, katexImport)
	menuJSON, err := json.Marshal(a.Menu)
	if err != nil {
		return errors.Wrap(err, "failed to encode astro navigation menu")
	}
	siteModule := fmt.Sprintf(
		"export const SITE_TITLE = %s;\nexport const SITE_DESCRIPTION = %s;\nexport const FEED = %t;\nexport const PAGE_SIZE = 10;\nexport const MENU = %s;\n",
		jsonString(a.Title), jsonString(a.Description), a.Feed, menuJSON,
	)
	urlModule := `export function withBase(path: string): string {
  if (/^(?:[a-z]+:)?\/\//i.test(path) || path.startsWith("#")) return path;
  const base = import.meta.env.BASE_URL.endsWith("/")
    ? import.meta.env.BASE_URL
    : import.meta.env.BASE_URL + "/";
  return base + path.replace(/^\/+/, "");
}

export function tagSlug(value: string): string {
  return value
    .normalize("NFKC")
    .trim()
    .toLocaleLowerCase("en-US")
    .replace(/[^\p{Letter}\p{Number}]+/gu, "-")
    .replace(/^-+|-+$/g, "");
}
`
	tsconfig := `{
  "extends": "astro/tsconfigs/strict"
}
`
	gitignore := "node_modules/\ndist/\n.astro/\n"

	files := map[string][]byte{
		"package.json":                   packageJSON,
		"astro.config.mjs":               []byte(astroConfig),
		"tsconfig.json":                  []byte(tsconfig),
		".gitignore":                     []byte(gitignore),
		"src/components/PostList.astro":  []byte(astroPostList),
		"src/content.config.ts":          []byte(astroContentConfig),
		"src/layouts/Base.astro":         []byte(baseLayout),
		"src/lib/image-optimizer.mjs":    []byte(astroImageOptimizer),
		"src/lib/site.ts":                []byte(siteModule),
		"src/lib/urls.ts":                []byte(urlModule),
		"src/pages/index.astro":          []byte(astroIndexPage),
		"src/pages/issue-[number].astro": []byte(astroIssuePage),
		"src/pages/page/[page].astro":    []byte(astroPaginationPage),
		"src/pages/tags/[tag].astro":     []byte(astroTagPage),
		"src/pages/tags/index.astro":     []byte(astroTagsPage),
		"src/styles/global.css":          []byte(astroGlobalCSS),
	}
	if a.Feed {
		files["src/pages/rss.xml.js"] = []byte(astroRSSPage)
	} else if err := os.Remove(filepath.Join(path, "src/pages/rss.xml.js")); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to remove disabled astro RSS page")
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(path, name), content, 0644); err != nil {
			return errors.Wrapf(err, "failed to write astro file %s", name)
		}
	}
	return nil
}

func (a *Astro) generatePosts(path string, issues []models.Issue) error {
	for _, issue := range issues {
		post := astroPost{
			Number:      issue.Number,
			Title:       issue.Title,
			Description: issue.Title,
			CreatedAt:   issue.CreatedAt,
			UpdatedAt:   issue.UpdatedAt,
			Author: astroAuthor{
				Name: issue.User.Login, URL: issue.User.URL, AvatarURL: issue.User.AvatarURL,
			},
			IssueURL: issue.URL,
			Tags:     make([]string, 0, len(issue.Labels)),
			Reactions: astroReactions{
				ThumbsUp: issue.Reactions.ThumbUp, ThumbsDown: issue.Reactions.ThumbDown,
				Laugh: issue.Reactions.Laugh, Hooray: issue.Reactions.Hooray,
				Confused: issue.Reactions.Confused, Heart: issue.Reactions.Heart,
				Rocket: issue.Reactions.Rocket, Eyes: issue.Reactions.Eyes,
			},
			Comments: make([]astroComment, 0, len(issue.Comments)),
		}
		for _, label := range issue.Labels {
			post.Tags = append(post.Tags, label.Name)
		}
		for _, comment := range issue.Comments {
			post.Comments = append(post.Comments, astroComment{
				URL: comment.HTMLURL, AuthorName: comment.User.Login, AuthorAvatar: comment.User.AvatarURL,
				Content: comment.Body, UpdatedAt: comment.UpdatedAt,
			})
		}
		frontMatter, err := json.MarshalIndent(post, "", "  ")
		if err != nil {
			return errors.Wrapf(err, "failed to encode astro post for issue #%d", issue.Number)
		}
		content := append([]byte("---\n"), frontMatter...)
		content = append(content, []byte("\n---\n\n"+issue.Body+"\n")...)
		name := filepath.Join(path, "src", "content", "issues", fmt.Sprintf("issue-%d.md", issue.Number))
		if err := os.WriteFile(name, content, 0644); err != nil {
			return errors.Wrapf(err, "failed to write astro post for issue #%d", issue.Number)
		}
	}
	return nil
}

func (a *Astro) loadConfig() error {
	if a.ConfigPath == "" {
		return nil
	}
	content, err := os.ReadFile(a.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to read astro config compatibility file %s", a.ConfigPath)
	}

	config := astroLegacyConfig{}
	if err := toml.Unmarshal(content, &config); err != nil {
		return errors.Wrapf(err, "failed to parse astro config compatibility file %s", a.ConfigPath)
	}
	if config.Extra.EvenTitle != "" {
		a.Title = config.Extra.EvenTitle
	}
	if len(config.Extra.EvenMenu) > 0 {
		a.Menu = make([]astroMenuItem, 0, len(config.Extra.EvenMenu))
		for _, item := range config.Extra.EvenMenu {
			item.URL = strings.TrimPrefix(strings.ReplaceAll(item.URL, "$BASE_URL", ""), "/")
			a.Menu = append(a.Menu, item)
		}
	}
	return nil
}

func astroDeployment(baseURL string) (site, base string) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", "/"
	}
	parsed, err := url.Parse(baseURL)
	if err == nil && parsed.IsAbs() && parsed.Host != "" {
		site = parsed.Scheme + "://" + parsed.Host
		base = parsed.EscapedPath()
	} else {
		base = baseURL
	}
	if base == "" {
		base = "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if base != "/" {
		base = strings.TrimSuffix(base, "/")
	}
	return site, base
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
