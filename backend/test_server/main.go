package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TestArticle represents a test article for the RSS feed
type TestArticle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	GUID        string    `json:"guid"`
	PublishedAt time.Time `json:"publishedAt"`
}

var (
	predefinedArticles = []TestArticle{
		{
			ID:          "1",
			Title:       "技术架构更新公告",
			Description: "本次更新包含多项性能优化和新功能，提升系统整体稳定性",
			Content:     "详细的技术更新内容，包括数据库优化、缓存策略调整、前端渲染改进等多个方面的优化。",
			GUID:        "predefined-1",
			PublishedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:          "2",
			Title:       "新品发布预告",
			Description: "即将发布全新产品线，敬请期待",
			Content:     "我们很高兴宣布新产品即将面世。这次发布将带来革命性的用户体验和强大的功能特性。",
			GUID:        "predefined-2",
			PublishedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			ID:          "3",
			Title:       "社区活动总结",
			Description: "上周末的开发者大会圆满结束",
			Content:     "感谢所有参与者的支持本次活动吸引了超过500名开发者参与，涵盖了前端、后端、云原生等多个主题演讲。",
			GUID:        "predefined-3",
			PublishedAt: time.Now().Add(-72 * time.Hour),
		},
		{
			ID:          "4",
			Title:       "安全漏洞修复通知",
			Description: "已修复近期发现的安全问题",
			Content:     "本次安全更新修复了可能导致数据泄露的漏洞，建议所有用户尽快更新到最新版本。",
			GUID:        "predefined-4",
			PublishedAt: time.Now().Add(-96 * time.Hour),
		},
		{
			ID:          "5",
			Title:       "测试文章：正常标题",
			Description: "这是一篇用于测试的正常文章，包含普通文本内容",
			Content:     "这是测试文章的正文内容，用于验证RSS阅读器的各项功能是否正常工作。",
			GUID:        "predefined-5",
			PublishedAt: time.Now().Add(-120 * time.Hour),
		},
	}

	dynamicArticles = struct {
		sync.RWMutex
		articles []TestArticle
	}{}

	titleTemplates = []string{
		"最新资讯：%s 发布",
		"专题报道：%s 深度解析",
		"%s 专题",
		"关于 %s 的最新消息",
		"%s 相关内容汇总",
	}

	keywords = []string{
		"Go语言", "Kubernetes", "Docker", "微服务", "云原生",
		"机器学习", "人工智能", "深度学习", "数据分析", "大数据",
		"区块链", "Web3", "去中心化", "智能合约", "NFT",
		"前端开发", "React", "Vue", "TypeScript", "JavaScript",
		"后端开发", "REST API", "GraphQL", "gRPC", "分布式系统",
	}

	contentTemplates = []string{
		"本文深入探讨了 %s 的核心技术原理和最佳实践，适合开发者学习参考。",
		"%s 是当前技术领域的热门话题，本文将带你全面了解其发展趋势。",
		"关于 %s 的最新研究显示，该技术正在快速普及并改变行业格局。",
		"本文介绍如何使用 %s 构建高效、可靠的系统架构。",
		"%s 相关的开源项目持续增长，社区活跃度不断提升。",
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateRandomArticle(index int) TestArticle {
	titleTpl := titleTemplates[rand.Intn(len(titleTemplates))]
	keyword := keywords[rand.Intn(len(keywords))]
	title := fmt.Sprintf(titleTpl, keyword)

	contentTpl := contentTemplates[rand.Intn(len(contentTemplates))]
	content := fmt.Sprintf(contentTpl, keyword)

	desc := content
	if len(desc) > 50 {
		desc = desc[:50] + "..."
	}

	pubDate := time.Now().Add(-time.Duration(rand.Intn(168)) * time.Hour)

	return TestArticle{
		ID:          fmt.Sprintf("random-%d", index),
		Title:       title,
		Link:        fmt.Sprintf("http://example.com/random/%d", rand.Intn(10000)),
		Description: desc,
		Content:     content,
		GUID:        fmt.Sprintf("random-guid-%d-%d", time.Now().UnixNano(), index),
		PublishedAt: pubDate,
	}
}

func generateRSS(items []TestArticle, title, link, description string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">` + "\n")
	sb.WriteString(`  <channel>` + "\n")
	sb.WriteString(fmt.Sprintf(`    <title>%s</title>`+"\n", title))
	sb.WriteString(fmt.Sprintf(`    <link>%s</link>`+"\n", link))
	sb.WriteString(fmt.Sprintf(`    <description>%s</description>`+"\n", description))
	sb.WriteString(fmt.Sprintf(`    <atom:link href="%s" rel="self" type="application/rss+xml"/>`+"\n", link))
	sb.WriteString(fmt.Sprintf(`    <language>zh-cn</language>`+"\n"))
	sb.WriteString(fmt.Sprintf(`    <lastBuildDate>%s</lastBuildDate>`+"\n", time.Now().Format(time.RFC1123Z)))

	for _, item := range items {
		sb.WriteString(`    <item>` + "\n")
		sb.WriteString(fmt.Sprintf(`      <title><![CDATA[%s]]></title>`+"\n", item.Title))
		sb.WriteString(fmt.Sprintf(`      <link>%s</link>`+"\n", item.Link))
		sb.WriteString(fmt.Sprintf(`      <description><![CDATA[%s]]></description>`+"\n", item.Description))
		sb.WriteString(fmt.Sprintf(`      <content:encoded><![CDATA[%s]]></content:encoded>`+"\n", item.Content))
		sb.WriteString(fmt.Sprintf(`      <guid isPermaLink="false">%s</guid>`+"\n", item.GUID))
		sb.WriteString(fmt.Sprintf(`      <pubDate>%s</pubDate>`+"\n", item.PublishedAt.Format(time.RFC1123Z)))
		sb.WriteString(`    </item>` + "\n")
	}

	sb.WriteString(`  </channel>` + "\n")
	sb.WriteString(`</rss>`)

	return sb.String()
}

func getHTMLForm() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>测试 RSS 服务器 - 文章发布</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 20px; text-align: center; }
        .card { background: white; border-radius: 8px; padding: 24px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h2 { color: #555; margin-bottom: 16px; font-size: 18px; border-bottom: 1px solid #eee; padding-bottom: 8px; }
        .form-group { margin-bottom: 16px; }
        label { display: block; margin-bottom: 6px; color: #666; font-weight: 500; }
        input[type="text"], textarea { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
        textarea { height: 120px; resize: vertical; font-family: inherit; }
        button { background: #4CAF50; color: white; border: none; padding: 12px 24px; border-radius: 4px; cursor: pointer; font-size: 16px; }
        button:hover { background: #45a049; }
        .btn-delete { background: #f44336; padding: 8px 16px; font-size: 14px; }
        .btn-delete:hover { background: #da190b; }
        .btn-reset { background: #ff9800; padding: 8px 16px; font-size: 14px; }
        .btn-reset:hover { background: #e68a00; }
        .btn-group { display: flex; gap: 10px; margin-top: 16px; }
        .article-list { margin-top: 16px; }
        .article-item { background: #f9f9f9; padding: 12px; border-radius: 4px; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: start; }
        .article-info h3 { font-size: 14px; color: #333; margin-bottom: 4px; }
        .article-info p { font-size: 12px; color: #666; }
        .article-meta { font-size: 11px; color: #999; margin-top: 4px; }
        .success { color: #4CAF50; padding: 10px; background: #e8f5e9; border-radius: 4px; margin-bottom: 16px; display: none; }
        .error { color: #f44336; padding: 10px; background: #ffebee; border-radius: 4px; margin-bottom: 16px; display: none; }
        .feeds-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
        .feed-item { background: #f0f0f0; padding: 10px; border-radius: 4px; font-size: 13px; }
        .feed-item a { color: #1976D2; text-decoration: none; }
        .feed-item a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>测试 RSS 服务器</h1>

        <div class="card">
            <h2>可用 Feeds</h2>
            <div class="feeds-list">
                <div class="feed-item"><a href="/feeds/simple">/feeds/simple</a> - 5篇固定文章</div>
                <div class="feed-item"><a href="/feeds/random">/feeds/random</a> - 5-20篇随机文章</div>
                <div class="feed-item"><a href="/feeds/empty">/feeds/empty</a> - 空列表</div>
                <div class="feed-item"><a href="/feeds/single">/feeds/single</a> - 单篇文章</div>
                <div class="feed-item"><a href="/feeds/custom?count=10">/feeds/custom?count=10</a> - 自定义数量</div>
            </div>
        </div>

        <div class="card">
            <h2>发布测试文章</h2>
            <div id="success-msg" class="success"></div>
            <div id="error-msg" class="error"></div>
            <form id="article-form">
                <div class="form-group">
                    <label>标题</label>
                    <input type="text" name="title" required placeholder="输入文章标题">
                </div>
                <div class="form-group">
                    <label>描述</label>
                    <textarea name="description" placeholder="简短描述（可选）"></textarea>
                </div>
                <div class="form-group">
                    <label>正文内容</label>
                    <textarea name="content" required placeholder="输入文章正文内容，用于测试AI摘要功能" style="height: 200px;"></textarea>
                </div>
                <button type="submit">发布文章</button>
            </form>
        </div>

        <div class="card">
            <h2>已发布的文章</h2>
            <div class="btn-group">
                <button type="button" onclick="loadArticles()" class="btn-reset">刷新列表</button>
                <button type="button" onclick="resetArticles()" class="btn-reset">清空所有</button>
            </div>
            <div id="article-list" class="article-list"></div>
        </div>
    </div>

    <script>
        async function loadArticles() {
            var res = await fetch('/feeds/articles');
            var articles = await res.json();
            var list = document.getElementById('article-list');
            if (articles.length === 0) {
                list.innerHTML = '<p style="color:#999;text-align:center;padding:20px;">暂无已发布的文章</p>';
                return;
            }
            var html = '';
            for (var i = 0; i < articles.length; i++) {
                var a = articles[i];
                html += '<div class="article-item">' +
                    '<div class="article-info">' +
                    '<h3>' + escapeHtml(a.title) + '</h3>' +
                    '<p>' + escapeHtml(a.description || '无描述') + '</p>' +
                    '<div class="article-meta">ID: ' + a.id + ' | GUID: ' + a.guid + '</div>' +
                    '</div>' +
                    '<button onclick="deleteArticle(\'' + a.id + '\')" class="btn-delete">删除</button>' +
                    '</div>';
            }
            list.innerHTML = html;
        }

        function escapeHtml(text) {
            var div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        async function deleteArticle(id) {
            await fetch('/feeds/articles/' + id, { method: 'DELETE' });
            loadArticles();
        }

        async function resetArticles() {
            if (!confirm('确定要清空所有已发布的文章吗？')) return;
            await fetch('/feeds/reset', { method: 'POST' });
            loadArticles();
        }

        document.getElementById('article-form').onsubmit = async function(e) {
            e.preventDefault();
            var form = new FormData(e.target);
            var data = {
                title: form.get('title'),
                description: form.get('description'),
                content: form.get('content')
            };
            try {
                var res = await fetch('/feeds/articles', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (res.ok) {
                    var article = await res.json();
                    document.getElementById('success-msg').textContent = '发布成功！ID: ' + article.id;
                    document.getElementById('success-msg').style.display = 'block';
                    document.getElementById('error-msg').style.display = 'none';
                    e.target.reset();
                    loadArticles();
                } else {
                    throw new Error(await res.text());
                }
            } catch (err) {
                document.getElementById('error-msg').textContent = '发布失败: ' + err.message;
                document.getElementById('error-msg').style.display = 'block';
                document.getElementById('success-msg').style.display = 'none';
            }
        };

        loadArticles();
    </script>
</body>
</html>`
}

func setupRoutes(r *gin.Engine) {
	feeds := r.Group("/feeds")
	{
		feeds.GET("/simple", func(c *gin.Context) {
			items := make([]TestArticle, len(predefinedArticles))
			copy(items, predefinedArticles)
			rss := generateRSS(items, "Test RSS Feed - Simple",
				"http://localhost:8095/feeds/simple",
				"A simple test feed for development")
			c.Header("Content-Type", "application/rss+xml; charset=utf-8")
			c.String(200, rss)
		})

		feeds.GET("/empty", func(c *gin.Context) {
			rss := generateRSS([]TestArticle{}, "Test RSS Feed - Empty",
				"http://localhost:8095/feeds/empty",
				"An empty test feed with no articles")
			c.Header("Content-Type", "application/rss+xml; charset=utf-8")
			c.String(200, rss)
		})

		feeds.GET("/single", func(c *gin.Context) {
			rss := generateRSS([]TestArticle{predefinedArticles[0]}, "Test RSS Feed - Single",
				"http://localhost:8095/feeds/single",
				"A test feed with a single article")
			c.Header("Content-Type", "application/rss+xml; charset=utf-8")
			c.String(200, rss)
		})

		feeds.GET("/random", func(c *gin.Context) {
			count := 5 + rand.Intn(16)
			items := make([]TestArticle, count)
			for i := 0; i < count; i++ {
				items[i] = generateRandomArticle(i)
			}
			rss := generateRSS(items, "Test RSS Feed - Random",
				"http://localhost:8095/feeds/random",
				fmt.Sprintf("A test feed with %d random articles", count))
			c.Header("Content-Type", "application/rss+xml; charset=utf-8")
			c.String(200, rss)
		})

		feeds.GET("/custom", func(c *gin.Context) {
			count := 5
			if ct := c.Query("count"); ct != "" {
				fmt.Sscanf(ct, "%d", &count)
			}
			if count <= 0 {
				count = 0
			}
			if count > 1000 {
				count = 1000
			}

			dynamicArticles.RLock()
			allArticles := make([]TestArticle, 0, len(predefinedArticles)+len(dynamicArticles.articles))
			allArticles = append(allArticles, predefinedArticles...)
			allArticles = append(allArticles, dynamicArticles.articles...)
			dynamicArticles.RUnlock()

			items := make([]TestArticle, count)
			for i := 0; i < count; i++ {
				if i < len(allArticles) {
					items[i] = allArticles[i]
				} else {
					items[i] = generateRandomArticle(i)
				}
			}
			rss := generateRSS(items, "Test RSS Feed - Custom",
				"http://localhost:8095/feeds/custom",
				fmt.Sprintf("A test feed with %d articles", count))
			c.Header("Content-Type", "application/rss+xml; charset=utf-8")
			c.String(200, rss)
		})

		feeds.POST("/articles", func(c *gin.Context) {
			var article TestArticle
			if err := c.ShouldBindJSON(&article); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			if article.GUID == "" {
				article.GUID = fmt.Sprintf("dynamic-%d", time.Now().UnixNano())
			}
			if article.ID == "" {
				article.ID = fmt.Sprintf("dynamic-%d", time.Now().UnixNano())
			}
			if article.PublishedAt.IsZero() {
				article.PublishedAt = time.Now()
			}

			dynamicArticles.Lock()
			dynamicArticles.articles = append(dynamicArticles.articles, article)
			dynamicArticles.Unlock()

			c.JSON(201, article)
		})

		feeds.GET("/articles", func(c *gin.Context) {
			dynamicArticles.RLock()
			articles := make([]TestArticle, len(dynamicArticles.articles))
			copy(articles, dynamicArticles.articles)
			dynamicArticles.RUnlock()
			c.JSON(200, articles)
		})

		feeds.DELETE("/articles/:id", func(c *gin.Context) {
			id := c.Param("id")
			dynamicArticles.Lock()
			found := false
			newArticles := make([]TestArticle, 0, len(dynamicArticles.articles))
			for _, a := range dynamicArticles.articles {
				if a.ID == id {
					found = true
				} else {
					newArticles = append(newArticles, a)
				}
			}
			dynamicArticles.articles = newArticles
			dynamicArticles.Unlock()

			if !found {
				c.JSON(404, gin.H{"error": "Article not found"})
				return
			}
			c.JSON(200, gin.H{"status": "ok"})
		})

		feeds.POST("/reset", func(c *gin.Context) {
			dynamicArticles.Lock()
			dynamicArticles.articles = nil
			dynamicArticles.Unlock()
			c.JSON(200, gin.H{"status": "ok", "message": "All dynamic articles reset"})
		})
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, getHTMLForm())
	})
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	setupRoutes(r)

	addr := os.Getenv("TEST_SERVER_ADDR")
	if addr == "" {
		addr = ":8095"
	}

	fmt.Printf("Test RSS Server starting on %s\n", addr)
	fmt.Printf("Available feeds:\n")
	fmt.Printf("  - http://%s/feeds/simple (5 fixed articles)\n", addr)
	fmt.Printf("  - http://%s/feeds/random (5-20 random articles)\n", addr)
	fmt.Printf("  - http://%s/feeds/empty (no articles)\n", addr)
	fmt.Printf("  - http://%s/feeds/single (1 article)\n", addr)
	fmt.Printf("  - http://%s/feeds/custom?count=N (N articles)\n", addr)
	fmt.Printf("  - http://%s/ (Web UI)\n", addr)
	r.Run(addr)
}
