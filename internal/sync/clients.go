package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Константы источников данных (значения source в метаданных, согласно ТЗ).
const (
	SourceGitLab     = "gitlab"
	SourceEvaProject = "evaproject"
)

// httpClient общий HTTP-клиент для источников данных.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// ---------------------------------------------------------------------------
// EvaProject
// ---------------------------------------------------------------------------

// EvaClient — клиент API EvaProject для получения задач.
type EvaClient struct {
	baseURL string
	token   string
}

// NewEvaClient создаёт клиент EvaProject.
func NewEvaClient(baseURL, token string) *EvaClient {
	return &EvaClient{baseURL: strings.TrimRight(baseURL, "/"), token: token}
}

// Enabled возвращает true, если клиент настроен (указан URL).
func (c *EvaClient) Enabled() bool {
	return c.baseURL != ""
}

// evaTask представляет задачу из EvaProject.
type evaTask struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	// UpdatedAt принимает различные форматы дат; храним как строку
	UpdatedAt string `json:"updated_at"`
}

// FetchAll возвращает все задачи EvaProject.
func (c *EvaClient) FetchAll(ctx context.Context) ([]ContentItem, error) {
	return c.fetch(ctx, "")
}

// FetchSince возвращает задачи, обновлённые после указанного времени.
func (c *EvaClient) FetchSince(ctx context.Context, since time.Time) ([]ContentItem, error) {
	return c.fetch(ctx, since.Format(time.RFC3339))
}

// fetch выполняет запрос к подходящему эндпоинту EvaProject.
func (c *EvaClient) fetch(ctx context.Context, since string) ([]ContentItem, error) {
	if !c.Enabled() {
		return nil, nil
	}

	var endpoint string
	if since != "" {
		endpoint = fmt.Sprintf("%s/api/tasks?updated_after=%s", c.baseURL, url.QueryEscape(since))
	} else {
		endpoint = c.baseURL + "/api/tasks"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("EvaProject: ошибка создания запроса: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("EvaProject: ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("EvaProject: статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("EvaProject: ошибка чтения ответа: %w", err)
	}

	var tasks []evaTask
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("EvaProject: ошибка разбора ответа: %w", err)
	}

	items := make([]ContentItem, 0, len(tasks))
	for _, t := range tasks {
		text := t.Title
		if t.Content != "" {
			text = t.Title + "\n\n" + t.Content
		}
		items = append(items, ContentItem{
			Source:    SourceEvaProject,
			Title:     t.Title,
			URL:       fmt.Sprintf("%s/tasks/%d", c.baseURL, t.ID),
			Text:      text,
			UpdatedAt: t.UpdatedAt,
		})
	}

	return items, nil
}

// ---------------------------------------------------------------------------
// GitLab
// ---------------------------------------------------------------------------

// GitLabClient — клиент GitLab API для получения проектов и Wiki.
type GitLabClient struct {
	baseURL string
	token   string
}

// NewGitLabClient создаёт клиент GitLab.
func NewGitLabClient(baseURL, token string) *GitLabClient {
	return &GitLabClient{baseURL: strings.TrimRight(baseURL, "/"), token: token}
}

// Enabled возвращает true, если клиент настроен (указан URL).
func (c *GitLabClient) Enabled() bool {
	return c.baseURL != ""
}

// gitlabProject — проект GitLab.
type gitlabProject struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	WebURL         string `json:"web_url"`
	Description    string `json:"description"`
	LastActivityAt string `json:"last_activity_at"`
}

// gitlabWikiPage — страница Wiki.
type gitlabWikiPage struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Content     string `json:"content"`
	WebURL      string `json:"web_url"`
	LastUpdated string `json:"last_updated_at"`
}

// FetchAll возвращает все проекты и их Wiki.
func (c *GitLabClient) FetchAll(ctx context.Context) ([]ContentItem, error) {
	return c.fetch(ctx, "")
}

// FetchSince возвращает проекты, обновлённые после указанного времени.
func (c *GitLabClient) FetchSince(ctx context.Context, since time.Time) ([]ContentItem, error) {
	return c.fetch(ctx, since.Format(time.RFC3339))
}

// fetch запрашивает проекты GitLab, доступные пользователю, и их Wiki.
func (c *GitLabClient) fetch(ctx context.Context, since string) ([]ContentItem, error) {
	if !c.Enabled() {
		return nil, nil
	}

	var endpoint string
	if since != "" {
		endpoint = fmt.Sprintf("%s/api/v4/projects?membership=true&last_activity_after=%s&per_page=100",
			c.baseURL, url.QueryEscape(since))
	} else {
		endpoint = fmt.Sprintf("%s/api/v4/projects?membership=true&simple=true&per_page=100", c.baseURL)
	}

	projects, err := c.fetchProjects(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var items []ContentItem

	for _, p := range projects {
		text := strings.TrimSpace(p.Description)
		if text == "" {
			// если описания нет, хотя бы имя проекта как содержимое
			text = p.Name
		} else {
			text = p.Name + "\n\n" + text
		}

		items = append(items, ContentItem{
			Source:    SourceGitLab,
			Title:     p.Name,
			URL:       p.WebURL,
			Text:      text,
			UpdatedAt: p.LastActivityAt,
		})

		// Wiki проекта (если доступно)
		wikiItems, err := c.fetchWiki(ctx, p.ID)
		if err != nil {
			continue // wiki может быть недоступен — пропускаем, не фатально
		}
		items = append(items, wikiItems...)
	}

	return items, nil
}

// fetchProjects выполняет GET-запрос и разбирает список проектов.
func (c *GitLabClient) fetchProjects(ctx context.Context, endpoint string) ([]gitlabProject, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("GitLab: ошибка создания запроса: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitLab: ошибка запроса проектов: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitLab: статус проектов %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GitLab: ошибка чтения проектов: %w", err)
	}

	var projects []gitlabProject
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("GitLab: ошибка разбора проектов: %w", err)
	}
	return projects, nil
}

// fetchWiki получает страницы Wiki проекта.
func (c *GitLabClient) fetchWiki(ctx context.Context, projectID int) ([]ContentItem, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%d/wikis", c.baseURL, projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitLab: wiki статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pages []struct {
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		WebURL      string `json:"web_url"`
		LastUpdated string `json:"last_updated_at"`
	}
	if err := json.Unmarshal(body, &pages); err != nil {
		return nil, fmt.Errorf("GitLab: ошибка разбора wiki: %w", err)
	}

	result := make([]ContentItem, 0, len(pages))
	for _, page := range pages {
		content, err := c.fetchWikiContent(ctx, projectID, page.Slug)
		if err != nil {
			continue
		}
		result = append(result, ContentItem{
			Source:    SourceGitLab,
			Title:     page.Title,
			URL:       page.WebURL,
			Text:      content,
			UpdatedAt: page.LastUpdated,
		})
	}
	return result, nil
}

// fetchWikiContent получает полное содержимое страницы Wiki.
func (c *GitLabClient) fetchWikiContent(ctx context.Context, projectID int, slug string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%d/wikis/%s?render=false",
		c.baseURL, projectID, url.PathEscape(slug))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitLab: wiki content статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var page struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return "", fmt.Errorf("GitLab: ошибка разбора wiki content: %w", err)
	}
	return page.Content, nil
}
