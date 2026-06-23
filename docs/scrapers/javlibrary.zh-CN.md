# JavLibrary Scraper 模块实现文档

本文聚焦 `internal/scraper/javlibrary`，从模块边界、实例构造、请求流程、页面结构假设、字段提取规则和维护要点几个角度说明 JavLibrary scraper 的实现。

## 1. 模块定位

JavLibrary scraper 是一个站点适配模块，负责把 JavLibrary 搜索页和详情页转换为统一的 `models.ScraperResult`。

它遵循仓库的 scraper 注册式架构：

- 通过 `init()` 调用 `scraperutil.RegisterModule()` 注册模块。
- 运行期由 scraper registry 根据配置构造实例。
- 对外实现 `models.Scraper`，并额外实现 URL 识别、直链抓取和媒体下载代理解析能力。
- 只依赖 `config`、`models`、`httpclient`、`ratelimit`、`imageutil`、`scraperutil` 等底层公共模块，不依赖 aggregator、organizer、API 或前端。

源码范围：

| 文件 | 职责 |
| --- | --- |
| `internal/scraper/javlibrary/module.go` | 注册 scraper 模块、声明前端配置元数据、默认值、优先级和扁平配置构造逻辑。 |
| `internal/scraper/javlibrary/config.go` | 定义 `JavLibraryConfig`，并校验语言、通用请求配置和 `base_url`。 |
| `internal/scraper/javlibrary/javlibrary.go` | 核心实现：构造 HTTP client、搜索、直链抓取、详情页解析、字段提取、图片处理、FlareSolverr 兜底和错误分类。 |
| `internal/scraper/javlibrary/*_test.go` | 覆盖配置映射、URL 处理、搜索流程、详情页解析、图片/评分/演员提取和 FlareSolverr 集成入口。 |

## 2. 模块注册与配置

`module.go` 注册的模块名是 `javlibrary`，描述为 `JavLibrary`，优先级为 `80`。默认 scraper 配置为：

```go
config.ScraperSettings{
    Enabled:   false,
    Language:  "en",
    RateLimit: 1000,
}
```

配置项在 UI/API 配置元数据中暴露为：

| 配置项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `language` | select | UI 元数据为 `ja`，scraper 默认和运行期空值回退为 `en` | 支持 `en`、`ja`、`cn`、`tw`，决定搜索 URL 的语言路径和结果语言。 |
| `request_delay` | number | `1000` | 请求间隔，单位毫秒。 |
| `base_url` | string | 运行期空值回退到 `http://www.javlibrary.com` | JavLibrary base URL，可用于镜像站。 |
| `use_flaresolverr` | boolean | `false` | 是否在直接请求遇到 Cloudflare challenge 时使用全局 FlareSolverr。 |

`JavLibraryConfig` 额外支持：

| 字段 | 说明 |
| --- | --- |
| `cookies` | flatten 阶段会映射到 `ScraperSettings.Cookies`，可配置 `cf_clearance`、`cf_bm` 等站点 cookie；但当前 `New()` 构造 `configForHTTP` 时未带上 `Cookies`，所以运行期不会自动注入这些 cookies。 |
| `proxy` | scraper 级代理配置。 |
| `download_proxy` | 媒体下载专用代理配置。 |
| `user_agent`、`timeout`、`max_retries` | 来自 `BaseScraperConfig` 的通用请求配置。 |

`config.go` 中的 `ValidateConfig` 会校验：

- 通用 scraper 配置，例如启用状态、请求延迟、重试、超时等。
- `language` 必须为空、`en`、`ja`、`cn` 或 `tw`，校验时大小写不敏感。
- `base_url` 必须是合法 HTTP/HTTPS URL。

配置示例位于 `internal/config/config.yaml.example`：

```yaml
javlibrary:
  enabled: false
  language: ja
  request_delay: 1000
  base_url: "http://www.javlibrary.com"
  cookies:
    cf_clearance: ""
    cf_bm: ""
  user_agent: ""
  proxy:
    enabled: false
  use_flaresolverr: false
```

需要注意两个默认值细节：

- `module.go` 的 `ScraperOptions` 将语言选项默认值标成 `ja`，但 `ScraperDefaults` 和 `New()` 的空值回退都是 `en`。
- `New()` 内部在 `base_url` 为空时回退到 `http://www.javlibrary.com`。
- `New()` 内部在 `language` 为空或不支持时回退到 `en`，其中不支持语言会写 warning 日志。

## 3. 实例构造

`New(settings, globalProxy, globalFlareSolverr)` 完成运行期对象构造。

主要步骤：

1. 从 `settings` 中提取 HTTP 相关字段，构造 `configForHTTP`。
2. 通过 `httpclient.FromScraperSettings(...).BuildWithFlareSolverr()` 创建 Resty client 和可选 FlareSolverr client。
3. 请求头使用 `httpclient.StandardHTMLHeaders()`，并叠加 `httpclient.UserAgentHeader(settings.UserAgent)`。
4. 合并 scraper 级代理、全局代理和全局 FlareSolverr 配置。
5. 如果 HTTP client 构造失败，则记录错误并回退到显式无代理 Resty client，同时禁用 FlareSolverr 句柄。
6. 归一化 `baseURL` 和 `language`。
7. 使用 `settings.RateLimit` 构造 `ratelimit.Limiter`。

构造后的 `Scraper` 保存以下运行期状态：

| 字段 | 作用 |
| --- | --- |
| `client` | Resty HTTP client。 |
| `flaresolverr` | 可选 FlareSolverr client，只在 scraper 配置启用且全局 FlareSolverr 启用时可用。 |
| `enabled` | 是否启用。 |
| `baseURL` | 搜索页和详情页构造的基础 URL。 |
| `language` | 运行期语言段。 |
| `proxyOverride` | scraper 级代理，用于下载代理解析兜底。 |
| `downloadProxy` | 媒体下载专用代理。 |
| `rateLimiter` | 每次页面请求前的节流器。 |
| `settings` | 保存完整配置，`Config()` 返回其深拷贝。 |
| `cookieMu` | FlareSolverr 返回 cookie 后写入共享 Resty client 时使用的互斥锁。 |

## 4. 对外接口

JavLibrary scraper 实现的能力如下：

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| `models.Scraper` | `Name()`、`Search()`、`GetURL()`、`IsEnabled()`、`Config()`、`Close()` | 标准 scraper 生命周期与搜索入口。 |
| `models.URLHandler` | `CanHandleURL()`、`ExtractIDFromURL()` | 判断 URL 是否属于 JavLibrary，并从 URL 中提取 ID 或 `v` 参数。 |
| `models.DirectURLScraper` | `ScrapeURL()` | 对用户直接输入的 JavLibrary URL 执行抓取。 |
| `models.ScraperDownloadProxyResolver` | `ResolveDownloadProxyForHost()` | 为 JavLibrary 图片和 CDN 媒体下载选择代理。 |

`ResolveDownloadProxyForHost()` 支持以下 host：

- `javlibrary.com` 及其子域名，例如 `www.javlibrary.com`、`img.javlibrary.com`。
- `c.impact.jp` 及其子域名，这是 JavLibrary 页面中常见的图片 CDN。

匹配后返回 `downloadProxy` 和 `proxyOverride`，由下载器按统一优先级决定实际代理。

## 5. 抓取总流程

JavLibrary 模块有两个主要入口：按 ID 搜索和按 URL 直抓。

### 5.1 按 ID 搜索

`Search(ctx, id)` 是常规元数据抓取入口。

```mermaid
flowchart TD
    Start["Search(ctx, id)"] --> Enabled{"scraper enabled?"}
    Enabled -- "否" --> Disabled["返回 disabled error"]
    Enabled -- "是" --> BuildSearchURL["getURLCtx(ctx, id)"]
    BuildSearchURL --> FetchSearch["fetchPageCtx(searchURL)"]
    FetchSearch --> DirectDetail{"HTML 包含 id=\"video_info\"?"}
    DirectDetail -- "是" --> ParseDirect["parseDetailPage(searchHTML, id, searchURL, language)"]
    DirectDetail -- "否" --> ExtractLink["extractMovieURLFromHTML(searchHTML, id)"]
    ExtractLink --> Found{"找到详情路径?"}
    Found -- "否" --> NotFound["返回 not found"]
    Found -- "是" --> BuildDetailURL["构造 detailURL"]
    BuildDetailURL --> FetchDetail["fetchPageCtx(detailURL)"]
    FetchDetail --> ParseDetail["parseDetailPage(detailHTML, id, detailURL, language)"]
    ParseDirect --> Result["models.ScraperResult"]
    ParseDetail --> Result
```

流程要点：

- `Search()` 首先检查 scraper 是否启用。
- 搜索 URL 固定由 `GetURL()`/`getURLCtx()` 构造，格式为 `{baseURL}/{language}/vl_searchbyid.php?keyword={id}`。
- JavLibrary 搜索页有时会直接跳到详情页；实现通过 `id="video_info"` 判断这种情况。
- 如果返回的是结果列表，则使用 `extractMovieURLFromHTML()` 找到详情页 `?v=...`。
- 详情页最终统一进入 `parseDetailPage()`。

### 5.2 搜索结果页链接发现

`extractMovieURLFromHTML(html, searchID)` 负责从搜索结果页找到详情页路径。当前支持两类页面结构。

第一类是新版缩略图列表：

```html
<div class="video" id="vid_javliat76u">
  <a href="./javliat76u.html" title="ONED-025 Play Erotic Woman">
    <img src="https://pics.dmm.co.jp/.../oned025ps.jpg">
    <div class="id">ONED-025</div>
    <div class="title">Play Erotic Woman</div>
  </a>
</div>
```

Chrome 中打开的中文演员列表页确认了这个结构：外层有 `.videothumblist .videos`，每个结果是 `div.video#vid_xxx`，`div.id` 是展示 ID，`div.title` 是展示标题，`a[href]` 通常是 `./javli6ll5y.html` 这种相对详情路径，`a[title]` 是 `ID + 标题`，缩略图多为 DMM `ps.jpg`。

解析规则：

1. 用 `reVideoThumbDiv` 找到 `class="video"` 的块，并读取其中 `<div class="id">`。
2. 用 `reVideoThumbID` 从同一个块的 `id="vid_xxx"` 中提取 JavLibrary 内部视频 ID；真实列表中的 `href="./xxx.html"` 与这个内部 ID 一致，但当前实现直接转成 `?v={vidID}`。
3. 将页面 ID 和搜索 ID 都转大写并移除 `-` 后比较；这是列表页最可靠的匹配条件。
4. 精确匹配时返回 `?v={vidID}`。
5. 如果没有精确匹配，但列表中存在结果，则记录 debug 日志并返回第一个结果作为兜底。

第二类是旧版链接：

```html
<a href="/en/?v=javli43uqe">result</a>
<a href="?v=javli43uqe">result</a>
```

解析规则：

- 优先匹配带语言段的 `/(en|ja|cn|tw)/?v=...`。
- 再匹配相对查询串 `?v=...`。

### 5.3 详情 URL 构造

搜索结果返回的 `detailPath` 可能是绝对 URL，也可能是 `?v=...` 或 `/en/?v=...` 这样的相对路径。

构造规则：

| `detailPath` 形态 | 构造方式 |
| --- | --- |
| 以 `http` 开头 | 直接使用。 |
| 去掉前导 `/` 后以 `{language}/` 或 `{language}?` 开头 | 拼成 `{baseURL}/{detailPath}`，避免重复语言段。 |
| 其他相对路径 | 拼成 `{baseURL}/{language}/{detailPath}`。 |

例如 `baseURL=http://www.javlibrary.com`、`language=en`：

| 输入 | 输出 |
| --- | --- |
| `?v=javli43uqe` | `http://www.javlibrary.com/en/?v=javli43uqe` |
| `/en/?v=javli43uqe` | `http://www.javlibrary.com/en/?v=javli43uqe` |

### 5.4 按 URL 直抓

`ScrapeURL(ctx, rawURL)` 用于处理用户直接传入 JavLibrary URL 的情况。

```mermaid
flowchart TD
    Start["ScrapeURL(ctx, rawURL)"] --> CanHandle{"CanHandleURL(rawURL)?"}
    CanHandle -- "否" --> NotHandled["返回 not found: URL not handled"]
    CanHandle -- "是" --> ExtractID["ExtractIDFromURL(rawURL)"]
    ExtractID --> DetailURL{"URL 自身带 v 参数?"}
    DetailURL -- "是" --> UseRaw["detailURL = rawURL，并从路径解析语言段"]
    DetailURL -- "否" --> Build["detailURL = baseURL/language/?v=id"]
    UseRaw --> Fetch["fetchPageCtx(detailURL)"]
    Build --> Fetch
    Fetch --> HasInfo{"HTML 包含 id=\"video_info\"?"}
    HasInfo -- "否" --> NoInfo["返回 not found: page does not contain video info"]
    HasInfo -- "是" --> Parse["parseDetailPage(html, id, detailURL, resultLanguage)"]
```

`CanHandleURL()` 只接受 `javlibrary.com` 及其子域名。

`ExtractIDFromURL()` 的提取优先级如下：

| 来源 | 规则 |
| --- | --- |
| 查询参数 `v` | 直接返回原值，例如 `javli123`。 |
| 查询参数 `keyword` | 返回大写后的搜索关键词，例如 `IPX-123`。 |
| 路径段 | 从后往前找长度大于 4 的非空 path segment 并返回。 |

由于 JavLibrary 详情页核心标识是 `v` 参数，按 URL 直抓时如果 URL 带 `v`，`ScrapeURL()` 会保留原 URL，并尝试从路径首段识别语言；否则按配置语言构造 `/{language}/?v={id}`。

## 6. HTTP、限流与挑战页处理

所有页面请求都经过 `fetchPageCtx(ctx, targetURL)`。

该函数负责：

1. 调用 `rateLimiter.Wait(ctx)` 等待请求额度。
2. 使用 Resty 绑定上下文并发起直接 `GET`。
3. 如果直接请求返回 `200` 且不是 Cloudflare challenge 页面，立即返回 HTML。
4. 如果直接请求返回 `200` 但内容是 Cloudflare challenge，则尝试升级到 FlareSolverr。
5. 如果直接请求为非 `200`，记录状态并在后续按状态码返回 typed status error。
6. 如果 FlareSolverr 可用，调用 `ResolveURL(targetURL)`。
7. FlareSolverr 成功后，如果内容仍是 challenge，则返回 `models.NewScraperChallengeError`。
8. FlareSolverr 成功且返回 cookies 时，用 `cookieMu` 保护并写入共享 Resty client，供后续请求复用。
9. FlareSolverr 失败时，回退到直接请求结果继续判断。

错误分类：

| 场景 | 返回 |
| --- | --- |
| rate limiter 被 context 取消 | 普通 error，消息包含 `rate limit wait failed`。 |
| 直接请求网络错误且 FlareSolverr 不可用或失败 | 原始网络 error。 |
| HTTP 状态码非 `200` | `models.NewScraperStatusError("JavLibrary", statusCode, message)`。 |
| 直接请求或 FlareSolverr 响应仍为 Cloudflare challenge | `models.NewScraperChallengeError("JavLibrary", message)`。 |

这个实现是“直接请求优先，遇到挑战再升级”的策略；开启 `use_flaresolverr` 并不意味着每个请求都会先走 FlareSolverr。

## 7. 详情页页面分析

`parseDetailPage(html, id, sourceURL, language)` 是详情页字段提取中心。它会先构造基础结果，再同时使用 regexp 和 goquery 读取字段。

当前实现假设 JavLibrary 详情页具有以下结构：

| 页面区域 | 选择器/结构 | 用途 |
| --- | --- | --- |
| 页面标题 | `<title>IPX-123 Title - JAVLibrary</title>` | 影片标题。 |
| 标题块 | `#video_title h3 a` | 详情页正文标题；真实中英文详情页均存在，可作为比 `<title>` 更局部的标题来源。 |
| 详情页标记 | `id="video_info"` | 判断搜索结果是否直接落到详情页，以及直链页是否有效。 |
| 封面图 | `#video_jacket_img[src]` | 首选封面 URL；打开的详情页中 `#video_jacket` 是 `div`，没有 `href`。 |
| 基础信息 | `#video_info > div.item`，例如 `#video_id`、`#video_date`、`#video_length` | ID、发售日、时长等文本字段。 |
| 链接字段 | `#video_director a`、`#video_maker a`、`#video_label a`、`#video_series a` | 导演、制作商、发行商、系列。 |
| 类型 | `#video_genres .genre a` | `Genres`，限定在 `#video_genres` 内可避免误收页面其他链接。 |
| 演员 | `#video_cast .star a` | `Actresses`，限定在 `#video_cast` 内可避免收藏/评论区干扰。 |
| 简介 | `meta[property=og:description]`、`meta[name=description]` | 详情页可见样本没有 `meta[name=description]`，但有 `og:description`；`#video_review` 实际是评分控件，不应当作简介主来源。 |
| 评分 | JS 变量 `$rating`、`#video_review` 中显示的 `(8.30)` | `Rating.Score`；真实页面存在 `$rating = "8"`，展示文本是 `User Rating: ... (8.30)`。 |
| 样例图 | `#video_images` 或明确的样例图链接 | 打开的详情页中 `#video_images` 为空；不要把封面、编辑图标或评论区图片当样例图。 |
| 预告片 | 明确的 `video/source` 或 sample movie 元素 | 打开的详情页没有官方预告片；评论区/跳转链接里的 `.mp4` 不应作为 trailer。 |

## 8. 字段提取规则

### 8.1 基础字段

`parseDetailPage()` 初始化的字段：

| 字段 | 来源 |
| --- | --- |
| `Source` | 固定为 `javlibrary`。 |
| `SourceURL` | 当前详情页 URL 或直抓 URL。 |
| `Language` | 搜索配置语言，或直链路径中的语言段。 |
| `ID` | 搜索传入 ID，或从 URL 提取出的 `v`/`keyword`/路径段。 |

### 8.2 标题

`extractTitle(html, id)` 只从 `<title>` 中提取标题。

真实详情页还提供 `#video_title h3 a`，正文标题不带 ` - JAVLibrary` 后缀，也不包含浏览器标题里的站点名。若后续优化实现，建议优先读取 `#video_title h3 a`，再回退到当前 `<title>` 规则。

规则：

1. 匹配 `<title>(...)</title>`。
2. 去掉尾部 ` - JAVLibrary`。
3. 去掉开头的 `{id} ` 前缀。
4. 返回 trim 后的文本。

例如：

```html
<title>IPX-123 Sample Title - JAVLibrary</title>
```

解析结果为 `Sample Title`。

### 8.3 封面与海报

`extractCoverURL(html)` 的优先级：

1. `id="video_jacket_img"` 元素的 `src`。
2. `id="video_jacket"` 元素的 `href`。

如果 fallback URL 是协议相对地址，例如 `//images.example.com/ipx123pl.jpg`，会补成 `https://...`。

拿到封面后，`parseDetailPage()` 会继续：

1. 调用 `imageutil.NormalizeDMMScreenshotURL()` 归一化 DMM 图片 URL。
2. 调用 `imageutil.UpgradeCoverResolution()` 将常见低清后缀升级到更高分辨率，例如 `ps.jpg` -> `pl.jpg`。
3. 调用 `imageutil.GetOptimalPosterURL()` 尝试根据封面推导更合适的 poster URL。
4. 根据返回值设置 `PosterURL` 和 `ShouldCropPoster`。

### 8.4 发售日

`extractReleaseDate(html)` 的优先级：

| 优先级 | 模式 | 说明 |
| --- | --- | --- |
| 1 | `id="video_date"...class="text">YYYY-MM-DD<` | JavLibrary 标准结构。 |
| 2 | `Release Date: ... YYYY-MM-DD` | 文本兜底。 |

解析使用 `time.Parse("2006-01-02", value)`，失败时返回 `nil`。

### 8.5 时长

`extractRuntime(html)` 的优先级：

| 优先级 | 模式 | 说明 |
| --- | --- | --- |
| 1 | `id="video_length"...class="text">120<` | JavLibrary 标准结构。 |
| 2 | `(Length|Duration): ... 120 min` | 文本兜底。 |

成功时返回分钟数，失败时返回 `0`。

### 8.6 导演、制作商、发行商

`extractField(html, divID)` 是通用函数，当前用于：

| 结果字段 | `divID` |
| --- | --- |
| `Director` | `video_director` |
| `Maker` | `video_maker` |
| `Label` | `video_label` |

规则是匹配目标 `div` 内第一个 anchor 文本：

```html
<div id="video_maker">
  <a href="/maker/test">Maker Test</a>
</div>
```

解析结果为 `Maker Test`。

### 8.7 系列

`extractSeries(html)` 的优先级：

1. `id="video_series"` 内第一个 anchor 文本。
2. 任意 `Series:` 附近的第一个 anchor 文本。

没有匹配时返回空字符串。

### 8.8 类型

`extractGenres(html)` 当前匹配所有 `.genre a`。真实详情页的影片类型位于 `#video_genres`：

```html
<div id="video_genres" class="item">
  <span class="genre"><a href="vl_genre.php?g=ky">Creampie</a></span>
  <span class="genre"><a href="vl_genre.php?g=lq">Solowork</a></span>
</div>
```

规则：

- 优先读取 `#video_genres .genre a` 文本。
- 当前实现的全局 `.genre a` 在打开的详情页上也能得到正确结果，但限定到 `#video_genres` 更稳。
- 去除首尾空白。
- 用 `seen` map 去重。
- 保持页面出现顺序。

### 8.9 演员

`extractActresses(html)` 当前匹配所有 `.star a`。真实详情页的影片演员位于 `#video_cast`：

```html
<div id="video_cast" class="item">
  <span class="star"><a href="vl_star.php?s=aejdw">Aine Maria</a></span>
</div>
```

规则：

- 优先读取 `#video_cast .star a` 文本。
- 当前实现的全局 `.star a` 在打开的详情页上也能得到正确结果，但限定到 `#video_cast` 可以避开非影片演员区域。
- 去重并保持页面顺序。
- 使用 `strings.Fields()` 按空白拆分名称。
- 第一个 token 写入 `ActressInfo.FirstName`。
- 剩余 token 用空格拼回 `ActressInfo.LastName`。

例如 `Jane Mary Doe` 会解析为：

```json
{
  "first_name": "Jane",
  "last_name": "Mary Doe"
}
```

当前实现没有从 JavLibrary 演员页补充 `JapaneseName`、`ThumbURL` 或 `DMMID`。

### 8.10 简介

`extractDescription(html)` 的优先级：

| 优先级 | 模式 | 过滤 |
| --- | --- | --- |
| 1 | `<meta name="description" content="...">` | 当前实现的主路径；打开的中英文详情页未提供这个标签。 |
| 2 | `#video_review` 内 `class="text"` 的 table cell | 当前实现的 fallback，但真实页面这里是评分控件，通常会被 `star-rating-control` 过滤。 |
| 3 | `#video_review` 整体文本 | 当前实现的最后 fallback，同样需要过滤评分控件。 |

页面确认后，简介更合理的优化顺序是：

1. 优先读取 `meta[property="og:description"]`，打开的中英文详情页均存在。
2. 再读取 `meta[name="description"]`。
3. 最后才考虑真正的 review 文本；不要把 `#video_review` 的评分控件当剧情简介。

JavLibrary 详情页通常没有稳定剧情简介，因此该字段更多是 meta description 或 review 文本兜底。

### 8.11 评分

`extractRating(html, doc)` 的优先级：

1. 正则匹配 JS 变量 `$rating = "7"` 或 `$rating = "8.5"`。
2. 如果没有 JS 变量，则读取 `#video_rating span.num` 文本。

打开的中英文详情页均有 `$rating = "8"`，同时 `#video_review` 展示 `(8.30)`。当前实现会选择 JS 变量，因此得到的是整数评分；如果希望更贴近页面显示分数，后续可增加从 `#video_review .score` 或括号文本解析小数的 fallback。

成功时返回：

```go
&models.Rating{Score: score}
```

当前不解析投票数，因此 `Votes` 保持零值。

### 8.12 截图

`extractScreenshotURLs(html)` 会从多个来源收集样例图 URL，并用 `seen` map 去重。

打开的 JavLibrary 详情页里 `#video_images` 为空，没有可抓取样例图。为了避免误收，规则应当优先限定在明确 gallery/sample 区域；宽泛扫描整页 `.jpg` 时必须过滤封面、站点图标、编辑图标、广告和评论区内容。

收集来源按顺序包括：

| 来源 | 示例 |
| --- | --- |
| lazy loading | `data-src="...jpg"` |
| sample 图片 | `src="...sample...jpg"` |
| DMM/JavLibrary 样例图 | `src="...jp-1.jpg"`、`src="...jp-2.jpg"` |
| pic 序列 | `src="...pic01.jpg"` |
| `c.impact.jp` 数字图 | `src=".../04.jpg"`、`src=".../001.jpg"` |
| gallery href | `#video_gallery ... href="...jpg"` |
| href fallback | `href="...sample...jpg"`、`href="...jp-1.jpg"`、`href="...impact.jp...jpg"`、`href=".../04.jpg"` |

过滤规则：

- 跳过空 URL 和重复 URL。
- 跳过 `redirect.php`、`redirect%`。
- 跳过 `loading`、`blank`、`placeholder`、`icon`、`head2.jpg`。
- 跳过 DMM 封面/海报后缀 `pl.jpg`、`ps.jpg`，避免把封面混入截图。

`parseDetailPage()` 拿到截图后会：

1. 对每个 URL 调用 `imageutil.NormalizeDMMScreenshotURL()`。
2. 如果截图列表中存在和 `CoverURL` 完全相同的 URL，则移除。
3. 如果截图等于 `CoverURL` 的 `pl.jpg` -> `ps.jpg` 变体，也移除。

### 8.13 预告片

`extractTrailerURL(html)` 的优先级：

| 优先级 | 模式 |
| --- | --- |
| 1 | 任意 `src="...sample...mp4"`。 |
| 2 | `<video src="...">` 且标签内容附近有 `sample`。 |
| 3 | 任意 `href="...sample_movie...mp4"`。 |
| 4 | 任意 `href="...mp4"`。 |

打开的详情页没有官方 trailer；页面评论区可能出现网盘跳转链接或 `redirect.php?...mp4`。这些不是预告片，优化规则应过滤 `redirect.php`、评论区和外部下载链接，只接受 `video/source`、明确 sample movie 容器或站点定义的预告片字段。

没有匹配时返回空字符串。

## 9. 媒体 URL 与代理处理

JavLibrary 页面上的图片可能来自 JavLibrary 自身、`c.impact.jp`，也可能是 DMM 图片 CDN。

实现中的媒体处理分两层：

| 层级 | 处理 |
| --- | --- |
| 字段解析层 | 从 HTML 中提取原始 cover、screenshot、trailer URL。 |
| 图片规范化层 | 对 DMM 相关 URL 做协议、host、后缀和分辨率处理。 |

下载代理解析由 `ResolveDownloadProxyForHost(host)` 完成，只认定 JavLibrary 自有 host 和 `c.impact.jp`。DMM 图片 host 的处理依赖其他公共下载逻辑，不在 JavLibrary scraper 内声明为自有 host。

## 10. 错误语义

JavLibrary scraper 中常见错误路径如下：

| 场景 | 位置 | 结果 |
| --- | --- | --- |
| scraper 未启用 | `Search()` | 普通 error：`JavLibrary scraper is disabled`。 |
| URL 不属于 JavLibrary | `ScrapeURL()` | `models.NewScraperNotFoundError("JavLibrary", "URL not handled by JavLibrary scraper")`。 |
| URL 无法提取 ID | `ScrapeURL()` | 包装后的普通 error。 |
| 搜索页没有详情链接 | `Search()` | `models.NewScraperNotFoundError("JavLibrary", "movie ... not found on JavLibrary")`。 |
| 直链页没有 `video_info` | `ScrapeURL()` | `models.NewScraperNotFoundError("JavLibrary", "page does not contain video info")`。 |
| HTTP 非 200 | `fetchPageCtx()` | `models.NewScraperStatusError("JavLibrary", statusCode, message)`。 |
| Cloudflare challenge | `fetchPageCtx()` | `models.NewScraperChallengeError("JavLibrary", message)`。 |
| HTML 无法被 goquery 解析 | `parseDetailPage()` | 包装后的普通 error。 |

## 11. 测试覆盖

当前测试重点：

| 测试文件 | 覆盖内容 |
| --- | --- |
| `config_test.go` | flatten 配置、cookies/proxy 透传、语言/base_url/通用配置校验。 |
| `javlibrary_test.go` | URL 识别、ID 提取、接口实现、默认语言、`GetURL()`、禁用状态和集成测试入口。 |
| `javlibrary_parser_test.go` | 搜索结果列表解析、旧版 href 解析、详情页解析、描述/系列/评分/截图/预告片提取、下载代理 host 识别。 |

集成测试 `TestIntegration_Search` 默认跳过，需要：

```bash
JAVINIZER_RUN_FLARESOLVERR_TESTS=1 go test -v -timeout 120s ./internal/scraper/javlibrary/... -run TestIntegration_Search
```

这类测试依赖外部 JavLibrary 页面和 FlareSolverr 可用性，不适合普通 CI 稳定运行。

## 12. 维护要点

- JavLibrary 页面结构不稳定时，优先扩展 `extractMovieURLFromHTML()` 和各字段 extractor 的兜底模式，并补充 parser 单元测试。
- 新增字段提取时，应保持 `parseDetailPage()` 只编排提取流程，把具体规则放在独立 `extract*` 函数中。
- 修改截图规则时要同时检查封面过滤，避免 `pl.jpg`、`ps.jpg` 被写入 `ScreenshotURL`。
- 修改 FlareSolverr 行为时要保留“直接请求成功则不升级”的语义，避免无谓增加浏览器挑战绕过成本。
- 修改 cookie 写入逻辑时要保留 `cookieMu`，因为 scraper 实例可能被并发调用。
- 新增媒体 host 时，同步评估 `ResolveDownloadProxyForHost()`，否则下载器可能不会按 scraper 的代理配置处理这些资源。
